// SPDX-FileCopyrightText: SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package audit

import (
	"bytes"
	"context"
	"io"
	"math/rand"
	"net/http"
	"net/http/httptest"
	"time"

	"github.com/go-logr/logr"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	dto "github.com/prometheus/client_model/go"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apiserver/pkg/apis/audit"

	"github.com/gardener/auditlog-forwarder/internal/helper"
	"github.com/gardener/auditlog-forwarder/internal/output"
	outputfactory "github.com/gardener/auditlog-forwarder/internal/output/factory"
	"github.com/gardener/auditlog-forwarder/internal/processor"
	"github.com/gardener/auditlog-forwarder/internal/processor/annotation"
	configv1alpha1 "github.com/gardener/auditlog-forwarder/pkg/apis/config/v1alpha1"
)

// testProcessor is a lightweight processor used only for chaining regression tests.
type testProcessor struct {
	name      string
	transform func([]byte) []byte
}

func (t *testProcessor) Process(_ context.Context, data []byte) ([]byte, error) {
	return t.transform(data), nil
}

func (t *testProcessor) Name() string { return t.name }

var _ = Describe("Handler", func() {
	var (
		logger      logr.Logger
		annotations map[string]string
		processors  []processor.Processor
		outputInsts []output.Output
		handler     *Handler
		testServer  *httptest.Server
		response    []byte
	)

	BeforeEach(func() {
		logger = logr.Discard()
		annotations = map[string]string{
			"test-key": "test-value",
		}
		processors = []processor.Processor{
			annotation.New(annotations),
		}

		testServer = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			body, err := io.ReadAll(r.Body)
			Expect(err).NotTo(HaveOccurred())
			response = body
			w.WriteHeader(http.StatusOK)
		}))

		outputConfigs := []configv1alpha1.Output{
			{
				HTTP: &configv1alpha1.OutputHTTP{
					URL: testServer.URL,
				},
			},
		}

		var err error
		outputInsts, err = outputfactory.NewFromConfigs(outputConfigs)
		Expect(err).NotTo(HaveOccurred())

		// reinitialize metrics before each test
		auditReceived = promauto.NewCounter(prometheus.CounterOpts{Name: randString(10)})
		auditSucceeded = promauto.NewCounter(prometheus.CounterOpts{Name: randString(10)})
		auditFailed = promauto.NewCounter(prometheus.CounterOpts{Name: randString(10)})
	})

	AfterEach(func() {
		if testServer != nil {
			testServer.Close()
		}
	})

	Describe("NewHandler", func() {
		It("should create a handler with output clients", func() {
			var err error
			handler, err = NewHandler(logger, processors, outputInsts)
			Expect(err).NotTo(HaveOccurred())
			Expect(handler).NotTo(BeNil())
			Expect(handler.outputs).To(HaveLen(1))
			Expect(handler.outputs[0].Name()).To(Equal(testServer.URL))
		})

		It("should return error when no outputs configured", func() {
			var err error
			handler, err = NewHandler(logger, processors, []output.Output{})
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("no outputs configured"))
			Expect(handler).To(BeNil())
		})
	})

	Describe("ServeHTTP", func() {
		BeforeEach(func() {
			var err error
			handler, err = NewHandler(logger, processors, outputInsts)
			Expect(err).NotTo(HaveOccurred())
		})

		It("should chain multiple processors passing transformed data between them", func() {
			p1 := processor.Processor(&testProcessor{
				name:      "p1",
				transform: func(data []byte) []byte { return append(data, []byte("->B")...) },
			})
			p2 := processor.Processor(&testProcessor{
				name: "p2",
				transform: func(data []byte) []byte {
					Expect(string(data)).To(ContainSubstring("A->B"))
					return append(data, []byte("->C")...)
				},
			})

			processorsChained := []processor.Processor{p1, p2}
			var err error
			handler, err = NewHandler(logger, processorsChained, outputInsts)
			Expect(err).NotTo(HaveOccurred())

			initial := []byte("A")
			req := httptest.NewRequest(http.MethodPost, "/audit", bytes.NewReader(initial))
			w := httptest.NewRecorder()
			handler.ServeHTTP(w, req)
			Expect(w.Code).To(Equal(http.StatusOK))

			Eventually(func() bool { return len(response) > 0 }, time.Millisecond*100).Should(BeTrue())
			Expect(string(response)).To(Equal("A->B->C"))

			Expect(getMetricValue(auditReceived)).To(Equal(1.0))
			Expect(getMetricValue(auditSucceeded)).To(Equal(1.0))
			Expect(getMetricValue(auditFailed)).To(Equal(0.0))
		})

		It("should process audit events and forward to outputs", func() {
			eventList := &audit.EventList{
				TypeMeta: metav1.TypeMeta{
					APIVersion: "audit.k8s.io/v1",
					Kind:       "EventList",
				},
				Items: []audit.Event{
					{
						Verb: "create",
						ObjectRef: &audit.ObjectReference{
							Namespace: "test-namespace",
							Name:      "test-pod",
						},
						Annotations: map[string]string{
							"existing": "annotation",
						},
					},
				},
			}

			body, err := helper.EncodeEventList(eventList)
			Expect(err).NotTo(HaveOccurred())

			req := httptest.NewRequest(http.MethodPost, "/audit", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			handler.ServeHTTP(w, req)

			Expect(w.Code).To(Equal(http.StatusOK))

			Eventually(func() bool {
				return len(response) > 0
			}, time.Millisecond*100).Should(BeTrue())

			forwardedEventList, err := helper.DecodeEventList(response)
			Expect(err).NotTo(HaveOccurred())

			Expect(forwardedEventList.Items).To(HaveLen(1))
			event := forwardedEventList.Items[0]
			Expect(event.Annotations).To(HaveKeyWithValue("test-key", "test-value"))
			Expect(event.Annotations).To(HaveKeyWithValue("existing", "annotation"))

			Expect(getMetricValue(auditReceived)).To(Equal(1.0))
			Expect(getMetricValue(auditSucceeded)).To(Equal(1.0))
			Expect(getMetricValue(auditFailed)).To(Equal(0.0))
		})

		It("should return error when output fails", func() {
			// Close the test server to simulate output failure
			testServer.Close()

			eventList := &audit.EventList{
				TypeMeta: metav1.TypeMeta{
					APIVersion: "audit.k8s.io/v1",
					Kind:       "EventList",
				},
				Items: []audit.Event{
					{
						Verb: "create",
					},
				},
			}

			body, err := helper.EncodeEventList(eventList)
			Expect(err).NotTo(HaveOccurred())

			req := httptest.NewRequest(http.MethodPost, "/audit", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			handler.ServeHTTP(w, req)

			Expect(w.Code).To(Equal(http.StatusInternalServerError))
			Expect(w.Body.String()).To(ContainSubstring("failed to forward audit events"))

			Expect(getMetricValue(auditReceived)).To(Equal(1.0))
			Expect(getMetricValue(auditSucceeded)).To(Equal(0.0))
			Expect(getMetricValue(auditFailed)).To(Equal(1.0))
		})

		It("should handle malformed request body", func() {
			req := httptest.NewRequest(http.MethodPost, "/audit", bytes.NewReader([]byte("invalid json")))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			handler.ServeHTTP(w, req)

			Expect(w.Code).To(Equal(http.StatusInternalServerError))
			Expect(w.Body.String()).To(ContainSubstring("failed processing audit events"))

			Expect(getMetricValue(auditReceived)).To(Equal(1.0))
			Expect(getMetricValue(auditSucceeded)).To(Equal(0.0))
			Expect(getMetricValue(auditFailed)).To(Equal(1.0))
		})

		It("should return error when reading request fails", func() {
			req := httptest.NewRequest(http.MethodPost, "/audit", io.NopCloser(&errorReader{}))
			w := httptest.NewRecorder()

			handler.ServeHTTP(w, req)

			Expect(w.Code).To(Equal(http.StatusInternalServerError))
			Expect(w.Body.String()).To(ContainSubstring("failed reading body request"))

			Expect(getMetricValue(auditReceived)).To(Equal(1.0))
			Expect(getMetricValue(auditSucceeded)).To(Equal(0.0))
			Expect(getMetricValue(auditFailed)).To(Equal(1.0))
		})
	})
})

// errorReader is a reader that always returns an error
type errorReader struct{}

func (e *errorReader) Read(p []byte) (n int, err error) {
	return 0, io.ErrUnexpectedEOF
}

// getMetricValue returns the sum of the Counter metrics associated with the Collector
// e.g. the metric for a non-vector, or the sum of the metrics for vector labels.
// If the metric is a Histogram then number of samples is used.
func getMetricValue(col prometheus.Collector) float64 {
	var total float64
	collect(col, func(m *dto.Metric) {
		if h := m.GetHistogram(); h != nil {
			total += float64(h.GetSampleCount())
		} else {
			total += m.GetCounter().GetValue()
		}
	})
	return total
}

// collect calls the function for each metric associated with the Collector
func collect(col prometheus.Collector, do func(*dto.Metric)) {
	c := make(chan prometheus.Metric)
	go func(c chan prometheus.Metric) {
		col.Collect(c)
		close(c)
	}(c)
	for x := range c { // eg range across distinct label vector values
		m := dto.Metric{}
		_ = x.Write(&m)
		do(&m)
	}
}

var letterRunes = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")

func randString(n int) string {
	b := make([]rune, n)
	for i := range b {
		b[i] = letterRunes[rand.Intn(len(letterRunes))]
	}
	return string(b)
}

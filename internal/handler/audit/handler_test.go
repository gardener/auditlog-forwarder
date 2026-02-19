// SPDX-FileCopyrightText: SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package audit

import (
	"bytes"
	"context"
	"io"
	"math/rand/v2"
	"net/http"
	"net/http/httptest"
	"time"

	"github.com/go-logr/logr"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	prommodels "github.com/prometheus/client_model/go"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apiserver/pkg/apis/audit"

	"github.com/gardener/auditlog-forwarder/internal/helper"
	"github.com/gardener/auditlog-forwarder/internal/metrics"
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
				DeliveryMode: configv1alpha1.DeliveryModeGuaranteed,
				HTTP: &configv1alpha1.OutputHTTP{
					URL: testServer.URL,
				},
			},
		}

		var err error
		outputInsts, err = outputfactory.NewHttpOutputsWithOptions(outputConfigs, configv1alpha1.DeliveryModeGuaranteed)
		Expect(err).NotTo(HaveOccurred())

		// reinitialize metrics before each test
		metrics.AuditReceived = promauto.NewCounter(prometheus.CounterOpts{Name: randString()})
		metrics.AuditSucceeded = promauto.NewCounter(prometheus.CounterOpts{Name: randString()})
		metrics.AuditFailed = promauto.NewCounter(prometheus.CounterOpts{Name: randString()})
		metrics.OutputSucceeded = promauto.NewCounterVec(prometheus.CounterOpts{Name: randString()}, []string{"output", "delivery_mode"})
		metrics.OutputFailed = promauto.NewCounterVec(prometheus.CounterOpts{Name: randString()}, []string{"output", "delivery_mode"})
	})

	AfterEach(func() {
		if testServer != nil {
			testServer.Close()
		}
	})

	Describe("NewHandler", func() {
		It("should create a handler with output clients", func() {
			var err error
			handler, err = NewHandler(logger, processors, outputInsts, nil)
			Expect(err).NotTo(HaveOccurred())
			Expect(handler).NotTo(BeNil())
			Expect(handler.guaranteedOutputs).To(HaveLen(1))
			Expect(handler.guaranteedOutputs[0].Name()).To(Equal(testServer.URL))
		})

		It("should return error when no outputs configured", func() {
			var err error
			handler, err = NewHandler(logger, processors, []output.Output{}, nil)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("at least one guaranteed output must be configured"))
			Expect(handler).To(BeNil())
		})
	})

	Describe("ServeHTTP", func() {
		BeforeEach(func() {
			var err error
			handler, err = NewHandler(logger, processors, outputInsts, nil)
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
			handler, err = NewHandler(logger, processorsChained, outputInsts, nil)
			Expect(err).NotTo(HaveOccurred())

			initial := []byte("A")
			req := httptest.NewRequest(http.MethodPost, "/audit", bytes.NewReader(initial))
			w := httptest.NewRecorder()
			handler.ServeHTTP(w, req)
			Expect(w.Code).To(Equal(http.StatusOK))

			Eventually(func() bool { return len(response) > 0 }, time.Millisecond*100).Should(BeTrue())
			Expect(string(response)).To(Equal("A->B->C"))

			Expect(getMetricValue(metrics.AuditReceived)).To(Equal(1.0))
			Expect(getMetricValue(metrics.AuditSucceeded)).To(Equal(1.0))
			Expect(getMetricValue(metrics.AuditFailed)).To(Equal(0.0))
			Expect(getMetricValue(metrics.OutputSucceeded)).To(Equal(1.0))
			Expect(getMetricValue(metrics.OutputFailed)).To(Equal(0.0))
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

			Expect(getMetricValue(metrics.AuditReceived)).To(Equal(1.0))
			Expect(getMetricValue(metrics.AuditSucceeded)).To(Equal(1.0))
			Expect(getMetricValue(metrics.AuditFailed)).To(Equal(0.0))
			Expect(getMetricValue(metrics.OutputSucceeded)).To(Equal(1.0))
			Expect(getMetricValue(metrics.OutputFailed)).To(Equal(0.0))
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
			Expect(w.Body.String()).To(ContainSubstring("failed forwarding audit events"))

			Expect(getMetricValue(metrics.AuditReceived)).To(Equal(1.0))
			Expect(getMetricValue(metrics.AuditSucceeded)).To(Equal(0.0))
			Expect(getMetricValue(metrics.AuditFailed)).To(Equal(1.0))
			Expect(getMetricValue(metrics.OutputSucceeded)).To(Equal(0.0))
			Expect(getMetricValue(metrics.OutputFailed)).To(Equal(1.0))
		})

		It("should handle malformed request body", func() {
			req := httptest.NewRequest(http.MethodPost, "/audit", bytes.NewReader([]byte("invalid json")))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			handler.ServeHTTP(w, req)

			Expect(w.Code).To(Equal(http.StatusInternalServerError))
			Expect(w.Body.String()).To(ContainSubstring("failed processing audit events"))

			Expect(getMetricValue(metrics.AuditReceived)).To(Equal(1.0))
			Expect(getMetricValue(metrics.AuditSucceeded)).To(Equal(0.0))
			Expect(getMetricValue(metrics.AuditFailed)).To(Equal(1.0))
			Expect(getMetricValue(metrics.OutputSucceeded)).To(Equal(0.0))
			Expect(getMetricValue(metrics.OutputFailed)).To(Equal(0.0))
		})

		It("should return error when reading request fails", func() {
			req := httptest.NewRequest(http.MethodPost, "/audit", io.NopCloser(&errorReader{}))
			w := httptest.NewRecorder()

			handler.ServeHTTP(w, req)

			Expect(w.Code).To(Equal(http.StatusInternalServerError))
			Expect(w.Body.String()).To(ContainSubstring("failed reading body request"))

			Expect(getMetricValue(metrics.AuditReceived)).To(Equal(1.0))
			Expect(getMetricValue(metrics.AuditSucceeded)).To(Equal(0.0))
			Expect(getMetricValue(metrics.AuditFailed)).To(Equal(1.0))
			Expect(getMetricValue(metrics.OutputSucceeded)).To(Equal(0.0))
			Expect(getMetricValue(metrics.OutputFailed)).To(Equal(0.0))
		})
	})

	Describe("Shutdown", func() {
		var (
			bestEffortServer   *httptest.Server
			bestEffortResponse []byte
		)

		BeforeEach(func() {
			bestEffortServer = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				body, err := io.ReadAll(r.Body)
				Expect(err).NotTo(HaveOccurred())
				bestEffortResponse = body
				w.WriteHeader(http.StatusOK)
			}))

			outputConfigs := []configv1alpha1.Output{
				{
					DeliveryMode: configv1alpha1.DeliveryModeBestEffort,
					HTTP: &configv1alpha1.OutputHTTP{
						URL: bestEffortServer.URL,
					},
				},
			}

			var err error
			bestEffortOutputs, err := outputfactory.NewHttpOutputsWithOptions(outputConfigs, configv1alpha1.DeliveryModeBestEffort)
			Expect(err).NotTo(HaveOccurred())

			handler, err = NewHandler(logger, processors, outputInsts, bestEffortOutputs)
			Expect(err).NotTo(HaveOccurred())
		})

		AfterEach(func() {
			if bestEffortServer != nil {
				bestEffortServer.Close()
			}
		})

		It("should wait for best-effort outputs to complete during shutdown", func() {
			eventList := &audit.EventList{
				TypeMeta: metav1.TypeMeta{
					APIVersion: "audit.k8s.io/v1",
					Kind:       "EventList",
				},
				Items: []audit.Event{{Verb: "create"}},
			}

			body, err := helper.EncodeEventList(eventList)
			Expect(err).NotTo(HaveOccurred())

			req := httptest.NewRequest(http.MethodPost, "/audit", bytes.NewReader(body))
			w := httptest.NewRecorder()

			handler.ServeHTTP(w, req)

			Expect(w.Code).To(Equal(http.StatusOK))

			// Shutdown should wait for best-effort output to complete
			Expect(handler.Shutdown(2 * time.Second)).To(Succeed())
			Eventually(func() bool { return len(bestEffortResponse) > 0 }, 50*time.Millisecond).Should(BeTrue())
		})

		It("should return timeout error if best-effort outputs take too long", func() {
			slowServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				time.Sleep(time.Second) // Longer than shutdown timeout
				w.WriteHeader(http.StatusOK)
			}))
			defer slowServer.Close()

			slowOutputConfigs := []configv1alpha1.Output{
				{
					DeliveryMode: configv1alpha1.DeliveryModeBestEffort,
					HTTP: &configv1alpha1.OutputHTTP{
						URL: slowServer.URL,
					},
				},
			}

			slowOutputs, err := outputfactory.NewHttpOutputsWithOptions(slowOutputConfigs, configv1alpha1.DeliveryModeBestEffort)
			Expect(err).NotTo(HaveOccurred())

			handler, err = NewHandler(logger, processors, outputInsts, slowOutputs)
			Expect(err).NotTo(HaveOccurred())

			eventList := &audit.EventList{
				TypeMeta: metav1.TypeMeta{
					APIVersion: "audit.k8s.io/v1",
					Kind:       "EventList",
				},
				Items: []audit.Event{{Verb: "create"}},
			}

			body, err := helper.EncodeEventList(eventList)
			Expect(err).NotTo(HaveOccurred())

			req := httptest.NewRequest(http.MethodPost, "/audit", bytes.NewReader(body))
			w := httptest.NewRecorder()

			handler.ServeHTTP(w, req)
			Expect(w.Code).To(Equal(http.StatusOK))

			// Shutdown with short timeout should timeout
			err = handler.Shutdown(100 * time.Millisecond)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("shutdown timeout exceeded"))
		})

		It("should complete shutdown immediately when no best-effort outputs are in flight", func() {
			Expect(handler.Shutdown(5 * time.Second)).To(Succeed())
		})

		It("should cancel shutdown context after successful shutdown", func() {
			initialCtx := handler.shutdownCtx
			Expect(initialCtx.Err()).To(BeNil())

			Expect(handler.Shutdown(1 * time.Second)).To(Succeed())
			Expect(initialCtx.Err()).To(Equal(context.Canceled))
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
	collect(col, func(m *prommodels.Metric) {
		if h := m.GetHistogram(); h != nil {
			total += float64(h.GetSampleCount())
		} else {
			total += m.GetCounter().GetValue()
		}
	})
	return total
}

// collect calls the function for each metric associated with the Collector
func collect(col prometheus.Collector, do func(*prommodels.Metric)) {
	c := make(chan prometheus.Metric)
	go func(c chan prometheus.Metric) {
		col.Collect(c)
		close(c)
	}(c)
	for x := range c { // eg range across distinct label vector values
		m := prommodels.Metric{}
		Expect(x.Write(&m)).To(Succeed())
		do(&m)
	}
}

var letterRunes = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")

func randString() string {
	b := make([]rune, 10)
	for i := range b {
		b[i] = letterRunes[rand.IntN(len(letterRunes))] //nolint:gosec
	}
	return string(b)
}

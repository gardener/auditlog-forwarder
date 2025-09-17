// SPDX-FileCopyrightText: SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package audit

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"time"

	"github.com/go-logr/logr"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
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
		})

		It("should handle malformed request body", func() {
			req := httptest.NewRequest(http.MethodPost, "/audit", bytes.NewReader([]byte("invalid json")))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			handler.ServeHTTP(w, req)

			Expect(w.Code).To(Equal(http.StatusInternalServerError))
		})
	})
})

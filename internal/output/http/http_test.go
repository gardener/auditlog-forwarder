// SPDX-FileCopyrightText: SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package http_test

import (
	"bytes"
	"compress/gzip"
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	httpoutput "github.com/gardener/auditlog-forwarder/internal/output/http"
	configv1alpha1 "github.com/gardener/auditlog-forwarder/pkg/apis/config/v1alpha1"
)

var _ = Describe("HTTP Output", func() {
	var (
		testServer   *httptest.Server
		response     []byte
		responseCode int
		httpOutput   *httpoutput.Output
	)

	BeforeEach(func() {
		response = []byte{}
		responseCode = http.StatusOK

		testServer = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			body, err := io.ReadAll(r.Body)
			Expect(err).NotTo(HaveOccurred())
			response = body

			w.WriteHeader(responseCode)
		}))
	})

	AfterEach(func() {
		if testServer != nil {
			testServer.Close()
		}
	})

	Describe("New", func() {
		It("should create a new HTTP output", func() {
			config := &configv1alpha1.OutputHTTP{
				URL: testServer.URL,
			}

			var err error
			httpOutput, err = httpoutput.New(config)
			Expect(err).NotTo(HaveOccurred())
			Expect(httpOutput).NotTo(BeNil())
			Expect(httpOutput.Name()).To(Equal(testServer.URL))
		})

		It("should handle nil config", func() {
			httpOutput, err := httpoutput.New(nil)
			Expect(err).To(HaveOccurred())
			Expect(err).To(MatchError(ContainSubstring("is nil")))
			Expect(httpOutput).To(BeNil())
		})

		It("should create output with TLS config", func() {
			config := &configv1alpha1.OutputHTTP{
				URL: testServer.URL,
				TLS: &configv1alpha1.ClientTLS{
					CAFile: "/nonexistent/ca.pem",
				},
			}

			httpOutput, err := httpoutput.New(config)
			Expect(err).To(HaveOccurred())
			Expect(err).To(MatchError(ContainSubstring("failed to read CA certificate file")))
			Expect(err).To(MatchError(ContainSubstring("/nonexistent/ca.pem")))
			Expect(httpOutput).To(BeNil())
		})
	})

	Describe("SendEvents", func() {
		BeforeEach(func() {
			config := &configv1alpha1.OutputHTTP{
				URL: testServer.URL,
			}

			var err error
			httpOutput, err = httpoutput.New(config)
			Expect(err).NotTo(HaveOccurred())
		})

		It("should send events successfully", func() {
			testData := []byte(`{"events": ["test"]}`)

			Expect(httpOutput.Send(context.Background(), testData)).To(Succeed())

			Expect(response).To(Equal(testData))
		})

		It("should send events compressed with gzip when configured", func() {
			// recreate test server to inspect compression
			testServer.Close()
			var receivedEncoding string
			var receivedBody []byte
			testServer = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				receivedEncoding = r.Header.Get("Content-Encoding")
				body, err := io.ReadAll(r.Body)
				Expect(err).NotTo(HaveOccurred())
				// decompress
				gzReader, err := gzip.NewReader(bytes.NewReader(body))
				Expect(err).NotTo(HaveOccurred())
				decompressed, err := io.ReadAll(gzReader)
				Expect(err).NotTo(HaveOccurred())
				Expect(gzReader.Close()).To(Succeed())
				receivedBody = decompressed
				w.WriteHeader(http.StatusOK)
			}))

			config := &configv1alpha1.OutputHTTP{
				URL:         testServer.URL,
				Compression: "gzip",
			}
			var err error
			httpOutput, err = httpoutput.New(config)
			Expect(err).NotTo(HaveOccurred())

			testData := []byte(`{"events": ["test"]}`)
			Expect(httpOutput.Send(context.Background(), testData)).To(Succeed())

			Expect(receivedEncoding).To(Equal("gzip"))
			Expect(receivedBody).To(Equal(testData))
		})

		It("should handle server errors", func() {
			responseCode = http.StatusInternalServerError

			testData := []byte(`{"events": ["test"]}`)

			err := httpOutput.Send(context.Background(), testData)
			Expect(err).To(HaveOccurred())
			Expect(err).To(MatchError(ContainSubstring(("output returned status 500"))))
		})

		It("should handle context cancellation", func() {
			ctx, cancel := context.WithCancel(context.Background())
			cancel() // Cancel immediately

			testData := []byte(`{"events": ["test"]}`)

			err := httpOutput.Send(ctx, testData)
			Expect(err).To(HaveOccurred())
			Expect(err).To(MatchError(ContainSubstring("context canceled")))
		})

		It("should retry on retryable status codes", func() {
			var attempts int32
			originalBackoff := httpoutput.BackoffFunc
			originalSleep := httpoutput.SleepFunc
			httpoutput.BackoffFunc = func(_ int, _, _ time.Duration) time.Duration { return 0 }
			httpoutput.SleepFunc = func(_ context.Context, _ time.Duration) error { return nil }
			DeferCleanup(func() {
				httpoutput.BackoffFunc = originalBackoff
				httpoutput.SleepFunc = originalSleep
			})

			testServer.Close()
			testServer = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				count := atomic.AddInt32(&attempts, 1)
				if count < 3 {
					w.WriteHeader(http.StatusInternalServerError)
					return
				}
				w.WriteHeader(http.StatusOK)
			}))

			config := &configv1alpha1.OutputHTTP{
				URL: testServer.URL,
			}

			var err error
			httpOutput, err = httpoutput.New(config)
			Expect(err).NotTo(HaveOccurred())

			testData := []byte(`{"events": ["test"]}`)
			Expect(httpOutput.Send(context.Background(), testData)).To(Succeed())
			Expect(atomic.LoadInt32(&attempts)).To(Equal(int32(3)))
		})
	})
})

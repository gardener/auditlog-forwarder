// SPDX-FileCopyrightText: SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package http

import (
	"bytes"
	"compress/gzip"
	"context"
	"io"
	"net/http"
	"net/http/httptest"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	configv1alpha1 "github.com/gardener/auditlog-forwarder/pkg/apis/config/v1alpha1"
)

var _ = Describe("HTTP Output", func() {
	var (
		testServer   *httptest.Server
		response     []byte
		responseCode int
		output       *Output
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
			output, err = New(config)
			Expect(err).NotTo(HaveOccurred())
			Expect(output).NotTo(BeNil())
			Expect(output.Name()).To(Equal(testServer.URL))
		})

		It("should handle nil config", func() {
			output, err := New(nil)
			Expect(err).To(HaveOccurred())
			Expect(err).To(MatchError(ContainSubstring("is nil")))
			Expect(output).To(BeNil())
		})

		It("should create output with TLS config", func() {
			config := &configv1alpha1.OutputHTTP{
				URL: testServer.URL,
				TLS: &configv1alpha1.ClientTLS{
					CAFile: "/nonexistent/ca.pem",
				},
			}

			output, err := New(config)
			Expect(err).To(HaveOccurred())
			Expect(err).To(MatchError(ContainSubstring("failed to read CA certificate file")))
			Expect(err).To(MatchError(ContainSubstring("/nonexistent/ca.pem")))
			Expect(output).To(BeNil())
		})
	})

	Describe("SendEvents", func() {
		BeforeEach(func() {
			config := &configv1alpha1.OutputHTTP{
				URL: testServer.URL,
			}

			var err error
			output, err = New(config)
			Expect(err).NotTo(HaveOccurred())
		})

		It("should send events successfully", func() {
			testData := []byte(`{"events": ["test"]}`)

			err := output.Send(context.Background(), testData)
			Expect(err).NotTo(HaveOccurred())

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
			output, err = New(config)
			Expect(err).NotTo(HaveOccurred())

			testData := []byte(`{"events": ["test"]}`)
			err = output.Send(context.Background(), testData)
			Expect(err).NotTo(HaveOccurred())

			Expect(receivedEncoding).To(Equal("gzip"))
			Expect(receivedBody).To(Equal(testData))
		})

		It("should handle server errors", func() {
			responseCode = http.StatusInternalServerError

			testData := []byte(`{"events": ["test"]}`)

			err := output.Send(context.Background(), testData)
			Expect(err).To(HaveOccurred())
			Expect(err).To(MatchError(ContainSubstring(("output returned status 500"))))
		})

		It("should handle context cancellation", func() {
			ctx, cancel := context.WithCancel(context.Background())
			cancel() // Cancel immediately

			testData := []byte(`{"events": ["test"]}`)

			err := output.Send(ctx, testData)
			Expect(err).To(HaveOccurred())
			Expect(err).To(MatchError(ContainSubstring("context canceled")))
		})
	})
})

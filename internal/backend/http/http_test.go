// SPDX-FileCopyrightText: SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package http

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	configv1alpha1 "github.com/gardener/auditlog-forwarder/pkg/apis/config/v1alpha1"
)

var _ = Describe("HTTP Backend", func() {
	var (
		testServer   *httptest.Server
		response     []byte
		responseCode int
		backend      *Backend
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
		It("should create a new HTTP backend", func() {
			config := &configv1alpha1.HTTPBackend{
				URL: testServer.URL,
			}

			var err error
			backend, err = New(config)
			Expect(err).NotTo(HaveOccurred())
			Expect(backend).NotTo(BeNil())
			Expect(backend.Name()).To(Equal(testServer.URL))
		})

		It("should handle nil config", func() {
			backend, err := New(nil)
			Expect(err).To(HaveOccurred())
			Expect(err).To(MatchError(ContainSubstring("is nil")))
			Expect(backend).To(BeNil())
		})

		It("should create backend with TLS config", func() {
			config := &configv1alpha1.HTTPBackend{
				URL: testServer.URL,
				TLS: &configv1alpha1.ClientTLSConfig{
					CAFile: "/nonexistent/ca.pem",
				},
			}

			backend, err := New(config)
			Expect(err).To(HaveOccurred())
			Expect(err).To(MatchError(ContainSubstring("failed to read CA certificate file")))
			Expect(err).To(MatchError(ContainSubstring("/nonexistent/ca.pem")))
			Expect(backend).To(BeNil())
		})
	})

	Describe("SendEvents", func() {
		BeforeEach(func() {
			config := &configv1alpha1.HTTPBackend{
				URL: testServer.URL,
			}

			var err error
			backend, err = New(config)
			Expect(err).NotTo(HaveOccurred())
		})

		It("should send events successfully", func() {
			testData := []byte(`{"events": ["test"]}`)

			err := backend.Send(context.Background(), testData)
			Expect(err).NotTo(HaveOccurred())

			Expect(response).To(Equal(testData))
		})

		It("should handle server errors", func() {
			responseCode = http.StatusInternalServerError

			testData := []byte(`{"events": ["test"]}`)

			err := backend.Send(context.Background(), testData)
			Expect(err).To(HaveOccurred())
			Expect(err).To(MatchError(ContainSubstring(("backend returned status 500"))))
		})

		It("should handle context cancellation", func() {
			ctx, cancel := context.WithCancel(context.Background())
			cancel() // Cancel immediately

			testData := []byte(`{"events": ["test"]}`)

			err := backend.Send(ctx, testData)
			Expect(err).To(HaveOccurred())
			Expect(err).To(MatchError(ContainSubstring("context canceled")))
		})
	})
})

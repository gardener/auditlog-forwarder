// SPDX-FileCopyrightText: SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package factory_test

import (
	"net/http/httptest"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/gardener/auditlog-forwarder/internal/backend/factory"
	configv1alpha1 "github.com/gardener/auditlog-forwarder/pkg/apis/config/v1alpha1"
)

var _ = Describe("Backend Factory", func() {
	var testServer *httptest.Server

	BeforeEach(func() {
		testServer = httptest.NewServer(nil)
	})

	AfterEach(func() {
		if testServer != nil {
			testServer.Close()
		}
	})

	Describe("NewFromConfig", func() {
		It("should create HTTP backend from config", func() {
			config := configv1alpha1.Backend{
				HTTP: &configv1alpha1.HTTPBackend{
					URL: testServer.URL,
				},
			}

			backend, err := factory.NewFromConfig(config)
			Expect(err).NotTo(HaveOccurred())
			Expect(backend).NotTo(BeNil())
			Expect(backend.Name()).To(Equal(testServer.URL))
		})

		It("should return error for empty config", func() {
			config := configv1alpha1.Backend{}

			backend, err := factory.NewFromConfig(config)
			Expect(err).To(HaveOccurred())
			Expect(err).To(MatchError(ContainSubstring("no supported backend type configured")))
			Expect(backend).To(BeNil())
		})
	})

	Describe("NewFromConfigs", func() {
		It("should create multiple backends from configs", func() {
			configs := []configv1alpha1.Backend{
				{
					HTTP: &configv1alpha1.HTTPBackend{
						URL: testServer.URL + "/endpoint1",
					},
				},
				{
					HTTP: &configv1alpha1.HTTPBackend{
						URL: testServer.URL + "/endpoint2",
					},
				},
			}

			backends, err := factory.NewFromConfigs(configs)
			Expect(err).NotTo(HaveOccurred())
			Expect(backends).To(HaveLen(2))
			Expect(backends[0].Name()).To(Equal(testServer.URL + "/endpoint1"))
			Expect(backends[1].Name()).To(Equal(testServer.URL + "/endpoint2"))
		})

		It("should handle empty configs slice", func() {
			backends, err := factory.NewFromConfigs([]configv1alpha1.Backend{})
			Expect(err).NotTo(HaveOccurred())
			Expect(backends).To(HaveLen(0))
		})

		It("should return error if any backend fails to create", func() {
			configs := []configv1alpha1.Backend{
				{
					HTTP: &configv1alpha1.HTTPBackend{
						URL: testServer.URL,
					},
				},
				{}, // Invalid config
			}

			backends, err := factory.NewFromConfigs(configs)
			Expect(err).To(HaveOccurred())
			Expect(err).To(MatchError(ContainSubstring("no supported backend type configured")))
			Expect(backends).To(BeNil())
		})
	})
})

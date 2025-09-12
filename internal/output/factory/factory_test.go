// SPDX-FileCopyrightText: SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package factory_test

import (
	"net/http/httptest"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/gardener/auditlog-forwarder/internal/output/factory"
	configv1alpha1 "github.com/gardener/auditlog-forwarder/pkg/apis/config/v1alpha1"
)

var _ = Describe("Output Factory", func() {
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
		It("should create HTTP output from config", func() {
			config := configv1alpha1.Output{
				HTTP: &configv1alpha1.OutputHTTP{
					URL: testServer.URL,
				},
			}

			output, err := factory.NewFromConfig(config)
			Expect(err).NotTo(HaveOccurred())
			Expect(output).NotTo(BeNil())
			Expect(output.Name()).To(Equal(testServer.URL))
		})

		It("should return error for empty config", func() {
			config := configv1alpha1.Output{}

			output, err := factory.NewFromConfig(config)
			Expect(err).To(HaveOccurred())
			Expect(err).To(MatchError(ContainSubstring("no supported output type configured")))
			Expect(output).To(BeNil())
		})
	})

	Describe("NewFromConfigs", func() {
		It("should create multiple outputs from configs", func() {
			configs := []configv1alpha1.Output{
				{
					HTTP: &configv1alpha1.OutputHTTP{
						URL: testServer.URL + "/endpoint1",
					},
				},
				{
					HTTP: &configv1alpha1.OutputHTTP{
						URL: testServer.URL + "/endpoint2",
					},
				},
			}

			outputs, err := factory.NewFromConfigs(configs)
			Expect(err).NotTo(HaveOccurred())
			Expect(outputs).To(HaveLen(2))
			Expect(outputs[0].Name()).To(Equal(testServer.URL + "/endpoint1"))
			Expect(outputs[1].Name()).To(Equal(testServer.URL + "/endpoint2"))
		})

		It("should handle empty configs slice", func() {
			outputs, err := factory.NewFromConfigs([]configv1alpha1.Output{})
			Expect(err).NotTo(HaveOccurred())
			Expect(outputs).To(HaveLen(0))
		})

		It("should return error if any output fails to create", func() {
			configs := []configv1alpha1.Output{
				{
					HTTP: &configv1alpha1.OutputHTTP{
						URL: testServer.URL,
					},
				},
				{}, // Invalid config
			}

			outputs, err := factory.NewFromConfigs(configs)
			Expect(err).To(HaveOccurred())
			Expect(err).To(MatchError(ContainSubstring("no supported output type configured")))
			Expect(outputs).To(BeNil())
		})
	})
})

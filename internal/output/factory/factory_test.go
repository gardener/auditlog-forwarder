// SPDX-FileCopyrightText: SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package factory_test

import (
	"net/http/httptest"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/gardener/auditlog-forwarder/internal/output/factory"
	httpoutput "github.com/gardener/auditlog-forwarder/internal/output/http"
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

	Describe("NewHttpOutputsWithOptions", func() {
		It("should create HTTP outputs with Guaranteed delivery mode", func() {
			outputs := []configv1alpha1.Output{
				{
					DeliveryMode: configv1alpha1.DeliveryModeGuaranteed,
					HTTP: &configv1alpha1.OutputHTTP{
						URL: testServer.URL,
					},
				},
				{
					DeliveryMode: configv1alpha1.DeliveryModeBestEffort,
					HTTP: &configv1alpha1.OutputHTTP{
						URL: testServer.URL + "/other",
					},
				},
			}

			result, err := factory.NewHTTPOutputsWithOptions(outputs, configv1alpha1.DeliveryModeGuaranteed)
			Expect(err).NotTo(HaveOccurred())
			Expect(result).To(HaveLen(1))
			Expect(result[0].Name()).To(Equal(testServer.URL))
		})

		It("should create HTTP outputs with BestEffort delivery mode", func() {
			outputs := []configv1alpha1.Output{
				{
					DeliveryMode: configv1alpha1.DeliveryModeGuaranteed,
					HTTP: &configv1alpha1.OutputHTTP{
						URL: testServer.URL,
					},
				},
				{
					DeliveryMode: configv1alpha1.DeliveryModeBestEffort,
					HTTP: &configv1alpha1.OutputHTTP{
						URL: testServer.URL + "/best-effort",
					},
				},
			}

			result, err := factory.NewHTTPOutputsWithOptions(outputs, configv1alpha1.DeliveryModeBestEffort)
			Expect(err).NotTo(HaveOccurred())
			Expect(result).To(HaveLen(1))
			Expect(result[0].Name()).To(Equal(testServer.URL + "/best-effort"))
		})

		It("should filter out non-HTTP outputs", func() {
			outputs := []configv1alpha1.Output{
				{
					DeliveryMode: configv1alpha1.DeliveryModeGuaranteed,
					HTTP:         nil, // non-HTTP output
				},
				{
					DeliveryMode: configv1alpha1.DeliveryModeGuaranteed,
					HTTP: &configv1alpha1.OutputHTTP{
						URL: testServer.URL,
					},
				},
			}

			result, err := factory.NewHTTPOutputsWithOptions(outputs, configv1alpha1.DeliveryModeGuaranteed)
			Expect(err).NotTo(HaveOccurred())
			Expect(result).To(HaveLen(1))
		})

		It("should filter by delivery mode", func() {
			outputs := []configv1alpha1.Output{
				{
					DeliveryMode: configv1alpha1.DeliveryModeGuaranteed,
					HTTP: &configv1alpha1.OutputHTTP{
						URL: testServer.URL + "/guaranteed-1",
					},
				},
				{
					DeliveryMode: configv1alpha1.DeliveryModeGuaranteed,
					HTTP: &configv1alpha1.OutputHTTP{
						URL: testServer.URL + "/guaranteed-2",
					},
				},
				{
					DeliveryMode: configv1alpha1.DeliveryModeBestEffort,
					HTTP: &configv1alpha1.OutputHTTP{
						URL: testServer.URL + "/best-effort",
					},
				},
			}

			result, err := factory.NewHTTPOutputsWithOptions(outputs, configv1alpha1.DeliveryModeGuaranteed)
			Expect(err).NotTo(HaveOccurred())
			Expect(result).To(HaveLen(2))
			Expect(result[0].Name()).To(Equal(testServer.URL + "/guaranteed-1"))
			Expect(result[1].Name()).To(Equal(testServer.URL + "/guaranteed-2"))
		})

		It("should return empty slice when no outputs match", func() {
			outputs := []configv1alpha1.Output{
				{
					DeliveryMode: configv1alpha1.DeliveryModeBestEffort,
					HTTP: &configv1alpha1.OutputHTTP{
						URL: testServer.URL,
					},
				},
			}

			result, err := factory.NewHTTPOutputsWithOptions(outputs, configv1alpha1.DeliveryModeGuaranteed)
			Expect(err).NotTo(HaveOccurred())
			Expect(result).To(BeEmpty())
		})

		It("should handle empty outputs slice", func() {
			outputs := []configv1alpha1.Output{}

			result, err := factory.NewHTTPOutputsWithOptions(outputs, configv1alpha1.DeliveryModeGuaranteed)
			Expect(err).NotTo(HaveOccurred())
			Expect(result).To(BeEmpty())
		})

		It("should apply HTTP options to created outputs", func() {
			outputs := []configv1alpha1.Output{
				{
					DeliveryMode: configv1alpha1.DeliveryModeGuaranteed,
					HTTP: &configv1alpha1.OutputHTTP{
						URL: testServer.URL,
					},
				},
			}

			result, err := factory.NewHTTPOutputsWithOptions(
				outputs,
				configv1alpha1.DeliveryModeGuaranteed,
				httpoutput.WithMaxSendAttempts(10),
				httpoutput.WithBaseBackoff(2*time.Second),
				httpoutput.WithMaxBackoff(20*time.Second),
			)
			Expect(err).NotTo(HaveOccurred())
			Expect(result).To(HaveLen(1))
			Expect(result[0].Name()).To(Equal(testServer.URL))
		})

		It("should apply different options to multiple outputs", func() {
			outputs := []configv1alpha1.Output{
				{
					DeliveryMode: configv1alpha1.DeliveryModeGuaranteed,
					HTTP: &configv1alpha1.OutputHTTP{
						URL: testServer.URL + "/output-1",
					},
				},
				{
					DeliveryMode: configv1alpha1.DeliveryModeGuaranteed,
					HTTP: &configv1alpha1.OutputHTTP{
						URL: testServer.URL + "/output-2",
					},
				},
			}

			result, err := factory.NewHTTPOutputsWithOptions(
				outputs,
				configv1alpha1.DeliveryModeGuaranteed,
				httpoutput.WithMaxSendAttempts(5),
			)
			Expect(err).NotTo(HaveOccurred())
			Expect(result).To(HaveLen(2))
			Expect(result[0].Name()).To(Equal(testServer.URL + "/output-1"))
			Expect(result[1].Name()).To(Equal(testServer.URL + "/output-2"))
		})

		It("should return error when HTTP output creation fails", func() {
			outputs := []configv1alpha1.Output{
				{
					DeliveryMode: configv1alpha1.DeliveryModeGuaranteed,
					HTTP: &configv1alpha1.OutputHTTP{
						URL: testServer.URL,
						TLS: &configv1alpha1.ClientTLS{
							CAFile: "/nonexistent/ca.pem",
						},
					},
				},
			}

			result, err := factory.NewHTTPOutputsWithOptions(outputs, configv1alpha1.DeliveryModeGuaranteed)
			Expect(err).To(HaveOccurred())
			Expect(err).To(MatchError(ContainSubstring("failed to create HTTP output")))
			Expect(result).To(BeNil())
		})

		It("should handle mixed HTTP and non-HTTP outputs with filtering", func() {
			outputs := []configv1alpha1.Output{
				{
					DeliveryMode: configv1alpha1.DeliveryModeGuaranteed,
					HTTP:         nil, // non-HTTP
				},
				{
					DeliveryMode: configv1alpha1.DeliveryModeGuaranteed,
					HTTP: &configv1alpha1.OutputHTTP{
						URL: testServer.URL + "/http-1",
					},
				},
				{
					DeliveryMode: configv1alpha1.DeliveryModeBestEffort,
					HTTP: &configv1alpha1.OutputHTTP{
						URL: testServer.URL + "/http-2",
					},
				},
				{
					DeliveryMode: configv1alpha1.DeliveryModeGuaranteed,
					HTTP: &configv1alpha1.OutputHTTP{
						URL: testServer.URL + "/http-3",
					},
				},
			}

			result, err := factory.NewHTTPOutputsWithOptions(outputs, configv1alpha1.DeliveryModeGuaranteed)
			Expect(err).NotTo(HaveOccurred())
			Expect(result).To(HaveLen(2))
			Expect(result[0].Name()).To(Equal(testServer.URL + "/http-1"))
			Expect(result[1].Name()).To(Equal(testServer.URL + "/http-3"))
		})
	})
})

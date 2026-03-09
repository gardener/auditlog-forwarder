// SPDX-FileCopyrightText: SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package v1alpha1_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	. "github.com/gardener/auditlog-forwarder/pkg/apis/config/v1alpha1"
)

var _ = Describe("Defaults", func() {
	var (
		obj *AuditlogForwarder
	)

	BeforeEach(func() {
		obj = &AuditlogForwarder{}
	})

	Describe("#SetDefaults_AuditlogForwarder", func() {
		It("should default the log level and format", func() {
			SetDefaults_AuditlogForwarder(obj)

			Expect(obj.Log.Level).To(Equal(LogLevelInfo))
			Expect(obj.Log.Format).To(Equal(LogFormatJSON))
		})

		It("should default the server port", func() {
			SetDefaults_AuditlogForwarder(obj)

			Expect(obj.Server.Port).To(Equal(int32(10443)))
		})

		It("should default the server metrics port", func() {
			SetDefaults_AuditlogForwarder(obj)

			Expect(obj.Server.MetricsPort).To(Equal(int32(8080)))
		})

		It("should not override existing values", func() {
			obj.Log.Level = LogLevelDebug
			obj.Log.Format = LogFormatText
			obj.Server.Port = 8080
			obj.Server.MetricsPort = 9090

			SetDefaults_AuditlogForwarder(obj)

			Expect(obj.Log.Level).To(Equal(LogLevelDebug))
			Expect(obj.Log.Format).To(Equal(LogFormatText))
			Expect(obj.Server.Port).To(Equal(int32(8080)))
			Expect(obj.Server.MetricsPort).To(Equal(int32(9090)))
		})
	})

	Describe("#SetDefaults_Log", func() {
		var (
			logConfig *Log
		)

		BeforeEach(func() {
			logConfig = &Log{}
		})

		It("should default the level to info", func() {
			SetDefaults_Log(logConfig)

			Expect(logConfig.Level).To(Equal(LogLevelInfo))
		})

		It("should default the format to json", func() {
			SetDefaults_Log(logConfig)

			Expect(logConfig.Format).To(Equal(LogFormatJSON))
		})

		It("should not override existing values", func() {
			logConfig.Level = LogLevelDebug
			logConfig.Format = LogFormatText

			SetDefaults_Log(logConfig)

			Expect(logConfig.Level).To(Equal(LogLevelDebug))
			Expect(logConfig.Format).To(Equal(LogFormatText))
		})
	})

	Describe("#SetDefaults_Server", func() {
		var (
			serverConfig *Server
		)

		BeforeEach(func() {
			serverConfig = &Server{}
		})

		It("should default the port to 10443", func() {
			SetDefaults_Server(serverConfig)

			Expect(serverConfig.Port).To(Equal(int32(10443)))
		})

		It("should default the metrics port to 8080", func() {
			SetDefaults_Server(serverConfig)

			Expect(serverConfig.MetricsPort).To(Equal(int32(8080)))
		})

		It("should not override existing port value", func() {
			serverConfig.Port = 8080

			SetDefaults_Server(serverConfig)

			Expect(serverConfig.Port).To(Equal(int32(8080)))
		})

		It("should not override existing metrics port value", func() {
			serverConfig.MetricsPort = 8090

			SetDefaults_Server(serverConfig)

			Expect(serverConfig.MetricsPort).To(Equal(int32(8090)))
		})

		It("should not set defaults for address (should remain empty)", func() {
			SetDefaults_Server(serverConfig)

			Expect(serverConfig.Address).To(BeEmpty())
		})

		It("should not set defaults for TLS configuration", func() {
			SetDefaults_Server(serverConfig)

			Expect(serverConfig.TLS.CertFile).To(BeEmpty())
			Expect(serverConfig.TLS.KeyFile).To(BeEmpty())
		})
	})

	Describe("#SetDefaults_Outputs", func() {
		It("should default single output without delivery mode to Guaranteed", func() {
			outputs := []Output{
				{
					HTTP: &OutputHTTP{URL: "http://example.com"},
				},
			}

			SetDefaults_Outputs(outputs)

			Expect(outputs[0].DeliveryMode).To(Equal(DeliveryModeGuaranteed))
		})

		It("should not override delivery mode for single output when already set", func() {
			outputs := []Output{
				{
					HTTP:         &OutputHTTP{URL: "http://example.com"},
					DeliveryMode: DeliveryModeBestEffort,
				},
			}

			SetDefaults_Outputs(outputs)

			Expect(outputs[0].DeliveryMode).To(Equal(DeliveryModeBestEffort))
		})

		It("should default multiple outputs without delivery modes to BestEffort", func() {
			outputs := []Output{
				{HTTP: &OutputHTTP{URL: "http://example1.com"}},
				{HTTP: &OutputHTTP{URL: "http://example2.com"}},
				{HTTP: &OutputHTTP{URL: "http://example3.com"}},
			}

			SetDefaults_Outputs(outputs)

			Expect(outputs[0].DeliveryMode).To(Equal(DeliveryModeBestEffort))
			Expect(outputs[1].DeliveryMode).To(Equal(DeliveryModeBestEffort))
			Expect(outputs[2].DeliveryMode).To(Equal(DeliveryModeBestEffort))
		})

		It("should not override delivery modes for multiple outputs when already set", func() {
			outputs := []Output{
				{
					HTTP:         &OutputHTTP{URL: "http://example1.com"},
					DeliveryMode: DeliveryModeGuaranteed,
				},
				{
					HTTP:         &OutputHTTP{URL: "http://example2.com"},
					DeliveryMode: DeliveryModeGuaranteed,
				},
			}

			SetDefaults_Outputs(outputs)

			Expect(outputs[0].DeliveryMode).To(Equal(DeliveryModeGuaranteed))
			Expect(outputs[1].DeliveryMode).To(Equal(DeliveryModeGuaranteed))
		})

		It("should default only unset delivery modes for multiple outputs", func() {
			outputs := []Output{
				{
					HTTP:         &OutputHTTP{URL: "http://example1.com"},
					DeliveryMode: DeliveryModeGuaranteed,
				},
				{
					HTTP: &OutputHTTP{URL: "http://example2.com"},
					// No delivery mode set
				},
				{
					HTTP:         &OutputHTTP{URL: "http://example3.com"},
					DeliveryMode: DeliveryModeGuaranteed,
				},
			}

			SetDefaults_Outputs(outputs)

			Expect(outputs[0].DeliveryMode).To(Equal(DeliveryModeGuaranteed))
			Expect(outputs[1].DeliveryMode).To(Equal(DeliveryModeBestEffort))
			Expect(outputs[2].DeliveryMode).To(Equal(DeliveryModeGuaranteed))
		})
	})
})

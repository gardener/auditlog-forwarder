// SPDX-FileCopyrightText: SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package validation_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/util/validation/field"

	configv1alpha1 "github.com/gardener/auditlog-forwarder/pkg/apis/config/v1alpha1"
	. "github.com/gardener/auditlog-forwarder/pkg/apis/config/v1alpha1/validation"
)

var _ = Describe("#ValidateAuditlogForwarderConfiguration", func() {
	var config *configv1alpha1.AuditlogForwarderConfiguration

	BeforeEach(func() {
		config = &configv1alpha1.AuditlogForwarderConfiguration{
			Log: configv1alpha1.LogConfiguration{
				Level:  configv1alpha1.LogLevelInfo,
				Format: configv1alpha1.LogFormatJSON,
			},
			Server: configv1alpha1.ServerConfiguration{
				Port: 10443,
				TLS: configv1alpha1.TLSConfig{
					CertFile: "/path/to/cert.pem",
					KeyFile:  "/path/to/key.pem",
				},
			},
		}
	})

	Context("when configuration is valid", func() {
		It("should return no errors", func() {
			errs := ValidateAuditlogForwarderConfiguration(config)
			Expect(errs).To(BeEmpty())
		})
	})

	Context("when server port is missing", func() {
		It("should return an error", func() {
			config.Server.Port = 0

			errs := ValidateAuditlogForwarderConfiguration(config)
			Expect(errs).To(HaveLen(1))
			Expect(errs[0].Type).To(Equal(field.ErrorTypeRequired))
			Expect(errs[0].Field).To(Equal("server.port"))
		})
	})

	Context("when TLS cert file is missing", func() {
		It("should return an error", func() {
			config.Server.TLS.CertFile = ""

			errs := ValidateAuditlogForwarderConfiguration(config)
			Expect(errs).To(HaveLen(1))
			Expect(errs[0].Type).To(Equal(field.ErrorTypeRequired))
			Expect(errs[0].Field).To(Equal("server.tls.certFile"))
		})
	})

	Context("when TLS key file is missing", func() {
		It("should return an error", func() {
			config.Server.TLS.KeyFile = ""

			errs := ValidateAuditlogForwarderConfiguration(config)
			Expect(errs).To(HaveLen(1))
			Expect(errs[0].Type).To(Equal(field.ErrorTypeRequired))
			Expect(errs[0].Field).To(Equal("server.tls.keyFile"))
		})
	})

	Context("when both TLS files are missing", func() {
		It("should return multiple errors", func() {
			config.Server.TLS.CertFile = ""
			config.Server.TLS.KeyFile = ""

			errs := ValidateAuditlogForwarderConfiguration(config)
			Expect(errs).To(HaveLen(2))
		})
	})

	Context("when log level is invalid", func() {
		It("should return an error", func() {
			config.Log.Level = "invalid"

			errs := ValidateAuditlogForwarderConfiguration(config)
			Expect(errs).To(HaveLen(1))
			Expect(errs[0].Type).To(Equal(field.ErrorTypeNotSupported))
			Expect(errs[0].Field).To(Equal("log.level"))
		})
	})

	Context("when log format is invalid", func() {
		It("should return an error", func() {
			config.Log.Format = "invalid"

			errs := ValidateAuditlogForwarderConfiguration(config)
			Expect(errs).To(HaveLen(1))
			Expect(errs[0].Type).To(Equal(field.ErrorTypeNotSupported))
			Expect(errs[0].Field).To(Equal("log.format"))
		})
	})

	Context("when log configuration is empty", func() {
		It("should not return errors (defaults will be applied)", func() {
			config.Log = configv1alpha1.LogConfiguration{}

			errs := ValidateAuditlogForwarderConfiguration(config)
			Expect(errs).To(BeEmpty())
		})
	})
})

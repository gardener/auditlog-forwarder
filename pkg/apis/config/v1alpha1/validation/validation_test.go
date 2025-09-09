// SPDX-FileCopyrightText: SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package validation_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gstruct"
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
			Backends: []configv1alpha1.Backend{
				{
					HTTP: &configv1alpha1.HTTPBackend{
						URL: "https://example.com/audit",
					},
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
			Expect(errs).To(ConsistOf(PointTo(MatchFields(IgnoreExtras, Fields{
				"Type":  Equal(field.ErrorTypeRequired),
				"Field": Equal("server.port"),
			}))))
		})
	})

	Context("when TLS cert file is missing", func() {
		It("should return an error", func() {
			config.Server.TLS.CertFile = ""

			errs := ValidateAuditlogForwarderConfiguration(config)
			Expect(errs).To(ConsistOf(PointTo(MatchFields(IgnoreExtras, Fields{
				"Type":  Equal(field.ErrorTypeRequired),
				"Field": Equal("server.tls.certFile"),
			}))))
		})
	})

	Context("when TLS key file is missing", func() {
		It("should return an error", func() {
			config.Server.TLS.KeyFile = ""

			errs := ValidateAuditlogForwarderConfiguration(config)
			Expect(errs).To(ConsistOf(PointTo(MatchFields(IgnoreExtras, Fields{
				"Type":  Equal(field.ErrorTypeRequired),
				"Field": Equal("server.tls.keyFile"),
			}))))
		})
	})

	Context("when both TLS files are missing", func() {
		It("should return multiple errors", func() {
			config.Server.TLS.CertFile = ""
			config.Server.TLS.KeyFile = ""

			errs := ValidateAuditlogForwarderConfiguration(config)
			Expect(errs).To(ConsistOf(
				PointTo(MatchFields(IgnoreExtras, Fields{
					"Type":  Equal(field.ErrorTypeRequired),
					"Field": Equal("server.tls.certFile"),
				})),
				PointTo(MatchFields(IgnoreExtras, Fields{
					"Type":  Equal(field.ErrorTypeRequired),
					"Field": Equal("server.tls.keyFile"),
				})),
			))
		})
	})

	Context("when log level is invalid", func() {
		It("should return an error", func() {
			config.Log.Level = "invalid"

			errs := ValidateAuditlogForwarderConfiguration(config)
			Expect(errs).To(ConsistOf(PointTo(MatchFields(IgnoreExtras, Fields{
				"Type":  Equal(field.ErrorTypeNotSupported),
				"Field": Equal("log.level"),
			}))))
		})
	})

	Context("when log format is invalid", func() {
		It("should return an error", func() {
			config.Log.Format = "invalid"

			errs := ValidateAuditlogForwarderConfiguration(config)
			Expect(errs).To(ConsistOf(PointTo(MatchFields(IgnoreExtras, Fields{
				"Type":  Equal(field.ErrorTypeNotSupported),
				"Field": Equal("log.format"),
			}))))
		})
	})

	Context("when log configuration is empty", func() {
		It("should not return errors (defaults will be applied)", func() {
			config.Log = configv1alpha1.LogConfiguration{}

			errs := ValidateAuditlogForwarderConfiguration(config)
			Expect(errs).To(BeEmpty())
		})
	})

	Context("backends validation", func() {
		Context("when no backends are configured", func() {
			It("should return an error", func() {
				config.Backends = []configv1alpha1.Backend{}

				errs := ValidateAuditlogForwarderConfiguration(config)
				Expect(errs).To(ConsistOf(PointTo(MatchFields(IgnoreExtras, Fields{
					"Type":  Equal(field.ErrorTypeRequired),
					"Field": Equal("backends"),
				}))))
			})
		})

		Context("when multiple backends are configured", func() {
			It("should return an error", func() {
				config.Backends = []configv1alpha1.Backend{
					{
						HTTP: &configv1alpha1.HTTPBackend{
							URL: "https://example1.com/audit",
						},
					},
					{
						HTTP: &configv1alpha1.HTTPBackend{
							URL: "https://example2.com/audit",
						},
					},
				}

				errs := ValidateAuditlogForwarderConfiguration(config)
				Expect(errs).To(ConsistOf(PointTo(MatchFields(IgnoreExtras, Fields{
					"Type":  Equal(field.ErrorTypeInvalid),
					"Field": Equal("backends"),
				}))))
			})
		})

		Context("when backend has no type specified", func() {
			It("should return an error", func() {
				config.Backends = []configv1alpha1.Backend{
					{
						// No HTTP field specified
					},
				}

				errs := ValidateAuditlogForwarderConfiguration(config)
				Expect(errs).To(ConsistOf(PointTo(MatchFields(IgnoreExtras, Fields{
					"Type":  Equal(field.ErrorTypeRequired),
					"Field": Equal("backends[0]"),
				}))))
			})
		})

		Context("when HTTP backend has empty URL", func() {
			It("should return an error", func() {
				config.Backends[0].HTTP.URL = ""

				errs := ValidateAuditlogForwarderConfiguration(config)
				Expect(errs).To(ConsistOf(PointTo(MatchFields(IgnoreExtras, Fields{
					"Type":  Equal(field.ErrorTypeRequired),
					"Field": Equal("backends[0].http.url"),
				}))))
			})
		})

		Context("when HTTP backend has malformed URL", func() {
			It("should return an error", func() {
				config.Backends[0].HTTP.URL = "://invalid-url"

				errs := ValidateAuditlogForwarderConfiguration(config)
				Expect(errs).To(ConsistOf(PointTo(MatchFields(IgnoreExtras, Fields{
					"Type":  Equal(field.ErrorTypeInvalid),
					"Field": Equal("backends[0].http.url"),
				}))))
			})
		})

		Context("when HTTP backend URL is not HTTPS", func() {
			It("should return an error", func() {
				config.Backends[0].HTTP.URL = "http://example.com/audit"

				errs := ValidateAuditlogForwarderConfiguration(config)
				Expect(errs).To(ConsistOf(PointTo(MatchFields(IgnoreExtras, Fields{
					"Type":   Equal(field.ErrorTypeInvalid),
					"Field":  Equal("backends[0].http.url"),
					"Detail": ContainSubstring("URL scheme must be 'https'"),
				}))))
			})
		})

		Context("when HTTP backend URL contains query parameters", func() {
			It("should return an error", func() {
				config.Backends[0].HTTP.URL = "https://example.com/audit?param=value"

				errs := ValidateAuditlogForwarderConfiguration(config)
				Expect(errs).To(ConsistOf(PointTo(MatchFields(IgnoreExtras, Fields{
					"Type":   Equal(field.ErrorTypeInvalid),
					"Field":  Equal("backends[0].http.url"),
					"Detail": ContainSubstring("URL must not contain query parameters"),
				}))))
			})
		})

		Context("when HTTP backend URL contains fragments", func() {
			It("should return an error", func() {
				config.Backends[0].HTTP.URL = "https://example.com/audit#fragment"

				errs := ValidateAuditlogForwarderConfiguration(config)
				Expect(errs).To(ConsistOf(PointTo(MatchFields(IgnoreExtras, Fields{
					"Type":   Equal(field.ErrorTypeInvalid),
					"Field":  Equal("backends[0].http.url"),
					"Detail": ContainSubstring("URL must not contain fragments"),
				}))))
			})
		})

		Context("when HTTP backend URL contains user information", func() {
			It("should return an error", func() {
				config.Backends[0].HTTP.URL = "https://user:pass@example.com/audit"

				errs := ValidateAuditlogForwarderConfiguration(config)
				Expect(errs).To(ConsistOf(PointTo(MatchFields(IgnoreExtras, Fields{
					"Type":   Equal(field.ErrorTypeInvalid),
					"Field":  Equal("backends[0].http.url"),
					"Detail": ContainSubstring("URL must not contain user information"),
				}))))
			})
		})

		Context("when HTTP backend has valid TLS configuration", func() {
			It("should return no errors", func() {
				config.Backends[0].HTTP.TLS = &configv1alpha1.ClientTLSConfig{
					CAFile:   "/path/to/ca.pem",
					CertFile: "/path/to/client-cert.pem",
					KeyFile:  "/path/to/client-key.pem",
				}

				errs := ValidateAuditlogForwarderConfiguration(config)
				Expect(errs).To(BeEmpty())
			})
		})

		Context("when HTTP backend has only cert file without key file", func() {
			It("should return an error", func() {
				config.Backends[0].HTTP.TLS = &configv1alpha1.ClientTLSConfig{
					CertFile: "/path/to/client-cert.pem",
					// KeyFile is missing
				}

				errs := ValidateAuditlogForwarderConfiguration(config)
				Expect(errs).To(ConsistOf(PointTo(MatchFields(IgnoreExtras, Fields{
					"Type":  Equal(field.ErrorTypeRequired),
					"Field": Equal("backends[0].http.tls.keyFile"),
				}))))
			})
		})

		Context("when HTTP backend has only key file without cert file", func() {
			It("should return an error", func() {
				config.Backends[0].HTTP.TLS = &configv1alpha1.ClientTLSConfig{
					KeyFile: "/path/to/client-key.pem",
				}

				errs := ValidateAuditlogForwarderConfiguration(config)
				Expect(errs).To(ConsistOf(PointTo(MatchFields(IgnoreExtras, Fields{
					"Type":  Equal(field.ErrorTypeRequired),
					"Field": Equal("backends[0].http.tls.certFile"),
				}))))
			})
		})

		Context("when HTTP backend does not configure client authentication", func() {
			It("should return no errors", func() {
				config.Backends[0].HTTP.TLS = &configv1alpha1.ClientTLSConfig{
					CAFile: "/path/to/ca.pem",
				}

				errs := ValidateAuditlogForwarderConfiguration(config)
				Expect(errs).To(BeEmpty())
			})
		})
	})
})

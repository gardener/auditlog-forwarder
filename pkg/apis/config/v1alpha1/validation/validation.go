// SPDX-FileCopyrightText: SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package validation

import (
	"strings"

	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/apimachinery/pkg/util/validation/field"

	configv1alpha1 "github.com/gardener/auditlog-forwarder/pkg/apis/config/v1alpha1"
)

var (
	validLogLevels = sets.NewString(
		configv1alpha1.LogLevelDebug,
		configv1alpha1.LogLevelInfo,
		configv1alpha1.LogLevelError,
	)
	validLogFormats = sets.NewString(
		configv1alpha1.LogFormatJSON,
		configv1alpha1.LogFormatText,
	)
)

// ValidateAuditlogForwarderConfiguration validates the given [*configv1alpha1.AuditlogForwarderConfiguration].
func ValidateAuditlogForwarderConfiguration(cfg *configv1alpha1.AuditlogForwarderConfiguration) field.ErrorList {
	allErrs := field.ErrorList{}

	allErrs = append(allErrs, validateLogConfiguration(&cfg.Log, field.NewPath("log"))...)
	allErrs = append(allErrs, validateServerConfig(&cfg.Server, field.NewPath("server"))...)

	return allErrs
}

// validateLogConfiguration validates the log configuration.
func validateLogConfiguration(logConfig *configv1alpha1.LogConfiguration, fldPath *field.Path) field.ErrorList {
	allErrs := field.ErrorList{}

	if logConfig.Level != "" {
		if !validLogLevels.Has(logConfig.Level) {
			allErrs = append(allErrs, field.NotSupported(fldPath.Child("level"), logConfig.Level, validLogLevels.List()))
		}
	}

	if logConfig.Format != "" {
		if !validLogFormats.Has(logConfig.Format) {
			allErrs = append(allErrs, field.NotSupported(fldPath.Child("format"), logConfig.Format, validLogFormats.List()))
		}
	}

	return allErrs
}

// validateServerConfig validates the server configuration.
func validateServerConfig(serverConfig *configv1alpha1.ServerConfiguration, fldPath *field.Path) field.ErrorList {
	allErrs := field.ErrorList{}

	if serverConfig.Port == 0 {
		allErrs = append(allErrs, field.Required(fldPath.Child("port"), "port is required"))
	}

	allErrs = append(allErrs, validateTLSConfig(&serverConfig.TLS, fldPath.Child("tls"))...)

	return allErrs
}

// validateTLSConfig validates the TLS configuration.
func validateTLSConfig(tlsConfig *configv1alpha1.TLSConfig, fldPath *field.Path) field.ErrorList {
	allErrs := field.ErrorList{}

	if strings.TrimSpace(tlsConfig.CertFile) == "" {
		allErrs = append(allErrs, field.Required(fldPath.Child("certFile"), "TLS certificate file is required"))
	}

	if strings.TrimSpace(tlsConfig.KeyFile) == "" {
		allErrs = append(allErrs, field.Required(fldPath.Child("keyFile"), "TLS private key file is required"))
	}

	return allErrs
}

// SPDX-FileCopyrightText: SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package validation

import (
	"net/url"
	"strings"

	apivalidation "k8s.io/apimachinery/pkg/api/validation"
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
	allErrs = append(allErrs, validateBackends(cfg.Backends, field.NewPath("backends"))...)
	allErrs = append(allErrs, validateInjectAnnotations(cfg.InjectAnnotations, field.NewPath("injectAnnotations"))...)

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

// validateBackends validates the backends configuration.
func validateBackends(backends []configv1alpha1.Backend, fldPath *field.Path) field.ErrorList {
	allErrs := field.ErrorList{}

	if len(backends) == 0 {
		allErrs = append(allErrs, field.Required(fldPath, "at least one backend must be configured"))
		return allErrs
	}

	// TODO: remove this limitation in the future
	if len(backends) != 1 {
		allErrs = append(allErrs, field.Invalid(fldPath, len(backends), "exactly one backend must be configured"))
		return allErrs
	}

	for i, backend := range backends {
		backendPath := fldPath.Index(i)
		allErrs = append(allErrs, validateBackend(&backend, backendPath)...)
	}

	return allErrs
}

// validateBackend validates a single backend configuration.
func validateBackend(backend *configv1alpha1.Backend, fldPath *field.Path) field.ErrorList {
	allErrs := field.ErrorList{}

	// Count the number of backend types configured
	backendTypes := 0
	if backend.HTTP != nil {
		backendTypes++
	}

	if backendTypes == 0 {
		allErrs = append(allErrs, field.Required(fldPath, "backend type must be specified (currently only 'http' is supported)"))
		return allErrs
	}

	if backendTypes > 1 {
		allErrs = append(allErrs, field.Invalid(fldPath, backendTypes, "exactly one backend type must be specified"))
		return allErrs
	}

	if backend.HTTP != nil {
		allErrs = append(allErrs, validateHTTPBackend(backend.HTTP, fldPath.Child("http"))...)
	}

	return allErrs
}

// validateHTTPBackend validates the HTTP backend configuration.
func validateHTTPBackend(httpBackend *configv1alpha1.HTTPBackend, fldPath *field.Path) field.ErrorList {
	allErrs := field.ErrorList{}

	urlValue := strings.TrimSpace(httpBackend.URL)
	if urlValue == "" {
		allErrs = append(allErrs, field.Required(fldPath.Child("url"), "URL is required for HTTP backend"))
	} else {
		// Validate URL format
		if backendURL, err := url.Parse(urlValue); err != nil {
			allErrs = append(allErrs, field.Invalid(fldPath.Child("url"), urlValue, "invalid URL format"))
		} else {
			if backendURL.Scheme != "https" {
				allErrs = append(allErrs, field.Invalid(fldPath.Child("url"), urlValue, "URL scheme must be 'https'"))
			}

			if backendURL.RawQuery != "" {
				allErrs = append(allErrs, field.Invalid(fldPath.Child("url"), urlValue, "URL must not contain query parameters"))
			}

			if backendURL.Fragment != "" {
				allErrs = append(allErrs, field.Invalid(fldPath.Child("url"), urlValue, "URL must not contain fragments"))
			}

			if backendURL.User != nil {
				allErrs = append(allErrs, field.Invalid(fldPath.Child("url"), urlValue, "URL must not contain user information"))
			}
		}
	}

	if httpBackend.TLS != nil {
		allErrs = append(allErrs, validateClientTLSConfig(httpBackend.TLS, fldPath.Child("tls"))...)
	}

	return allErrs
}

// validateClientTLSConfig validates the client TLS configuration.
func validateClientTLSConfig(tlsConfig *configv1alpha1.ClientTLSConfig, fldPath *field.Path) field.ErrorList {
	allErrs := field.ErrorList{}

	// Both certFile and keyFile must be specified together for client authentication
	certFileSpecified := strings.TrimSpace(tlsConfig.CertFile) != ""
	keyFileSpecified := strings.TrimSpace(tlsConfig.KeyFile) != ""

	if certFileSpecified && !keyFileSpecified {
		allErrs = append(allErrs, field.Required(fldPath.Child("keyFile"), "keyFile is required when certFile is specified"))
	}

	if !certFileSpecified && keyFileSpecified {
		allErrs = append(allErrs, field.Required(fldPath.Child("certFile"), "certFile is required when keyFile is specified"))
	}

	return allErrs
}

// validateInjectAnnotations validates the inject annotations configuration.
func validateInjectAnnotations(annotations map[string]string, fldPath *field.Path) field.ErrorList {
	allErrs := field.ErrorList{}

	allErrs = append(allErrs, apivalidation.ValidateAnnotations(annotations, fldPath)...)

	for key, value := range annotations {
		if strings.TrimSpace(value) == "" {
			keyPath := fldPath.Key(key)
			allErrs = append(allErrs, field.Required(keyPath, "annotation value cannot be empty"))
		}
	}

	return allErrs
}

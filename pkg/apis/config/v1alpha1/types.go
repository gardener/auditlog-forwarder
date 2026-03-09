// SPDX-FileCopyrightText: SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	// LogLevelDebug is the debug log level, i.e. the most verbose.
	LogLevelDebug = "debug"
	// LogLevelInfo is the default log level.
	LogLevelInfo = "info"
	// LogLevelError is a log level where only errors are logged.
	LogLevelError = "error"

	// LogFormatJSON is the output type that produces a JSON object per log line.
	LogFormatJSON = "json"
	// LogFormatText outputs the log as human-readable text.
	LogFormatText = "text"
)

// DeliveryMode defines how messages are delivered to an output.
type DeliveryMode string

const (
	// DeliveryModeGuaranteed indicates that delivery to this output is required for request success.
	// Messages will be retried on failure.
	DeliveryModeGuaranteed DeliveryMode = "Guaranteed"
	// DeliveryModeBestEffort indicates that delivery is attempted but failures don't affect request success.
	// Messages may be delivered multiple times or not at all.
	DeliveryModeBestEffort DeliveryMode = "BestEffort"
)

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// AuditlogForwarder defines the configuration for the audit log forwarder.
type AuditlogForwarder struct {
	metav1.TypeMeta `json:",inline"`

	// Log contains the logging configuration for the audit log forwarder.
	Log Log `json:"log"`
	// Server contains the server configuration for the audit log forwarder.
	Server Server `json:"server"`
	// Outputs contains the list of outputs to forward audit logs to.
	Outputs []Output `json:"outputs"`
	// InjectAnnotations contains annotations to be injected into audit events.
	// +optional
	InjectAnnotations map[string]string `json:"injectAnnotations,omitempty"`
}

// Log defines the logging configuration for the audit log forwarder.
type Log struct {
	// Level is the level/severity for the logs. Must be one of [info,debug,error].
	// +optional
	Level string `json:"level,omitempty"`
	// Format is the output format for the logs. Must be one of [text,json].
	// +optional
	Format string `json:"format,omitempty"`
}

// Server defines the server configuration for the audit log forwarder.
type Server struct {
	// Port is the port that the server will listen on.
	// Defaults to 10443.
	// +optional
	Port int32 `json:"port,omitempty"`
	// MetricsPort is the port that the server will listen on to serve metrics.
	// Defaults to 8080.
	// +optional
	MetricsPort int32 `json:"metricsPort,omitempty"`
	// Address is the IP address that the server will listen on.
	// If unspecified all interfaces will be used.
	// +optional
	Address string `json:"address,omitempty"`
	// TLS contains the TLS configuration for the server.
	TLS TLS `json:"tls"`
}

// TLS defines the TLS configuration for the server.
type TLS struct {
	// CertFile is the file containing the x509 Certificate for HTTPS.
	CertFile string `json:"certFile"`
	// KeyFile is the file containing the x509 private key matching the certificate.
	KeyFile string `json:"keyFile"`
	// ClientCAFile is the file containing the Certificate Authority to verify client certificates.
	// If specified, client certificate verification will be enabled with RequireAndVerifyClientCert policy.
	// +optional
	ClientCAFile string `json:"clientCAFile,omitempty"`
}

// Output defines an output to forward audit logs to.
type Output struct {
	// DeliveryMode specifies how messages are delivered to this output.
	// "Guaranteed" means the request is considered successful only if this output succeeds.
	// "BestEffort" means delivery is attempted but failures don't affect request success.
	// When only one output is configured, it is implicitly "Guaranteed".
	// When multiple outputs are configured, exactly one must be "Guaranteed".
	// +optional
	DeliveryMode DeliveryMode `json:"deliveryMode,omitempty"`
	// HTTP contains the HTTP output configuration.
	// +optional
	HTTP *OutputHTTP `json:"http,omitempty"`
}

// OutputHTTP defines the configuration for an HTTP output.
type OutputHTTP struct {
	// URL is the endpoint URL to send audit logs to.
	URL string `json:"url"`
	// TLS contains the TLS configuration for client.
	// +optional
	TLS *ClientTLS `json:"tls,omitempty"`
	// Compression defines the compression algorithm to use for the HTTP request body.
	// Currently only "gzip" is supported. If empty, no compression is applied.
	// +optional
	Compression string `json:"compression,omitempty"`
}

// ClientTLS defines the TLS configuration for client.
type ClientTLS struct {
	// CAFile is the file containing the Certificate Authority to verify the server certificate.
	// +optional
	CAFile string `json:"caFile,omitempty"`
	// CertFile is the file containing the client certificate for mutual TLS.
	// +optional
	CertFile string `json:"certFile,omitempty"`
	// KeyFile is the file containing the client private key for mutual TLS.
	// +optional
	KeyFile string `json:"keyFile,omitempty"`
}

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
	// +optional
	Port uint `json:"port,omitempty"`
	// Address is the IP address that the server will listen on.
	// If unspecified all interfaces will be used.
	// +optional
	Address string `json:"address,omitempty"`
	// TLS contains the TLS configuration for the server.
	TLS TLS `json:"tls"`
	// MetricsPort is the port that the server will listen on for metrics.
	// +optional
	MetricsPort uint `json:"metricsPort,omitempty"`
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

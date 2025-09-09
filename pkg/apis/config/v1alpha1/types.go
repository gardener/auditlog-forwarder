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

// AuditlogForwarderConfiguration defines the configuration for the audit log forwarder.
type AuditlogForwarderConfiguration struct {
	metav1.TypeMeta `json:",inline"`

	// Log contains the logging configuration for the audit log forwarder.
	Log LogConfiguration `json:"log"`
	// Server contains the server configuration for the audit log forwarder.
	Server ServerConfiguration `json:"server"`
	// Backends contains the list of backends to forward audit logs to.
	Backends []Backend `json:"backends"`
	// InjectAnnotations contains annotations to be injected into audit events.
	// +optional
	InjectAnnotations map[string]string `json:"injectAnnotations,omitempty"`
}

// LogConfiguration defines the logging configuration for the audit log forwarder.
type LogConfiguration struct {
	// Level is the level/severity for the logs. Must be one of [info,debug,error].
	// +optional
	Level string `json:"level,omitempty"`
	// Format is the output format for the logs. Must be one of [text,json].
	// +optional
	Format string `json:"format,omitempty"`
}

// ServerConfiguration defines the server configuration for the audit log forwarder.
type ServerConfiguration struct {
	// Port is the port that the server will listen on.
	// +optional
	Port uint `json:"port,omitempty"`
	// Address is the IP address that the server will listen on.
	// If unspecified all interfaces will be used.
	// +optional
	Address string `json:"address,omitempty"`
	// TLS contains the TLS configuration for the server.
	TLS TLSConfig `json:"tls"`
}

// TLSConfig defines the TLS configuration for the server.
type TLSConfig struct {
	// CertFile is the file containing the x509 Certificate for HTTPS.
	CertFile string `json:"certFile"`
	// KeyFile is the file containing the x509 private key matching the certificate.
	KeyFile string `json:"keyFile"`
}

// Backend defines a backend to forward audit logs to.
type Backend struct {
	// HTTP contains the HTTP backend configuration.
	// +optional
	HTTP *HTTPBackend `json:"http,omitempty"`
}

// HTTPBackend defines the configuration for an HTTP backend.
type HTTPBackend struct {
	// URL is the endpoint URL to send audit logs to.
	URL string `json:"url"`
	// TLS contains the TLS configuration for client.
	// +optional
	TLS *ClientTLSConfig `json:"tls,omitempty"`
}

// ClientTLSConfig defines the TLS configuration for client.
type ClientTLSConfig struct {
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

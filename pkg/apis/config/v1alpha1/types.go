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

// SPDX-FileCopyrightText: SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package v1alpha1

// SetDefaults_AuditlogForwarder sets defaults for the configuration of the audit log forwarder.
func SetDefaults_AuditlogForwarder(obj *AuditlogForwarder) {
	SetDefaults_Log(&obj.Log)
	SetDefaults_Server(&obj.Server)
}

// SetDefaults_Log sets defaults for the logging configuration.
func SetDefaults_Log(obj *Log) {
	if obj.Level == "" {
		obj.Level = LogLevelInfo
	}
	if obj.Format == "" {
		obj.Format = LogFormatJSON
	}
}

// SetDefaults_Server sets defaults for the server configuration.
func SetDefaults_Server(obj *Server) {
	if obj.Port == 0 {
		obj.Port = 10443
	}
}

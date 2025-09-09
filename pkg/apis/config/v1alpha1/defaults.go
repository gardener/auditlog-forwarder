// SPDX-FileCopyrightText: SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package v1alpha1

// SetDefaults_AuditlogForwarderConfiguration sets defaults for the configuration of the audit log forwarder.
func SetDefaults_AuditlogForwarderConfiguration(obj *AuditlogForwarderConfiguration) {
	SetDefaults_LogConfiguration(&obj.Log)
	SetDefaults_ServerConfiguration(&obj.Server)
}

// SetDefaults_LogConfiguration sets defaults for the logging configuration.
func SetDefaults_LogConfiguration(obj *LogConfiguration) {
	if obj.Level == "" {
		obj.Level = LogLevelInfo
	}
	if obj.Format == "" {
		obj.Format = LogFormatJSON
	}
}

// SetDefaults_ServerConfiguration sets defaults for the server configuration.
func SetDefaults_ServerConfiguration(obj *ServerConfiguration) {
	if obj.Port == 0 {
		obj.Port = 10443
	}
}

// SPDX-FileCopyrightText: SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package v1alpha1

// SetDefaults_AuditlogForwarderConfiguration sets defaults for the configuration of the audit log forwarder.
func SetDefaults_AuditlogForwarderConfiguration(obj *AuditlogForwarderConfiguration) {
	if obj.LogLevel == "" {
		obj.LogLevel = LogLevelInfo
	}
	if obj.LogFormat == "" {
		obj.LogFormat = LogFormatJSON
	}
}

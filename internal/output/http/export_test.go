// SPDX-FileCopyrightText: SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package http

import (
	"time"

	configv1alpha1 "github.com/gardener/auditlog-forwarder/pkg/apis/config/v1alpha1"
)

var (
	BackoffFunc = backoffDuration
	SleepFunc   = sleepWithContext
)

// SetTLSReloadDebounceForTest overrides the debounce duration for TLS reload and returns a cleanup function.
func SetTLSReloadDebounceForTest(d time.Duration) func() {
	old := tlsReloadDebounce
	tlsReloadDebounce = d
	return func() { tlsReloadDebounce = old }
}

// ReloadTLSClientForTest triggers a synchronous TLS client reload with the given config.
func (o *Output) ReloadTLSClientForTest(tlsConfig *configv1alpha1.ClientTLS) {
	o.reloadTLSClient(tlsConfig)
}

// SPDX-FileCopyrightText: SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

const (
	namespace          = "auditlog_forwarder"
	subsystemReceived  = "received"
	subsystemSucceeded = "succeeded"
	subsystemFailed    = "failed"
	name               = "total"
)

var (
	AuditReceived = promauto.NewCounter(prometheus.CounterOpts{
		Namespace: namespace,
		Subsystem: subsystemReceived,
		Name:      name,
		Help:      "Total number of received audit requests.",
	})

	AuditSucceeded = promauto.NewCounter(prometheus.CounterOpts{
		Namespace: namespace,
		Subsystem: subsystemSucceeded,
		Name:      name,
		Help:      "Total number of successfully processed audit requests.",
	})

	AuditFailed = promauto.NewCounter(prometheus.CounterOpts{
		Namespace: namespace,
		Subsystem: subsystemFailed,
		Name:      name,
		Help:      "Total number of failed processed audit requests.",
	})
)

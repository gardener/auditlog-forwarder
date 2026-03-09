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
	subsystemOutput    = "output"
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

	OutputSucceeded = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: namespace,
		Subsystem: subsystemOutput,
		Name:      "succeeded_total",
		Help:      "Total number of successful sends per output.",
	}, []string{"output", "delivery_mode"})

	OutputFailed = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: namespace,
		Subsystem: subsystemOutput,
		Name:      "failed_total",
		Help:      "Total number of failed sends per output.",
	}, []string{"output", "delivery_mode"})
)

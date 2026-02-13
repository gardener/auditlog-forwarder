// SPDX-FileCopyrightText: SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package http

var (
	BackoffFunc = backoffDuration
	SleepFunc   = sleepWithContext
)

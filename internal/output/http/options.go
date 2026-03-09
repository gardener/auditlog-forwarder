// SPDX-FileCopyrightText: SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package http

import "time"

// Option is a functional option for configuring an HTTP Output.
type Option func(*Output) error

// WithMaxSendAttempts sets the maximum number of attempts to send data to the HTTP endpoint.
// This includes the initial attempt plus any retries.
func WithMaxSendAttempts(attempts int) Option {
	return func(o *Output) error {
		o.maxSendAttempts = attempts
		return nil
	}
}

// WithBaseBackoff sets the initial backoff duration between retry attempts.
// The actual backoff may be calculated based on this value for exponential backoff strategies.
func WithBaseBackoff(backoff time.Duration) Option {
	return func(o *Output) error {
		o.baseBackoff = backoff
		return nil
	}
}

// WithMaxBackoff sets the maximum backoff duration between retry attempts.
// This caps the backoff duration to prevent excessively long wait times.
func WithMaxBackoff(backoff time.Duration) Option {
	return func(o *Output) error {
		o.maxBackoff = backoff
		return nil
	}
}

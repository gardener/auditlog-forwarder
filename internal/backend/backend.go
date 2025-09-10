// SPDX-FileCopyrightText: SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package backend

import (
	"context"
)

// Backend represents a backend for forwarding audit events.
type Backend interface {
	// Send sends data to the backend.
	// The context may contain a logger.
	Send(ctx context.Context, data []byte) error

	// Name returns a human-readable name/identifier for this backend.
	Name() string
}

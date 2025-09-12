// SPDX-FileCopyrightText: SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package output

import (
	"context"
)

// Output represents an output for forwarding audit events.
type Output interface {
	// Send sends data to the output.
	// The context may contain a logger.
	Send(ctx context.Context, data []byte) error

	// Name returns a human-readable name/identifier for this output.
	Name() string
}

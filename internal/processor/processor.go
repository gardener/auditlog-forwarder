// SPDX-FileCopyrightText: Contributors to the Gardener project
//
// SPDX-License-Identifier: Apache-2.0

package processor

import "context"

// Processor processes audit event data.
type Processor interface {
	// Process takes audit event data as input and returns processed data.
	// The context may contain a logger.
	Process(ctx context.Context, data []byte) ([]byte, error)

	// Name returns the name of the processor.
	Name() string
}

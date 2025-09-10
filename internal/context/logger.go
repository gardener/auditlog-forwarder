// SPDX-FileCopyrightText: SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package context

import (
	"context"

	"github.com/go-logr/logr"
)

// contextKey is the key type for storing logger in context
type contextKey string

const loggerKey contextKey = "logger"

// WithLogger adds a logger to the context.
func WithLogger(ctx context.Context, logger logr.Logger) context.Context {
	return context.WithValue(ctx, loggerKey, logger)
}

// LoggerFromContext retrieves the logger from the context.
// If no logger is found, it returns a discard logger.
func LoggerFromContext(ctx context.Context) logr.Logger {
	if logger, ok := ctx.Value(loggerKey).(logr.Logger); ok {
		return logger
	}
	return logr.Discard()
}

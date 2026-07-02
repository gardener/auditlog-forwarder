// SPDX-FileCopyrightText: SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package factory

import (
	"context"
	"errors"
	"fmt"

	"github.com/gardener/auditlog-forwarder/internal/output"
	"github.com/gardener/auditlog-forwarder/internal/output/http"
	configv1alpha1 "github.com/gardener/auditlog-forwarder/pkg/apis/config/v1alpha1"
)

// NewHTTPOutputsWithOptions filters outputs by delivery mode and creates HTTP outputs with the given options.
// It extracts only HTTP outputs matching the specified delivery mode and configures them with the provided options.
func NewHTTPOutputsWithOptions(ctx context.Context, allOutputs []configv1alpha1.Output, deliveryMode configv1alpha1.DeliveryMode, httpOpts ...http.Option) ([]output.Output, error) {
	var outputs []output.Output
	for _, outputConfig := range allOutputs {
		if outputConfig.HTTP != nil && outputConfig.DeliveryMode == deliveryMode {
			httpOutput, err := http.New(ctx, outputConfig.HTTP, httpOpts...)
			if err != nil {
				// Preserve the primary cause but surface any secondary damage from
				// closing outputs we already built.
				return nil, errors.Join(fmt.Errorf("failed to create HTTP output: %w", err), closeOutputs(outputs))
			}
			outputs = append(outputs, httpOutput)
		}
	}

	return outputs, nil
}

// closeOutputs releases resources of the given outputs, joining any errors.
func closeOutputs(outputs []output.Output) error {
	var errs []error
	for _, o := range outputs {
		if err := o.Close(); err != nil {
			errs = append(errs, fmt.Errorf("failed to close output %q: %w", o.Name(), err))
		}
	}
	return errors.Join(errs...)
}

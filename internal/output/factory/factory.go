// SPDX-FileCopyrightText: SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package factory

import (
	"fmt"

	"github.com/gardener/auditlog-forwarder/internal/output"
	"github.com/gardener/auditlog-forwarder/internal/output/http"
	configv1alpha1 "github.com/gardener/auditlog-forwarder/pkg/apis/config/v1alpha1"
)

// NewHttpOutputsWithOptions filters outputs by delivery mode and creates HTTP outputs with the given options.
// It extracts only HTTP outputs matching the specified delivery mode and configures them with the provided options.
func NewHttpOutputsWithOptions(allOutputs []configv1alpha1.Output, deliveryMode configv1alpha1.DeliveryMode, httpOpts ...http.Option) ([]output.Output, error) {
	var outputs []output.Output
	for _, outputConfig := range allOutputs {
		if outputConfig.HTTP != nil && outputConfig.DeliveryMode == deliveryMode {
			if http, err := http.New(outputConfig.HTTP, httpOpts...); err == nil {
				outputs = append(outputs, http)
			} else {
				return nil, fmt.Errorf("failed to create HTTP output: %w", err)
			}
		}
	}

	return outputs, nil
}

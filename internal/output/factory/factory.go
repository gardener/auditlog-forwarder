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

// NewFromConfig creates a output from the given configuration.
func NewFromConfig(config configv1alpha1.Output) (output.Output, error) {
	if config.HTTP != nil {
		return http.New(config.HTTP)
	}

	return nil, fmt.Errorf("no supported output type configured")
}

// NewFromConfigs creates a slice of outputs from the given configurations.
func NewFromConfigs(configs []configv1alpha1.Output) ([]output.Output, error) {
	var outputs []output.Output

	for i, config := range configs {
		output, err := NewFromConfig(config)
		if err != nil {
			return nil, fmt.Errorf("failed to create output at index %d: %w", i, err)
		}
		outputs = append(outputs, output)
	}

	return outputs, nil
}

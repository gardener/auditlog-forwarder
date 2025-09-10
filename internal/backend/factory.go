// SPDX-FileCopyrightText: SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package backend

import (
	"fmt"

	"github.com/go-logr/logr"

	"github.com/gardener/auditlog-forwarder/internal/backend/http"
	configv1alpha1 "github.com/gardener/auditlog-forwarder/pkg/apis/config/v1alpha1"
)

// NewFromConfig creates a backend from the given configuration.
func NewFromConfig(config configv1alpha1.Backend, logger logr.Logger) (Backend, error) {
	if config.HTTP != nil {
		return http.New(config.HTTP)
	}

	return nil, fmt.Errorf("no supported backend type configured")
}

// NewFromConfigs creates a slice of backends from the given configurations.
func NewFromConfigs(configs []configv1alpha1.Backend, logger logr.Logger) ([]Backend, error) {
	var backends []Backend

	for i, config := range configs {
		backend, err := NewFromConfig(config, logger)
		if err != nil {
			return nil, fmt.Errorf("failed to create backend at index %d: %w", i, err)
		}
		backends = append(backends, backend)
	}

	return backends, nil
}

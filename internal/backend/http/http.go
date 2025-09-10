// SPDX-FileCopyrightText: SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package http

import (
	"bytes"
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	loggerctx "github.com/gardener/auditlog-forwarder/internal/context"
	configv1alpha1 "github.com/gardener/auditlog-forwarder/pkg/apis/config/v1alpha1"
)

const (
	headerContentType = "Content-Type"
	mimeAppJSON       = "application/json"
)

// Backend represents an HTTP backend for forwarding audit events.
type Backend struct {
	url    string
	client *http.Client
}

// New creates a new HTTP backend with the given configuration.
func New(config *configv1alpha1.HTTPBackend) (*Backend, error) {
	if config == nil {
		return nil, fmt.Errorf("HTTP backend configuration is nil")
	}

	client, err := createHTTPClient(config.TLS)
	if err != nil {
		return nil, fmt.Errorf("failed to create HTTP client: %w", err)
	}

	return &Backend{
		url:    config.URL,
		client: client,
	}, nil
}

// Send sends data to the HTTP backend.
func (b *Backend) Send(ctx context.Context, data []byte) error {
	logger := loggerctx.LoggerFromContext(ctx).WithName("http-backend").WithValues("url", b.url)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, b.url, bytes.NewReader(data))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set(headerContentType, mimeAppJSON)

	resp, err := b.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			logger.Error(err, "failed closing body")
		}
	}()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		if body, err := io.ReadAll(resp.Body); err != nil {
			logger.Error(err, "failed reading body")
		} else {
			return fmt.Errorf("backend returned status %d: %s", resp.StatusCode, string(body))
		}
	}

	return nil
}

// Name returns the URL of this HTTP backend.
func (b *Backend) Name() string {
	return b.url
}

// createHTTPClient creates an HTTP client with optional TLS configuration.
func createHTTPClient(tlsConfig *configv1alpha1.ClientTLSConfig) (*http.Client, error) {
	client := &http.Client{
		Timeout: 15 * time.Second,
	}

	if tlsConfig == nil {
		return client, nil
	}

	transport := &http.Transport{
		TLSClientConfig: &tls.Config{
			MinVersion: tls.VersionTLS12,
		},
	}

	if tlsConfig.CAFile != "" {
		caCert, err := os.ReadFile(tlsConfig.CAFile)
		if err != nil {
			return nil, fmt.Errorf("failed to read CA certificate file: %w", err)
		}

		caCertPool := x509.NewCertPool()
		if !caCertPool.AppendCertsFromPEM(caCert) {
			return nil, fmt.Errorf("failed to parse CA certificate")
		}
		transport.TLSClientConfig.RootCAs = caCertPool
	}

	if tlsConfig.CertFile != "" && tlsConfig.KeyFile != "" {
		cert, err := tls.LoadX509KeyPair(tlsConfig.CertFile, tlsConfig.KeyFile)
		if err != nil {
			return nil, fmt.Errorf("failed to load client certificate: %w", err)
		}
		transport.TLSClientConfig.Certificates = []tls.Certificate{cert}
	}

	client.Transport = transport
	return client, nil
}

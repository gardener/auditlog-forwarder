// SPDX-FileCopyrightText: SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package http

import (
	"bytes"
	"compress/gzip"
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"

	loggerctx "github.com/gardener/auditlog-forwarder/internal/context"
	configv1alpha1 "github.com/gardener/auditlog-forwarder/pkg/apis/config/v1alpha1"
)

const (
	headerContentType = "Content-Type"
	mimeAppJSON       = "application/json"

	headerContentEncoding = "Content-Encoding"
	contentEncodingGzip   = "gzip"
)

// Output represents an HTTP output for forwarding audit events.
type Output struct {
	url    string
	client *http.Client
	// compression algorithm to use (currently only "gzip" or empty for none)
	compression string
}

// New creates a new HTTP output with the given configuration.
func New(config *configv1alpha1.OutputHTTP) (*Output, error) {
	if config == nil {
		return nil, fmt.Errorf("HTTP output configuration is nil")
	}

	client, err := createHTTPClient(config.TLS)
	if err != nil {
		return nil, fmt.Errorf("failed to create HTTP client: %w", err)
	}

	return &Output{
		url:         config.URL,
		client:      client,
		compression: config.Compression,
	}, nil
}

// Send sends data to the HTTP output.
func (o *Output) Send(ctx context.Context, data []byte) error {
	logger := loggerctx.LoggerFromContext(ctx).WithName("http").WithValues("url", o.url)

	var bodyReader io.Reader
	if o.compression == contentEncodingGzip {
		var buf bytes.Buffer
		gz := gzip.NewWriter(&buf)
		if _, err := gz.Write(data); err != nil {
			return fmt.Errorf("failed to gzip data: %w", err)
		}
		if err := gz.Close(); err != nil { // flush
			return fmt.Errorf("failed to finalize gzip writer: %w", err)
		}
		bodyReader = &buf
	} else {
		bodyReader = bytes.NewReader(data)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, o.url, bodyReader)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set(headerContentType, mimeAppJSON)
	if o.compression == contentEncodingGzip {
		req.Header.Set(headerContentEncoding, contentEncodingGzip)
	}

	resp, err := o.client.Do(req)
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
			return fmt.Errorf("output returned status %d: %s", resp.StatusCode, string(body))
		}
	}

	return nil
}

// Name returns the URL of this HTTP output.
func (o *Output) Name() string {
	return o.url
}

// createHTTPClient creates an HTTP client with optional TLS configuration.
func createHTTPClient(tlsConfig *configv1alpha1.ClientTLS) (*http.Client, error) {
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
		caCert, err := os.ReadFile(filepath.Clean(tlsConfig.CAFile))
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

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

	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/util/wait"

	loggerctx "github.com/gardener/auditlog-forwarder/internal/context"
	"github.com/gardener/auditlog-forwarder/internal/output"
	configv1alpha1 "github.com/gardener/auditlog-forwarder/pkg/apis/config/v1alpha1"
)

const (
	headerContentType = "Content-Type"
	mimeAppJSON       = "application/json"

	headerContentEncoding = "Content-Encoding"
	contentEncodingGzip   = "gzip"
)

var _ output.Output = (*Output)(nil)

// Output represents an HTTP output for forwarding audit events.
type Output struct {
	url    string
	client *http.Client
	// compression algorithm to use (currently only "gzip" or empty for none)
	compression string

	maxSendAttempts int
	baseBackoff     time.Duration
	maxBackoff      time.Duration
}

// New creates a new HTTP output with the given configuration.
func New(config *configv1alpha1.OutputHTTP, options ...Option) (*Output, error) {
	if config == nil {
		return nil, fmt.Errorf("HTTP output configuration is nil")
	}

	client, err := createHTTPClient(config.TLS)
	if err != nil {
		return nil, fmt.Errorf("failed to create HTTP client: %w", err)
	}

	output := &Output{
		url:             config.URL,
		client:          client,
		compression:     config.Compression,
		maxSendAttempts: 4,
		baseBackoff:     500 * time.Millisecond,
		maxBackoff:      3 * time.Second,
	}

	for _, opt := range options {
		if err := opt(output); err != nil {
			return nil, fmt.Errorf("failed to apply option: %w", err)
		}
	}

	return output, nil
}

// Send sends data to the HTTP output.
func (o *Output) Send(ctx context.Context, data []byte) error {
	logger := loggerctx.LoggerFromContext(ctx).WithName("http").WithValues("url", o.url)

	payload := data
	if o.compression == contentEncodingGzip {
		var buf bytes.Buffer
		gz := gzip.NewWriter(&buf)
		if _, err := gz.Write(data); err != nil {
			// call gz.Close for the sake of completeness
			// ignore the error as this would probably be the same error as the error returned by gz.Write
			_ = gz.Close()
			return fmt.Errorf("failed to gzip data: %w", err)
		}
		// explicitly close the writer in order to make it flush residual data and write the gzip footer
		// we do not use defer here because we want to write all data to the buffer before passing it to the http request
		if err := gz.Close(); err != nil { // flush
			return fmt.Errorf("failed to finalize gzip writer: %w", err)
		}
		payload = buf.Bytes()
	}

	var lastErr error
	for attempt := 1; attempt <= o.maxSendAttempts; attempt++ {
		bodyReader := bytes.NewReader(payload)
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
			lastErr = fmt.Errorf("failed to send request: %w", err)
		} else {
			body, readErr := readAndCloseBody(resp, logger)
			if readErr != nil {
				return readErr
			}

			if resp.StatusCode >= 200 && resp.StatusCode < 300 {
				return nil
			}

			reqErr := fmt.Errorf("output returned status %d: %s", resp.StatusCode, string(body))
			if !isRetryableStatus(resp.StatusCode) {
				return reqErr
			}
			lastErr = reqErr
		}

		if attempt < o.maxSendAttempts {
			if err := sleepWithContext(ctx, backoffDuration(attempt, o.baseBackoff, o.maxBackoff)); err != nil {
				return fmt.Errorf("request canceled while retrying: %w, previous attempt failed with: %w", err, lastErr)
			}
		}
	}

	return lastErr
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

func isRetryableStatus(statusCode int) bool {
	return statusCode == http.StatusTooManyRequests || statusCode >= http.StatusInternalServerError
}

func readAndCloseBody(resp *http.Response, logger logr.Logger) ([]byte, error) {
	defer func() {
		if err := resp.Body.Close(); err != nil {
			logger.Error(err, "failed closing body")
		}
	}()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	return body, nil
}

func backoffDuration(attempt int, baseBackoff, maxBackoff time.Duration) time.Duration {
	if attempt <= 1 {
		return baseBackoff
	}

	backoff := baseBackoff * time.Duration(1<<int64(attempt-1))
	return wait.Jitter(min(backoff, maxBackoff), 0.05)
}

func sleepWithContext(ctx context.Context, d time.Duration) error {
	select {
	case <-time.After(d):
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

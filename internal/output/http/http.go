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
	"sync/atomic"
	"time"

	"github.com/fsnotify/fsnotify"
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

	// defaultTLSReloadDebounce is the default delay after a filesystem event before reloading TLS credentials.
	// Kubernetes secret updates produce multiple events in rapid succession; this coalesces them.
	defaultTLSReloadDebounce = 500 * time.Millisecond
)

var _ output.Output = (*Output)(nil)

// Output represents an HTTP output for forwarding audit events.
type Output struct {
	url    string
	client atomic.Pointer[http.Client]
	// compression algorithm to use (currently only "gzip" or empty for none)
	compression string

	maxSendAttempts int
	baseBackoff     time.Duration
	maxBackoff      time.Duration

	// tlsReloadDebounce is the delay before reloading TLS credentials after a filesystem event
	tlsReloadDebounce time.Duration
	// logger is used for logging TLS reload events in the background watcher
	logger logr.Logger
	// watcher is the fsnotify watcher for TLS credential files (nil if TLS is not configured)
	watcher *fsnotify.Watcher
}

// New creates a new HTTP output with the given configuration.
// The context controls the lifetime of the TLS credential file watcher.
func New(ctx context.Context, config *configv1alpha1.OutputHTTP, options ...Option) (*Output, error) {
	if config == nil {
		return nil, fmt.Errorf("HTTP output configuration is nil")
	}

	client, err := createHTTPClient(config.TLS)
	if err != nil {
		return nil, fmt.Errorf("failed to create HTTP client: %w", err)
	}

	o := &Output{
		url:               config.URL,
		compression:       config.Compression,
		maxSendAttempts:   4,
		baseBackoff:       500 * time.Millisecond,
		maxBackoff:        3 * time.Second,
		tlsReloadDebounce: defaultTLSReloadDebounce,
		logger:            logr.Discard(),
	}
	o.client.Store(client)

	for _, opt := range options {
		if err := opt(o); err != nil {
			return nil, fmt.Errorf("failed to apply option: %w", err)
		}
	}

	if config.TLS != nil {
		if err := o.startTLSWatcher(ctx, config.TLS); err != nil {
			return nil, fmt.Errorf("failed to start TLS file watcher: %w", err)
		}
	}

	return o, nil
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

		resp, err := o.client.Load().Do(req)
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

// Close triggers shutdown of the TLS file watcher goroutine and releases resources.
// It is safe to call multiple times.
func (o *Output) Close() error {
	if o.watcher == nil {
		return nil
	}
	err := o.watcher.Close()
	o.watcher = nil
	return err
}

// startTLSWatcher begins watching the directories containing TLS credential files.
// When files change, the HTTP client is rebuilt with freshly-loaded credentials.
func (o *Output) startTLSWatcher(ctx context.Context, tlsConfig *configv1alpha1.ClientTLS) error {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return fmt.Errorf("failed to create file watcher: %w", err)
	}

	// Watch the parent directories of all configured TLS files.
	// This handles Kubernetes secret mounts where files are symlinks that get atomically swapped.
	dirs := tlsDirectories(tlsConfig)
	for _, dir := range dirs {
		if err := watcher.Add(dir); err != nil {
			_ = watcher.Close()
			return fmt.Errorf("failed to watch directory %s: %w", dir, err)
		}
	}

	o.watcher = watcher

	go o.watchTLSFiles(ctx, tlsConfig)
	return nil
}

// watchTLSFiles is the event loop for the TLS file watcher.
func (o *Output) watchTLSFiles(ctx context.Context, tlsConfig *configv1alpha1.ClientTLS) {
	var debounceTimer *time.Timer

	for {
		select {
		case <-ctx.Done():
			if debounceTimer != nil {
				debounceTimer.Stop()
			}
			return

		case event, ok := <-o.watcher.Events:
			if !ok {
				return
			}
			// Only react to events that indicate file content changed
			if !event.Has(fsnotify.Write) && !event.Has(fsnotify.Create) && !event.Has(fsnotify.Remove) {
				continue
			}

			// Debounce: reset timer on each event
			if debounceTimer != nil {
				debounceTimer.Stop()
			}
			debounceTimer = time.AfterFunc(o.tlsReloadDebounce, func() {
				o.reloadTLSClient(tlsConfig)
			})

		case err, ok := <-o.watcher.Errors:
			if !ok {
				return
			}
			o.logger.Error(err, "File watcher error")
		}
	}
}

// reloadTLSClient rebuilds the HTTP client with freshly-loaded TLS credentials.
// On failure, the existing client is kept.
func (o *Output) reloadTLSClient(tlsConfig *configv1alpha1.ClientTLS) {
	client, err := createHTTPClient(tlsConfig)
	if err != nil {
		o.logger.Error(err, "Failed to reload TLS credentials, keeping existing client")
		return
	}
	o.client.Store(client)
	o.logger.Info("Reloaded TLS credentials")
}

// tlsDirectories returns the unique parent directories of all configured TLS files.
func tlsDirectories(tlsConfig *configv1alpha1.ClientTLS) []string {
	seen := make(map[string]struct{})
	var dirs []string

	for _, file := range []string{tlsConfig.CAFile, tlsConfig.CertFile, tlsConfig.KeyFile} {
		if file == "" {
			continue
		}
		dir := filepath.Dir(file)
		if _, ok := seen[dir]; !ok {
			seen[dir] = struct{}{}
			dirs = append(dirs, dir)
		}
	}

	return dirs
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
		caCertPool, err := loadCACertPool(tlsConfig.CAFile)
		if err != nil {
			return nil, err
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

// loadCACertPool reads a PEM-encoded CA certificate file and returns a cert pool.
func loadCACertPool(caFile string) (*x509.CertPool, error) {
	caCert, err := os.ReadFile(filepath.Clean(caFile))
	if err != nil {
		return nil, fmt.Errorf("failed to read CA certificate file: %w", err)
	}

	caCertPool := x509.NewCertPool()
	if !caCertPool.AppendCertsFromPEM(caCert) {
		return nil, fmt.Errorf("failed to parse CA certificate")
	}
	return caCertPool, nil
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

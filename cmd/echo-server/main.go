// SPDX-FileCopyrightText: SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"crypto/tls"
	"crypto/x509"
	"flag"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

var (
	logBody     = flag.Bool("log-body", false, "Should the request body be logged")
	tlsCertFile = flag.String("tls-cert-file", "", "Path to TLS certificate file (required for HTTPS)")
	tlsKeyFile  = flag.String("tls-key-file", "", "Path to TLS private key file (required for HTTPS)")
	tlsCAFile   = flag.String("tls-ca-file", "", "Path to TLS CA certificate file for client certificate verification (enables mTLS)")
	healthPort  = flag.Int("health-port", 8080, "Port for HTTP health endpoint")
)

func main() {
	port := flag.Int("port", 8000, "Port to listen on")
	flag.Parse()

	// Start health server in a goroutine
	healthSrv := &http.Server{
		Handler:           http.HandlerFunc(healthHandler),
		Addr:              ":" + fmt.Sprint(*healthPort),
		WriteTimeout:      15 * time.Second,
		ReadTimeout:       15 * time.Second,
		ReadHeaderTimeout: 15 * time.Second,
	}

	go func() {
		slog.Info("Starting HTTP health server", "port", *healthPort)
		if err := healthSrv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("Health server failed", "error", err)
			os.Exit(1)
		}
	}()

	// Main server for audit endpoints
	srv := &http.Server{
		Handler:           http.HandlerFunc(auditHandler),
		Addr:              ":" + fmt.Sprint(*port),
		WriteTimeout:      15 * time.Second,
		ReadTimeout:       15 * time.Second,
		ReadHeaderTimeout: 15 * time.Second,
	}

	if *tlsCertFile != "" && *tlsKeyFile != "" {
		tlsConfig, err := setupTLSConfig(*tlsCertFile, *tlsKeyFile, *tlsCAFile)
		if err != nil {
			slog.Error("Failed to setup TLS configuration", "error", err)
			os.Exit(1)
		}
		srv.TLSConfig = tlsConfig

		if *tlsCAFile != "" {
			slog.Info("Starting HTTPS echo server with mTLS", "port", *port)
		} else {
			slog.Info("Starting HTTPS echo server", "port", *port)
		}
		if err := srv.ListenAndServeTLS("", ""); err != nil && err != http.ErrServerClosed {
			slog.Error("HTTPS echo server failed", "error", err)
			os.Exit(1)
		}
		slog.Info("HTTPS echo server shutdown completed")
		os.Exit(0)
	}

	slog.Info("Starting HTTP echo server", "port", *port)
	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		slog.Error("HTTP echo server failed", "error", err)
		os.Exit(1)
	}
	slog.Info("HTTP echo server shutdown completed")
	os.Exit(0)
}

// setupTLSConfig creates a TLS configuration with optional client certificate verification
func setupTLSConfig(certFile, keyFile, caFile string) (*tls.Config, error) {
	// Load server certificate and key
	serverCert, err := tls.LoadX509KeyPair(certFile, keyFile)
	if err != nil {
		return nil, fmt.Errorf("failed to load server certificate and key: %w", err)
	}

	tlsConfig := &tls.Config{
		Certificates: []tls.Certificate{serverCert},
		MinVersion:   tls.VersionTLS12,
	}

	// If CA file is provided, enable client certificate verification
	if caFile != "" {
		caCert, err := os.ReadFile(filepath.Clean(caFile))
		if err != nil {
			return nil, fmt.Errorf("failed to read CA certificate file: %w", err)
		}

		caCertPool := x509.NewCertPool()
		if !caCertPool.AppendCertsFromPEM(caCert) {
			return nil, fmt.Errorf("failed to parse CA certificate")
		}

		tlsConfig.ClientCAs = caCertPool
		tlsConfig.ClientAuth = tls.RequireAndVerifyClientCert

		slog.Info("Client certificate verification enabled", "caFilePath", caFile)
	}

	return tlsConfig, nil
}

// healthHandler handles only health check requests on HTTP
func healthHandler(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path == "/health" {
		w.Header().Add("Content-Type", "application/json")
		_, err := w.Write([]byte(`{"status": "ok"}`))
		if err != nil {
			slog.Error("Failed to write health response", "error", err)
		}
		return
	}

	// For non-health paths, return 404
	http.NotFound(w, r)
}

// auditHandler handles audit requests (renamed from genericHandler)
func auditHandler(w http.ResponseWriter, r *http.Request) {
	logArgs := []any{"method", r.Method, "path", r.URL.Path}

	// Log client certificate information if present
	if r.TLS != nil && len(r.TLS.PeerCertificates) > 0 {
		cert := r.TLS.PeerCertificates[0]
		logArgs = append(logArgs, "client", cert.Subject.CommonName, "issuer", cert.Issuer.CommonName)
	}

	if *logBody && r.Method == http.MethodPost {
		defer func() {
			if err := r.Body.Close(); err != nil {
				slog.Info("Error closing body:", "error", err)
			}
		}()
		body, err := io.ReadAll(r.Body)
		if err != nil {
			w.Header().Add("Content-Type", "application/json")
			http.Error(w, `{"success": false}`, http.StatusInternalServerError)
			return
		}

		logArgs = append(logArgs, "body", string(body))
	}

	slog.Info("Handled request", logArgs...) //#nosec // G706: echo server intentionally logs request data, slog takes care of escaping special symbols to avoid log injection

	w.Header().Add("Content-Type", "application/json")
	_, err := w.Write([]byte(`{"success": true}`))
	if err != nil {
		slog.Info("Failed to write response", "error", err)
	}
}

// SPDX-FileCopyrightText: SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package http_test

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	httpoutput "github.com/gardener/auditlog-forwarder/internal/output/http"
	configv1alpha1 "github.com/gardener/auditlog-forwarder/pkg/apis/config/v1alpha1"
)

var _ = Describe("TLS Hot Reload", func() {
	var (
		tmpDir     string
		ctx        context.Context
		cancel     context.CancelFunc
		httpOutput *httpoutput.Output
	)

	BeforeEach(func() {
		var err error
		tmpDir, err = os.MkdirTemp("", "tls-reload-test-*")
		Expect(err).NotTo(HaveOccurred())

		ctx, cancel = context.WithCancel(context.Background())
	})

	AfterEach(func() {
		cancel()
		if httpOutput != nil {
			Expect(httpOutput.Close()).To(Succeed())
		}
		Expect(os.RemoveAll(tmpDir)).To(Succeed())
	})

	It("should reload client certificates when files change", func() {
		// Generate CA
		caKey, caCert, caPEM := generateCA("Test CA")

		// Generate initial client cert
		clientCertPEM1, clientKeyPEM1 := generateClientCert(caKey, caCert, "client-1")

		// Write initial certs to files
		caFile := filepath.Join(tmpDir, "ca.crt")
		certFile := filepath.Join(tmpDir, "client.crt")
		keyFile := filepath.Join(tmpDir, "client.key")

		Expect(os.WriteFile(caFile, caPEM, 0600)).To(Succeed())
		Expect(os.WriteFile(certFile, clientCertPEM1, 0600)).To(Succeed())
		Expect(os.WriteFile(keyFile, clientKeyPEM1, 0600)).To(Succeed())

		// Create a TLS test server that requires client certs and captures the subject
		caCertPool := x509.NewCertPool()
		caCertPool.AddCert(caCert)

		var receivedSubject string
		testServer := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.TLS != nil && len(r.TLS.PeerCertificates) > 0 {
				receivedSubject = r.TLS.PeerCertificates[0].Subject.CommonName
			}
			w.WriteHeader(http.StatusOK)
		}))
		testServer.TLS = &tls.Config{
			ClientAuth: tls.RequireAndVerifyClientCert,
			ClientCAs:  caCertPool,
			MinVersion: tls.VersionTLS12,
		}
		testServer.StartTLS()
		defer testServer.Close()

		// Extract server's CA for the client to trust
		serverCAPEM := pem.EncodeToMemory(&pem.Block{
			Type:  "CERTIFICATE",
			Bytes: testServer.TLS.Certificates[0].Certificate[0],
		})
		serverCAFile := filepath.Join(tmpDir, "server-ca.crt")
		Expect(os.WriteFile(serverCAFile, serverCAPEM, 0600)).To(Succeed())

		// Create output with TLS config and short debounce for testing
		config := &configv1alpha1.OutputHTTP{
			URL: testServer.URL,
			TLS: &configv1alpha1.ClientTLS{
				CAFile:   serverCAFile,
				CertFile: certFile,
				KeyFile:  keyFile,
			},
		}

		var err error
		httpOutput, err = httpoutput.New(ctx, config, httpoutput.WithTLSReloadDebounce(50*time.Millisecond))
		Expect(err).NotTo(HaveOccurred())

		// First request should use client-1 cert
		Expect(httpOutput.Send(context.Background(), []byte(`{"test": 1}`))).To(Succeed())
		Expect(receivedSubject).To(Equal("client-1"))

		// Generate a new client cert with a different CN
		clientCertPEM2, clientKeyPEM2 := generateClientCert(caKey, caCert, "client-2")

		// Write new certs to the same files
		Expect(os.WriteFile(certFile, clientCertPEM2, 0600)).To(Succeed())
		Expect(os.WriteFile(keyFile, clientKeyPEM2, 0600)).To(Succeed())

		// Wait for the watcher to pick up the change (debounce + processing)
		Eventually(func() string {
			_ = httpOutput.Send(context.Background(), []byte(`{"test": 2}`))
			return receivedSubject
		}, 2*time.Second, 100*time.Millisecond).Should(Equal("client-2"))
	})

	It("should reload CA certificate when file changes", func() {
		// Generate two different CAs
		caKey1, caCert1, caPEM1 := generateCA("Test CA 1")
		caKey2, caCert2, caPEM2 := generateCA("Test CA 2")

		// Generate server certs signed by each CA
		serverCert1 := generateServerCert(caKey1, caCert1, "127.0.0.1")
		serverCert2 := generateServerCert(caKey2, caCert2, "127.0.0.1")

		// Start with CA 1 as the trusted CA on the client side
		caFile := filepath.Join(tmpDir, "ca.crt")
		Expect(os.WriteFile(caFile, caPEM1, 0600)).To(Succeed())

		// Create a test server using cert signed by CA 1
		testServer := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
		}))
		testServer.TLS = &tls.Config{
			Certificates: []tls.Certificate{serverCert1},
			MinVersion:   tls.VersionTLS12,
		}
		testServer.StartTLS()
		defer testServer.Close()

		// Create output trusting CA 1 with short debounce for testing
		config := &configv1alpha1.OutputHTTP{
			URL: testServer.URL,
			TLS: &configv1alpha1.ClientTLS{
				CAFile: caFile,
			},
		}

		var err error
		httpOutput, err = httpoutput.New(ctx, config, httpoutput.WithTLSReloadDebounce(50*time.Millisecond))
		Expect(err).NotTo(HaveOccurred())

		// Request should succeed (server cert signed by CA 1, client trusts CA 1)
		Expect(httpOutput.Send(context.Background(), []byte(`{"test": 1}`))).To(Succeed())

		// Now switch the server to use a cert signed by CA 2
		testServer.Close()

		testServer2 := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
		}))
		testServer2.TLS = &tls.Config{
			Certificates: []tls.Certificate{serverCert2},
			MinVersion:   tls.VersionTLS12,
		}
		testServer2.Listener, err = net.Listen("tcp", testServer.Listener.Addr().String())
		Expect(err).NotTo(HaveOccurred())
		testServer2.StartTLS()
		defer testServer2.Close()

		// Request should fail (server cert signed by CA 2, client still trusts CA 1)
		err = httpOutput.Send(context.Background(), []byte(`{"test": 2}`))
		Expect(err).To(HaveOccurred())

		// Update the CA file to trust CA 2
		Expect(os.WriteFile(caFile, caPEM2, 0600)).To(Succeed())

		// Wait for the watcher to reload, then verify requests succeed
		Eventually(func() error {
			return httpOutput.Send(context.Background(), []byte(`{"test": 3}`))
		}, 2*time.Second, 100*time.Millisecond).Should(Succeed())
	})

	It("should keep the old client when new cert files are invalid", func() {
		// Generate CA and valid client cert
		caKey, caCert, caPEM := generateCA("Test CA")
		clientCertPEM, clientKeyPEM := generateClientCert(caKey, caCert, "valid-client")

		caFile := filepath.Join(tmpDir, "ca.crt")
		certFile := filepath.Join(tmpDir, "client.crt")
		keyFile := filepath.Join(tmpDir, "client.key")

		Expect(os.WriteFile(caFile, caPEM, 0600)).To(Succeed())
		Expect(os.WriteFile(certFile, clientCertPEM, 0600)).To(Succeed())
		Expect(os.WriteFile(keyFile, clientKeyPEM, 0600)).To(Succeed())

		// Create a TLS test server
		caCertPool := x509.NewCertPool()
		caCertPool.AddCert(caCert)

		testServer := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
		}))
		testServer.TLS = &tls.Config{
			ClientAuth: tls.RequireAndVerifyClientCert,
			ClientCAs:  caCertPool,
			MinVersion: tls.VersionTLS12,
		}
		testServer.StartTLS()
		defer testServer.Close()

		// Extract server CA
		serverCAPEM := pem.EncodeToMemory(&pem.Block{
			Type:  "CERTIFICATE",
			Bytes: testServer.TLS.Certificates[0].Certificate[0],
		})
		serverCAFile := filepath.Join(tmpDir, "server-ca.crt")
		Expect(os.WriteFile(serverCAFile, serverCAPEM, 0600)).To(Succeed())

		config := &configv1alpha1.OutputHTTP{
			URL: testServer.URL,
			TLS: &configv1alpha1.ClientTLS{
				CAFile:   serverCAFile,
				CertFile: certFile,
				KeyFile:  keyFile,
			},
		}

		var err error
		httpOutput, err = httpoutput.New(ctx, config, httpoutput.WithTLSReloadDebounce(50*time.Millisecond))
		Expect(err).NotTo(HaveOccurred())

		// Initial request succeeds
		Expect(httpOutput.Send(context.Background(), []byte(`{"test": 1}`))).To(Succeed())

		// Write invalid cert data
		Expect(os.WriteFile(certFile, []byte("not a valid cert"), 0600)).To(Succeed())

		// Verify requests keep succeeding (old client kept on reload failure)
		Consistently(func() error {
			return httpOutput.Send(context.Background(), []byte(`{"test": 2}`))
		}, 500*time.Millisecond, 50*time.Millisecond).Should(Succeed())
	})
})

// generateCA creates a self-signed CA certificate.
func generateCA(cn string) (*ecdsa.PrivateKey, *x509.Certificate, []byte) {
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	Expect(err).NotTo(HaveOccurred())

	template := &x509.Certificate{
		SerialNumber:          big.NewInt(1),
		Subject:               pkix.Name{CommonName: cn},
		NotBefore:             time.Now().Add(-1 * time.Hour),
		NotAfter:              time.Now().Add(24 * time.Hour),
		IsCA:                  true,
		KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageCRLSign,
		BasicConstraintsValid: true,
	}

	certDER, err := x509.CreateCertificate(rand.Reader, template, template, &key.PublicKey, key)
	Expect(err).NotTo(HaveOccurred())

	cert, err := x509.ParseCertificate(certDER)
	Expect(err).NotTo(HaveOccurred())

	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: certDER})
	return key, cert, certPEM
}

// generateClientCert creates a client certificate signed by the given CA.
func generateClientCert(caKey *ecdsa.PrivateKey, caCert *x509.Certificate, cn string) (certPEM, keyPEM []byte) {
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	Expect(err).NotTo(HaveOccurred())

	template := &x509.Certificate{
		SerialNumber: big.NewInt(time.Now().UnixNano()),
		Subject:      pkix.Name{CommonName: cn},
		NotBefore:    time.Now().Add(-1 * time.Hour),
		NotAfter:     time.Now().Add(24 * time.Hour),
		KeyUsage:     x509.KeyUsageDigitalSignature,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
	}

	certDER, err := x509.CreateCertificate(rand.Reader, template, caCert, &key.PublicKey, caKey)
	Expect(err).NotTo(HaveOccurred())

	certPEM = pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: certDER})

	keyDER, err := x509.MarshalECPrivateKey(key)
	Expect(err).NotTo(HaveOccurred())
	keyPEM = pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: keyDER})

	return certPEM, keyPEM
}

// generateServerCert creates a server certificate signed by the given CA for the specified IP.
func generateServerCert(caKey *ecdsa.PrivateKey, caCert *x509.Certificate, ip string) tls.Certificate {
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	Expect(err).NotTo(HaveOccurred())

	template := &x509.Certificate{
		SerialNumber: big.NewInt(time.Now().UnixNano()),
		Subject:      pkix.Name{CommonName: ip},
		NotBefore:    time.Now().Add(-1 * time.Hour),
		NotAfter:     time.Now().Add(24 * time.Hour),
		KeyUsage:     x509.KeyUsageDigitalSignature,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		IPAddresses:  []net.IP{net.ParseIP(ip)},
	}

	certDER, err := x509.CreateCertificate(rand.Reader, template, caCert, &key.PublicKey, caKey)
	Expect(err).NotTo(HaveOccurred())

	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: certDER})

	keyDER, err := x509.MarshalECPrivateKey(key)
	Expect(err).NotTo(HaveOccurred())
	keyPEM := pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: keyDER})

	tlsCert, err := tls.X509KeyPair(certPEM, keyPEM)
	Expect(err).NotTo(HaveOccurred())

	return tlsCert
}

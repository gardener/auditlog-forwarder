// SPDX-FileCopyrightText: SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package factory_test

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"errors"
	"math/big"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/gardener/auditlog-forwarder/internal/output"
	"github.com/gardener/auditlog-forwarder/internal/output/factory"
	httpoutput "github.com/gardener/auditlog-forwarder/internal/output/http"
	configv1alpha1 "github.com/gardener/auditlog-forwarder/pkg/apis/config/v1alpha1"
)

var _ = Describe("Output Factory", func() {
	var testServer *httptest.Server

	BeforeEach(func() {
		testServer = httptest.NewServer(nil)
	})

	AfterEach(func() {
		if testServer != nil {
			testServer.Close()
		}
	})

	Describe("NewHttpOutputsWithOptions", func() {
		It("should create HTTP outputs with Guaranteed delivery mode", func() {
			outputs := []configv1alpha1.Output{
				{
					DeliveryMode: configv1alpha1.DeliveryModeGuaranteed,
					HTTP: &configv1alpha1.OutputHTTP{
						URL: testServer.URL,
					},
				},
				{
					DeliveryMode: configv1alpha1.DeliveryModeBestEffort,
					HTTP: &configv1alpha1.OutputHTTP{
						URL: testServer.URL + "/other",
					},
				},
			}

			result, err := factory.NewHTTPOutputsWithOptions(context.Background(), outputs, configv1alpha1.DeliveryModeGuaranteed)
			Expect(err).NotTo(HaveOccurred())
			Expect(result).To(HaveLen(1))
			Expect(result[0].Name()).To(Equal(testServer.URL))
		})

		It("should create HTTP outputs with BestEffort delivery mode", func() {
			outputs := []configv1alpha1.Output{
				{
					DeliveryMode: configv1alpha1.DeliveryModeGuaranteed,
					HTTP: &configv1alpha1.OutputHTTP{
						URL: testServer.URL,
					},
				},
				{
					DeliveryMode: configv1alpha1.DeliveryModeBestEffort,
					HTTP: &configv1alpha1.OutputHTTP{
						URL: testServer.URL + "/best-effort",
					},
				},
			}

			result, err := factory.NewHTTPOutputsWithOptions(context.Background(), outputs, configv1alpha1.DeliveryModeBestEffort)
			Expect(err).NotTo(HaveOccurred())
			Expect(result).To(HaveLen(1))
			Expect(result[0].Name()).To(Equal(testServer.URL + "/best-effort"))
		})

		It("should filter out non-HTTP outputs", func() {
			outputs := []configv1alpha1.Output{
				{
					DeliveryMode: configv1alpha1.DeliveryModeGuaranteed,
					HTTP:         nil, // non-HTTP output
				},
				{
					DeliveryMode: configv1alpha1.DeliveryModeGuaranteed,
					HTTP: &configv1alpha1.OutputHTTP{
						URL: testServer.URL,
					},
				},
			}

			result, err := factory.NewHTTPOutputsWithOptions(context.Background(), outputs, configv1alpha1.DeliveryModeGuaranteed)
			Expect(err).NotTo(HaveOccurred())
			Expect(result).To(HaveLen(1))
		})

		It("should filter by delivery mode", func() {
			outputs := []configv1alpha1.Output{
				{
					DeliveryMode: configv1alpha1.DeliveryModeGuaranteed,
					HTTP: &configv1alpha1.OutputHTTP{
						URL: testServer.URL + "/guaranteed-1",
					},
				},
				{
					DeliveryMode: configv1alpha1.DeliveryModeGuaranteed,
					HTTP: &configv1alpha1.OutputHTTP{
						URL: testServer.URL + "/guaranteed-2",
					},
				},
				{
					DeliveryMode: configv1alpha1.DeliveryModeBestEffort,
					HTTP: &configv1alpha1.OutputHTTP{
						URL: testServer.URL + "/best-effort",
					},
				},
			}

			result, err := factory.NewHTTPOutputsWithOptions(context.Background(), outputs, configv1alpha1.DeliveryModeGuaranteed)
			Expect(err).NotTo(HaveOccurred())
			Expect(result).To(HaveLen(2))
			Expect(result[0].Name()).To(Equal(testServer.URL + "/guaranteed-1"))
			Expect(result[1].Name()).To(Equal(testServer.URL + "/guaranteed-2"))
		})

		It("should return empty slice when no outputs match", func() {
			outputs := []configv1alpha1.Output{
				{
					DeliveryMode: configv1alpha1.DeliveryModeBestEffort,
					HTTP: &configv1alpha1.OutputHTTP{
						URL: testServer.URL,
					},
				},
			}

			result, err := factory.NewHTTPOutputsWithOptions(context.Background(), outputs, configv1alpha1.DeliveryModeGuaranteed)
			Expect(err).NotTo(HaveOccurred())
			Expect(result).To(BeEmpty())
		})

		It("should handle empty outputs slice", func() {
			outputs := []configv1alpha1.Output{}

			result, err := factory.NewHTTPOutputsWithOptions(context.Background(), outputs, configv1alpha1.DeliveryModeGuaranteed)
			Expect(err).NotTo(HaveOccurred())
			Expect(result).To(BeEmpty())
		})

		It("should apply HTTP options to created outputs", func() {
			outputs := []configv1alpha1.Output{
				{
					DeliveryMode: configv1alpha1.DeliveryModeGuaranteed,
					HTTP: &configv1alpha1.OutputHTTP{
						URL: testServer.URL,
					},
				},
			}

			result, err := factory.NewHTTPOutputsWithOptions(
				context.Background(),
				outputs,
				configv1alpha1.DeliveryModeGuaranteed,
				httpoutput.WithMaxSendAttempts(10),
				httpoutput.WithBaseBackoff(2*time.Second),
				httpoutput.WithMaxBackoff(20*time.Second),
			)
			Expect(err).NotTo(HaveOccurred())
			Expect(result).To(HaveLen(1))
			Expect(result[0].Name()).To(Equal(testServer.URL))
		})

		It("should apply different options to multiple outputs", func() {
			outputs := []configv1alpha1.Output{
				{
					DeliveryMode: configv1alpha1.DeliveryModeGuaranteed,
					HTTP: &configv1alpha1.OutputHTTP{
						URL: testServer.URL + "/output-1",
					},
				},
				{
					DeliveryMode: configv1alpha1.DeliveryModeGuaranteed,
					HTTP: &configv1alpha1.OutputHTTP{
						URL: testServer.URL + "/output-2",
					},
				},
			}

			result, err := factory.NewHTTPOutputsWithOptions(
				context.Background(),
				outputs,
				configv1alpha1.DeliveryModeGuaranteed,
				httpoutput.WithMaxSendAttempts(5),
			)
			Expect(err).NotTo(HaveOccurred())
			Expect(result).To(HaveLen(2))
			Expect(result[0].Name()).To(Equal(testServer.URL + "/output-1"))
			Expect(result[1].Name()).To(Equal(testServer.URL + "/output-2"))
		})

		It("should return error when HTTP output creation fails", func() {
			outputs := []configv1alpha1.Output{
				{
					DeliveryMode: configv1alpha1.DeliveryModeGuaranteed,
					HTTP: &configv1alpha1.OutputHTTP{
						URL: testServer.URL,
						TLS: &configv1alpha1.ClientTLS{
							CAFile: "/nonexistent/ca.pem",
						},
					},
				},
			}

			result, err := factory.NewHTTPOutputsWithOptions(context.Background(), outputs, configv1alpha1.DeliveryModeGuaranteed)
			Expect(err).To(HaveOccurred())
			Expect(err).To(MatchError(ContainSubstring("failed to create HTTP output")))
			Expect(result).To(BeNil())
		})

		It("should handle mixed HTTP and non-HTTP outputs with filtering", func() {
			outputs := []configv1alpha1.Output{
				{
					DeliveryMode: configv1alpha1.DeliveryModeGuaranteed,
					HTTP:         nil, // non-HTTP
				},
				{
					DeliveryMode: configv1alpha1.DeliveryModeGuaranteed,
					HTTP: &configv1alpha1.OutputHTTP{
						URL: testServer.URL + "/http-1",
					},
				},
				{
					DeliveryMode: configv1alpha1.DeliveryModeBestEffort,
					HTTP: &configv1alpha1.OutputHTTP{
						URL: testServer.URL + "/http-2",
					},
				},
				{
					DeliveryMode: configv1alpha1.DeliveryModeGuaranteed,
					HTTP: &configv1alpha1.OutputHTTP{
						URL: testServer.URL + "/http-3",
					},
				},
			}

			result, err := factory.NewHTTPOutputsWithOptions(context.Background(), outputs, configv1alpha1.DeliveryModeGuaranteed)
			Expect(err).NotTo(HaveOccurred())
			Expect(result).To(HaveLen(2))
			Expect(result[0].Name()).To(Equal(testServer.URL + "/http-1"))
			Expect(result[1].Name()).To(Equal(testServer.URL + "/http-3"))
		})

		It("should close already-created outputs when a later one fails", func() {
			// Write a valid CA file so the first output can be built (its TLS setup
			// spawns a watcher goroutine); the second output references a nonexistent
			// CA file and must fail, at which point the first output should be closed
			// and its goroutine reaped rather than leaked.
			tmpDir, err := os.MkdirTemp("", "factory-cleanup-*")
			Expect(err).NotTo(HaveOccurred())
			defer func() { _ = os.RemoveAll(tmpDir) }()

			caFile := filepath.Join(tmpDir, "ca.crt")
			Expect(os.WriteFile(caFile, generateTestCAPEM(), 0600)).To(Succeed())

			outputs := []configv1alpha1.Output{
				{
					DeliveryMode: configv1alpha1.DeliveryModeGuaranteed,
					HTTP: &configv1alpha1.OutputHTTP{
						URL: testServer.URL + "/ok",
						TLS: &configv1alpha1.ClientTLS{CAFile: caFile},
					},
				},
				{
					DeliveryMode: configv1alpha1.DeliveryModeGuaranteed,
					HTTP: &configv1alpha1.OutputHTTP{
						URL: testServer.URL + "/broken",
						TLS: &configv1alpha1.ClientTLS{CAFile: "/nonexistent/ca.pem"},
					},
				},
			}

			// Sample the baseline after any test-framework goroutines have settled.
			Eventually(runtime.NumGoroutine).WithTimeout(500 * time.Millisecond).Should(BeNumerically(">", 0))
			baseline := runtime.NumGoroutine()

			result, err := factory.NewHTTPOutputsWithOptions(context.Background(), outputs, configv1alpha1.DeliveryModeGuaranteed)
			Expect(err).To(HaveOccurred())
			Expect(err).To(MatchError(ContainSubstring("failed to create HTTP output")))
			Expect(result).To(BeNil())

			// The first output's watcher goroutine must have been closed; the goroutine
			// count should return to (approximately) baseline. Some slack is allowed
			// for scheduler / GC goroutines that come and go around the test boundary.
			Eventually(runtime.NumGoroutine).WithTimeout(2 * time.Second).WithPolling(20 * time.Millisecond).
				Should(BeNumerically("<=", baseline+1),
					"watcher goroutine of successfully-created output should have been closed on partial failure")
		})
	})

	Describe("CloseOutputs", func() {
		It("should return nil when all outputs close successfully", func() {
			outputs := []output.Output{
				&fakeOutput{name: "a"},
				&fakeOutput{name: "b"},
			}
			Expect(factory.CloseOutputs(outputs)).To(Succeed())
		})

		It("should join errors from failing closes and continue closing the rest", func() {
			errA := errors.New("boom-A")
			errC := errors.New("boom-C")
			outB := &fakeOutput{name: "b"}
			outputs := []output.Output{
				&fakeOutput{name: "a", closeErr: errA},
				outB,
				&fakeOutput{name: "c", closeErr: errC},
			}

			err := factory.CloseOutputs(outputs)
			Expect(err).To(HaveOccurred())
			// Both underlying errors must be wrapped in the joined result — this is
			// what guarantees a leaking fsnotify handle (or similar) is not silently
			// swallowed when NewHTTPOutputsWithOptions fails partway through.
			Expect(errors.Is(err, errA)).To(BeTrue(), "joined error must wrap errA")
			Expect(errors.Is(err, errC)).To(BeTrue(), "joined error must wrap errC")
			Expect(err.Error()).To(ContainSubstring(`"a"`))
			Expect(err.Error()).To(ContainSubstring(`"c"`))
			// A single failing Close must NOT short-circuit the rest — every output
			// still gets its Close call.
			Expect(outB.closed).To(BeTrue(), "close of intermediate output must still happen")
		})
	})
})

// generateTestCAPEM returns a minimal self-signed CA in PEM form, suitable
// for satisfying x509.CertPool.AppendCertsFromPEM in tests. Kept local to
// this file so the factory package's test does not pull in the full cert
// helpers from the http package's test file.
func generateTestCAPEM() []byte {
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	Expect(err).NotTo(HaveOccurred())

	template := &x509.Certificate{
		SerialNumber:          big.NewInt(1),
		Subject:               pkix.Name{CommonName: "factory-test-ca"},
		NotBefore:             time.Now().Add(-1 * time.Hour),
		NotAfter:              time.Now().Add(1 * time.Hour),
		IsCA:                  true,
		KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageCRLSign,
		BasicConstraintsValid: true,
	}
	der, err := x509.CreateCertificate(rand.Reader, template, template, &key.PublicKey, key)
	Expect(err).NotTo(HaveOccurred())

	return pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der})
}

// fakeOutput is a stub output.Output used to exercise closeOutputs' error
// aggregation without spinning up real HTTP outputs.
type fakeOutput struct {
	name     string
	closeErr error
	closed   bool
}

func (f *fakeOutput) Send(_ context.Context, _ []byte) error { return nil }
func (f *fakeOutput) Name() string                           { return f.name }
func (f *fakeOutput) Close() error {
	f.closed = true
	return f.closeErr
}

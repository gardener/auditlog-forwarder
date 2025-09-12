// SPDX-FileCopyrightText: SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package options

import (
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strconv"

	"github.com/spf13/pflag"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"

	"github.com/gardener/auditlog-forwarder/internal/backend"
	backendfactory "github.com/gardener/auditlog-forwarder/internal/backend/factory"
	configv1alpha1 "github.com/gardener/auditlog-forwarder/pkg/apis/config/v1alpha1"
	"github.com/gardener/auditlog-forwarder/pkg/apis/config/v1alpha1/validation"
)

var configDecoder runtime.Decoder

func init() {
	configScheme := runtime.NewScheme()
	utilruntime.Must(configv1alpha1.AddToScheme(configScheme))
	configDecoder = serializer.NewCodecFactory(configScheme).UniversalDecoder()
}

// Options contain the server options.
type Options struct {
	ConfigFile string
	Config     *configv1alpha1.AuditlogForwarder
}

// NewOptions return options with default values.
func NewOptions() *Options {
	opts := &Options{}
	return opts
}

// AddFlags adds server options to flagset
func (o *Options) AddFlags(fs *pflag.FlagSet) {
	fs.StringVar(&o.ConfigFile, "config", o.ConfigFile, "Path to configuration file.")
}

// Complete loads the configuration from file and applies defaults.
func (o *Options) Complete() error {
	if len(o.ConfigFile) == 0 {
		return errors.New("missing config file")
	}

	data, err := os.ReadFile(filepath.Clean(o.ConfigFile))
	if err != nil {
		return fmt.Errorf("error reading config file: %w", err)
	}

	o.Config = &configv1alpha1.AuditlogForwarder{}
	if err = runtime.DecodeInto(configDecoder, data, o.Config); err != nil {
		return fmt.Errorf("error decoding config: %w", err)
	}

	return nil
}

// Validate validates the configuration.
func (o *Options) Validate() error {
	if errs := validation.ValidateAuditlogForwarder(o.Config); len(errs) > 0 {
		return errs.ToAggregate()
	}
	return nil
}

// LogConfig returns the log level and format from the configuration.
func (o *Options) LogConfig() (string, string) {
	return o.Config.Log.Level, o.Config.Log.Format
}

// ApplyTo applies the options to the config.
func (o *Options) ApplyTo(server *Config) error {
	if err := o.applyServerConfigToServing(&server.Serving); err != nil {
		return err
	}

	server.InjectAnnotations = o.Config.InjectAnnotations

	backends, err := backendfactory.NewFromConfigs(o.Config.Backends)
	if err != nil {
		return fmt.Errorf("failed to create backends: %w", err)
	}
	server.Backends = backends

	return nil
}

// applyServerConfigToServing applies server configuration to serving config
func (o *Options) applyServerConfigToServing(serving *Serving) error {
	serverConfig := o.Config.Server
	serving.Address = net.JoinHostPort(serverConfig.Address, strconv.FormatUint(uint64(serverConfig.Port), 10))

	serverCert, err := tls.LoadX509KeyPair(serverConfig.TLS.CertFile, serverConfig.TLS.KeyFile)
	if err != nil {
		return fmt.Errorf("failed to parse server certificates: %w", err)
	}

	tlsConfig := &tls.Config{
		Certificates: []tls.Certificate{serverCert},
		MinVersion:   tls.VersionTLS12,
	}

	// Configure client certificate verification if specified
	if len(serverConfig.TLS.ClientCAFile) > 0 {
		if err := o.configureClientAuth(tlsConfig, serverConfig.TLS.ClientCAFile); err != nil {
			return fmt.Errorf("failed to configure client certificate verification: %w", err)
		}
	}

	serving.TLSConfig = tlsConfig
	return nil
}

// configureClientAuth configures client certificate authentication for the TLS config.
func (o *Options) configureClientAuth(tlsConfig *tls.Config, clientCAFile string) error {
	caCert, err := os.ReadFile(filepath.Clean(clientCAFile))
	if err != nil {
		return fmt.Errorf("failed to read CA file: %w", err)
	}

	caCertPool := x509.NewCertPool()
	if !caCertPool.AppendCertsFromPEM(caCert) {
		return fmt.Errorf("failed to parse CA certificate from %s", clientCAFile)
	}

	tlsConfig.ClientCAs = caCertPool
	tlsConfig.ClientAuth = tls.RequireAndVerifyClientCert

	return nil
}

// Config has all the context to run an auditlog forwarder.
type Config struct {
	Serving           Serving
	InjectAnnotations map[string]string
	Backends          []backend.Backend
}

// Serving contains the configuration for the auditlog forwarder.
type Serving struct {
	TLSConfig *tls.Config
	Address   string
}

// SPDX-FileCopyrightText: SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package options

import (
	"crypto/tls"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/spf13/pflag"
)

// Options contain the server options.
type Options struct {
	ServingOptions ServingOptions
}

// ServingOptions are options applied to the authentication webhook server.
type ServingOptions struct {
	TLSCertFile string
	TLSKeyFile  string

	Address string
	Port    uint
}

// AddFlags adds server options to flagset
func (s *ServingOptions) AddFlags(fs *pflag.FlagSet) {
	fs.StringVar(&s.TLSCertFile, "tls-cert-file", s.TLSCertFile, "File containing the x509 Certificate for HTTPS.")
	fs.StringVar(&s.TLSKeyFile, "tls-private-key-file", s.TLSKeyFile, "File containing the x509 private key matching --tls-cert-file.")

	fs.StringVar(&s.Address, "address", "", "The IP address that the server will listen on. If unspecified all interfaces will be used.")
	fs.UintVar(&s.Port, "port", 10443, "The port that the server will listen on.")
}

// Validate validates the serving options.
func (s *ServingOptions) Validate() []error {
	errs := []error{}
	if strings.TrimSpace(s.TLSCertFile) == "" {
		errs = append(errs, errors.New("--tls-cert-file is required"))
	}

	if strings.TrimSpace(s.TLSKeyFile) == "" {
		errs = append(errs, errors.New("--tls-private-key-file is required"))
	}

	return errs
}

// ApplyTo applies the serving options to the authentication server configuration.
func (s *ServingOptions) ApplyTo(c *Serving) error {
	c.Address = fmt.Sprintf("%s:%s", s.Address, strconv.FormatUint(uint64(s.Port), 10))
	serverCert, err := tls.LoadX509KeyPair(s.TLSCertFile, s.TLSKeyFile)
	if err != nil {
		return fmt.Errorf("failed to parse server certificates: %w", err)
	}

	c.TLSConfig = &tls.Config{
		Certificates: []tls.Certificate{serverCert},
		MinVersion:   tls.VersionTLS12,
	}

	return nil
}

// NewOptions return options with default values.
func NewOptions() *Options {
	opts := &Options{}
	return opts
}

// AddFlags adds server options to flagset
func (o *Options) AddFlags(fs *pflag.FlagSet) {
	o.ServingOptions.AddFlags(fs)
}

// ApplyTo applies the options to the config.
func (o *Options) ApplyTo(server *Config) error {
	if err := o.ServingOptions.ApplyTo(&server.Serving); err != nil {
		return err
	}

	return nil
}

// Validate checks if options are valid
func (o *Options) Validate() []error {
	return o.ServingOptions.Validate()
}

// Config has all the context to run an auditlog forwarder.
type Config struct {
	Serving Serving
}

// Serving contains the configuration for the auditlog forwarder.
type Serving struct {
	TLSConfig *tls.Config
	Address   string
}

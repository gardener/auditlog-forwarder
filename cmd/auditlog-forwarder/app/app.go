// SPDX-FileCopyrightText: SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package app

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/go-logr/logr"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"k8s.io/component-base/version"
	"k8s.io/component-base/version/verflag"

	"github.com/gardener/auditlog-forwarder/cmd/auditlog-forwarder/app/options"
	"github.com/gardener/auditlog-forwarder/internal/handler/audit"
	"github.com/gardener/auditlog-forwarder/internal/processor"
	"github.com/gardener/auditlog-forwarder/internal/processor/annotation"
	configv1alpha1 "github.com/gardener/auditlog-forwarder/pkg/apis/config/v1alpha1"
)

// AppName is the name of the application.
const AppName = "auditlog-forwarder"

// NewCommand is the root command for the auditlog forwarder.
func NewCommand() *cobra.Command {
	opt := options.NewOptions()

	cmd := &cobra.Command{
		Use: AppName,
		RunE: func(cmd *cobra.Command, _ []string) error {
			if err := opt.Complete(); err != nil {
				return fmt.Errorf("cannot complete options: %w", err)
			}

			if err := opt.Validate(); err != nil {
				return fmt.Errorf("cannot validate options: %w", err)
			}

			level, format := opt.LogConfig()
			log := setupLogging(level, format)

			log.Info("Starting application", "app", AppName, "version", version.Get())
			cmd.Flags().VisitAll(func(flag *pflag.Flag) {
				log.Info("Flag", "name", flag.Name, "value", flag.Value, "default", flag.DefValue)
			})

			conf := &options.Config{}
			if err := opt.ApplyTo(conf); err != nil {
				return fmt.Errorf("cannot apply options: %w", err)
			}

			return run(cmd.Context(), log, conf)
		},
		PreRunE: func(_ *cobra.Command, _ []string) error {
			verflag.PrintAndExitIfRequested()
			return nil
		},
	}

	fs := cmd.Flags()
	verflag.AddFlags(fs)
	opt.AddFlags(fs)
	fs.AddGoFlagSet(flag.CommandLine)

	return cmd
}

func run(ctx context.Context, log logr.Logger, conf *options.Config) error {
	// Create processors
	var processors []processor.Processor
	if len(conf.InjectAnnotations) > 0 {
		processors = append(processors, annotation.New(conf.InjectAnnotations))
	}

	auditHandler, err := audit.NewHandler(log, processors, conf.Outputs)
	if err != nil {
		return fmt.Errorf("failed to create audit handler: %w", err)
	}

	mux := http.NewServeMux()
	mux.Handle("POST /audit", auditHandler)

	srv := &http.Server{
		Addr:         conf.Serving.Address,
		Handler:      mux,
		TLSConfig:    conf.Serving.TLSConfig,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
	}

	return runServer(ctx, log, srv)
}

// runServer starts the auditlog forwarder server. It returns if the context is canceled or the server cannot start initially.
func runServer(ctx context.Context, log logr.Logger, srv *http.Server) error {
	log = log.WithName("auditlog-forwarder")
	errCh := make(chan error)
	go func(errCh chan<- error) {
		log.Info("Starts listening", "address", srv.Addr)
		defer close(errCh)
		if err := srv.ListenAndServeTLS("", ""); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- fmt.Errorf("failed serving content: %w", err)
		} else {
			log.Info("Server stopped listening")
		}
	}(errCh)

	select {
	case err := <-errCh:
		return err
	case <-ctx.Done():
		log.Info("Shutting down")
		cancelCtx, cancel := context.WithTimeout(context.Background(), time.Second*10)
		defer cancel()
		err := srv.Shutdown(cancelCtx)
		if err != nil {
			return fmt.Errorf("auditlog forwarder server failed graceful shutdown: %w", err)
		}
		log.Info("Shutdown successful")
		return nil
	}
}

// setupLogging configures logging based on the level and format from configuration.
func setupLogging(level, format string) logr.Logger {
	var slogLevel slog.Level
	switch level {
	case configv1alpha1.LogLevelDebug:
		slogLevel = slog.LevelDebug
	case configv1alpha1.LogLevelError:
		slogLevel = slog.LevelError
	default:
		slogLevel = slog.LevelInfo
	}

	var handler slog.Handler
	handlerOptions := &slog.HandlerOptions{Level: slogLevel}

	switch format {
	case configv1alpha1.LogFormatText:
		handler = slog.NewTextHandler(os.Stdout, handlerOptions)
	default:
		handler = slog.NewJSONHandler(os.Stdout, handlerOptions)
	}

	return logr.FromSlogHandler(handler)
}

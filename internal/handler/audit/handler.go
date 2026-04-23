// SPDX-FileCopyrightText: SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package audit

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"

	"github.com/go-logr/logr"
	"github.com/google/uuid"

	loggerctx "github.com/gardener/auditlog-forwarder/internal/context"
	"github.com/gardener/auditlog-forwarder/internal/metrics"
	"github.com/gardener/auditlog-forwarder/internal/output"
	"github.com/gardener/auditlog-forwarder/internal/processor"
	configv1alpha1 "github.com/gardener/auditlog-forwarder/pkg/apis/config/v1alpha1"
)

const (
	headerContentType = "Content-Type"
	mimeAppJSON       = "application/json"
)

// Handler handles incoming audit events.
// It processes events through configured processors and sends them to configured outputs.
type Handler struct {
	logger            logr.Logger
	processors        []processor.Processor
	guaranteedOutputs []output.Output
	bestEffortOutputs []output.Output
	shutdownCtx       context.Context
	shutdownCancel    context.CancelFunc
	bestEffortWg      sync.WaitGroup
}

// NewHandler creates a new [Handler].
func NewHandler(logger logr.Logger, processors []processor.Processor, guaranteedOutputs, bestEffortOutputs []output.Output) (*Handler, error) {
	if len(guaranteedOutputs) == 0 {
		return nil, errors.New("at least one Guaranteed output must be configured")
	}

	shutdownCtx, shutdownCancel := context.WithCancel(context.Background()) //#nosec // G118: Handler.Shutdown method is calling the Cancel func.

	return &Handler{
		logger:            logger,
		processors:        processors,
		guaranteedOutputs: guaranteedOutputs,
		bestEffortOutputs: bestEffortOutputs,
		shutdownCtx:       shutdownCtx,
		shutdownCancel:    shutdownCancel,
	}, nil
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	metrics.AuditReceived.Inc()

	log := h.logger.WithValues("req_id", uuid.NewString())
	body, err := io.ReadAll(r.Body)
	if err != nil {
		log.Error(err, "Reading request body")
		w.Header().Set(headerContentType, mimeAppJSON)
		w.WriteHeader(http.StatusInternalServerError)
		writeErrorResponse(w, log, http.StatusInternalServerError, "failed reading body request")
		metrics.AuditFailed.Inc()
		return
	}

	ctx := loggerctx.WithLogger(r.Context(), log)
	log.Info("Received audit events")

	processedData := body
	for _, processor := range h.processors {
		processedData, err = processor.Process(ctx, processedData)
		if err != nil {
			log.Error(err, "Processing audit events", "processor", processor.Name())
			w.Header().Set(headerContentType, mimeAppJSON)
			w.WriteHeader(http.StatusInternalServerError)
			writeErrorResponse(w, log, http.StatusInternalServerError, "failed processing audit events")
			metrics.AuditFailed.Inc()
			return
		}
	}

	// Send to Guaranteed outputs first - these must succeed for request to be successful
	if err := forwardToGuaranteedOutputs(ctx, processedData, h.guaranteedOutputs, log); err != nil {
		log.Error(err, "Failed to forward audit events to Guaranteed outputs")
		w.Header().Set(headerContentType, mimeAppJSON)
		w.WriteHeader(http.StatusInternalServerError)
		writeErrorResponse(w, log, http.StatusInternalServerError, "failed forwarding audit events")
		metrics.AuditFailed.Inc()
		return
	}

	// Fire off BestEffort outputs asynchronously - they don't block the response
	if len(h.bestEffortOutputs) > 0 {
		h.bestEffortWg.Go(func() {
			forwardToBestEffortOutputs(h.shutdownCtx, processedData, h.bestEffortOutputs, log)
		})
	}

	log.Info("Forwarded audit events to Guaranteed outputs")
	w.WriteHeader(http.StatusOK)
	metrics.AuditSucceeded.Inc()
}

// Shutdown initiates graceful shutdown of the handler, waiting for in-flight
// BestEffort outputs to complete within the given timeout.
// It waits for all active BestEffort goroutines to finish, canceling the
// shutdown context only after timeout to stop any remaining work.
func (h *Handler) Shutdown(timeout time.Duration) error {
	h.logger.Info("Initiating handler shutdown", "timeout", timeout.String())

	done := make(chan struct{})
	go func() {
		h.bestEffortWg.Wait()
		close(done)
	}()

	select {
	case <-done:
		h.logger.Info("All BestEffort outputs completed successfully or maximum retries reached")
		h.shutdownCancel()
		return nil
	case <-time.After(timeout):
		// Cancel shutdown context to stop any ongoing retries
		h.shutdownCancel()
		return fmt.Errorf("shutdown timeout exceeded after %s, some BestEffort outputs may not have completed", timeout)
	}
}

func writeErrorResponse(w http.ResponseWriter, log logr.Logger, statusCode int, message string) {
	if _, err := fmt.Fprintf(w, `{"code":%d,"message":"%s"}`, statusCode, message); err != nil {
		log.Error(err, "Writing response body")
	}
}

// forwardToGuaranteedOutputs forwards audit events to Guaranteed outputs.
// All Guaranteed outputs must succeed for the request to be considered successful.
func forwardToGuaranteedOutputs(ctx context.Context,
	data []byte,
	outputs []output.Output,
	log logr.Logger,
) error {
	logAndMeterOutputErr := func(out output.Output, err error) error {
		log.Error(err, "Failed to forward to Guaranteed output", "output", out.Name())
		metrics.OutputFailed.WithLabelValues(out.Name(), string(configv1alpha1.DeliveryModeGuaranteed)).Inc()
		return fmt.Errorf("output %s failed: %w", out.Name(), err)
	}

	// Single output, no need to initialize a wait group and spawn goroutines
	if len(outputs) == 1 {
		out := outputs[0]
		if err := out.Send(ctx, data); err != nil {
			return logAndMeterOutputErr(out, err)
		}
		metrics.OutputSucceeded.WithLabelValues(out.Name(), string(configv1alpha1.DeliveryModeGuaranteed)).Inc()
		return nil
	}

	var wg sync.WaitGroup
	errCh := make(chan error, len(outputs))

	for _, out := range outputs {
		wg.Add(1)
		go func(o output.Output) {
			defer wg.Done()
			if err := o.Send(ctx, data); err != nil {
				errCh <- logAndMeterOutputErr(o, err)
			} else {
				metrics.OutputSucceeded.WithLabelValues(o.Name(), string(configv1alpha1.DeliveryModeGuaranteed)).Inc()
			}
		}(out)
	}

	wg.Wait()
	close(errCh)

	// Check if any output failed
	var errs []error
	for err := range errCh {
		errs = append(errs, err)
	}

	if len(errs) > 0 {
		return fmt.Errorf("one or more Guaranteed outputs failed: %w", errors.Join(errs...))
	}

	return nil
}

// forwardToBestEffortOutputs forwards audit events to BestEffort outputs asynchronously.
// Failures are logged and tracked in metrics but do not affect the request status.
func forwardToBestEffortOutputs(
	ctx context.Context,
	data []byte,
	outputs []output.Output,
	log logr.Logger,
) {
	ctx = loggerctx.WithLogger(ctx, log)

	var wg sync.WaitGroup
	for _, out := range outputs {
		wg.Add(1)
		go func(o output.Output) {
			defer wg.Done()
			if err := o.Send(ctx, data); err != nil {
				log.Error(err, "Failed to forward to BestEffort output", "output", o.Name())
				metrics.OutputFailed.WithLabelValues(o.Name(), string(configv1alpha1.DeliveryModeBestEffort)).Inc()
			} else {
				log.Info("Successfully forwarded to BestEffort output", "output", o.Name())
				metrics.OutputSucceeded.WithLabelValues(o.Name(), string(configv1alpha1.DeliveryModeBestEffort)).Inc()
			}
		}(out)
	}

	wg.Wait()
}

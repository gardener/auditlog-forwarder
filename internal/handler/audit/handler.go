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

	"github.com/go-logr/logr"
	"github.com/google/uuid"

	loggerctx "github.com/gardener/auditlog-forwarder/internal/context"
	"github.com/gardener/auditlog-forwarder/internal/output"
	"github.com/gardener/auditlog-forwarder/internal/processor"
)

const (
	headerContentType = "Content-Type"
	mimeAppJSON       = "application/json"
)

// Handler handles incoming audit events.
// It processes events through configured processors and sends them to configured outputs.
type Handler struct {
	logger     logr.Logger
	processors []processor.Processor
	outputs    []output.Output
}

// NewHandler creates a new [Handler].
func NewHandler(logger logr.Logger, processors []processor.Processor, outputs []output.Output) (*Handler, error) {
	if len(outputs) == 0 {
		return nil, errors.New("no outputs configured")
	}
	return &Handler{
		logger:     logger,
		processors: processors,
		outputs:    outputs,
	}, nil
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	log := h.logger.WithValues("req_id", uuid.NewString())
	body, err := io.ReadAll(r.Body)
	if err != nil {
		log.Error(err, "Reading request body")
		w.Header().Set(headerContentType, mimeAppJSON)
		w.WriteHeader(http.StatusInternalServerError)
		if _, err := w.Write([]byte(`{"code":500,"message":"failed reading body request"}`)); err != nil {
			log.Error(err, "Writing response body")
			return
		}
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
			if _, err := w.Write([]byte(`{"code":500,"message":"failed processing audit events"}`)); err != nil {
				log.Error(err, "Writing response body")
				return
			}
			return
		}
	}

	if err := forwardToOutputs(ctx, processedData, h.outputs, log); err != nil {
		log.Error(err, "Failed to forward audit events to outputs")
		w.Header().Set(headerContentType, mimeAppJSON)
		w.WriteHeader(http.StatusInternalServerError)
		if _, err := w.Write([]byte(`{"code":500,"message":"failed to forward audit events"}`)); err != nil {
			log.Error(err, "Writing response body")
			return
		}
		return
	}

	log.Info("Forwarded audit events to all outputs")
	w.WriteHeader(http.StatusOK)
}

// forwardToOutputs forwards audit events to all configured outputs in parallel.
func forwardToOutputs(ctx context.Context,
	data []byte,
	outputs []output.Output,
	log logr.Logger,
) error {
	// Single output, no need to initialize a wait group and spawn goroutines
	if len(outputs) == 1 {
		output := outputs[0]
		if err := output.Send(ctx, data); err != nil {
			log.Error(err, "Failed to forward to output", "output", output.Name())
			return fmt.Errorf("output %s failed: %w", output.Name(), err)
		}
		return nil
	}

	var wg sync.WaitGroup
	errCh := make(chan error, len(outputs))

	for _, out := range outputs {
		wg.Add(1)
		go func(o output.Output) {
			defer wg.Done()
			if err := o.Send(ctx, data); err != nil {
				log.Error(err, "Failed to forward to output", "output", o.Name())
				errCh <- fmt.Errorf("output %s failed: %w", o.Name(), err)
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
		return fmt.Errorf("one or more outputs failed: %v", errs)
	}

	return nil
}

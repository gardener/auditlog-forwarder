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

	"github.com/gardener/auditlog-forwarder/internal/backend"
	loggerctx "github.com/gardener/auditlog-forwarder/internal/context"
	"github.com/gardener/auditlog-forwarder/internal/processor"
)

const (
	headerContentType = "Content-Type"
	mimeAppJSON       = "application/json"
)

// Handler handles incoming audit events.
// It processes events through configured processors and sends them to configured backends.
type Handler struct {
	logger     logr.Logger
	processors []processor.Processor
	backends   []backend.Backend
}

// NewHandler creates a new [Handler].
func NewHandler(logger logr.Logger, processors []processor.Processor, backends []backend.Backend) (*Handler, error) {
	if len(backends) == 0 {
		return nil, errors.New("no backends configured")
	}
	return &Handler{
		logger:     logger,
		processors: processors,
		backends:   backends,
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
		processedData, err = processor.Process(ctx, body)
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

	if err := forwardToBackends(ctx, processedData, h.backends, log); err != nil {
		log.Error(err, "Failed to forward audit events to backends")
		w.Header().Set(headerContentType, mimeAppJSON)
		w.WriteHeader(http.StatusInternalServerError)
		if _, err := w.Write([]byte(`{"code":500,"message":"failed to forward audit events"}`)); err != nil {
			log.Error(err, "Writing response body")
			return
		}
		return
	}

	log.Info("Forwarded audit events to all backends")
	w.WriteHeader(http.StatusOK)
}

// forwardToBackends forwards audit events to all configured backends in parallel.
func forwardToBackends(ctx context.Context,
	data []byte,
	backends []backend.Backend,
	log logr.Logger,
) error {
	// Single backend, no need to initialize a wait group and spawn goroutines
	if len(backends) == 1 {
		backend := backends[0]
		if err := backend.Send(ctx, data); err != nil {
			log.Error(err, "Failed to forward to backend", "backend", backend.Name())
			return fmt.Errorf("backend %s failed: %w", backend.Name(), err)
		}
		return nil
	}

	var wg sync.WaitGroup
	errCh := make(chan error, len(backends))

	for _, be := range backends {
		wg.Add(1)
		go func(b backend.Backend) {
			defer wg.Done()
			if err := b.Send(ctx, data); err != nil {
				log.Error(err, "Failed to forward to backend", "backend", b.Name())
				errCh <- fmt.Errorf("backend %s failed: %w", b.Name(), err)
			}
		}(be)
	}

	wg.Wait()
	close(errCh)

	// Check if any backend failed
	var errs []error
	for err := range errCh {
		errs = append(errs, err)
	}

	if len(errs) > 0 {
		return fmt.Errorf("one or more backends failed: %v", errs)
	}

	return nil
}

// SPDX-FileCopyrightText: SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package audit

import (
	"context"
	"errors"
	"fmt"
	"io"
	"maps"
	"net/http"
	"sync"

	"github.com/go-logr/logr"
	"github.com/google/uuid"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/apiserver/pkg/apis/audit"
	v1 "k8s.io/apiserver/pkg/apis/audit/v1"

	"github.com/gardener/auditlog-forwarder/internal/backend"
	loggerctx "github.com/gardener/auditlog-forwarder/internal/context"
)

const (
	headerContentType = "Content-Type"
	mimeAppJSON       = "application/json"
)

var (
	runtimeScheme = runtime.NewScheme()
	codecs        = serializer.NewCodecFactory(runtimeScheme)
	decoder       = codecs.UniversalDecoder()
)

func init() {
	_ = v1.AddToScheme(runtimeScheme)
	_ = audit.AddToScheme(runtimeScheme)
}

// Handler handles incoming audit events.
// It additionally enriches the events with metadata and sends them to configured backednds.
type Handler struct {
	logger      logr.Logger
	annotations map[string]string
	backends    []backend.Backend
}

// NewHandler creates a new [Handler].
func NewHandler(logger logr.Logger, annotations map[string]string, backends []backend.Backend) (*Handler, error) {
	if len(backends) == 0 {
		return nil, errors.New("no backends configured")
	}
	return &Handler{
		logger:      logger,
		annotations: annotations,
		backends:    backends,
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

	eventList, err := decode(body)
	if err != nil {
		log.Error(err, "Decoding body")
		w.Header().Set(headerContentType, mimeAppJSON)
		w.WriteHeader(http.StatusBadRequest)
		if _, err := w.Write([]byte(`{"code":400,"message":"invalid body format"}`)); err != nil {
			log.Error(err, "Writing response body")
			return
		}
		return
	}

	for i := range eventList.Items {
		if eventList.Items[i].Annotations == nil {
			eventList.Items[i].Annotations = make(map[string]string)
		}
		maps.Insert(eventList.Items[i].Annotations, maps.All(h.annotations))
	}

	respBody, err := runtime.Encode(codecs.LegacyCodec(v1.SchemeGroupVersion), eventList)
	if err != nil {
		log.Error(err, "Encoding response body")
		w.Header().Set(headerContentType, mimeAppJSON)
		w.WriteHeader(http.StatusInternalServerError)
		if _, err := w.Write([]byte(`{"code":500,"message":"failed encoding response body"}`)); err != nil {
			log.Error(err, "Writing response body")
			return
		}
		return
	}

	log.Info("Received audit events", "count", len(eventList.Items))

	ctx := loggerctx.WithLogger(r.Context(), log)
	if err := forwardToBackends(ctx, respBody, h.backends, log); err != nil {
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

func decode(data []byte) (*audit.EventList, error) {
	internal, schemaVersion, err := decoder.Decode(data, nil, nil)
	if err != nil {
		return nil, err
	}

	out, ok := internal.(*audit.EventList)
	if !ok {
		return nil, fmt.Errorf("failure to cast to auditlog internal; type: %v", schemaVersion)
	}
	return out, nil
}

// SPDX-FileCopyrightText: SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package audit

import (
	"fmt"
	"io"
	"maps"
	"net/http"

	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/apiserver/pkg/apis/audit"
	v1 "k8s.io/apiserver/pkg/apis/audit/v1"
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
}

// NewHandler creates a new [Handler].
func NewHandler(logger logr.Logger, annotations map[string]string) (*Handler, error) {
	return &Handler{
		logger:      logger,
		annotations: annotations,
	}, nil
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		h.logger.Error(err, "Reading request body")
		w.Header().Set(headerContentType, mimeAppJSON)
		w.WriteHeader(http.StatusInternalServerError)
		if _, err := w.Write([]byte(`{"code":500,"message":"failed reading body request"}`)); err != nil {
			h.logger.Error(err, "Writing response body")
			return
		}
		return
	}

	eventList, err := decode(body)
	if err != nil {
		h.logger.Error(err, "Decoding body")
		w.Header().Set(headerContentType, mimeAppJSON)
		w.WriteHeader(http.StatusBadRequest)
		if _, err := w.Write([]byte(`{"code":400,"message":"invalid body format"}`)); err != nil {
			h.logger.Error(err, "Writing response body")
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

	_, err = runtime.Encode(codecs.LegacyCodec(v1.SchemeGroupVersion), eventList)
	if err != nil {
		h.logger.Error(err, "Encoding response body")
		w.Header().Set(headerContentType, mimeAppJSON)
		w.WriteHeader(http.StatusInternalServerError)
		if _, err := w.Write([]byte(`{"code":500,"message":"failed encoding response body"}`)); err != nil {
			h.logger.Error(err, "Writing response body")
			return
		}
		return
	}

	h.logger.Info("Received audit events", "count", len(eventList.Items))

	// w.Header().Set(headerContentType, mimeAppJSON)
	// w.Write(respBody)
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

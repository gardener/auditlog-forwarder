// SPDX-FileCopyrightText: SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package annotation

import (
	"context"
	"maps"

	"github.com/gardener/auditlog-forwarder/internal/helper"
	"github.com/gardener/auditlog-forwarder/internal/processor"
)

var _ processor.Processor = (*Injector)(nil)

// Injector implements Processor and injects annotations into audit events.
type Injector struct {
	annotations map[string]string
}

// New creates a new annotation Injector with the given annotations.
func New(annotations map[string]string) *Injector {
	return &Injector{
		annotations: annotations,
	}
}

// Process injects annotations into the audit events.
func (a *Injector) Process(_ context.Context, data []byte) ([]byte, error) {
	if len(a.annotations) == 0 {
		return data, nil
	}

	eventList, err := helper.DecodeEventList(data)
	if err != nil {
		return nil, err
	}

	for i := range eventList.Items {
		if eventList.Items[i].Annotations == nil {
			eventList.Items[i].Annotations = make(map[string]string)
		}
		maps.Insert(eventList.Items[i].Annotations, maps.All(a.annotations))
	}

	return helper.EncodeEventList(eventList)
}

// Name returns the name of the processor.
func (a *Injector) Name() string {
	return "audit-event-annotation-injector"
}

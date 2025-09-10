// SPDX-FileCopyrightText: SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package annotation

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apiserver/pkg/apis/audit"

	"github.com/gardener/auditlog-forwarder/internal/helper"
)

var _ = Describe("Injector", func() {
	var (
		injector    *Injector
		annotations map[string]string
		ctx         context.Context
	)

	BeforeEach(func() {
		annotations = map[string]string{
			"test-key":    "test-value",
			"another-key": "another-value",
		}
		ctx = context.Background()
		injector = New(annotations)
	})

	Describe("Process", func() {
		It("should inject annotations into audit events", func() {
			eventList := &audit.EventList{
				TypeMeta: metav1.TypeMeta{
					APIVersion: "audit.k8s.io/v1",
					Kind:       "EventList",
				},
				Items: []audit.Event{
					{
						Verb: "create",
						ObjectRef: &audit.ObjectReference{
							Resource: "pods",
						},
					},
					{
						Verb: "update",
						ObjectRef: &audit.ObjectReference{
							Resource: "services",
						},
						Annotations: map[string]string{
							"existing-key": "existing-value",
						},
					},
				},
			}

			inputData, err := helper.EncodeEventList(eventList)
			Expect(err).NotTo(HaveOccurred())

			processedData, err := injector.Process(ctx, inputData)
			Expect(err).NotTo(HaveOccurred())

			processedEventList, err := helper.DecodeEventList(processedData)
			Expect(err).NotTo(HaveOccurred())

			Expect(processedEventList.Items).To(HaveLen(2))

			Expect(processedEventList.Items[0].Annotations).To(HaveKeyWithValue("test-key", "test-value"))
			Expect(processedEventList.Items[0].Annotations).To(HaveKeyWithValue("another-key", "another-value"))

			Expect(processedEventList.Items[1].Annotations).To(HaveKeyWithValue("existing-key", "existing-value"))
			Expect(processedEventList.Items[1].Annotations).To(HaveKeyWithValue("test-key", "test-value"))
			Expect(processedEventList.Items[1].Annotations).To(HaveKeyWithValue("another-key", "another-value"))
		})

		It("should handle empty annotations", func() {
			emptyInjector := New(map[string]string{})

			eventList := &audit.EventList{
				TypeMeta: metav1.TypeMeta{
					APIVersion: "audit.k8s.io/v1",
					Kind:       "EventList",
				},
				Items: []audit.Event{
					{
						Verb: "create",
					},
				},
			}

			inputData, err := helper.EncodeEventList(eventList)
			Expect(err).NotTo(HaveOccurred())

			processedData, err := emptyInjector.Process(ctx, inputData)
			Expect(err).NotTo(HaveOccurred())

			Expect(processedData).To(Equal(inputData))
		})

		It("should handle invalid input data", func() {
			invalidData := []byte("invalid json")

			_, err := injector.Process(ctx, invalidData)
			Expect(err).To(HaveOccurred())
		})
	})
})

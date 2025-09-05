// SPDX-FileCopyrightText: SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package validation

import (
	configv1alpha1 "github.com/gardener/auditlog-forwarder/pkg/apis/config/v1alpha1"
	"k8s.io/apimachinery/pkg/util/validation/field"
)

// ValidateAuditlogForwarderConfiguration validates the given [*configv1alpha1.AuditlogForwarderConfiguration].
func ValidateAuditlogForwarderConfiguration(_ *configv1alpha1.AuditlogForwarderConfiguration) field.ErrorList {
	allErrs := field.ErrorList{}

	return allErrs
}

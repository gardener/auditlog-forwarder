// SPDX-FileCopyrightText: Contributors to the Gardener project
//
// SPDX-License-Identifier: Apache-2.0

package annotation

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestAnnotation(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Annotation Test Suite")
}

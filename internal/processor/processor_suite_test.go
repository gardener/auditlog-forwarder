// SPDX-FileCopyrightText: Contributors to the Gardener project
//
// SPDX-License-Identifier: Apache-2.0

package processor

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestProcessor(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Processor Test Suite")
}

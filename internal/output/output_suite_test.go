// SPDX-FileCopyrightText: Contributors to the Gardener project
//
// SPDX-License-Identifier: Apache-2.0

package output

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestOutput(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Output Test Suite")
}

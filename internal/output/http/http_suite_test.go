// SPDX-FileCopyrightText: Contributors to the Gardener project
//
// SPDX-License-Identifier: Apache-2.0

package http_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestHTTPOutput(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "HTTP Output Test Suite")
}

// SPDX-FileCopyrightText: SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package http

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestHTTPBackend(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "HTTP Backend Test Suite")
}

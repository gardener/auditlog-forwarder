// SPDX-FileCopyrightText: SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

//go:build tools
// +build tools

// This package imports things required by build scripts, to force `go mod` to see them as dependencies
package tools

import (
	_ "github.com/gardener/gardener/hack/.ci" // Keep it as dependency so that the scripts from 'github.com/gardener/gardener/hack' are considered dependency
	_ "github.com/gardener/gardener/hack/tools"
)

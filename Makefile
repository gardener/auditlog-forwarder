# SPDX-FileCopyrightText: 2023 SAP SE or an SAP affiliate company and Gardener contributors
#
# SPDX-License-Identifier: Apache-2.0

ENSURE_GARDENER_MOD    := $(shell go get github.com/gardener/gardener@$$(go list -m -f "{{.Version}}" github.com/gardener/gardener))
GARDENER_HACK_DIR      := $(shell go list -m -f "{{.Dir}}" github.com/gardener/gardener)/hack
REPO_ROOT              := $(shell dirname $(realpath $(lastword $(MAKEFILE_LIST))))
HACK_DIR               := $(REPO_ROOT)/hack
VERSION                := $(shell cat "$(REPO_ROOT)/VERSION")
EFFECTIVE_VERSION      := $(VERSION)-$(shell git rev-parse HEAD)
LD_FLAGS               := "-w $(shell bash $(GARDENER_HACK_DIR)/get-build-ld-flags.sh k8s.io/component-base $(REPO_ROOT)/VERSION $(NAME))"
KIND_LOCAL_KUBECONFIG  := $(REPO_ROOT)/dev/local/kind/kubeconfig

TOOLS_DIR := $(REPO_ROOT)/hack/tools
include $(GARDENER_HACK_DIR)/tools.mk

.PHONY: format
format: $(GOIMPORTS) $(GOIMPORTSREVISER)
	@bash $(GARDENER_HACK_DIR)/format.sh ./cmd ./internal

.PHONY: test
test:
	go test -cover ./...

.PHONY: clean
clean:
	@bash $(GARDENER_HACK_DIR)/clean.sh ./cmd/... ./internal/...

.PHONY: check
check: $(GOIMPORTS) $(GOLANGCI_LINT) $(TYPOS)
	go vet ./...
	@REPO_ROOT=$(REPO_ROOT) bash $(GARDENER_HACK_DIR)/check.sh --golangci-lint-config=./.golangci.yaml ./cmd/... ./internal/...

	@bash $(GARDENER_HACK_DIR)/check-typos.sh
	@bash $(GARDENER_HACK_DIR)/check-file-names.sh

.PHONY: generate
generate:
	$(MAKE) format

.PHONY: check-generate
check-generate:
	@bash $(GARDENER_HACK_DIR)/check-generate.sh $(REPO_ROOT)

.PHONY: sast
sast: tidy $(GOSEC)
	@$(HACK_DIR)/sast.sh

.PHONY: sast-report
sast-report: tidy $(GOSEC)
	@$(HACK_DIR)/sast.sh --gosec-report true

.PHONY: test-cov
test-cov:
	@bash $(GARDENER_HACK_DIR)/test-cover.sh ./cmd/... ./internal/...

.PHONY: test-clean
test-clean:
	@bash $(GARDENER_HACK_DIR)/test-cover-clean.sh

.PHONY: verify
verify: format check test sast

.PHONY: verify-extended
verify-extended: check-generate check format test test-cov test-clean sast-report

kind-up kind-down: export KIND_KUBECONFIG = $(KIND_LOCAL_KUBECONFIG)
kind-up kind-down server-up: export KUBECONFIG = $(KIND_LOCAL_KUBECONFIG)

.PHONY: kind-up
kind-up: $(KIND) $(KUBECTL) $(YQ)
	@bash $(HACK_DIR)/kind-up.sh

.PHONY: kind-down
kind-down: $(KIND)
	@bash $(HACK_DIR)/kind-down.sh

server-up: export LD_FLAGS = $(shell $(REPO_ROOT)/hack/get-build-ld-flags.sh k8s.io/component-base $(REPO_ROOT)/VERSION auditlog-forwarder $(BUILD_DATE))

.PHONY: server-up
server-up: $(SKAFFOLD) $(HELM) $(KUBECTL)
	$(SKAFFOLD) run

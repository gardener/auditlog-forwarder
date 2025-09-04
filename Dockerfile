# SPDX-FileCopyrightText: SAP SE or an SAP affiliate company and Gardener contributors
#
# SPDX-License-Identifier: Apache-2.0

FROM golang:1.25.1 AS go-builder

ARG TARGETARCH
WORKDIR /workspace

COPY . .

RUN CGO_ENABLED=0 GOOS=linux GOARCH=$TARGETARCH go build -a -ldflags="$(/workspace/hack/get-build-ld-flags.sh)" -o auditlog-forwarder cmd/auditlog-forwarder/main.go

FROM gcr.io/distroless/static-debian12:nonroot AS auditlog-forwarder
WORKDIR /
COPY --from=go-builder /workspace/auditlog-forwarder .

ENTRYPOINT ["/auditlog-forwarder"]

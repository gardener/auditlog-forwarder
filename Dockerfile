# SPDX-FileCopyrightText: SAP SE or an SAP affiliate company and Gardener contributors
#
# SPDX-License-Identifier: Apache-2.0

FROM golang:1.25.2 AS go-builder

ARG TARGETARCH
WORKDIR /go/src/github.com/gardener/auditlog-forwarder

# Copy go mod and sum files
COPY go.mod go.sum ./
# Download all dependencies. Dependencies will be cached if the go.mod and go.sum files are not changed
RUN go mod download

COPY . .

ARG EFFECTIVE_VERSION
RUN make install EFFECTIVE_VERSION=$EFFECTIVE_VERSION

FROM gcr.io/distroless/static-debian12:nonroot AS auditlog-forwarder
WORKDIR /

COPY --from=go-builder /go/bin/auditlog-forwarder /auditlog-forwarder
ENTRYPOINT ["/auditlog-forwarder"]

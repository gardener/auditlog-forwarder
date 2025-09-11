# SPDX-FileCopyrightText: SAP SE or an SAP affiliate company and Gardener contributors
#
# SPDX-License-Identifier: Apache-2.0

FROM golang:1.25.1 AS go-builder

ARG TARGETARCH
WORKDIR /workspace

COPY . .

ARG EFFECTIVE_VERSION
RUN make install EFFECTIVE_VERSION=$EFFECTIVE_VERSION

FROM gcr.io/distroless/static-debian12:nonroot AS auditlog-forwarder
WORKDIR /

COPY --from=go-builder /go/bin/auditlog-forwarder /auditlog-forwarder
ENTRYPOINT ["/auditlog-forwarder"]

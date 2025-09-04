#!/bin/bash

# SPDX-FileCopyrightText: SAP SE or an SAP affiliate company and Gardener contributors
#
# SPDX-License-Identifier: Apache-2.0
# 

set -o errexit
set -o pipefail

repo_root="$(readlink -f $(dirname ${0})/..)"
cert_dir="$repo_root/dev/local/certs"

mkdir -p "$cert_dir"

if [[ -s "$cert_dir/tls.key" ]]; then
    echo "Development certificate found at $cert_dir. Skipping generation..."
    exit 0
fi

echo "Generating development certificate..."
openssl req -x509 -newkey ec -pkeyopt ec_paramgen_curve:prime256v1 -days 365 \
  -nodes -keyout "$cert_dir/tls.key" -out "$cert_dir/tls.crt" \
  -subj "/CN=auditlog-forwarder" -addext "subjectAltName=DNS:localhost,DNS:auditlog-forwarder,DNS:auditlog-forwarder.kube-system,DNS:auditlog-forwarder.kube-system.svc,DNS:auditlog-forwarder.kube-system.svc.cluster.local,IP:127.0.0.1"

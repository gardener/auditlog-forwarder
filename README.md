# Auditlog Forwarder

[![REUSE status](https://api.reuse.software/badge/github.com/gardener/auditlog-forwarder)](https://api.reuse.software/info/github.com/gardener/auditlog-forwarder)
[![Build](https://github.com/gardener/auditlog-forwarder/actions/workflows/non-release.yaml/badge.svg)](https://github.com/gardener/auditlog-forwarder/actions/workflows/non-release.yaml)
[![Go Report Card](https://goreportcard.com/badge/github.com/gardener/auditlog-forwarder)](https://goreportcard.com/report/github.com/gardener/auditlog-forwarder)

A Kubernetes audit log forwarder that receives audit events from Kubernetes API servers via webhook, enriches them with metadata annotations, and forwards them to configured backends. This project is part of the [Gardener](https://gardener.cloud/) ecosystem for managing Kubernetes clusters.

## Overview

The auditlog-forwarder acts as a webhook endpoint that:

1. **Receives** Kubernetes audit events from API servers
2. **Processes** events through configurable processors (annotation injection, filtering, etc.)
3. **Forwards** enriched events to multiple backend systems (HTTP endpoints, OTLP, etc.)

### Key Features

- **Webhook Integration**: Seamless integration with Kubernetes audit webhook functionality
- **Annotation Injection**: Enrich audit events with custom metadata for better observability
- **Multiple Backends**: Forward to multiple destinations simultaneously (work in progress)
- **TLS Security**: Mutual TLS support for secure communication
- **Configurable Processing**: Pluggable processor architecture for extensible event handling

### Architecture

```
┌─────────────────┐     HTTP POST       ┌──────────────────────┐     Forward      ┌─────────────────┐
│ Kubernetes API  │────────────────────▶│  auditlog-forwarder  │─────────────────▶│    Backend 1    │
│    Server       │    /audit endpoint  │                      │                  │     (HTTPS)     │
└─────────────────┘                     │  - Receive events    │                  └─────────────────┘
                                        │  - Process & enrich  │
                                        │  - Forward to all    │     Forward      ┌─────────────────┐
                                        │    backends          │─────────────────▶│    Backend N    │
                                        │                      │                  │     (HTTPS)     │
                                        └──────────────────────┘                  └─────────────────┘
```

## Development

### Quick Start

For developers looking to get started quickly, please refer to the [Getting Started Locally guide](docs/getting-started-locally.md).
This guide provides step-by-step instructions for setting up and running the project locally with a complete KinD-based development environment.

### Testing

```bash
# Run all tests
make test

# Verify code conventions
make check
```

### Code Generation

```bash
# Generate deepcopy and default functions
make generate
```

## Contributing

Please see [CONTRIBUTING.md](CONTRIBUTING.md) for guidelines on how to contribute to this project.

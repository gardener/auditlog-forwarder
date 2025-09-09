# Getting Started Locally

## Local KinD Setup

This document will walk you through running a KinD cluster on your local machine and installing the auditlog-forwarder in it.

### 1. Create KinD cluster and deploy the auditlog-forwarder

```bash
make kind-up
make server-up
```

You can now target the KinD cluster.

```bash
export KUBECONFIG=$(pwd)/dev/local/kind/kubeconfig
```

This setup will deploy both the auditlog-forwarder and the echo-server in the `kube-system` namespace.
The echo-server acts as a dummy backend that receives and logs the enriched audit events forwarded by the auditlog-forwarder.

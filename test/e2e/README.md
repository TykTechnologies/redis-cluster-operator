# E2E Tests for Redis Cluster Operator

This directory contains end-to-end tests for the `redis-cluster-operator` using Ginkgo and the Operator SDK.

## What It Does

- Builds and loads the operator image into a Kind cluster
- Installs CRDs
- Deploys the operator
- Verifies the controller-manager pod is running
- Cleans up the test namespace after execution

## Prerequisites

- Docker  
- Kind (cluster named `test`)  
- Kubectl  
- Go  
- [Ginkgo CLI (optional)](https://onsi.github.io/ginkgo/#ginkgo-cli)

## Run the Tests

```bash
make test-e2e
```

## Notes

- Test namespace: `redis-cluster-operator-system`
- Image used: `tyk/redis-cluster-operator:v0.0.1`
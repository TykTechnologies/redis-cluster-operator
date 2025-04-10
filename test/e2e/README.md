# E2E Tests for Redis Cluster Operator

This directory contains end-to-end tests for the `redis-cluster-operator` using Ginkgo and the Operator SDK.

## What It Does

#### DRC operator test
- Builds and loads the operator image into a Kind cluster
- Installs CRDs
- Deploys the operator
- Verifies the controller-manager pod is running
- Cleans up the test namespace after execution
#### DRC CRUD test
Below is a concise documentation of what the tests verify:

- Creates a DistributedRedisCluster resource with a random name and password; waits for the cluster to stabilize and seeds test data.
- Changes the Redis configuration, updates the resource, and confirms consistent data state.
- Simulates master pod deletion and verifies automatic recovery and data integrity.
- Tests scaling operations by scaling up then scaling down, ensuring the cluster remains healthy.
- Resets the cluster password and performs a minor version rolling update, verifying stability and consistent data throughout.

## Prerequisites

- Docker  
- Kind (cluster named `test`)  
- Kubectl  
- Go  
- [Ginkgo CLI (optional)](https://onsi.github.io/ginkgo/#ginkgo-cli)

## Run the Tests

Below is a command-based short documentation on how to run the tests locally:

---

### How to Run the E2E Tests


1. **Set Up the Environment**  
   - **Create a Kind Cluster:**  
     ```bash
     kind create cluster --name e2e-test --image kindest/node:v1.31.6
     kubectl cluster-info && kubectl get nodes
     ```

2. **Run Operator E2E Tests**  
   ```bash
   go test ./test/e2e/drc_operator -v -ginkgo.v
   ```

3. **Deploy the Operator & Prepare CRUD Test**  
   ```bash
   # Load operator image into Kind 
   # The 'tykio/redis-cluster-operator:v0.0.0-teste2e' image is built as part of the Operator E2E Tests, so it does not need to be built again.
   kind load docker-image tykio/redis-cluster-operator:v0.0.0-teste2e --name e2e-test

   # Install CRDs and deploy the operator
   make install
   make deploy IMG=tykio/redis-cluster-operator:v0.0.0-teste2e

   # Build and load the CRUD test image
   make docker-build-e2e IMG=tykio/drc-crud-test:v0.0.0-teste2e
   kind load docker-image tykio/drc-crud-test:v0.0.0-teste2e --name e2e-test

   # Wait for the operator to be available
   kubectl wait --for=condition=available --timeout=90s deployment/redis-cluster-operator-controller-manager --namespace redis-cluster-operator-system
   ```

4. **Run and Monitor the CRUD E2E Test Job**  
   ```bash
   # Apply the job definition to run CRUD tests
   kubectl apply -f test/e2e/drc_crud/job.yaml

   # Optional: Get the pod name
   POD=$(kubectl get pods --namespace redis-cluster-operator-system -l job-name=drc-crud-e2e-job -o jsonpath="{.items[0].metadata.name}")

   # Stream logs from the test pod
   kubectl logs --namespace redis-cluster-operator-system -f "$POD"

   # Check job status (succeeded or failed)
   kubectl get job drc-crud-e2e-job --namespace redis-cluster-operator-system
   ```
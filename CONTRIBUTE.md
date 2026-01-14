# Contributing

## Development Environment Setup

1. Ensure Go 1.23+ is installed
2. Install [Kind](https://kind.sigs.k8s.io/), [Skaffold](https://skaffold.dev/) and [Helm](https://helm.sh/docs/intro/install)
3. Run `kind create cluster` to create a new Kubernetes cluster locally
4. Add a `Secret` object to the cluster containing your Cronitor API key according to the [instructions](./README.md#instructions)
5. In the main directory of this repository, run `skaffold dev`. Skaffold will build the Docker container and agent and push it to your local Kubernetes cluster. Anytime you make code changes it will rebuild and re-push.
6. If you'd like to add some sample `CronJob`s easily, you can run `kubectl apply -f e2e-test/resources/main/`, which will add a bunch of pre-written test resources.
7. When you're done, stop skaffold and clean up with `kind delete cluster`!

## Testing

### Unit Tests

Run all unit tests with:

```bash
go test ./...
```

The codebase has comprehensive unit tests covering:
- **Monitor sync** (`pkg/api/jobs_test.go`, `pkg/api/monitors_test.go`): Tests for CronJob → Cronitor monitor conversion, annotations, API error handling
- **Telemetry** (`pkg/api/telemetry_test.go`): Tests for telemetry URL construction, all query parameters (state, env, series, host, etc.)
- **Annotations** (`pkg/annotations_test.go`): Tests for annotation parsing and configuration

### E2E Tests

E2E tests run automatically in CI using a Kind cluster with a mock Cronitor API server. This tests:
- Agent starts successfully in Kubernetes
- Agent discovers CronJobs
- Agent makes correct HTTP requests to sync monitors

To run e2e tests locally:

```bash
# Create Kind cluster
kind create cluster --name cronitor-e2e

# Build and load images
docker build -t mock-cronitor-api:latest ./e2e-test/mock-server
docker build -t cronitor-kubernetes:e2e .
kind load docker-image mock-cronitor-api:latest --name cronitor-e2e
kind load docker-image cronitor-kubernetes:e2e --name cronitor-e2e

# Deploy mock server
kubectl apply -f ./e2e-test/mock-server/k8s-manifests.yaml
kubectl wait --for=condition=ready pod -l app=mock-cronitor-api -n cronitor-mock --timeout=60s

# Deploy agent pointing to mock server
helm install cronitor-agent ./charts/cronitor-kubernetes \
  --namespace cronitor --create-namespace \
  --set image=cronitor-kubernetes:e2e \
  --set imagePullPolicy=Never \
  --set credentials.createSecret.apiKey=e2e-test-key \
  --set config.hostnameOverride=http://mock-cronitor-api.cronitor-mock.svc.cluster.local \
  --set config.sentryEnabled=false

# Apply test CronJobs
kubectl apply -f ./e2e-test/resources/main/cronjob.yaml -n cronitor

# Run verification
./e2e-test/verify-e2e.sh

# Cleanup
kind delete cluster --name cronitor-e2e
```

## Making a Release

1. Update the `version` number in `charts/cronitor-kubernetes/Chart.yaml`
2. Update the `appVersion` number if any changes were made to the application itself
3. Push to the `main` branch
4. The CI pipeline will automatically build and publish the Docker image and Helm chart

## Code Structure

```
├── cmd/                    # CLI commands (agent, root)
├── pkg/
│   ├── api/               # Cronitor API client (monitors, telemetry)
│   ├── collector/         # Kubernetes watchers and sync logic
│   ├── normalizer/        # K8s version compatibility
│   └── annotations.go     # Annotation parsing
├── charts/                # Helm chart
├── e2e-test/
│   ├── mock-server/       # Mock Cronitor API for e2e tests
│   ├── resources/         # Test CronJob manifests
│   └── verify-e2e.sh      # E2E verification script
└── .github/workflows/     # CI/CD pipelines
```

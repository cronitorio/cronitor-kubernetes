# Cronitor Kubernetes Agent Helm Chart

This Helm chart deploys the Cronitor Kubernetes agent, which automatically monitors your Kubernetes CronJobs in Cronitor.

## Installation

```bash
helm repo add cronitor https://cronitorio.github.io/cronitor-kubernetes/

helm install cronitor-agent cronitor/cronitor-kubernetes \
  --namespace cronitor --create-namespace \
  --set credentials.createSecret.apiKey=YOUR_API_KEY
```

## Values

| Parameter | Description | Default |
|-----------|-------------|---------|
| `image` | Full image override (useful for testing) | `""` |
| `repository` | Docker image repository | `ghcr.io/cronitorio/cronitor-kubernetes` |
| `imageTag` | Docker image tag | Chart's `appVersion` |
| `imagePullPolicy` | Image pull policy | `Always` |
| `imagePullSecrets` | Image pull secrets | `[]` |
| `nameOverride` | Override chart name | `""` |
| `fullnameOverride` | Override full name | `""` |

### Credentials

| Parameter | Description | Default |
|-----------|-------------|---------|
| `credentials.createSecret.apiKey` | API key (creates Secret automatically) | `""` |
| `credentials.secretName` | Name of existing Secret | `null` |
| `credentials.secretKey` | Key in existing Secret | `null` |

### Configuration

| Parameter | Description | Default |
|-----------|-------------|---------|
| `config.default` | Default behavior for CronJobs (`include` or `exclude`) | `include` |
| `config.defaultEnvironment` | Default Cronitor environment name | `""` |
| `config.tags` | Tags to add to all monitors | `""` |
| `config.shipLogs` | Ship job logs to Cronitor | `true` |
| `config.sentryEnabled` | Enable Sentry telemetry | `true` |
| `config.yourEmail` | Your email for Cronitor support | `""` |
| `config.logLevel` | Agent log level (DEBUG, INFO, WARN, ERROR) | `""` |
| `config.podFilter` | Regex to filter pods by name | `""` |
| `config.hostnameOverride` | Override Cronitor API hostname (for testing) | `""` |

### RBAC

| Parameter | Description | Default |
|-----------|-------------|---------|
| `rbac.create` | Create RBAC resources | `true` |
| `rbac.clusterScope` | RBAC scope (`cluster` or `namespace`) | `cluster` |

### Service Account

| Parameter | Description | Default |
|-----------|-------------|---------|
| `serviceAccount.create` | Create service account | `true` |
| `serviceAccount.name` | Service account name | `""` (auto-generated) |

### Pod Settings

| Parameter | Description | Default |
|-----------|-------------|---------|
| `podSecurityContext` | Pod security context | `{}` |
| `securityContext` | Container security context | `{}` |
| `resources` | CPU/memory resources | `{}` |
| `nodeSelector` | Node selector | `{}` |
| `tolerations` | Tolerations | `[]` |
| `affinity` | Affinity rules | `{}` |

## CronJob Annotations

See the [main README](../../README.md#cronjob-annotations) for a full list of supported CronJob annotations.

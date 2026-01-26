# Cronitor for Kubernetes

![Test](https://github.com/cronitorio/cronitor-kubernetes/actions/workflows/publish.yml/badge.svg)
![E2E](https://github.com/cronitorio/cronitor-kubernetes/actions/workflows/e2e-tests.yml/badge.svg)

_Cronitor's Kubernetes agent and integration_

This repository contains the code and Helm chart for the Kubernetes agent for [Cronitor](cronitor.io), which provides simple monitoring for every type of application. The Cronitor Kubernetes agent helps you automatically instrument, track, and monitor your Kubernetes `CronJob`s in the Cronitor dashboard by automatically tracking every `CronJob` in Kubernetes and relaying related events like job successes and failures back to Cronitor.

Important note: by default, this chart enables Sentry for telemetry to help the Cronitor team identify and debug issues with the agent. If you would like to turn this off, set `config.sentryEnabled` to `false` in your `values.yaml` override.

### Instructions
To use the Helm chart:

    helm repo add cronitor https://cronitorio.github.io/cronitor-kubernetes/


A valid Cronitor API key with the ability to configure monitors is required. You can provide your API key in two ways:

#### Option 1: Chart-managed Secret (Recommended)
The simplest approach is to let the Helm chart create and manage the Secret for you:

```bash
helm upgrade --install <release name> cronitor/cronitor-kubernetes --namespace <namespace> \
    --set credentials.createSecret.apiKey=<your-api-key>
```

This approach is particularly useful when using tools like `helm-secrets` for encrypted deployments, as you can encrypt the API key directly in your values files.

#### Option 2: External Secret Management
Alternatively, you can create a Kubernetes `Secret` manually and reference it in the chart. Create the Secret in the namespace where you plan to deploy the Helm chart:

```bash
kubectl create secret generic cronitor-secret -n <namespace> --from-literal=CRONITOR_API_KEY=<api key>
```

Then deploy the chart referencing your existing Secret:

```bash
helm upgrade --install <release name> cronitor/cronitor-kubernetes --namespace <namespace> \
    --set credentials.secretName=cronitor-secret \
    --set credentials.secretKey=CRONITOR_API_KEY
```

You can customize your installation of the Cronitor Kubernetes agent by overriding the default values found in `values.yaml`, either with `--set` or by creating an additional values file of your own and passing it into Helm. For more information on this, see the [Helm documentation on values](https://helm.sh/docs/chart_template_guide/values_files/).

To learn what options are customizable in the chart, please see [this repository's documented `values.yaml`][1] file.


### CronJob annotations

The Cronitor Kubernetes agent's behavior has a number of defaults that are configurable via the chart's `values.yaml`. However, in certain situations you may want to override the defaults on a per-`CronJob` basis. You can do so using Kubernetes annotations on your `CronJob` objects as you create them.

#### Inclusion and exclusion

By default, all CronJobs are monitored. To exclude specific jobs, add:
```yaml
annotations:
  k8s.cronitor.io/exclude: "true"
```

Alternatively, you can set `config.default: exclude` in your Helm values to ignore all jobs by default, then opt-in specific jobs with:
```yaml
annotations:
  k8s.cronitor.io/include: "true"
```

#### Monitor identity

Control how the monitor is identified and named in Cronitor.

| Annotation | Description | Values | Default |
|------------|-------------|--------|---------|
| `k8s.cronitor.io/key` | Manually specify the monitor key. Use when migrating existing monitors to the agent, recreating CronJobs, or sharing a monitor across environments. | Any string | Auto-generated (see `key-inference`) |
| `k8s.cronitor.io/key-inference` | How the monitor key is auto-generated when `key` is not set. `k8s` uses the Kubernetes UID (unique per CronJob instance). `name` hashes the CronJob name (consistent across clusters/namespaces). | `"k8s"`, `"name"` | `"k8s"` |
| `k8s.cronitor.io/name` | Display name shown in the Cronitor dashboard. If using `key` to share a monitor, ensure all CronJobs with the same key use the same name. | Any string | `namespace/cronjob-name` |
| `k8s.cronitor.io/name-prefix` | Control the prefix added to auto-generated names. | `"namespace"`, `"none"`, or any custom string | `"namespace"` |

#### Organization

Control how the monitor is organized in the Cronitor dashboard.

| Annotation | Description | Values | Default |
|------------|-------------|--------|---------|
| `k8s.cronitor.io/env` | Environment name for this CronJob, shown in Cronitor. Overrides the chart-wide default. | Any string | Chart default (`config.defaultEnv`) |
| `k8s.cronitor.io/tags` | Tags for organizing and filtering monitors in Cronitor. | Comma-separated list | None |
| `k8s.cronitor.io/group` | Group key for organizing monitors within Cronitor. | Group key string | None |

#### Alerting

Control alert behavior for this monitor.

| Annotation | Description | Values | Default |
|------------|-------------|--------|---------|
| `k8s.cronitor.io/notify` | Notification lists to alert when the job fails or recovers. Use keys from your [Cronitor notification settings](https://cronitor.io/app/settings/alerts). | Comma-separated list of keys | Account default |
| `k8s.cronitor.io/grace-seconds` | Seconds to wait after the scheduled time before alerting on a missing job. No alert is sent if the job completes within this period. | Integer | Account default |

#### Additional configuration

| Annotation | Description | Values | Default |
|------------|-------------|--------|---------|
| `k8s.cronitor.io/note` | A note displayed on the monitor in the Cronitor dashboard. Useful for documentation or runbook links. | Any string | None |
| `k8s.cronitor.io/log-complete-event` | Send job completion as a log event instead of a state change. Use for async workflows where the actual task completion occurs outside the Kubernetes job. | `"true"`, `"false"` | `"false"` |

#### Legacy annotation names

For backwards compatibility, the following legacy annotation names are still supported but deprecated:

| Legacy | Preferred |
|--------|-----------|
| `k8s.cronitor.io/cronitor-id` | `k8s.cronitor.io/key` |
| `k8s.cronitor.io/cronitor-name` | `k8s.cronitor.io/name` |
| `k8s.cronitor.io/cronitor-group` | `k8s.cronitor.io/group` |
| `k8s.cronitor.io/cronitor-notify` | `k8s.cronitor.io/notify` |
| `k8s.cronitor.io/cronitor-grace-seconds` | `k8s.cronitor.io/grace-seconds` |
| `k8s.cronitor.io/id-inference` | `k8s.cronitor.io/key-inference` |


### Timezone support

The Cronitor Kubernetes agent automatically extracts the `timeZone` field from your Kubernetes CronJob spec (available in Kubernetes 1.24+) and sends it to Cronitor. This allows you to schedule jobs in specific timezones with proper daylight saving time handling.

Example CronJob with timezone:

```yaml
apiVersion: batch/v1
kind: CronJob
metadata:
  name: my-job
spec:
  schedule: "0 9 * * *"
  timeZone: "America/New_York"  # Automatically synced to Cronitor
  jobTemplate:
    spec:
      template:
        spec:
          containers:
          - name: hello
            image: busybox
            command: ["/bin/sh", "-c", "echo Hello"]
          restartPolicy: OnFailure
```

When the timezone is set, Cronitor will evaluate the cron schedule in that timezone rather than timezone set on your account (defaults to UTC).

### FAQ

**Does this pull in all my `CronJobs` across my cluster by default?**

By default, the agent will monitor all `CronJobs` in your Kubernetes cluster, but this is easily changeable. See below in the FAQ for additional information on how to handle various circumstances of `CronJob` inclusion or exclusion by annotation or namespace.

**The Kubernetes cluster I want to monitor is locked down with RBAC, and I only have access to one or a couple of namespaces. What do I do?**

You can configure the agent to only monitor a single namespace rather than the entire cluster. To do this, when deploying the agent, set `rbac.clusterScope` to `"namespace"` in [`values.yaml`][1]. In this setup, the agent will only monitor `CronJobs` within the namespace in which it is deployed, and it will not attempt to monitor anything outside of that namespace. It will not request permissions outside of its namespace either, using `Role` instead of `ClusterRole`.

If you have more than one namespace you need to monitor with this setup, you'll need to deploy multiple copies of the Cronitor Kubernetes agent, one in each namespace. When using chart-managed secrets (`credentials.createSecret.apiKey`), the Secret will be automatically created in each namespace during deployment. If using external Secret management, you will need to create a copy of the Secret containing your Cronitor API key in each namespace.

**What if I want just to try out this Kubernetes agent without pulling in all of my `CronJobs`? Can I do that?**

Yes, you definitely can! To exclude all of your Kubernetes `CronJobs` by default and only include the ones you explicitly choose, you can do the following:

1. When deploying the Cronitor Kubernetes agent, set `config.default` to `exclude`. You can do this in your custom `values.yaml` you use to deploy the Helm chart, or by passing the additional parameter `--set config.default=exclude` to Helm when you install or upgrade the release. This will exclude/ignore all of your cron jobs by default.
2. For any `CronJob` that you would like to be monitored by Cronitor, add the annotation `k8s.cronitor.io/include: true`. The agent honors any annotations explicitly set on `CronJobs` over whatever is set as the configuration default.

[1]: charts/cronitor-kubernetes/values.yaml

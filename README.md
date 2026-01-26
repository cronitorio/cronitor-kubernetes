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

Here is the list of supported annotations:
* `k8s.cronitor.io/include` - Override this CronJob to be explicitly tracked by Cronitor. Values are "true" or "false". (The agent default behavior is `config.default` in [`values.yaml`][1].)
* `k8s.cronitor.io/exclude` - Override this CronJob to be explicitly **not** tracked by Cronitor. Values are "true" or "false". (The agent default behavior is `config.default` in [`values.yaml`][1].)
* `k8s.cronitor.io/env` - Override the environment for this CronJob.
* `k8s.cronitor.io/tags` - Comma-separated list of tags for this cron job for use within the Cronitor application.
* `k8s.cronitor.io/key` - Manually specify a key for your monitor in Cronitor rather than autogenerating a new one. Use this when you already have jobs you are tracking in Cronitor that you want to keep the history of and you are migrating to the Cronitor agent, or if you are deleting and recreating your `CronJob` objects (e.g., you are migrating clusters or namespaces). You may also use this if you have a single CronJob that you run in different environments (staging, prod, etc.) and you want them all to report to the same monitor in Cronitor in different Cronitor environments.
* `k8s.cronitor.io/key-inference` - Specify how the Cronitor key is determined. Valid values are `k8s` (default) or `name`. `k8s` uses the Kubernetes UID. `name` hashes the name of the job itself (which is useful when you want standardization across environments).
* `k8s.cronitor.io/name` - Manually specify the name within the Cronitor dashboard for this CronJob. Please note if you are using `k8s.cronitor.io/key` you must ensure that CronJobs with the same key also have the same name, or the different names will overwrite each other.
* `k8s.cronitor.io/name-prefix` - Provides control over the prefix of the name. `none` uses the name as-is. `namespace` prepends the Kubernetes namespace. Any other string provided will be prepended to the name as-is.
* `k8s.cronitor.io/notify` - Comma-separated list of Notification List `key`s to assign alert destinations.
* `k8s.cronitor.io/group` - Group `key` attribute for grouping the monitor within the Cronitor application.
* `k8s.cronitor.io/grace-seconds` - The number of seconds that Cronitor should wait after the scheduled execution time before sending an alert. If the monitor is healthy at the end of the period no alert will be sent.
* `k8s.cronitor.io/note` - Default note for the monitor, displayed in the Cronitor dashboard.
* `k8s.cronitor.io/log-complete-event` - Controls whether the job completion event should be sent as a log record instead of a stateful completion. When set to `true`, the agent will not send telemetry events with state=complete, but will send a log event recording the completion. This supports async workflows where the actual task completion occurs outside the Kubernetes job. Valid values are `"true"` or `"false"`, default is `false`.

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

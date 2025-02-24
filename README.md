# Cronitor for Kubernetes

![Test](https://github.com/cronitorio/cronitor-kubernetes/actions/workflows/kubernetes.yaml/badge.svg)

_Cronitor's Kubernetes agent and integration_

This repository contains the code and Helm chart for the Kubernetes agent for [Cronitor](cronitor.io), which provides simple monitoring for every type of application. The Cronitor Kubernetes agent helps you automatically instrument, track, and monitor your Kubernetes `CronJob`s in the Cronitor dashboard by automatically tracking every `CronJob` in Kubernetes and relaying related events like job successes and failures back to Cronitor.

Important note: by default, this chart enables Sentry for telemetry to help the Cronitor team identify and debug issues with the agent. If you would like to turn this off, set `config.sentryEnabled` to `false` in your `values.yaml` override.

### Instructions
To use the Helm chart:

    helm repo add cronitor https://cronitorio.github.io/cronitor-kubernetes/


A valid Cronitor API key with the ability to configure monitors is required. Before deploying the agent, create a
Kubernetes `Secret` in the namespace in which you plan to deploy this Helm chart, and store your API key in that `Secret`. You will then put the name of the `Secret` and the key within the `Secret` at which the API key can be found into the following chart values:
* `credentials.secretName`
* `credentials.secretKey`

 This can be created easily using `kubectl`. Make sure that you create the Secret in the same namespace as where you plan to deploy the Helm chart for the agent. As an example:

```bash
kubectl create secret generic cronitor-secret -n <namespace> --from-literal=CRONITOR_API_KEY=<api key>
```

Deploy using Helm 2 or Helm 3, as in the following example:

```
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
* `k8s.cronitor.io/cronitor-id` - Manually specify an ID for your monitor in Cronitor rather than autogenerating a new one. Use this when you already have jobs you are tracking in Cronitor that you want to keep the history of and you are migrating to the Cronitor agent, or if you are deleting and recreating your `CronJob` objects (e.g., you are migrating clusters or namespaces). You may also use this if you have a single CronJob that you run in different environments (staging, prod, etc.) and you want them all to report to the same monitor in Cronitor in different Cronitor environments.
* `k8s.cronitor.io/id-inference` - Specify how the Cronitor ID is determined. `k8s` uses the Kubernetes UID. `name` hashes the name of the job itself (which is useful when you want standardization across environments)
* `k8s.cronitor.io/cronitor-name` - Manually specify the name within the Cronitor dashboard for this CronJob. Please note if you are using `k8s.cronitor.io/cronitor-id` you must ensure that CronJobs with the same ID also have the same name, or the different names will overwrite each other.
* `k8s.cronitor.io/name-prefix` - Provides control over the prefix of the name. `none` uses the name as-is. `namespace` prepends the Kubernetes namespace. Any other string provided will be prepended to the name as-is.
* `k8s.cronitor.io/cronitor-notify` - Comma-separated list of Notification List `key`s to assign alert destinations.
* `k8s.cronitor.io/cronitor-group` - Group `key` attribute for grouping the monitor within the Cronitor application.
* `k8s.cronitor.io/cronitor-grace-seconds` - The number of seconds that Cronitor should wait after the scheduled execution time before sending an alert. If the monitor is healthy at the end of the period no alert will be sent.
* `k8s.cronitor.io/auto-complete` - Controls whether the automatic completion telemetry should be disabled for this CronJob. When set to `false`, the agent will not send completion pings to Cronitor when the job finishes. Valid values are `"true"` or `"false"`.

### FAQ
<details>
    <summary>Does this pull in all my <code>CronJobs</code> across my cluster by default?</summary>

By default, the agent will monitor all `CronJobs` in your Kubernetes cluster, but this
is easily changeable. See below in the FAQ for additional information on how to handle various
circumstances of `CronJob` inclusion or exclusion by annotation or namespace.
</details>
<details>
    <summary>The Kubernetes cluster I want to monitor is locked down with RBAC, and I only have access
to one or a couple of namespaces. What do I do?</summary>

You can configure the agent to only monitor a single namespace rather than the entire cluster. To do this, when deploying the agent, set `rbac.clusterScope` to `"namespace"` in [`values.yaml`][1]. In this setup, the agent will only monitor `CronJobs` within the namespace in which it is deployed, and it will not attempt to monitor anything outside of that namespace. It will not request permissions outside of its namespace either, using `Role` instead of `ClusterRole`.

If you have more than one namespace you need to monitor with this setup, you'll need to deploy multiple copies of the Cronitor Kubernetes agent, one in each namespace. Please note that since Kubernetes Deployments can only access Secrets in the same namespace, you will also need to create a copy of the Secret containing your Cronitor API key in each namespace.

</details>
<details>
    <summary>What if I want just to try out this Kubernetes agent without pulling in <strong>all</strong> of my <code>CronJobs</code>? Can I do that?</summary>

Yes, you definitely can! To <strong>exclude</strong> all of your Kubernetes <code>CronJobs</code> by default and only include the ones you explicitly choose, you can do the following:

1. When deploying the Cronitor Kubernetes agent, set `config.default` to `exclude`. You can do this in your custom `values.yaml` you use to deploy the Helm chart, or by passing the additional parameter `--set config.default=exclude` to Helm when you install or upgrade the release. This will exclude/ignore all of your cron jobs by default.
2. For any `CronJob` that you would like to be monitored by Cronitor, add the annotation `k8s.cronitor.io/include: true`. The agent honors any annotations explicitly set on `CronJobs` over whatever is set as the configuration default.

</details>

[1]: charts/cronitor-kubernetes/values.yaml

# cronitor-kubernetes

_Cronitor's Kubernetes agent and integration_

### Instructions
To use the Helm chart:

    helm repo add cronitor https://cronitorio.github.io/cronitor-kubernetes/


A valid Cronitor API key is required. Before deploying the agent, create a
Kubernetes Secret in the namespace in which you plan to deploy this Helm chart, and
then put the name of the Secret and the key at which the API key be found in
the following chart values:
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

To learn what options are customizable in the chart, please see [this repository's documented `values.yaml`](charts/cronitor-kubernetes/values.yaml) file. 

### CronJob annotations

Annotations: 
* `k8s.cronitor.io/include` - Override this CronJob to be explicitly tracked by Cronitor. Values are "true" or "false"
* `k8s.cronitor.io/exclude` - Override this CronJob to be explicitly **not** tracked by Cronitor. Values are "true" or "false"
* `k8s.cronitor.io/env` - Override the environment for this CronJob. Shows up in the Cronitor dashboard as the `cluster-env` tag
* `k8s.cronitor.io/cronitor-id` - Manually specify an ID of an existing cron job in Cronitor rather than autogenerate a new one. Use this when you already have jobs you are tracking in Cronitor that you want to keep the history of and you are migrating to the Cronitor agent, or if you are deleting and recreating your `CronJob` objects (e.g., you are migrating clusters or namespaces)
* `k8s.cronitor.io/tags` - Comma-separated list of tags for this cron job for use within the Cronitor dashboard



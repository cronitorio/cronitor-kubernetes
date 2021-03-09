# cronitor-kubernetes

Cronitor's Kubernetes agent and integration

To use the Helm cart:

    helm repo add cronitor https://cronitorio.github.io/cronitor-kubernetes/


Annotations: 
* `k8s.cronitor.io/include` - "true" or "false"
* `k8s.cronitor.io/exclude` - "true" or "false"
* `k8s.cronitor.io/env`
* `k8s.cronitor.io/cronitor-id`
* `k8s.cronitor.io/tags`

Issues:
* Tags like `"kubernetes"` are not auto-created when submitted as part of a PUT request
* When loading the agent, sometimes we'll pick up events that are still present in Kubernetes but are actually
from sometime in the past. Can the telemetry API have a timestamp field added so that events from the past 
can be submitted?
  
Remaining to-dos:
* Set up publishing to Github Container Registry: https://github.com/docker/login-action#github-container-registry

Open questions:
* What should we do when watched CronJobs are deleted? Do we keep in Cronitor or remove?
* What should we do when the Cronitor k8s agent is redeployed with different watching rules, but 
no-longer-watched CronJobs still exist? Do we delete them or leave them be?

General notes:
* Jobs / CronJobs without a backoffLimit that are failing will retry indefinitely. A "failure" event never occurs, so
Cronitor would see this as the job never completing rather than as a failure. Default backoffLimit might be 6 though;
requires further investigation.
  

To make a new release:
1. Update the `version` number in the chart's `Chart.yaml`
2. Update the `appVersion` number if necessary
3. Push to the `main` branch
4. Profit
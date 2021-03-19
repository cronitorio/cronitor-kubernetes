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
* Because we're receiving both pod events and job events, particularly on start, the `Run` and `Fail` telemetry
event nearly always gets run twice in a row (though with the same series ID), once for the Job and 
  once for the Pod. Is this a problem?
* Can't push docker container
* When adding cronjobs, occasionally getting some weird 404s that I don't understand
  
Remaining to-dos:
* Set up publishing to Github Container Registry: https://github.com/docker/login-action#github-container-registry
* See if we can get the informers to limit information received at the server level
* Refactor log-fetching so it can happen asynchronously. We don't need the logs at the time
we send telemetry, they can be grabbed after-the-fact

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
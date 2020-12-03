# Cronitor-k8s
Cronitor's Kubernetes agent and integration


Annotations: 
* `k8s.cronitor.io/include` - "true" or "false"
* `k8s.cronitor.io/exclude` - "true" or "false"

Issues:
* Tags like `"kubernetes"` are not auto-created when submitted as part of a PUT request
* When loading the agent, sometimes we'll pick up events that are still present in Kubernetes but are actually
from sometime in the past. Can the telemetry API have a timestamp field added so that events from the past 
can be submitted?

Open questions:
* Where should we store the Docker builds? Docker Hub or Github registry?
* Where will building/CD take place? Github Actions?
* Creating Monitors in Cronitor. Can we store metdata to keep the k8s UID? Should we do it based on k8s name? 
Can we have a way that the canonical Cronitor ID be separate from the "name"?
* What should we do when watched CronJobs are deleted? Do we keep in Cronitor or remove?
* What should we do when the Cronitor k8s agent is redeployed with different watching rules, but 
no-longer-watched CronJobs still exist? Do we delete them or leave them be?
* Is it okay to use the Monitor API key for telemetry, or do we have to allow for Ping API key
use as well? It seems like we need to require both API keys in case the Monitor API is disallowed
for telemetry events.
* Pods can have more than one container, and so there may be more than one exit code. What should we do for exit
code selection?
* Allow selecting existing monitors in Cronitor for use as a CronJob by using a monitor ID as an annotation?

General notes:
* Jobs / CronJobs without a backoffLimit that are failing will retry indefinitely. A "failure" event never occurs, so
Cronitor would see this as the job never completing rather than as a failure. Default backoffLimit might be 6 though;
requires further investigation.

Todos:
* Combine the jobs watcher and cronjob watcher into one. Don't need to regenerate the jobs watcher
when the cronjobs change, just inspect events at time of receipt to see if they reference a watched
CronJob, and then discard then if they do not
* Handle whether one namespace or all
* Make it highly available? May need to handle leader election like [here][1]

[1]: https://github.com/opsgenie/kubernetes-event-exporter/blob/master/main.go
# Cronitor-k8s
Cronitor's Kubernetes agent and integration


Annotations: 
* `k8s.cronitor.io/include` - "true" or "false"
* `k8s.cronitor.io/exclude` - "true" or "false"

Todos:
* Combine the jobs watcher and cronjob watcher into one. Don't need to regenerate the jobs watcher
when the cronjobs change, just inspect events at time of receipt to see if they reference a watched
CronJob, and then discard then if they do not
* Handle whether one namespace or all
* Make it highly available? May need to handle leader election like [here][1]


[1]: https://github.com/opsgenie/kubernetes-event-exporter/blob/master/main.go
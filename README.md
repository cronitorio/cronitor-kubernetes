# Cronitor-k8s
Cronitor's Kubernetes agent and integration


Annotations: 
* `k8s.cronitor.io/include` - "true" or "false"
* `k8s.cronitor.io/exclude` - "true" or "false"

Todos:
* Add something to pick up all initial cronjobs during installation
* Handle whether one namespace or all
* Make it highly available? May need to handle leader election like [here][1]


[1]: https://github.com/opsgenie/kubernetes-event-exporter/blob/master/main.go
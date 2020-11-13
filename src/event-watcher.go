package src

import (
	"github.com/jdotjdot/Cronitor-k8s/src/collector"
	"k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"github.com/opsgenie/kubernetes-event-exporter/pkg/kube"
)

// TODO: We want to get all events for watched CronJobs.
// We can mainly do that by watching all cronjobs that are "Controlled by"
// any CronJob we are watching.

func NewEventWatcher() {
	clientset := collector.GetClientSet()
	factory := informers.NewSharedInformerFactory(clientset, 0)
	informer := factory.Core().V1().Events().Informer()

	stopper := make(chan struct{})
	defer close(stopper)
	defer runtime.HandleCrash()

	informer.Run(stopper)
}
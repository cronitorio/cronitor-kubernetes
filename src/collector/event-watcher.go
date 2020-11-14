package collector

import (
	"github.com/opsgenie/kubernetes-event-exporter/pkg/exporter"
	"github.com/opsgenie/kubernetes-event-exporter/pkg/kube"
	"github.com/opsgenie/kubernetes-event-exporter/pkg/sinks"
	log "github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/types"
)

// TODO: We want to get all events for watched CronJobs.
// We can mainly do that by watching all cronjobs that are "Controlled by"
// any CronJob we are watching

const ReceiverName = "in-memory"

func GetRouteForCronJobUID(uid types.UID) exporter.Route {
	return exporter.Route{
		Match: []exporter.Rule{{
			Kind:     "job",
			Labels: map[string]string{
				"controller-uid": string(uid),
			},
			Receiver: ReceiverName,
		}},
	}
}


func NewJobsEventWatcher(collection CronJobCollection) (*kube.EventWatcher, *exporter.Engine) {
	inMemory := new(sinks.InMemory)
	receiver := sinks.ReceiverConfig{
		Name:     ReceiverName,
		InMemory: &sinks.InMemoryConfig{
			Ref: inMemory,
		},
	}

	// Since we can't add our own stream, try abstracting over the InMemory sink
	// to grab the events that are coming from it and putting them in a channel
	eventStream := make(chan *kube.EnhancedEvent)
	go func(inMemory *sinks.InMemory, eventStream chan<- *kube.EnhancedEvent) {
		for {
			// Grab events, clear it out
			events := inMemory.Events
			inMemory.Events = []*kube.EnhancedEvent{}
			for _, event := range events {
				eventStream <- event
			}
		}
	}(inMemory, eventStream)

	go func(eventStream <-chan *kube.EnhancedEvent) {
		for {
			select {
				case event := <-eventStream:
					log.Debug(event)
			}
		}
	}(eventStream)

	var routes []exporter.Route

	for _, uid := range collection.GetAllWatchedCronJobUIDs() {
		routes = append(routes, GetRouteForCronJobUID(uid))
	}

	cfg := exporter.Config{
		Route: exporter.Route{Routes: routes},
		Receivers: []sinks.ReceiverConfig{receiver},
	}
	engine := exporter.NewEngine(&cfg, &exporter.ChannelBasedReceiverRegistry{})
	kubeconfig := GetConfig()
	w := kube.NewEventWatcher(kubeconfig, cfg.Namespace, engine.OnEvent)
	return w, engine
}
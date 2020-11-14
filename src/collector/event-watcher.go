package collector

import (
	"context"
	log "github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/tools/cache"
)

// TODO: We want to get all events for watched CronJobs.
// We can mainly do that by watching all cronjobs that are "Controlled by"
// any CronJob we are watching

type EventHandler struct {
	collection *CronJobCollection
	stopper chan struct{}
	informer cache.SharedInformer
}

func (e EventHandler) Start() {
	defer runtime.HandleCrash()
	log.Info("The jobs watcher is starting...")
	go e.informer.Run(e.stopper)
}

func (e *EventHandler) Stop() {
	log.Info("The jobs watcher is stopping...")
	close(e.stopper)
}

func (e EventHandler) CheckJobIsWatched(namespace string, name string) bool {
	// Grab the Job's information from the Kubernetes API.
	// Note: this might be a bit expensive, should we memoize it when possible?
	clientset := GetClientSet()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	jobsClient := clientset.BatchV1().Jobs(namespace)
	job, err := jobsClient.Get(ctx, name, v1.GetOptions{})
	if err != nil {
		// Job doesn't exist
		return false
	}

	ownerReference := job.ObjectMeta.OwnerReferences[0]
	if ownerReference.Kind != "CronJob" {
		return false
	}
	ownerUID := ownerReference.UID
	for _, b := range e.collection.GetAllWatchedCronJobUIDs() {
		if b == ownerUID {
			return true
		}
	}
	return false
}

func (e EventHandler) OnAdd(obj interface{}) {
	event := obj.(*corev1.Event)
	if event.InvolvedObject.Kind != "Job" {
		return
	}

	if e.CheckJobIsWatched(event.InvolvedObject.Namespace, event.InvolvedObject.Name) {
		log.WithFields(log.Fields{
			"name": event.InvolvedObject.Name,
			"kind": event.InvolvedObject.Kind,
			"eventMessage": event.Message,
		}).Info("We had a job watcher add event")
	}
}

// TODO: We may actually only need the OnAdd, since events don't seem to be deleted
// They are _sometimes_ updated when events are combined, but that seems to only happen on long-lived
// things like a CronJob, not the short-lived Jobs
func (e EventHandler) OnDelete(obj interface{}) {
	event := obj.(*corev1.Event)
	if event.InvolvedObject.Kind != "Job" {
		return
	}

	if e.CheckJobIsWatched(event.InvolvedObject.Namespace, event.InvolvedObject.Name) {
		log.WithFields(log.Fields{
			"name": event.InvolvedObject.Name,
			"kind": event.InvolvedObject.Kind,
			"eventMessage": event.Message,
		}).Info("We had a job watcher delete event")
	}
}

func (e EventHandler) OnUpdate(oldObj interface{}, newObj interface{}) {
	oldEvent := oldObj.(*corev1.Event)
	newEvent := newObj.(*corev1.Event)
	if newEvent.InvolvedObject.Kind != "Job" {
		return
	}

	if e.CheckJobIsWatched(newEvent.InvolvedObject.Namespace, newEvent.InvolvedObject.Name) {
		log.WithFields(log.Fields{
			"name": oldEvent.InvolvedObject.Name,
			"kind": newEvent.InvolvedObject.Kind,
			"eventMessage": newEvent.Message,
		}).Info("We had a job watcher update event... somehow?")
	}
}


func NewJobsEventWatcher(collection *CronJobCollection) *EventHandler {
	clientset := GetClientSet()
	factory := informers.NewSharedInformerFactory(clientset, 0)
	informer := factory.Core().V1().Events().Informer()

	eventHandler := &EventHandler{
		collection: collection,
		stopper: make(chan struct{}),
		informer: informer,
	}

	informer.AddEventHandler(eventHandler)
	return eventHandler
}

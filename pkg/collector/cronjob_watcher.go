package collector

import (
	log "github.com/sirupsen/logrus"
	"k8s.io/api/batch/v1beta1"
	"k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/tools/cache"
)

func onAdd(coll CronJobCollection, obj interface{}) {
	cronjob := obj.(*v1beta1.CronJob)
	configParser := NewCronitorConfigParser(cronjob)
	included, err := configParser.included()
	if err != nil {
		panic(err)
	}
	if !included {
		// If we aren't meant to include the CronJob in Cronitor,
		// there's nothing to do here.
		return
	}

	// TODO: Add the CronJob in Cronitor.
	coll.AddCronJob(cronjob)
}

func onUpdate(coll CronJobCollection, oldObj interface{}, newObj interface{}) {
	cronjobOld := oldObj.(*v1beta1.CronJob)
	cronjobNew := newObj.(*v1beta1.CronJob)
	configParserOld := NewCronitorConfigParser(cronjobOld)
	configParserNew := NewCronitorConfigParser(cronjobNew)
	wasIncluded, err := configParserOld.included()
	if err != nil {
		panic(err)
	}
	nowIncluded, err := configParserNew.included()
	if err != nil {
		panic(err)
	}
	if !wasIncluded && nowIncluded {
		onAdd(coll, cronjobNew)
	} else if wasIncluded && !nowIncluded {
		onDelete(coll, cronjobOld)
	} else if wasIncluded && nowIncluded {
		// Otherwise, if we're keeping it around, check if there are any changes to
		// configurable annotations and handle accordingly.
		// Right now we don't actually have any logic to put here, but we might down the line.
	}
}

func onDelete(coll CronJobCollection, obj interface{}) {
	cronjob := obj.(*v1beta1.CronJob)
	configParser := NewCronitorConfigParser(cronjob)
	included, err := configParser.included()
	if err != nil {
		panic(err)
	}
	if !included {
		// If the CronJob was never included in Cronitor, then nothing to do here.
		return
	}

	// TODO: Here we remove the CronJob from Cronitor, or at least notify that it's been removed
	coll.RemoveCronJob(cronjob)
}

type CronJobWatcher struct {
	informer    cache.SharedIndexInformer
	stopper     chan struct{}
	jobsWatcher *EventHandler
}

func (c CronJobWatcher) StartWatching() {
	defer runtime.HandleCrash()

	log.Info("The CronJob watcher is starting...")
	go c.informer.Run(c.stopper)
	go c.jobsWatcher.Start()
}

func (c CronJobWatcher) StopWatching() {
	close(c.stopper)
	c.jobsWatcher.Stop()
}

func NewCronJobWatcher(coll CronJobCollection) CronJobWatcher {
	clientset := coll.clientset
	factory := informers.NewSharedInformerFactory(clientset, 0)
	informer := factory.Batch().V1beta1().CronJobs().Informer()

	informer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			onAdd(coll, obj)
		},
		DeleteFunc: func(obj interface{}) {
			onDelete(coll, obj)
		},
		UpdateFunc: func(oldObj interface{}, newObj interface{}) {
			onUpdate(coll, oldObj, newObj)
		},
	})

	eventHandler := NewJobsEventWatcher(&coll)

	return CronJobWatcher{
		informer:    informer,
		stopper:     make(chan struct{}),
		jobsWatcher: eventHandler,
	}
}
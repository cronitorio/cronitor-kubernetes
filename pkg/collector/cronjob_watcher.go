package collector

import (
	"fmt"

	"github.com/cronitorio/cronitor-kubernetes/pkg"
	"github.com/cronitorio/cronitor-kubernetes/pkg/normalizer"
	log "github.com/sirupsen/logrus"
	v1 "k8s.io/api/batch/v1"
	"k8s.io/api/batch/v1beta1"
	"k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/tools/cache"
)

func onAdd(coll CronJobCollection, cronjob *v1.CronJob) {
	configParser := pkg.NewCronitorConfigParser(cronjob)
	included, err := configParser.IsCronJobIncluded()
	if err != nil {
		panic(err)
	}
	if !included {
		// If we aren't meant to include the CronJob in Cronitor,
		// there's nothing to do here.
		return
	}

	coll.AddCronJob(cronjob)
}

func onUpdate(coll CronJobCollection, cronjobOld *v1.CronJob, cronjobNew *v1.CronJob) {
	configParserOld := pkg.NewCronitorConfigParser(cronjobOld)
	configParserNew := pkg.NewCronitorConfigParser(cronjobNew)
	wasIncluded, err := configParserOld.IsCronJobIncluded()
	if err != nil {
		panic(err)
	}
	nowIncluded, err := configParserNew.IsCronJobIncluded()
	scheduleChanged := configParserOld.GetSchedule() != configParserNew.GetSchedule()
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
		if scheduleChanged {
			onAdd(coll, cronjobNew)
		}
	}
}

func onDelete(coll CronJobCollection, cronjob *v1.CronJob) {
	configParser := pkg.NewCronitorConfigParser(cronjob)
	included, err := configParser.IsCronJobIncluded()
	if err != nil {
		panic(err)
	}
	if !included {
		// If the CronJob was never included in Cronitor, then nothing to do here.
		return
	}

	coll.RemoveCronJob(cronjob)
}

type CronJobWatcher struct {
	informer    cache.SharedIndexInformer
	stopper     chan struct{}
	jobsWatcher *WatchWrapper
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

func coerceObjToV1CronJob(version string, obj interface{}) *v1.CronJob {
	var cronjob *v1.CronJob
	if version == "v1" {
		cronjob = obj.(*v1.CronJob)
	} else if version == "v1beta1" {
		temp := obj.(*v1beta1.CronJob)
		cronjob = normalizer.CronJobConvertV1Beta1ToV1(temp)
	}

	return cronjob
}

func NewCronJobWatcher(coll CronJobCollection) CronJobWatcher {
	clientset := coll.clientset
	var factory informers.SharedInformerFactory
	if coll.kubernetesNamespace == "" {
		factory = informers.NewSharedInformerFactory(clientset, 0)
	} else {
		factory = informers.NewSharedInformerFactoryWithOptions(clientset, 0, informers.WithNamespace(coll.kubernetesNamespace))
	}

	var informer cache.SharedIndexInformer
	version, err := coll.GetPreferredBatchApiVersion()
	if err != nil {
		panic(err)
	} else if version == "v1" {
		informer = factory.Batch().V1().CronJobs().Informer()
	} else if version == "v1beta1" {
		informer = factory.Batch().V1beta1().CronJobs().Informer()
	} else {
		panic(fmt.Sprintf("Invalid ApiVersion %s requested", version))
	}

	informer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			onAdd(coll, coerceObjToV1CronJob(version, obj))
		},
		DeleteFunc: func(obj interface{}) {
			onDelete(coll, coerceObjToV1CronJob(version, obj))
		},
		UpdateFunc: func(oldObj interface{}, newObj interface{}) {
			onUpdate(coll, coerceObjToV1CronJob(version, oldObj), coerceObjToV1CronJob(version, newObj))
		},
	})

	jobsWatcher := NewJobsEventWatcher(&coll)

	return CronJobWatcher{
		informer:    informer,
		stopper:     make(chan struct{}),
		jobsWatcher: jobsWatcher,
	}
}

package collector

import (
	"fmt"
	"log/slog"

	"github.com/cronitorio/cronitor-kubernetes/pkg"
	"github.com/cronitorio/cronitor-kubernetes/pkg/normalizer"
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

	// Skip if already tracked (was synced during initial LoadAllExistingCronJobs)
	if coll.IsTracked(cronjob.GetUID()) {
		slog.Debug("cronjob already tracked, skipping",
			"namespace", cronjob.Namespace,
			"name", cronjob.Name,
			"UID", cronjob.UID)
		return
	}

	if err := coll.AddCronJob(cronjob); err != nil {
		// Error is already logged and sent to Sentry in AddCronJob
		// Continue watching - the cronjob won't be tracked until a successful sync
		return
	}
}

func onUpdate(coll CronJobCollection, cronjobOld *v1.CronJob, cronjobNew *v1.CronJob) {
	configParserOld := pkg.NewCronitorConfigParser(cronjobOld)
	configParserNew := pkg.NewCronitorConfigParser(cronjobNew)
	wasIncluded, err := configParserOld.IsCronJobIncluded()
	if err != nil {
		panic(err)
	}
	nowIncluded, err := configParserNew.IsCronJobIncluded()
	if err != nil {
		panic(err)
	}
	if !wasIncluded && nowIncluded {
		// Newly included - sync it (bypass IsTracked check since it's a deliberate add)
		if err := coll.AddCronJob(cronjobNew); err != nil {
			return
		}
	} else if wasIncluded && !nowIncluded {
		onDelete(coll, cronjobOld)
	} else if wasIncluded && nowIncluded {
		slog.Info("cronjob updated",
			"namespace", cronjobNew.Namespace,
			"name", cronjobNew.Name,
			"UID", cronjobNew.UID,
			"oldSchedule", cronjobOld.Spec.Schedule,
			"newSchedule", cronjobNew.Spec.Schedule,
			"oldTimezone", configParserOld.GetTimezone(),
			"newTimezone", configParserNew.GetTimezone(),
			"configOld", configParserOld.GetSchedule(),
			"configNew", configParserNew.GetSchedule())

		// If the schedule or timezone is updated, sync the change
		if configParserOld.GetSchedule() != configParserNew.GetSchedule() ||
			configParserOld.GetTimezone() != configParserNew.GetTimezone() {
			if err := coll.AddCronJob(cronjobNew); err != nil {
				return
			}
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

	slog.Info("the CronJob watcher is starting...")
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
			cronjob := coerceObjToV1CronJob(version, obj)
			if cronjob == nil {
				slog.Error("failed to coerce object to CronJob", "version", version)
				return
			}
			onAdd(coll, cronjob)
		},
		DeleteFunc: func(obj interface{}) {
			cronjob := coerceObjToV1CronJob(version, obj)
			if cronjob == nil {
				slog.Error("failed to coerce object to CronJob", "version", version)
				return
			}
			onDelete(coll, cronjob)
		},
		UpdateFunc: func(oldObj interface{}, newObj interface{}) {
			oldCronjob := coerceObjToV1CronJob(version, oldObj)
			newCronjob := coerceObjToV1CronJob(version, newObj)
			if oldCronjob == nil || newCronjob == nil {
				slog.Error("failed to coerce object to CronJob", "version", version)
				return
			}
			onUpdate(coll, oldCronjob, newCronjob)
		},
	})

	jobsWatcher := NewJobsEventWatcher(&coll)

	return CronJobWatcher{
		informer:    informer,
		stopper:     make(chan struct{}),
		jobsWatcher: jobsWatcher,
	}
}

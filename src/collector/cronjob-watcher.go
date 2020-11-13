package collector

import (
	"encoding/json"
	log "github.com/sirupsen/logrus"
	"k8s.io/api/batch/v1beta1"
	"k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/tools/cache"
)

//&CronJob{ObjectMeta:{eventrouter-test-croonjob  cronitor /apis/batch/v1beta1/namespaces/cronitor/cronjobs/eventrouter-test-croonjob 7c56ca1c-f023-4ba2-94b8-657d4a291687 16727 0 2020-11-13 04:38:24 +0000 UTC <nil> <nil> map[app.kubernetes.io/managed-by:skaffold skaffold.dev/run-id:3a993098-5a27-45bb-aab9-24e2ce4910d5] map[] [] []  [{Go-http-client Update batch/v1beta1 2020-11-13 04:38:24 +0000 UTC FieldsV1 {"f:spec":{"f:concurrencyPolicy":{},"f:failedJobsHistoryLimit":{},"f:jobTemplate":{"f:spec":{"f:template":{"f:spec":{"f:containers":{"k:{\"name\":\"hello\"}":{".":{},"f:args":{},"f:image":{},"f:imagePullPolicy":{},"f:name":{},"f:resources":{},"f:terminationMessagePath":{},"f:terminationMessagePolicy":{}}},"f:dnsPolicy":{},"f:restartPolicy":{},"f:schedulerName":{},"f:securityContext":{},"f:terminationGracePeriodSeconds":{}}}}},"f:schedule":{},"f:successfulJobsHistoryLimit":{},"f:suspend":{}}}} {skaffold Update batch/v1beta1 2020-11-13 04:38:25 +0000 UTC FieldsV1 {"f:metadata":{"f:labels":{".":{},"f:app.kubernetes.io/managed-by":{},"f:skaffold.dev/run-id":{}}}}}]},Spec:CronJobSpec{Schedule:*/2 * * * *,StartingDeadlineSeconds:nil,ConcurrencyPolicy:Forbid,Suspend:*false,JobTemplate:JobTemplateSpec{ObjectMeta:{      0 0001-01-01 00:00:00 +0000 UTC <nil> <nil> map[] map[] [] []  []},Spec:{<nil> <nil> <nil> <nil> nil <nil> {{      0 0001-01-01 00:00:00 +0000 UTC <nil> <nil> map[] map[] [] []  []} {[] [] [{hello busybox [] [/bin/sh -c date ; echo Hello from k8s]  [] [] [] {map[] map[]} [] [] nil nil nil nil /dev/termination-log File Always nil false false false}] [] OnFailure 0xc00039d090 <nil> ClusterFirst map[]   <nil>  false false false <nil> PodSecurityContext{SELinuxOptions:nil,RunAsUser:nil,RunAsNonRoot:nil,SupplementalGroups:[],FSGroup:nil,RunAsGroup:nil,Sysctls:[]Sysctl{},WindowsOptions:nil,FSGroupChangePolicy:nil,SeccompProfile:nil,} []   nil default-scheduler [] []  <nil> nil [] <nil> <nil> <nil> map[] [] <nil>}} <nil>},},SuccessfulJobsHistoryLimit:*3,FailedJobsHistoryLimit:*1,},Status:CronJobStatus{Active:[]ObjectReference{},LastScheduleTime:<nil>,},}

func JsonAndPrint(input interface{}) string {
	item, _ := json.Marshal(input)
	return string(item)
}

func onAdd(coll CronJobCollection, obj interface{}) {
	// Cast as a CronJob
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
	informer cache.SharedIndexInformer
	stopper chan struct{}
}

func (c CronJobWatcher) StartWatching() {
	defer runtime.HandleCrash()

	log.Info("The CronJob watcher is starting...")
	go c.informer.Run(c.stopper)
}

func (c CronJobWatcher) StopWatching() {
	close(c.stopper)
}

func NewCronJobWatcher(coll CronJobCollection) CronJobWatcher {
	clientset := GetClientSet()
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

	return CronJobWatcher{
		informer: informer,
		stopper: make(chan struct{}),
	}
}

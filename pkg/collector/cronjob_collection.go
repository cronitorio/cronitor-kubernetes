package collector

import (
	"context"
	"github.com/cronitorio/cronitor-kubernetes/pkg/api"
	log "github.com/sirupsen/logrus"
	"k8s.io/api/batch/v1beta1"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
)

type CronJobCollection struct {
	clientset   *kubernetes.Clientset
	cronitorApi *api.CronitorApi
	cronjobs    map[types.UID]*v1beta1.CronJob
	loaded      bool
	stopper     func()
}

func NewCronJobCollection(pathToKubeconfig string, cronitorApi *api.CronitorApi) (*CronJobCollection, error) {
	config, err := GetConfig(pathToKubeconfig)
	if err != nil {
		return nil, err
	}
	clientset := GetClientSet(config)
	return &CronJobCollection{
		clientset:   clientset,
		cronitorApi: cronitorApi,
		cronjobs:    make(map[types.UID]*v1beta1.CronJob),
		loaded:      false,
	}, nil
}

func (coll *CronJobCollection) AddCronJob(cronjob *v1beta1.CronJob) {
	_, err := coll.cronitorApi.PutCronJob(cronjob)
	coll.cronjobs[cronjob.GetUID()] = cronjob
	if err != nil {
		log.WithFields(log.Fields{
			"namespace": cronjob.Namespace,
			"name":      cronjob.Name,
			"UID":       cronjob.UID,
		}).Errorf("Error adding cronjob to Cronitor: %s", err.Error())
	} else {
		log.WithFields(log.Fields{
			"namespace": cronjob.Namespace,
			"name":      cronjob.Name,
			"UID":       cronjob.UID,
		}).Info("Cronjob added to Cronitor")
	}
}

func (coll *CronJobCollection) RemoveCronJob(cronjob *v1beta1.CronJob) {
	delete(coll.cronjobs, cronjob.GetUID())
	log.WithFields(log.Fields{
		"namespace": cronjob.Namespace,
		"name":      cronjob.Name,
	}).Info("Cronjob no longer watched (Still present in Cronitor)")
}

func (coll *CronJobCollection) LoadAllExistingCronJobs() error {
	clientset := coll.clientset
	api := clientset.BatchV1beta1()
	listOptions := meta_v1.ListOptions{}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	cronjobs, err := api.CronJobs("").List(ctx, listOptions)
	if err != nil {
		return err
	}
	for _, cronjob := range cronjobs.Items {
		if included, err := NewCronitorConfigParser(&cronjob).included(); err == nil && included {
			coll.AddCronJob(&cronjob)
		}
	}
	coll.loaded = true
	log.Infof("Existing CronJobs have loaded. %d found; %d included based on configuration.", len(cronjobs.Items), len(coll.cronjobs))
	return nil
}

func (coll *CronJobCollection) StartWatchingAll() {
	cronJobWatcher := NewCronJobWatcher(*coll)

	coll.stopper = func() {
		cronJobWatcher.StopWatching()
	}

	cronJobWatcher.StartWatching()
}

func (coll CronJobCollection) StopWatchingAll() {
	if coll.stopper == nil {
		log.Warning("CronJobCollection.stopper() called, but it wasn't running")
	}
	coll.stopper()
	coll.stopper = nil
}

func (coll CronJobCollection) GetAllWatchedCronJobUIDs() []types.UID {
	var outList []types.UID
	for k, _ := range coll.cronjobs {
		outList = append(outList, k)
	}
	return outList
}

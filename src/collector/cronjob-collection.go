package collector

import (
	"context"
	log "github.com/sirupsen/logrus"
	"k8s.io/api/batch/v1beta1"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
)

type CronJobCollection struct {
	clientset *kubernetes.Clientset
	cronjobs  map[types.UID]*v1beta1.CronJob
	loaded    bool
	stopper   func()
}

func NewCronJobCollection(pathToKubeconfig string) (*CronJobCollection, error) {
	config, err := GetConfig(pathToKubeconfig)
	if err != nil {
		return nil, err
	}
	clientset := GetClientSet(config)
	return &CronJobCollection{
		clientset: clientset,
		cronjobs:  make(map[types.UID]*v1beta1.CronJob),
		loaded:    false,
	}, nil
}

func (coll *CronJobCollection) AddCronJob(cronjob *v1beta1.CronJob) {
	coll.cronjobs[cronjob.GetUID()] = cronjob
	log.WithFields(log.Fields{
		"namespace": cronjob.Namespace,
		"name":      cronjob.Name,
	}).Info("Cronjob added")
}

func (coll *CronJobCollection) RemoveCronJob(cronjob *v1beta1.CronJob) {
	delete(coll.cronjobs, cronjob.GetUID())
	log.WithFields(log.Fields{
		"namespace": cronjob.Namespace,
		"name":      cronjob.Name,
	}).Info("Cronjob removed")
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

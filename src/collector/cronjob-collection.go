package collector

import (
	"context"
	log "github.com/sirupsen/logrus"
	"k8s.io/api/batch/v1beta1"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

type CronJobCollection struct {
	cronjobs map[types.UID]*v1beta1.CronJob
	loaded bool
	watching bool
}

func NewCronJobCollection() CronJobCollection {
	return CronJobCollection{
		cronjobs: make(map[types.UID]*v1beta1.CronJob),
		loaded: false,
		watching: false,
	}
}

func (coll *CronJobCollection) AddCronJob(cronjob *v1beta1.CronJob) {
	coll.cronjobs[cronjob.GetUID()] = cronjob
	log.Debug("CronJob %s added", cronjob.Name)
}

func (coll *CronJobCollection) RemoveCronJob(cronjob *v1beta1.CronJob) {
	delete(coll.cronjobs, cronjob.GetUID())
}

func (coll *CronJobCollection) LoadAllExistingCronJobs() error {
	clientset := GetClientSet()
	api := clientset.BatchV1beta1()
	listOptions := meta_v1.ListOptions{}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	cronjobs, err := api.CronJobs("").List(ctx, listOptions)
	log.Debug("Cronjobs found: " + JsonAndPrint(cronjobs))
	if err != nil {
		return err
	}
	for _, cronjob := range cronjobs.Items {
		if included, err := NewCronitorConfigParser(&cronjob).included(); err != nil && included {
			coll.AddCronJob(&cronjob)
		}
	}
	coll.loaded = true
	log.Infof("Existing CronJobs have loaded. %d found.", len(coll.cronjobs))
	return nil
}

func (coll CronJobCollection) StartWatching() {
	NewCronJobWatcher(coll)
	coll.watching = true
}
package collector

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/cronitorio/cronitor-kubernetes/pkg"
	"github.com/cronitorio/cronitor-kubernetes/pkg/api"
	"github.com/cronitorio/cronitor-kubernetes/pkg/normalizer"
	"github.com/getsentry/sentry-go"
	v1 "k8s.io/api/batch/v1"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/version"
	"k8s.io/client-go/kubernetes"
)

type CronJobCollection struct {
	clientset           *kubernetes.Clientset
	serverVersion       *version.Info
	cronitorApi         *api.CronitorApi
	cronjobs            map[types.UID]*v1.CronJob
	kubernetesNamespace string
	loaded              bool
	stopper             func()
}

func NewCronJobCollection(pathToKubeconfig string, namespace string, cronitorApi *api.CronitorApi) (*CronJobCollection, error) {
	config, err := GetConfig(pathToKubeconfig)
	if err != nil {
		return nil, err
	}
	clientset := GetClientSet(config)
	discoveryClient := GetDiscoveryClient(config)
	serverVersion, err := discoveryClient.ServerVersion()
	if err != nil {
		return nil, err
	}
	return &CronJobCollection{
		clientset:           clientset,
		serverVersion:       serverVersion,
		cronitorApi:         cronitorApi,
		kubernetesNamespace: namespace,
		cronjobs:            make(map[types.UID]*v1.CronJob),
		loaded:              false,
	}, nil
}

func (coll *CronJobCollection) AddCronJob(cronjob *v1.CronJob) {
	_, err := coll.cronitorApi.PutCronJob(cronjob)
	coll.cronjobs[cronjob.GetUID()] = cronjob
	if err != nil {
		sentry.CaptureException(err)
		slog.Error("error adding cronjob to Cronitor",
			"namespace", cronjob.Namespace,
			"name", cronjob.Name,
			"UID", cronjob.UID,
			"error", err)
	} else {
		slog.Info("cronjob added to Cronitor",
			"namespace", cronjob.Namespace,
			"name", cronjob.Name,
			"UID", cronjob.UID)
	}
}

func (coll *CronJobCollection) RemoveCronJob(cronjob *v1.CronJob) {
	delete(coll.cronjobs, cronjob.GetUID())
	slog.Info("cronjob no longer watched (Still present in Cronitor)",
		"namespace", cronjob.Namespace,
		"name", cronjob.Name)
}

func (coll *CronJobCollection) LoadAllExistingCronJobs() error {
	clientset := coll.clientset
	listOptions := meta_v1.ListOptions{}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	// note that if it's global, coll.kubernetesNamespace will be "" (empty string)

	var cronjobs []v1.CronJob
	if version, err := coll.GetPreferredBatchApiVersion(); err != nil {
		return err
	} else if version == "v1" {
		api := clientset.BatchV1()
		cronJobList, err := api.CronJobs(coll.kubernetesNamespace).List(ctx, listOptions)
		if err != nil {
			return err
		}
		cronjobs = cronJobList.Items
	} else if version == "v1beta1" {
		api := clientset.BatchV1beta1()
		cronJobList, err := api.CronJobs(coll.kubernetesNamespace).List(ctx, listOptions)
		if err != nil {
			return err
		}
		for _, cj := range cronJobList.Items {
			cronjobs = append(cronjobs, *normalizer.CronJobConvertV1Beta1ToV1(&cj))
		}
	} else {
		return fmt.Errorf("unexpected apiVersion %s returned", version)
	}

	for _, cronjob := range cronjobs {
		if included, err := pkg.NewCronitorConfigParser(&cronjob).IsCronJobIncluded(); err == nil && included {
			coll.AddCronJob(&cronjob)
		}
	}
	coll.loaded = true
	slog.Info("existing CronJobs have loaded",
		"total_found", len(cronjobs),
		"included_count", len(coll.cronjobs))
	return nil
}

func (coll CronJobCollection) StartWatchingAll() {
	cronJobWatcher := NewCronJobWatcher(coll)

	coll.stopper = func() {
		cronJobWatcher.StopWatching()
	}

	cronJobWatcher.StartWatching()
}

func (coll CronJobCollection) StopWatchingAll() {
	if coll.stopper == nil {
		slog.Warn("CronJobCollection.stopper() called, but it wasn't running")
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

// CompareServerVersion will return 1 if the server version is higher than the compared version,
// -1 if it is lower than the compared version, or 0 if they are the same
func (coll CronJobCollection) CompareServerVersion(major int, minor int) (int, error) {
	serverVersionString := fmt.Sprintf("v%s.%s", coll.serverVersion.Major, coll.serverVersion.Minor)
	return version.CompareKubeAwareVersionStrings(fmt.Sprintf("v%d.%d", major, minor), serverVersionString), nil
}

func (coll CronJobCollection) GetPreferredBatchApiVersion() (string, error) {
	if result, err := coll.CompareServerVersion(1, 24); err != nil {
		return "", err
	} else if result >= 0 {
		return "v1", nil
	} else {
		return "v1beta1", nil
	}
}

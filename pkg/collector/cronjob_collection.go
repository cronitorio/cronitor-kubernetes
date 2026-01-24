package collector

import (
	"context"
	"fmt"
	"log/slog"
	"sync"

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
	cronjobsMu          sync.RWMutex // protects cronjobs map
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

func (coll *CronJobCollection) AddCronJob(cronjob *v1.CronJob) error {
	_, err := coll.cronitorApi.PutCronJob(cronjob)
	if err != nil {
		sentry.CaptureException(err)
		slog.Error("error adding cronjob to Cronitor",
			"namespace", cronjob.Namespace,
			"name", cronjob.Name,
			"UID", cronjob.UID,
			"error", err)
		return err
	}
	coll.cronjobsMu.Lock()
	coll.cronjobs[cronjob.GetUID()] = cronjob
	coll.cronjobsMu.Unlock()
	slog.Info("cronjob added to Cronitor",
		"namespace", cronjob.Namespace,
		"name", cronjob.Name,
		"UID", cronjob.UID)
	return nil
}

func (coll *CronJobCollection) RemoveCronJob(cronjob *v1.CronJob) {
	coll.cronjobsMu.Lock()
	delete(coll.cronjobs, cronjob.GetUID())
	coll.cronjobsMu.Unlock()
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

	// Collect all included cronjobs first
	var includedCronJobs []*v1.CronJob
	for i := range cronjobs {
		cronjob := &cronjobs[i]
		if included, err := pkg.NewCronitorConfigParser(cronjob).IsCronJobIncluded(); err == nil && included {
			includedCronJobs = append(includedCronJobs, cronjob)
		}
	}

	// Sync all cronjobs to Cronitor in a single batch API call
	if len(includedCronJobs) > 0 {
		_, err := coll.cronitorApi.PutCronJobs(includedCronJobs)
		if err != nil {
			sentry.CaptureException(err)
			slog.Error("failed to sync cronjobs to Cronitor - check your API key is a valid SDK key (not a telemetry key)",
				"cronjob_count", len(includedCronJobs),
				"error", err)
			return fmt.Errorf("failed to sync cronjobs to Cronitor: %w", err)
		}

		// Only add to local collection after successful API call
		coll.cronjobsMu.Lock()
		for _, cronjob := range includedCronJobs {
			coll.cronjobs[cronjob.GetUID()] = cronjob
			slog.Debug("cronjob synced to Cronitor",
				"namespace", cronjob.Namespace,
				"name", cronjob.Name,
				"UID", cronjob.UID)
		}
		coll.cronjobsMu.Unlock()
	}

	coll.loaded = true
	slog.Info("existing CronJobs have been synced to Cronitor",
		"total_found", len(cronjobs),
		"synced_count", len(coll.cronjobs))
	return nil
}

func (coll *CronJobCollection) StartWatchingAll() {
	cronJobWatcher := NewCronJobWatcher(*coll)

	coll.stopper = func() {
		cronJobWatcher.StopWatching()
	}

	cronJobWatcher.StartWatching()
}

func (coll *CronJobCollection) StopWatchingAll() {
	if coll.stopper == nil {
		slog.Warn("CronJobCollection.StopWatchingAll() called, but it wasn't running")
		return
	}
	coll.stopper()
	coll.stopper = nil
}

func (coll *CronJobCollection) GetAllWatchedCronJobUIDs() []types.UID {
	coll.cronjobsMu.RLock()
	defer coll.cronjobsMu.RUnlock()
	var outList []types.UID
	for k := range coll.cronjobs {
		outList = append(outList, k)
	}
	return outList
}

func (coll *CronJobCollection) IsTracked(uid types.UID) bool {
	coll.cronjobsMu.RLock()
	defer coll.cronjobsMu.RUnlock()
	_, exists := coll.cronjobs[uid]
	return exists
}

func (coll *CronJobCollection) GetCronJob(uid types.UID) (*v1.CronJob, bool) {
	coll.cronjobsMu.RLock()
	defer coll.cronjobsMu.RUnlock()
	cronjob, exists := coll.cronjobs[uid]
	return cronjob, exists
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

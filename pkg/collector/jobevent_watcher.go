package collector

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"regexp"
	"sync"
	"sync/atomic"

	"github.com/cronitorio/cronitor-kubernetes/pkg"
	"github.com/cronitorio/cronitor-kubernetes/pkg/api"
	"github.com/spf13/viper"
	v1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/runtime"
	apiWatch "k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/watch"
)

type EventHandler struct {
	collection     *CronJobCollection
	podFilter      *regexp.Regexp
	watchStartTime atomic.Pointer[meta_v1.Time]
}

func createPodFilter() *regexp.Regexp {
	if filterStr := viper.GetString("pod-filter"); filterStr != "" {
		slog.Debug("pod filter enabled", "filter", filterStr)
		return regexp.MustCompile(filterStr)
	}

	return nil
}

func (e EventHandler) fetchPod(namespace string, podName string) (*corev1.Pod, error) {
	clientset := e.collection.clientset
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	podsClient := clientset.CoreV1().Pods(namespace)
	pod, err := podsClient.Get(ctx, podName, meta_v1.GetOptions{})
	if err != nil {
		return nil, err
	}

	return pod, nil
}

// fetchPodByJobName grabs the Pod metadata from the Kubernetes API
func (e EventHandler) fetchPodByJobName(namespace string, jobName string) (*corev1.Pod, error) {
	clientset := e.collection.clientset
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	podsClient := clientset.CoreV1().Pods(namespace)
	listOptions := meta_v1.ListOptions{
		LabelSelector: fmt.Sprintf("job-name=%s", jobName),
	}
	pods, err := podsClient.List(ctx, listOptions)
	if err != nil {
		return nil, err
	}

	switch itemsLength := len(pods.Items); itemsLength {
	case 0:
		return nil, fmt.Errorf("no pod matching job name %s found", jobName)
	case 1:
		return &pods.Items[0], nil
	default:
		return nil, fmt.Errorf("more than one pod matching job name %s, %d found", jobName, itemsLength)
	}
}

// fetchJob gets the Job's information from the Kubernetes API.
func (e EventHandler) fetchJob(namespace string, name string) (*v1.Job, error) {
	clientset := e.collection.clientset
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	jobsClient := clientset.BatchV1().Jobs(namespace)
	job, err := jobsClient.Get(ctx, name, meta_v1.GetOptions{})
	if err != nil {
		return nil, JobNotFoundError{namespace, name, err}
	}
	return job, nil
}

func (e EventHandler) fetchCronJob(uid types.UID) (*v1.CronJob, error) {
	if cronjob, ok := e.collection.GetCronJob(uid); ok {
		return cronjob, nil
	}
	return nil, fmt.Errorf("cronjob %s not found in collection", string(uid))
}

func (e EventHandler) fetchPodLogs(pod *corev1.Pod) (string, error) {
	podLogOpts := corev1.PodLogOptions{}
	clientset := e.collection.clientset
	req := clientset.CoreV1().Pods(pod.Namespace).GetLogs(pod.Name, &podLogOpts)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	podLogs, err := req.Stream(ctx)
	if err != nil {
		return "", err
	}
	defer podLogs.Close()

	buf := new(bytes.Buffer)
	_, err = io.Copy(buf, podLogs)
	if err != nil {
		return "", err
	}

	str := buf.String()
	return str, nil
}

// FetchAndCheckJobEvent is a single-pass method that fetches a Job, validates
// the owner chain to a watched CronJob, and retrieves the associated Pod and logs.
// It replaces the old CheckJobIsWatched + FetchObjectsFromJobEvent combination,
// eliminating redundant API calls.
func (e EventHandler) FetchAndCheckJobEvent(namespace, jobName string, includeLogs bool) (pod *corev1.Pod, logs string, job *v1.Job, cronjob *v1.CronJob, watched bool, err error) {
	// 1. Single GET for the Job
	job, err = e.fetchJob(namespace, jobName)
	if err != nil {
		return
	}

	// 2. Validate owner chain -> CronJob
	if len(job.ObjectMeta.OwnerReferences) == 0 {
		return
	}
	ownerReference := job.ObjectMeta.OwnerReferences[0]
	if ownerReference.Kind != "CronJob" {
		return
	}

	// 3. Check if tracked (in-memory)
	ownerUID := ownerReference.UID
	if !e.collection.IsTracked(ownerUID) {
		return
	}
	watched = true

	// 4. Fetch CronJob (in-memory)
	cronjob, err = e.fetchCronJob(ownerUID)
	if err != nil {
		return
	}

	// 5. Single LIST for the Pod
	pod, err = e.fetchPodByJobName(namespace, jobName)
	if err != nil {
		return
	}

	// 6. Conditionally fetch logs (only on terminal events to avoid duplicates)
	if includeLogs && viper.GetBool("ship-logs") {
		logs, _ = e.fetchPodLogs(pod)
	}
	return
}

// FetchAndCheckPodEvent is a single-pass method that fetches a Pod, walks the
// owner chain through Job to CronJob, checks whether the CronJob is watched,
// and retrieves logs. It replaces the old fetchJobByPod + CheckJobIsWatched +
// FetchObjectsFromPodEvent combination, eliminating redundant API calls.
func (e EventHandler) FetchAndCheckPodEvent(namespace, podName string, includeLogs bool) (pod *corev1.Pod, logs string, job *v1.Job, cronjob *v1.CronJob, watched bool, err error) {
	// 1. Single GET for the Pod
	pod, err = e.fetchPod(namespace, podName)
	if err != nil {
		err = PodNotFoundError{namespace, podName, err}
		return
	}

	// 2. Extract Job owner from pod.OwnerReferences (in-memory)
	var jobOwnerRef *meta_v1.OwnerReference
	for i := range pod.OwnerReferences {
		if pod.OwnerReferences[i].Kind == "Job" {
			jobOwnerRef = &pod.OwnerReferences[i]
			break
		}
	}
	if jobOwnerRef == nil {
		// Pod is not owned by a Job — not a CronJob pod
		return
	}

	// 3. Single GET for the Job
	job, err = e.fetchJob(namespace, jobOwnerRef.Name)
	if err != nil {
		return
	}

	// 4. Validate owner chain -> CronJob
	if len(job.ObjectMeta.OwnerReferences) == 0 {
		return
	}
	ownerReference := job.ObjectMeta.OwnerReferences[0]
	if ownerReference.Kind != "CronJob" {
		return
	}

	// 5. Check if tracked (in-memory)
	ownerUID := ownerReference.UID
	if !e.collection.IsTracked(ownerUID) {
		return
	}
	watched = true

	// 6. Fetch CronJob (in-memory)
	cronjob, err = e.fetchCronJob(ownerUID)
	if err != nil {
		return
	}

	// 7. Conditionally fetch logs (only on terminal events to avoid duplicates)
	if includeLogs {
		logs, _ = e.fetchPodLogs(pod)
	}
	return
}

func (e EventHandler) CheckPodFilter(podName string) bool {
	if e.podFilter == nil {
		return true
	}
	return e.podFilter.MatchString(podName)
}

func (e EventHandler) OnAdd(obj interface{}) {
	event := obj.(*corev1.Event)
	eventTime := event.LastTimestamp

	switch event.InvolvedObject.Kind {
	case "Job":
		typedEvent := pkg.JobEvent(*event)

		// If this event is an older, stale event--e.g., it happened before this version of the agent started to run--
		// then ignore the event
		watchStart := e.watchStartTime.Load()
		if watchStart != nil && eventTime.Before(watchStart) {
			slog.Info("ignored event from the past",
				"name", typedEvent.InvolvedObject.Name,
				"kind", typedEvent.InvolvedObject.Kind,
				"eventMessage", typedEvent.Message,
				"eventReason", typedEvent.Reason,
				"eventTime", eventTime,
				"watchStartTime", *watchStart)
			return
		}

		// Only fetch logs on terminal events (Completed, BackoffLimitExceeded) to avoid
		// shipping duplicate/incomplete logs on SuccessfulCreate (run) events
		isTerminalEvent := typedEvent.Reason == "Completed" || typedEvent.Reason == "BackoffLimitExceeded"
		pod, logs, job, cronjob, watched, err := e.FetchAndCheckJobEvent(typedEvent.InvolvedObject.Namespace, typedEvent.InvolvedObject.Name, isTerminalEvent)
		if err != nil {
			slog.Warn("could not fetch objects related to event", "error", err)
			return
		}
		if watched {
			slog.Info("job event added",
				"name", typedEvent.InvolvedObject.Name,
				"kind", typedEvent.InvolvedObject.Kind,
				"eventMessage", typedEvent.Message,
				"eventReason", typedEvent.Reason)
			_ = e.collection.cronitorApi.MakeAndSendTelemetryJobEventAndLogs(&typedEvent, logs, pod, job, cronjob)
		}

	case "Pod":
		typedEvent := pkg.PodEvent(*event)

		// Before we do any additional work loading objects from the API, check our pod filter.
		// Users with busy clusters can use the pod filter to reduce API load
		if e.CheckPodFilter(typedEvent.InvolvedObject.Name) == false {
			slog.Debug("pod excluded by pod filter", "pod", typedEvent.InvolvedObject.Name)
			return
		}

		// If this event is an older, stale event--e.g., it happened before this version of the agent started to run--
		// then ignore the event
		watchStart := e.watchStartTime.Load()
		if watchStart != nil && eventTime.Before(watchStart) {
			slog.Info("ignored event from the past",
				"name", typedEvent.InvolvedObject.Name,
				"kind", typedEvent.InvolvedObject.Kind,
				"eventMessage", typedEvent.Message,
				"eventReason", typedEvent.Reason,
				"eventTime", eventTime,
				"watchStartTime", *watchStart)
			return
		}

		// If it's not an event we care about, we don't want to do all of the work of calling the Kubernetes API
		// to get all of the related objects, which would put heavy load on it given all of the pod events.
		// So we check early against our pod event list, even though it's somewhat redundant.
		if _, err := api.TranslatePodEventReasonToTelemetryEventStatus(&typedEvent); err != nil {
			return
		}

		podNamespace := typedEvent.InvolvedObject.Namespace
		podName := typedEvent.InvolvedObject.Name

		// Only fetch logs on terminal events (BackOff/fail) to avoid
		// shipping duplicate/incomplete logs on Started (run) events
		isTerminalEvent := typedEvent.Reason == "BackOff"
		pod, logs, job, cronjob, watched, err := e.FetchAndCheckPodEvent(podNamespace, podName, isTerminalEvent)
		if err != nil {
			switch t := err.(type) {
			case PodNotFoundError:
				slog.Debug("pod not found, probably a stale event",
					"namespace", t.podNamespace,
					"pod", t.podName,
					"error", errors.Unwrap(t))
			case JobNotFoundError:
				slog.Debug("job not found, probably a stale event",
					"namespace", t.jobNamespace,
					"job", t.jobName,
					"error", errors.Unwrap(t))
			default:
				slog.Error("unexpected error fetching the job for pod",
					"namespace", podNamespace,
					"pod", podName,
					"errorType", fmt.Sprintf("%T", err),
					"error", err)
			}
			return
		}

		if !watched {
			if job == nil {
				slog.Debug("pod does not belong to a job; discarded",
					"namespace", podNamespace,
					"pod", podName)
			}
			return
		}

		// By default, skip Pod "Started" → run events because the Job-level SuccessfulCreate
		// already sends a run event. Users can opt in via the send-pod-start-event annotation.
		if typedEvent.Reason == "Started" && !pkg.NewCronitorConfigParser(cronjob).SendPodStartEvent() {
			slog.Debug("pod Started event skipped (opt in via send-pod-start-event annotation)",
				"namespace", podNamespace,
				"pod", podName)
			return
		}

		slog.Info("pod event added",
			"name", typedEvent.InvolvedObject.Name,
			"kind", typedEvent.InvolvedObject.Kind,
			"eventMessage", typedEvent.Message,
			"eventReason", typedEvent.Reason,
			"eventTime", typedEvent.EventTime,
			"lastTimestamp", typedEvent.LastTimestamp)
		_ = e.collection.cronitorApi.MakeAndSendTelemetryPodEventAndLogs(&typedEvent, logs, pod, job, cronjob)

	default:
		return
	}
}

type WatchWrapper struct {
	watcher        apiWatch.Interface
	eventHandler   *EventHandler
	stopped        bool
	workerPoolSize int
}

func (w *WatchWrapper) Start() {
	defer runtime.HandleCrash()
	slog.Info("the jobs watcher is starting...")

	sem := make(chan struct{}, w.workerPoolSize)
	var wg sync.WaitGroup

	ch := w.watcher.ResultChan()
	for event := range ch {
		sem <- struct{}{}
		wg.Add(1)
		go func(obj interface{}) {
			defer func() { <-sem; wg.Done() }()
			w.eventHandler.OnAdd(obj)
		}(event.Object)
	}
	wg.Wait()
	if !w.stopped {
		slog.Error("the job watcher stopped unexpectedly")
	}
}

func (w *WatchWrapper) Stop() {
	slog.Info("the jobs watcher is stopping...")
	w.stopped = true
	w.watcher.Stop()
}

func NewJobsEventWatcher(collection *CronJobCollection) *WatchWrapper {
	clientset := collection.clientset
	namespace := corev1.NamespaceAll
	if collection.kubernetesNamespace != "" {
		namespace = collection.kubernetesNamespace
	}

	eventHandler := &EventHandler{
		collection: collection,
		podFilter:  createPodFilter(),
	}
	// Initialize watchStartTime
	now := meta_v1.Now()
	eventHandler.watchStartTime.Store(&now)

	watchFunc := func(options meta_v1.ListOptions) (apiWatch.Interface, error) {
		// Update watchStartTime when the watch restarts
		now := meta_v1.Now()
		eventHandler.watchStartTime.Store(&now)
		return clientset.CoreV1().Events(namespace).Watch(context.Background(), meta_v1.ListOptions{})
	}

	watcher, err := watch.NewRetryWatcher("1", &cache.ListWatch{WatchFunc: watchFunc})
	if err != nil {
		panic(err)
	}

	return &WatchWrapper{
		watcher:        watcher,
		eventHandler:   eventHandler,
		workerPoolSize: 4,
	}
}

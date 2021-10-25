package collector

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"github.com/cronitorio/cronitor-kubernetes/pkg"
	"github.com/cronitorio/cronitor-kubernetes/pkg/api"
	log "github.com/sirupsen/logrus"
	"io"
	v1 "k8s.io/api/batch/v1"
	"k8s.io/api/batch/v1beta1"
	corev1 "k8s.io/api/core/v1"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/tools/cache"
)

type EventHandler struct {
	collection *CronJobCollection
	stopper    chan struct{}
	informer   cache.SharedInformer
}

func (e EventHandler) Start() {
	defer runtime.HandleCrash()
	log.Info("The jobs watcher is starting...")
	go e.informer.Run(e.stopper)
}

func (e *EventHandler) Stop() {
	log.Info("The jobs watcher is stopping...")
	close(e.stopper)
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

func (e EventHandler) fetchJobByPod(namespace string, podName string) (*v1.Job, error) {
	clientset := e.collection.clientset
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	podsClient := clientset.CoreV1().Pods(namespace)
	pod, err := podsClient.Get(ctx, podName, meta_v1.GetOptions{})
	if err != nil {
		return nil, PodNotFoundError{namespace, podName, err}
	}

	var ownerReference = new(meta_v1.OwnerReference)
	for _, ref := range pod.OwnerReferences {
		if ref.Kind == "Job" {
			ownerReference = &ref
			break
		}
	}
	if ownerReference == nil {
		// If there is no job owning the pod at all,
		// then it's definitely not a CronJob pod, but it's
		// also not an error.
		return nil, nil
	}

	return e.fetchJob(namespace, ownerReference.Name)
}

// fetchPodByJobName grabs the Pod metadata from the Kubernetes API
func (e EventHandler) fetchPodByJobName(namespace string, jobName string) (*corev1.Pod, error) {
	// This could potentially be moved off of EventHandler into its own kube package
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

func (e EventHandler) fetchJob(namespace string, name string) (*v1.Job, error) {
	// Grab the Job's information from the Kubernetes API.
	// Note: this might be a bit expensive, should we memoize it when possible?
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

func (e EventHandler) fetchCronJob(uid types.UID) (*v1beta1.CronJob, error) {
	cronjobs := e.collection.cronjobs
	if val, ok := cronjobs[uid]; ok {
		return val, nil
	} else {
		return nil, fmt.Errorf("cronjob %s not found in collection", string(uid))
	}
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

func (e EventHandler) FetchObjectsFromPodEvent(event *pkg.PodEvent) (pod *corev1.Pod, logs string, job *v1.Job, cronjob *v1beta1.CronJob, err error) {
	namespace := event.InvolvedObject.Namespace
	podName := event.InvolvedObject.Name

	pod, err = e.fetchPod(namespace, podName)
	if err != nil {
		return
	}
	job, err = e.fetchJobByPod(namespace, podName)
	if err != nil {
		return
	}
	ownerReference := job.ObjectMeta.OwnerReferences[0]
	if ownerReference.Kind != "CronJob" {
		err = fmt.Errorf("expected ownerReference of CronJob, got %s", ownerReference.Kind)
	}
	ownerUID := ownerReference.UID
	cronjob, err = e.fetchCronJob(ownerUID)
	if err != nil {
		return
	}

	// Logs may not be available because the pod hasn't started yet, or maybe logs just aren't available.
	// In that case, ignore. Logs are retrieved on a best-effort basis.
	logs, _ = e.fetchPodLogs(pod)
	return
}

func (e EventHandler) FetchObjectsFromJobEvent(event *pkg.JobEvent) (pod *corev1.Pod, logs string, job *v1.Job, cronjob *v1beta1.CronJob, err error) {
	namespace := event.InvolvedObject.Namespace
	jobName := event.InvolvedObject.Name
	job, err = e.fetchJob(namespace, jobName)
	if err != nil {
		return
	}
	ownerReference := job.ObjectMeta.OwnerReferences[0]
	if ownerReference.Kind != "CronJob" {
		err = fmt.Errorf("expected ownerReference of CronJob, got %s", ownerReference.Kind)
	}
	ownerUID := ownerReference.UID
	cronjob, err = e.fetchCronJob(ownerUID)
	if err != nil {
		return
	}

	pod, err = e.fetchPodByJobName(namespace, jobName)
	if err != nil {
		return
	}

	// Logs may not be available because the pod hasn't started yet, or maybe logs just aren't available.
	// In that case, ignore. Logs are retrieved on a best-effort basis.
	logs, _ = e.fetchPodLogs(pod)
	return
}

func (e EventHandler) CheckJobIsWatched(jobNamespace string, jobName string) bool {
	job, err := e.fetchJob(jobNamespace, jobName)
	if err != nil {
		// Job doesn't exist
		return false
	}

	ownerReference := job.ObjectMeta.OwnerReferences[0]
	if ownerReference.Kind != "CronJob" {
		return false
	}
	ownerUID := ownerReference.UID
	for _, b := range e.collection.GetAllWatchedCronJobUIDs() {
		if b == ownerUID {
			return true
		}
	}
	return false
}

func (e EventHandler) OnAdd(obj interface{}) {
	event := obj.(*corev1.Event)

	switch event.InvolvedObject.Kind {
	case "Job":
		typedEvent := pkg.JobEvent(*event)
		if e.CheckJobIsWatched(typedEvent.InvolvedObject.Namespace, typedEvent.InvolvedObject.Name) {
			log.WithFields(log.Fields{
				"name":         typedEvent.InvolvedObject.Name,
				"kind":         typedEvent.InvolvedObject.Kind,
				"eventMessage": typedEvent.Message,
				"eventReason":  typedEvent.Reason,
			}).Info("Job event added")
			pod, logs, job, cronjob, err := e.FetchObjectsFromJobEvent(&typedEvent)
			if err != nil {
				log.Warnf("could not fetch objects related to event: %s", err.Error())
				return
			}
			_ = e.collection.cronitorApi.MakeAndSendTelemetryJobEventAndLogs(&typedEvent, logs, pod, job, cronjob)
		}

	case "Pod":
		typedEvent := pkg.PodEvent(*event)

		// If it's not an event we care about, we don't want to do all of the work of calling the Kubernetes API
		// to get all of the related objects, which would put heavy load on it given all of the pod events.
		// So we check early against our pod event list, even though it's somewhat redundant.
		if _, err := api.TranslatePodEventReasonToTelemteryEventStatus(&typedEvent); err != nil {
			return
		}
		podNamespace := typedEvent.InvolvedObject.Namespace
		podName := typedEvent.InvolvedObject.Name

		job, err := e.fetchJobByPod(podNamespace, podName)
		if err != nil {
			switch t := err.(type) {
			case PodNotFoundError:
				log.Debugf("pod %s/%s not found, probably a stale event: %v", t.podNamespace, t.podName, errors.Unwrap(t))
			case JobNotFoundError:
				log.Debugf("job %s/%s not found, probably a stale event: %v", t.jobNamespace, t.jobName, errors.Unwrap(t))
			default:
				log.Errorf("unexpected error fetching the job for pod %s/%s (error %T): %v", podNamespace, podName, err, err)
			}
			return
		} else if job == nil {
			log.Debugf("pod %s/%s does not belong to a job; discarded", podNamespace, podName)
			return
		}

		// Right now we end up fetching the job twice. We need to refactor out
		// the double-request, but this is acceptable for now.
		if e.CheckJobIsWatched(job.Namespace, job.Name) {
			log.WithFields(log.Fields{
				"name":         typedEvent.InvolvedObject.Name,
				"kind":         typedEvent.InvolvedObject.Kind,
				"eventMessage": typedEvent.Message,
				"eventReason":  typedEvent.Reason,
			}).Info("Pod event added")
			pod, logs, job, cronjob, err := e.FetchObjectsFromPodEvent(&typedEvent)
			if err != nil {
				log.Warnf("could not fetch objects related to event: %s", err.Error())
				return
			}
			_ = e.collection.cronitorApi.MakeAndSendTelemetryPodEventAndLogs(&typedEvent, logs, pod, job, cronjob)
		}

	default:
		return
	}
}

// TODO: We may actually only need the OnAdd, since events don't seem to be deleted
// They are _sometimes_ updated when events are combined, but that seems to only happen on long-lived
// things like a CronJob, not the short-lived Jobs
func (e EventHandler) OnDelete(obj interface{}) {
	event := obj.(*corev1.Event)
	if event.InvolvedObject.Kind != "Job" {
		return
	}

	if e.CheckJobIsWatched(event.InvolvedObject.Namespace, event.InvolvedObject.Name) {
		log.WithFields(log.Fields{
			"name":         event.InvolvedObject.Name,
			"kind":         event.InvolvedObject.Kind,
			"eventMessage": event.Message,
			"eventReason":  event.Reason,
		}).Info("Job event removed")
	}
}

func (e EventHandler) OnUpdate(oldObj interface{}, newObj interface{}) {
	oldEvent := oldObj.(*corev1.Event)
	newEvent := newObj.(*corev1.Event)
	if newEvent.InvolvedObject.Kind != "Job" {
		return
	}

	if e.CheckJobIsWatched(newEvent.InvolvedObject.Namespace, newEvent.InvolvedObject.Name) {
		log.WithFields(log.Fields{
			"name":         oldEvent.InvolvedObject.Name,
			"kind":         newEvent.InvolvedObject.Kind,
			"eventMessage": newEvent.Message,
			"eventReason":  newEvent.Reason,
		}).Info("Job event updated (...somehow?)")
	}
}

func NewJobsEventWatcher(collection *CronJobCollection) *EventHandler {
	clientset := collection.clientset
	factory := informers.NewSharedInformerFactory(clientset, 0)
	informer := factory.Core().V1().Events().Informer()

	eventHandler := &EventHandler{
		collection: collection,
		stopper:    make(chan struct{}),
		informer:   informer,
	}

	informer.AddEventHandler(eventHandler)
	return eventHandler
}
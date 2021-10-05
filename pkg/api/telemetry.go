package api

import (
	"bytes"
	"fmt"
	"github.com/cronitorio/cronitor-kubernetes/pkg"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"io/ioutil"
	v1 "k8s.io/api/batch/v1"
	"k8s.io/api/batch/v1beta1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

type TelemetryEventStatus string

const (
	Run      TelemetryEventStatus = "run"
	Complete TelemetryEventStatus = "complete"
	Fail     TelemetryEventStatus = "fail"
	Ok       TelemetryEventStatus = "ok"
)

type TelemetryEvent struct {
	CronJob   *v1beta1.CronJob
	Event     TelemetryEventStatus
	Message   string
	ErrorLogs string
	// Series is a UUID to distinguish different sets of pings in a series.
	// In Kubernetes, this is loosely analogous to a Job instance of a CronJob, so we use the
	// Job's UUID, which will stay stable even on multiple pod retries.
	Series   *types.UID
	ExitCode *int
	Env      string
	// Host is the Kubernetes node that the pod is running on
	Host string
}

func TranslatePodEventReasonToTelemteryEventStatus(event *pkg.PodEvent) (*TelemetryEventStatus, error) {
	var Event TelemetryEventStatus
	switch reason := event.Reason; reason {
	case "Started":
		Event = Run
	case "BackOff":
		Event = Fail
	default:
		return nil, fmt.Errorf("unknown pod event reason \"%s\" received", reason)
	}
	return &Event, nil
}

func translateJobEventReasonToTelemetryEventStatus(event *pkg.JobEvent) (*TelemetryEventStatus, error) {
	var Event TelemetryEventStatus
	switch reason := event.Reason; reason {
	case "SuccessfulCreate":
		Event = Run
	case "Completed":
		Event = Complete
	case "BackoffLimitExceeded":
		Event = Fail
	default:
		return nil, fmt.Errorf("unknown job event reason \"%s\" received", reason)
	}
	return &Event, nil
}

func NewTelemetryEventFromKubernetesPodEvent(event *pkg.PodEvent, logs string, pod *corev1.Pod, job *v1.Job, cronjob *v1beta1.CronJob) (*TelemetryEvent, error) {
	CronJob := cronjob
	Message := event.Message
	ErrorLogs := logs
	Series := job.UID

	Event, err := TranslatePodEventReasonToTelemteryEventStatus(event)
	if err != nil {
		return nil, err
	}

	Host := pod.Spec.NodeName

	telemetryEvent := TelemetryEvent{
		CronJob: CronJob,
		Event: *Event,
		Message: Message,
		ErrorLogs: ErrorLogs,
		Series: &Series,
		Host: Host,
	}

	if env := pkg.NewCronitorConfigParser(cronjob).GetEnvironment(); env != "" {
		telemetryEvent.Env = env
	}

	return &telemetryEvent, nil
}

func NewTelemetryEventFromKubernetesJobEvent(event *pkg.JobEvent, logs string, pod *corev1.Pod, job *v1.Job, cronjob *v1beta1.CronJob) (*TelemetryEvent, error) {
	CronJob := cronjob
	Message := event.Message
	ErrorLogs := logs
	Series := job.UID

	Event, err := translateJobEventReasonToTelemetryEventStatus(event)
	if err != nil {
		return nil, err
	}

	Host := pod.Spec.NodeName

	telemetryEvent := TelemetryEvent{
		CronJob:   CronJob,
		Event:     *Event,
		Message:   Message,
		ErrorLogs: ErrorLogs,
		Series:    &Series,
		Host:      Host,
	}

	if env := pkg.NewCronitorConfigParser(cronjob).GetEnvironment(); env != "" {
		telemetryEvent.Env = env
	}

	return &telemetryEvent, nil
}

func (t TelemetryEvent) Encode() string {
	q := url.Values{}
	if t.Message != "" {
		q.Add("message", t.Message)
	}
	if t.Series != nil {
		q.Add("series", string(*t.Series))
	}
	if t.ExitCode != nil {
		q.Add("exit_code", strconv.Itoa(*t.ExitCode))
	}
	if t.Env != "" {
		q.Add("env", t.Env)
	}
	if t.Host != "" {
		q.Add("host", t.Host)
	}
	return q.Encode()
}

// telemetryUrl generates the URL required to send events to the Telemetry API.
func (api CronitorApi) telemetryUrl(params *TelemetryEvent) string {
	cronitorID := pkg.NewCronitorConfigParser(params.CronJob).GetCronitorID()
	var hostname string
	if hostnameOverride := viper.GetString("hostname-override"); hostnameOverride != "" {
		hostname = hostnameOverride
	} else {
		hostname = "https://cronitor.link"
	}
	return fmt.Sprintf("%s/ping/%s/%s/%s", hostname, api.ApiKey, cronitorID, params.Event)
}

func (api CronitorApi) sendTelemetryPostRequest(params *TelemetryEvent) ([]byte, error) {
	telemetryUrl := api.telemetryUrl(params)
	req, err := http.NewRequest("POST", telemetryUrl, bytes.NewBuffer([]byte{}))
	if err != nil {
		return nil, err
	}
	req.URL.RawQuery = params.Encode()
	client := &http.Client{
		Timeout: 120 * time.Second,
	}
	response, err := client.Do(req)
	if err != nil {
		return nil, CronitorApiError{
			Err:      err,
			Response: response,
		}
	}
	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return nil, CronitorApiError{
			fmt.Errorf("error response code %d returned", response.StatusCode),
			response,
		}
	}

	body, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return nil, err
	}
	return body, nil
}

func (api CronitorApi) sendTelemetryEvent(t *TelemetryEvent) error {
	if api.DryRun {
		return nil
	}

	_, err := api.sendTelemetryPostRequest(t)
	if err != nil {
		return err
	}

	return nil
}

func (api CronitorApi) MakeAndSendTelemetryPodEventAndLogs(event *pkg.PodEvent, logs string, pod *corev1.Pod, job *v1.Job, cronjob *v1beta1.CronJob) error {
	telemetryEvent, err := NewTelemetryEventFromKubernetesPodEvent(event, logs, pod, job, cronjob)
	if err != nil {
		return err
	}

	defer func(telemetryEvent *TelemetryEvent, pod *corev1.Pod) {
		_, err := api.ShipLogData(telemetryEvent)
		if err != nil {
			if strings.Contains(err.Error(), "no such host") {
					// This error is due entirely to logs.cronitor.link not existing yet,
					// so discard for now
					return
			}
			log.Errorf("unexpected error sending log data for pod %s/%s: %v", pod.Namespace, pod.Name, err)
		}
	}(telemetryEvent, pod)

	return api.sendTelemetryEvent(telemetryEvent)
}

func (api CronitorApi) MakeAndSendTelemetryJobEventAndLogs(event *pkg.JobEvent, logs string, pod *corev1.Pod, job *v1.Job, cronjob *v1beta1.CronJob) error {
	telemetryEvent, err := NewTelemetryEventFromKubernetesJobEvent(event, logs, pod, job, cronjob)
	if err != nil {
		return err
	}

	defer func(telemetryEvent *TelemetryEvent, job *v1.Job) {
		_, err := api.ShipLogData(telemetryEvent)
		if err != nil {
			if strings.Contains(err.Error(), "no such host") {
				// This error is due entirely to logs.cronitor.link not existing yet,
				// so discard for now
				return
			}
			log.Errorf("unexpected error sending log data for job %s/%s: %v", job.Namespace, job.Name, err)
		}
	}(telemetryEvent, job)

	return api.sendTelemetryEvent(telemetryEvent)
}

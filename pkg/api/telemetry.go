package api

import (
	"bytes"
	"fmt"
	log "github.com/sirupsen/logrus"
	"io/ioutil"
	v1 "k8s.io/api/batch/v1"
	"k8s.io/api/batch/v1beta1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"net/http"
	"net/url"
	"strconv"
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
	CronJob  *v1beta1.CronJob
	Event    TelemetryEventStatus
	Message  string
	Series   *types.UID
	ExitCode *int
	// Metric
	Env  string
	Host string // need to fetch from Pod
}

func NewTelemetryEventFromKubernetesEvent(event *corev1.Event, pod *corev1.Pod, job *v1.Job, cronjob *v1beta1.CronJob) (*TelemetryEvent, error) {
	if event.InvolvedObject.Kind != "Job" {
		log.Fatal("an event was passed to telemetry that doesn't belong to a job")
	}

	CronJob := cronjob
	Message := event.Message

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

	Host := pod.Spec.NodeName
	telemetryEvent := TelemetryEvent{
		CronJob: CronJob,
		Event:   Event,
		Message: Message,
		Host:    Host,
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
	return fmt.Sprintf("https://cronitor.link/ping/%s/%s", api.ApiKey, string(params.CronJob.GetUID()))
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

func (api CronitorApi) MakeAndSendTelemetryEvent(event *corev1.Event, pod *corev1.Pod, job *v1.Job, cronjob *v1beta1.CronJob) error {
	telemetryEvent, err := NewTelemetryEventFromKubernetesEvent(event, pod, job, cronjob)
	if err != nil {
		return err
	}

	_, err = api.sendTelemetryPostRequest(telemetryEvent)
	if err != nil {
		return err
	}

	return nil
}
package api

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"log/slog"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/cronitorio/cronitor-kubernetes/pkg"
	"github.com/spf13/viper"
	v1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
)

type TelemetryEventStatus string

const LogsTruncationLength = 2000

const (
	Run      TelemetryEventStatus = "run"
	Complete TelemetryEventStatus = "complete"
	Fail     TelemetryEventStatus = "fail"
	Ok       TelemetryEventStatus = "ok"
	Logs     TelemetryEventStatus = "logs"
)

type TelemetryEvent struct {
	CronJob   *v1.CronJob
	Event     TelemetryEventStatus
	Message   string
	ErrorLogs string
	// Timestamp in seconds with 3 decimal places for microsecond
	Timestamp string
	// Series is a UUID to distinguish different sets of pings in a series.
	// In Kubernetes, this is loosely analogous to a Job instance of a CronJob, so we use the
	// Job's UUID, which will stay stable even on multiple pod retries.
	Series   *types.UID
	ExitCode *int
	Env      string
	// Host is the Kubernetes node that the pod is running on
	Host   string
	Metric string
}

func (t TelemetryEvent) CreateLogTelemetryEvent() *TelemetryEvent {
	t.Event = Logs
	if utf8.RuneCountInString(t.ErrorLogs) > LogsTruncationLength {
		t.Message = string([]rune(t.ErrorLogs)[:LogsTruncationLength])
	} else {
		t.Message = t.ErrorLogs
	}
	t.Metric = fmt.Sprintf("length:%d", len(t.ErrorLogs))
	return &t
}

func TranslatePodEventReasonToTelemetryEventStatus(event *pkg.PodEvent) (*TelemetryEventStatus, error) {
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

func NewTelemetryEventFromKubernetesPodEvent(event *pkg.PodEvent, logs string, pod *corev1.Pod, job *v1.Job, cronjob *v1.CronJob) (*TelemetryEvent, error) {
	CronJob := cronjob
	Message := event.Message
	ErrorLogs := logs
	Series := job.UID
	eventTime := event.LastTimestamp

	Event, err := TranslatePodEventReasonToTelemetryEventStatus(event)
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
		Timestamp: strconv.FormatInt(eventTime.Unix(), 10),
	}

	if env := pkg.NewCronitorConfigParser(cronjob).GetEnvironment(); env != "" {
		telemetryEvent.Env = env
	}

	return &telemetryEvent, nil
}

func NewTelemetryEventFromKubernetesJobEvent(event *pkg.JobEvent, logs string, pod *corev1.Pod, job *v1.Job, cronjob *v1.CronJob) (*TelemetryEvent, error) {
	CronJob := cronjob
	Message := event.Message
	ErrorLogs := logs
	Series := job.UID
	eventTime := event.LastTimestamp

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
		Timestamp: strconv.FormatInt(eventTime.Unix(), 10),
	}

	cronitorConfigParser := pkg.NewCronitorConfigParser(cronjob)

	if env := cronitorConfigParser.GetEnvironment(); env != "" {
		telemetryEvent.Env = env
	}

	if ok, _ := cronitorConfigParser.EnabledAutoComplete(); !ok && telemetryEvent.Event == Complete {
		telemetryEvent.Event = Logs
		telemetryEvent.Message = fmt.Sprintf("Job %s is completed with status %s", job.Name, Complete)
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
	if t.Timestamp != "" {
		q.Add("stamp", t.Timestamp)
	}
	if t.Metric != "" {
		q.Add("metric", t.Metric)
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
	req.Header.Add("User-Agent", api.UserAgent)
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
	defer response.Body.Close()
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

func (api CronitorApi) MakeAndSendTelemetryPodEventAndLogs(event *pkg.PodEvent, logs string, pod *corev1.Pod, job *v1.Job, cronjob *v1.CronJob) error {
	telemetryEvent, err := NewTelemetryEventFromKubernetesPodEvent(event, logs, pod, job, cronjob)
	if err != nil {
		return err
	}

	defer func(telemetryEvent *TelemetryEvent, pod *corev1.Pod) {
		if !viper.GetBool("ship-logs") || len(telemetryEvent.ErrorLogs) == 0 {
			return
		}
		_, err := api.ShipLogData(telemetryEvent)
		if err != nil {
			if strings.Contains(err.Error(), "no such host") {
				// This error is due entirely to logs.cronitor.link not existing yet,
				// so discard for now
				return
			}
			slog.Error("unexpected error sending log data for pod",
				"namespace", pod.Namespace,
				"pod", pod.Name,
				"error", err)
		}
		logTelemetryEvent := telemetryEvent.CreateLogTelemetryEvent()
		err = api.sendTelemetryEvent(logTelemetryEvent)
		if err != nil {
			slog.Error("unexpected error sending log telemetry event for pod",
				"namespace", pod.Namespace,
				"pod", pod.Name,
				"error", err)
		}
	}(telemetryEvent, pod)

	return api.sendTelemetryEvent(telemetryEvent)
}

func (api CronitorApi) MakeAndSendTelemetryJobEventAndLogs(event *pkg.JobEvent, logs string, pod *corev1.Pod, job *v1.Job, cronjob *v1.CronJob) error {
	telemetryEvent, err := NewTelemetryEventFromKubernetesJobEvent(event, logs, pod, job, cronjob)
	if err != nil {
		return err
	}

	defer func(telemetryEvent *TelemetryEvent, job *v1.Job) {
		if !viper.GetBool("ship-logs") || len(telemetryEvent.ErrorLogs) == 0 {
			return
		}
		_, err := api.ShipLogData(telemetryEvent)
		if err != nil {
			if strings.Contains(err.Error(), "no such host") {
				// This error is due entirely to logs.cronitor.link not existing yet,
				// so discard for now
				return
			}
			slog.Error("unexpected error sending log data for job",
				"namespace", job.Namespace,
				"job", job.Name,
				"error", err)
		}
		logTelemetryEvent := telemetryEvent.CreateLogTelemetryEvent()
		err = api.sendTelemetryEvent(logTelemetryEvent)
		if err != nil {
			slog.Error("unexpected error sending log telemetry event for pod",
				"namespace", pod.Namespace,
				"pod", pod.Name,
				"error", err)
		}
	}(telemetryEvent, job)

	return api.sendTelemetryEvent(telemetryEvent)
}

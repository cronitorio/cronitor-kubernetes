package api

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/cronitorio/cronitor-kubernetes/pkg"
	"github.com/spf13/viper"
	v1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

func TestTelemetryUrl(t *testing.T) {
	// Verify the base URL format: /ping/{api_key}/{monitor_key}
	// State is now passed as a query param, not in the path
	cronjob := &v1.CronJob{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-cronjob",
			Namespace: "default",
			UID:       "test-uid-123",
		},
	}

	api := CronitorApi{
		ApiKey:    "test-api-key",
		UserAgent: "test-agent",
	}

	telemetryEvent := &TelemetryEvent{
		CronJob: cronjob,
		Event:   Run,
	}

	telemetryUrl := api.telemetryUrl(telemetryEvent)

	// URL should be: https://cronitor.link/ping/{api_key}/{monitor_key}
	expectedUrl := "https://cronitor.link/ping/test-api-key/test-uid-123"
	if telemetryUrl != expectedUrl {
		t.Errorf("expected URL '%s', got '%s'", expectedUrl, telemetryUrl)
	}

	// Verify state is NOT in the path (it should be a query param now)
	if strings.Contains(telemetryUrl, "/run") {
		t.Error("state should not be in URL path, should be query param")
	}
}

func TestTelemetryUrlWithCustomCronitorID(t *testing.T) {
	cronjob := &v1.CronJob{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-cronjob",
			Namespace: "default",
			UID:       "test-uid-123",
			Annotations: map[string]string{
				"k8s.cronitor.io/cronitor-id": "custom-monitor-id",
			},
		},
	}

	api := CronitorApi{
		ApiKey:    "test-api-key",
		UserAgent: "test-agent",
	}

	telemetryEvent := &TelemetryEvent{
		CronJob: cronjob,
		Event:   Run,
	}

	telemetryUrl := api.telemetryUrl(telemetryEvent)

	expectedUrl := "https://cronitor.link/ping/test-api-key/custom-monitor-id"
	if telemetryUrl != expectedUrl {
		t.Errorf("expected URL '%s', got '%s'", expectedUrl, telemetryUrl)
	}
}

// TestTelemetryEncodeAllParams verifies that ALL telemetry parameters are correctly encoded
func TestTelemetryEncodeAllParams(t *testing.T) {
	series := types.UID("job-uid-456")
	exitCode := 1

	event := TelemetryEvent{
		Event:     Complete,
		Message:   "Job completed successfully",
		Series:    &series,
		ExitCode:  &exitCode,
		Env:       "production",
		Host:      "worker-node-1",
		Timestamp: "1234567890",
		Metric:    "duration:5000",
	}

	encoded := event.Encode()
	params, err := url.ParseQuery(encoded)
	if err != nil {
		t.Fatalf("failed to parse encoded query: %v", err)
	}

	// Verify ALL parameters are present and correct
	expectedParams := map[string]string{
		"state":     "complete",
		"message":   "Job completed successfully",
		"series":    "job-uid-456",
		"exit_code": "1",
		"env":       "production",
		"host":      "worker-node-1",
		"stamp":     "1234567890",
		"metric":    "duration:5000",
	}

	for key, expectedValue := range expectedParams {
		actualValue := params.Get(key)
		if actualValue != expectedValue {
			t.Errorf("param '%s': expected '%s', got '%s'", key, expectedValue, actualValue)
		}
	}
}

// TestTelemetryEncodeStateParam verifies state is correctly encoded for each event type
func TestTelemetryEncodeStateParam(t *testing.T) {
	tests := []struct {
		event         TelemetryEventStatus
		expectedState string
	}{
		{Run, "run"},
		{Complete, "complete"},
		{Fail, "fail"},
		{Ok, "ok"},
		{Logs, "logs"},
	}

	for _, tc := range tests {
		t.Run(string(tc.event), func(t *testing.T) {
			event := TelemetryEvent{Event: tc.event}
			encoded := event.Encode()
			params, _ := url.ParseQuery(encoded)

			if params.Get("state") != tc.expectedState {
				t.Errorf("expected state '%s', got '%s'", tc.expectedState, params.Get("state"))
			}
		})
	}
}

// TestTelemetryEncodeOptionalParams verifies optional params are only included when set
func TestTelemetryEncodeOptionalParams(t *testing.T) {
	// Minimal event - only state should be present
	event := TelemetryEvent{
		Event: Run,
	}

	encoded := event.Encode()
	params, _ := url.ParseQuery(encoded)

	// State is always required
	if params.Get("state") != "run" {
		t.Errorf("state should always be present, got '%s'", params.Get("state"))
	}

	// These should NOT be present when not set
	optionalParams := []string{"message", "series", "exit_code", "env", "host", "stamp", "metric"}
	for _, param := range optionalParams {
		if params.Get(param) != "" {
			t.Errorf("param '%s' should not be present when not set, got '%s'", param, params.Get(param))
		}
	}
}

func TestTranslatePodEventReasonToTelemetryEventStatus(t *testing.T) {
	tests := []struct {
		name           string
		reason         string
		expectedStatus TelemetryEventStatus
		expectError    bool
	}{
		{
			name:           "Started maps to run",
			reason:         "Started",
			expectedStatus: Run,
		},
		{
			name:           "BackOff maps to fail",
			reason:         "BackOff",
			expectedStatus: Fail,
		},
		{
			name:        "Unknown reason returns error",
			reason:      "SomeUnknownReason",
			expectError: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			event := &pkg.PodEvent{}
			event.Reason = tc.reason

			status, err := TranslatePodEventReasonToTelemetryEventStatus(event)

			if tc.expectError {
				if err == nil {
					t.Error("expected error but got none")
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if *status != tc.expectedStatus {
				t.Errorf("expected status '%s', got '%s'", tc.expectedStatus, *status)
			}
		})
	}
}

func TestTranslateJobEventReasonToTelemetryEventStatus(t *testing.T) {
	tests := []struct {
		name           string
		reason         string
		expectedStatus TelemetryEventStatus
		expectError    bool
	}{
		{
			name:           "SuccessfulCreate maps to run",
			reason:         "SuccessfulCreate",
			expectedStatus: Run,
		},
		{
			name:           "Completed maps to complete",
			reason:         "Completed",
			expectedStatus: Complete,
		},
		{
			name:           "BackoffLimitExceeded maps to fail",
			reason:         "BackoffLimitExceeded",
			expectedStatus: Fail,
		},
		{
			name:        "Unknown reason returns error",
			reason:      "SomeUnknownReason",
			expectError: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			event := &pkg.JobEvent{}
			event.Reason = tc.reason

			status, err := translateJobEventReasonToTelemetryEventStatus(event)

			if tc.expectError {
				if err == nil {
					t.Error("expected error but got none")
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if *status != tc.expectedStatus {
				t.Errorf("expected status '%s', got '%s'", tc.expectedStatus, *status)
			}
		})
	}
}

func TestTelemetryEventIncludesEnvironmentFromAnnotation(t *testing.T) {
	cronjob := &v1.CronJob{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-cronjob",
			Namespace: "default",
			UID:       "test-uid-123",
			Annotations: map[string]string{
				"k8s.cronitor.io/env": "staging",
			},
		},
	}

	job := &v1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-job",
			Namespace: "default",
			UID:       "job-uid-456",
		},
	}

	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-pod",
			Namespace: "default",
		},
		Spec: corev1.PodSpec{
			NodeName: "node-1",
		},
	}

	jobEvent := &pkg.JobEvent{}
	jobEvent.Reason = "Completed"
	jobEvent.Message = "Job completed successfully"

	telemetryEvent, err := NewTelemetryEventFromKubernetesJobEvent(jobEvent, "", pod, job, cronjob)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if telemetryEvent.Env != "staging" {
		t.Errorf("expected env 'staging', got '%s'", telemetryEvent.Env)
	}

	// Verify env is in encoded params
	params, _ := url.ParseQuery(telemetryEvent.Encode())
	if params.Get("env") != "staging" {
		t.Errorf("expected encoded env param 'staging', got '%s'", params.Get("env"))
	}
}

func TestTelemetryEventWithoutEnvironmentAnnotation(t *testing.T) {
	cronjob := &v1.CronJob{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-cronjob",
			Namespace: "default",
			UID:       "test-uid-123",
		},
	}

	job := &v1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-job",
			Namespace: "default",
			UID:       "job-uid-456",
		},
	}

	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-pod",
			Namespace: "default",
		},
		Spec: corev1.PodSpec{
			NodeName: "node-1",
		},
	}

	jobEvent := &pkg.JobEvent{}
	jobEvent.Reason = "Completed"

	telemetryEvent, err := NewTelemetryEventFromKubernetesJobEvent(jobEvent, "", pod, job, cronjob)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if telemetryEvent.Env != "" {
		t.Errorf("expected empty env, got '%s'", telemetryEvent.Env)
	}

	// Verify env is NOT in encoded params
	params, _ := url.ParseQuery(telemetryEvent.Encode())
	if params.Get("env") != "" {
		t.Errorf("expected no env param, got '%s'", params.Get("env"))
	}
}

// TestSendTelemetryRequestVerifiesAllParams is a comprehensive test that verifies
// the full HTTP request is constructed correctly with ALL parameters
func TestSendTelemetryRequestVerifiesAllParams(t *testing.T) {
	var capturedRequest *http.Request

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedRequest = r
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	}))
	defer server.Close()

	viper.Set("hostname-override", server.URL)
	defer viper.Set("hostname-override", "")

	cronjob := &v1.CronJob{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-cronjob",
			Namespace: "default",
			UID:       "test-uid-123",
		},
	}

	series := types.UID("job-series-789")
	exitCode := 0
	telemetryEvent := &TelemetryEvent{
		CronJob:   cronjob,
		Event:     Complete,
		Message:   "Job completed successfully",
		Env:       "production",
		Host:      "worker-node-1",
		Timestamp: "1234567890",
		Series:    &series,
		ExitCode:  &exitCode,
		Metric:    "duration:5000",
	}

	api := CronitorApi{
		ApiKey:    "test-api-key",
		UserAgent: "cronitor-kubernetes/test",
	}

	_, err := api.sendTelemetryPostRequest(telemetryEvent)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify request method
	if capturedRequest.Method != "POST" {
		t.Errorf("expected POST method, got %s", capturedRequest.Method)
	}

	// Verify URL path format: /ping/{api_key}/{monitor_key}
	expectedPath := "/ping/test-api-key/test-uid-123"
	if capturedRequest.URL.Path != expectedPath {
		t.Errorf("expected path '%s', got '%s'", expectedPath, capturedRequest.URL.Path)
	}

	// Verify state is NOT in path (should be query param)
	if strings.Contains(capturedRequest.URL.Path, "complete") {
		t.Error("state should not be in URL path")
	}

	// Verify ALL query params
	queryParams := capturedRequest.URL.Query()

	expectedQueryParams := map[string]string{
		"state":     "complete",
		"message":   "Job completed successfully",
		"env":       "production",
		"host":      "worker-node-1",
		"stamp":     "1234567890",
		"series":    "job-series-789",
		"exit_code": "0",
		"metric":    "duration:5000",
	}

	for key, expectedValue := range expectedQueryParams {
		actualValue := queryParams.Get(key)
		if actualValue != expectedValue {
			t.Errorf("query param '%s': expected '%s', got '%s'", key, expectedValue, actualValue)
		}
	}

	// Verify User-Agent header
	if capturedRequest.Header.Get("User-Agent") != "cronitor-kubernetes/test" {
		t.Errorf("expected User-Agent 'cronitor-kubernetes/test', got '%s'", capturedRequest.Header.Get("User-Agent"))
	}
}

// TestSendTelemetryRequestMinimalParams verifies request with only required params
func TestSendTelemetryRequestMinimalParams(t *testing.T) {
	var capturedRequest *http.Request

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedRequest = r
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	viper.Set("hostname-override", server.URL)
	defer viper.Set("hostname-override", "")

	cronjob := &v1.CronJob{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-cronjob",
			Namespace: "default",
			UID:       "test-uid-123",
		},
	}

	// Minimal event - only state
	telemetryEvent := &TelemetryEvent{
		CronJob: cronjob,
		Event:   Run,
	}

	api := CronitorApi{
		ApiKey:    "test-api-key",
		UserAgent: "test-agent",
	}

	_, err := api.sendTelemetryPostRequest(telemetryEvent)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	queryParams := capturedRequest.URL.Query()

	// State is required
	if queryParams.Get("state") != "run" {
		t.Errorf("expected state 'run', got '%s'", queryParams.Get("state"))
	}

	// Optional params should not be present
	optionalParams := []string{"message", "series", "exit_code", "env", "host", "stamp", "metric"}
	for _, param := range optionalParams {
		if queryParams.Get(param) != "" {
			t.Errorf("param '%s' should not be present, got '%s'", param, queryParams.Get(param))
		}
	}
}

func TestTelemetryFailureReturnsError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(`{"error": "Invalid API key"}`))
	}))
	defer server.Close()

	viper.Set("hostname-override", server.URL)
	defer viper.Set("hostname-override", "")

	cronjob := &v1.CronJob{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-cronjob",
			Namespace: "default",
			UID:       "test-uid-123",
		},
	}

	telemetryEvent := &TelemetryEvent{
		CronJob: cronjob,
		Event:   Run,
	}

	api := CronitorApi{
		ApiKey:    "invalid-key",
		UserAgent: "test-agent",
	}

	_, err := api.sendTelemetryPostRequest(telemetryEvent)
	if err == nil {
		t.Error("expected error for 401 response, got nil")
	}

	apiErr, ok := err.(CronitorApiError)
	if !ok {
		t.Fatalf("expected CronitorApiError, got %T", err)
	}

	if apiErr.Response.StatusCode != http.StatusUnauthorized {
		t.Errorf("expected status code 401, got %d", apiErr.Response.StatusCode)
	}
}

// TestTelemetryDryRunSkipsRequest verifies DryRun mode doesn't send requests
func TestTelemetryDryRunSkipsRequest(t *testing.T) {
	requestMade := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestMade = true
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	viper.Set("hostname-override", server.URL)
	defer viper.Set("hostname-override", "")

	cronjob := &v1.CronJob{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-cronjob",
			Namespace: "default",
			UID:       "test-uid-123",
		},
	}

	telemetryEvent := &TelemetryEvent{
		CronJob: cronjob,
		Event:   Run,
	}

	api := CronitorApi{
		ApiKey:    "test-api-key",
		UserAgent: "test-agent",
		DryRun:    true,
	}

	err := api.sendTelemetryEvent(telemetryEvent)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if requestMade {
		t.Error("request should not be made in DryRun mode")
	}
}

// =============================================================================
// Integration tests for MakeAndSendTelemetry* functions
// These test the full path from Kubernetes events to HTTP requests
// =============================================================================

// TestMakeAndSendTelemetryJobEvent_CompletedJob verifies the full integration path
// when a job completes successfully
func TestMakeAndSendTelemetryJobEvent_CompletedJob(t *testing.T) {
	var capturedRequest *http.Request

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedRequest = r
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	}))
	defer server.Close()

	viper.Set("hostname-override", server.URL)
	defer viper.Set("hostname-override", "")

	cronjob := &v1.CronJob{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "my-scheduled-job",
			Namespace: "production",
			UID:       "cronjob-uid-abc123",
			Annotations: map[string]string{
				"k8s.cronitor.io/env": "production",
			},
		},
	}

	job := &v1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "my-scheduled-job-28571234",
			Namespace: "production",
			UID:       "job-uid-xyz789",
		},
	}

	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "my-scheduled-job-28571234-abc",
			Namespace: "production",
		},
		Spec: corev1.PodSpec{
			NodeName: "worker-node-3",
		},
	}

	jobEvent := &pkg.JobEvent{}
	jobEvent.Reason = "Completed"
	jobEvent.Message = "Job completed successfully"

	api := CronitorApi{
		ApiKey:    "my-api-key",
		UserAgent: "cronitor-kubernetes/test",
	}

	err := api.MakeAndSendTelemetryJobEventAndLogs(jobEvent, "", pod, job, cronjob)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify request was made
	if capturedRequest == nil {
		t.Fatal("expected request to be made")
	}

	// Verify URL path: /ping/{api_key}/{monitor_key}
	// monitor_key should be the cronjob UID since no cronitor-id annotation
	expectedPath := "/ping/my-api-key/cronjob-uid-abc123"
	if capturedRequest.URL.Path != expectedPath {
		t.Errorf("expected path '%s', got '%s'", expectedPath, capturedRequest.URL.Path)
	}

	// Verify query params
	params := capturedRequest.URL.Query()

	if params.Get("state") != "complete" {
		t.Errorf("expected state 'complete', got '%s'", params.Get("state"))
	}

	if params.Get("env") != "production" {
		t.Errorf("expected env 'production', got '%s'", params.Get("env"))
	}

	if params.Get("series") != "job-uid-xyz789" {
		t.Errorf("expected series 'job-uid-xyz789', got '%s'", params.Get("series"))
	}

	if params.Get("host") != "worker-node-3" {
		t.Errorf("expected host 'worker-node-3', got '%s'", params.Get("host"))
	}

	if params.Get("message") != "Job completed successfully" {
		t.Errorf("expected message 'Job completed successfully', got '%s'", params.Get("message"))
	}
}

// TestMakeAndSendTelemetryJobEvent_WithCustomCronitorID verifies the monitor key
// is taken from the cronitor-id annotation when present
func TestMakeAndSendTelemetryJobEvent_WithCustomCronitorID(t *testing.T) {
	var capturedPath string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedPath = r.URL.Path
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	viper.Set("hostname-override", server.URL)
	defer viper.Set("hostname-override", "")

	cronjob := &v1.CronJob{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "my-scheduled-job",
			Namespace: "production",
			UID:       "cronjob-uid-abc123",
			Annotations: map[string]string{
				"k8s.cronitor.io/cronitor-id": "my-custom-monitor-key",
			},
		},
	}

	job := &v1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "my-scheduled-job-28571234",
			Namespace: "production",
			UID:       "job-uid-xyz789",
		},
	}

	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "my-scheduled-job-28571234-abc",
			Namespace: "production",
		},
		Spec: corev1.PodSpec{
			NodeName: "worker-node-1",
		},
	}

	jobEvent := &pkg.JobEvent{}
	jobEvent.Reason = "Completed"

	api := CronitorApi{
		ApiKey:    "my-api-key",
		UserAgent: "cronitor-kubernetes/test",
	}

	err := api.MakeAndSendTelemetryJobEventAndLogs(jobEvent, "", pod, job, cronjob)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify the custom cronitor-id is used as the monitor key
	expectedPath := "/ping/my-api-key/my-custom-monitor-key"
	if capturedPath != expectedPath {
		t.Errorf("expected path '%s', got '%s'", expectedPath, capturedPath)
	}
}

// TestMakeAndSendTelemetryJobEvent_FailedJob verifies failure events are sent correctly
func TestMakeAndSendTelemetryJobEvent_FailedJob(t *testing.T) {
	var capturedState string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedState = r.URL.Query().Get("state")
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	viper.Set("hostname-override", server.URL)
	defer viper.Set("hostname-override", "")

	cronjob := &v1.CronJob{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "failing-job",
			Namespace: "default",
			UID:       "cronjob-uid-fail",
		},
	}

	job := &v1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "failing-job-123",
			Namespace: "default",
			UID:       "job-uid-fail",
		},
	}

	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "failing-job-123-pod",
			Namespace: "default",
		},
		Spec: corev1.PodSpec{
			NodeName: "node-1",
		},
	}

	jobEvent := &pkg.JobEvent{}
	jobEvent.Reason = "BackoffLimitExceeded"
	jobEvent.Message = "Job has reached the specified backoff limit"

	api := CronitorApi{
		ApiKey:    "test-key",
		UserAgent: "test-agent",
	}

	err := api.MakeAndSendTelemetryJobEventAndLogs(jobEvent, "", pod, job, cronjob)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if capturedState != "fail" {
		t.Errorf("expected state 'fail', got '%s'", capturedState)
	}
}

// TestMakeAndSendTelemetryPodEvent_Started verifies pod start events are sent correctly
func TestMakeAndSendTelemetryPodEvent_Started(t *testing.T) {
	var capturedRequest *http.Request

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedRequest = r
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	viper.Set("hostname-override", server.URL)
	defer viper.Set("hostname-override", "")

	cronjob := &v1.CronJob{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "scheduled-task",
			Namespace: "apps",
			UID:       "cronjob-uid-pod-test",
			Annotations: map[string]string{
				"k8s.cronitor.io/cronitor-id": "my-task-monitor",
				"k8s.cronitor.io/env":         "staging",
			},
		},
	}

	job := &v1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "scheduled-task-12345",
			Namespace: "apps",
			UID:       "job-uid-pod-test",
		},
	}

	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "scheduled-task-12345-xyz",
			Namespace: "apps",
		},
		Spec: corev1.PodSpec{
			NodeName: "node-2",
		},
	}

	podEvent := &pkg.PodEvent{}
	podEvent.Reason = "Started"
	podEvent.Message = "Started container"

	api := CronitorApi{
		ApiKey:    "pod-test-key",
		UserAgent: "cronitor-kubernetes/test",
	}

	err := api.MakeAndSendTelemetryPodEventAndLogs(podEvent, "", pod, job, cronjob)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify request was made
	if capturedRequest == nil {
		t.Fatal("expected request to be made")
	}

	// Verify URL uses custom cronitor-id
	expectedPath := "/ping/pod-test-key/my-task-monitor"
	if capturedRequest.URL.Path != expectedPath {
		t.Errorf("expected path '%s', got '%s'", expectedPath, capturedRequest.URL.Path)
	}

	params := capturedRequest.URL.Query()

	if params.Get("state") != "run" {
		t.Errorf("expected state 'run', got '%s'", params.Get("state"))
	}

	if params.Get("env") != "staging" {
		t.Errorf("expected env 'staging', got '%s'", params.Get("env"))
	}

	if params.Get("series") != "job-uid-pod-test" {
		t.Errorf("expected series 'job-uid-pod-test', got '%s'", params.Get("series"))
	}
}

// TestMakeAndSendTelemetryPodEvent_BackOff verifies pod backoff events are sent as failures
func TestMakeAndSendTelemetryPodEvent_BackOff(t *testing.T) {
	var capturedState string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedState = r.URL.Query().Get("state")
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	viper.Set("hostname-override", server.URL)
	defer viper.Set("hostname-override", "")

	cronjob := &v1.CronJob{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "flaky-job",
			Namespace: "default",
			UID:       "cronjob-uid-backoff",
		},
	}

	job := &v1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "flaky-job-999",
			Namespace: "default",
			UID:       "job-uid-backoff",
		},
	}

	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "flaky-job-999-pod",
			Namespace: "default",
		},
		Spec: corev1.PodSpec{
			NodeName: "node-1",
		},
	}

	podEvent := &pkg.PodEvent{}
	podEvent.Reason = "BackOff"
	podEvent.Message = "Back-off restarting failed container"

	api := CronitorApi{
		ApiKey:    "test-key",
		UserAgent: "test-agent",
	}

	err := api.MakeAndSendTelemetryPodEventAndLogs(podEvent, "", pod, job, cronjob)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if capturedState != "fail" {
		t.Errorf("expected state 'fail', got '%s'", capturedState)
	}
}

// TestTelemetryURLVerification is a comprehensive test that explicitly documents
// and verifies the expected Cronitor API endpoints
func TestTelemetryURLVerification(t *testing.T) {
	t.Run("default telemetry URL is cronitor.link", func(t *testing.T) {
		// Clear any override
		viper.Set("hostname-override", "")

		cronjob := &v1.CronJob{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-job",
				Namespace: "default",
				UID:       "test-uid",
			},
		}

		api := CronitorApi{
			ApiKey: "my-api-key",
		}

		telemetryEvent := &TelemetryEvent{
			CronJob: cronjob,
			Event:   Run,
		}

		url := api.telemetryUrl(telemetryEvent)

		// CRITICAL: This verifies we're hitting the right production endpoint
		expectedURL := "https://cronitor.link/ping/my-api-key/test-uid"
		if url != expectedURL {
			t.Errorf("CRITICAL: Telemetry URL mismatch!\n  Expected: %s\n  Got: %s", expectedURL, url)
		}
	})

	t.Run("hostname-override redirects telemetry", func(t *testing.T) {
		viper.Set("hostname-override", "http://mock-server.local")
		defer viper.Set("hostname-override", "")

		cronjob := &v1.CronJob{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-job",
				Namespace: "default",
				UID:       "test-uid",
			},
		}

		api := CronitorApi{
			ApiKey: "my-api-key",
		}

		telemetryEvent := &TelemetryEvent{
			CronJob: cronjob,
			Event:   Run,
		}

		url := api.telemetryUrl(telemetryEvent)

		expectedURL := "http://mock-server.local/ping/my-api-key/test-uid"
		if url != expectedURL {
			t.Errorf("hostname-override not working!\n  Expected: %s\n  Got: %s", expectedURL, url)
		}
	})
}

// =============================================================================
// Async log shipping tests
// =============================================================================

// TestMakeAndSendTelemetryJobEventAndLogs_AsyncLogShipping verifies that
// MakeAndSendTelemetryJobEventAndLogs returns quickly even when log shipping
// would be slow, because log shipping now runs asynchronously in a goroutine.
func TestMakeAndSendTelemetryJobEventAndLogs_AsyncLogShipping(t *testing.T) {
	// goroutineDone signals when the async goroutine has finished all its work.
	// The goroutine flow: presign (slow) → error (empty URL) → log telemetry ping.
	// We track the last request (log telemetry ping) to know when it's safe to clean up.
	goroutineDone := make(chan struct{}, 1)
	var presignHit int32

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "/logs/presign") {
			// Simulate slow presign endpoint
			time.Sleep(200 * time.Millisecond)
			atomic.StoreInt32(&presignHit, 1)
			w.WriteHeader(http.StatusOK)
			// Return empty URL — ShipLogData will error, goroutine falls through
			// to sendTelemetryEvent for the log telemetry event
			w.Write([]byte(`{"url": ""}`))
			return
		}
		if strings.Contains(r.URL.Path, "/ping/") {
			// Could be the initial telemetry ping OR the log telemetry ping from the goroutine.
			// The goroutine's log telemetry ping arrives after presign, so signal done.
			if atomic.LoadInt32(&presignHit) == 1 {
				select {
				case goroutineDone <- struct{}{}:
				default:
				}
			}
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	}))
	defer server.Close()

	viper.Set("hostname-override", server.URL)
	viper.Set("ship-logs", true)

	cronjob := &v1.CronJob{
		ObjectMeta: metav1.ObjectMeta{
			Name: "async-test-job", Namespace: "default", UID: "async-uid",
		},
	}
	job := &v1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name: "async-test-job-123", Namespace: "default", UID: "async-job-uid",
		},
	}
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{Name: "async-pod", Namespace: "default"},
		Spec:       corev1.PodSpec{NodeName: "node-1"},
	}

	jobEvent := &pkg.JobEvent{}
	jobEvent.Reason = "Completed"
	jobEvent.Message = "Job completed"

	cronitorApi := CronitorApi{ApiKey: "test-key", UserAgent: "test-agent"}

	start := time.Now()
	err := cronitorApi.MakeAndSendTelemetryJobEventAndLogs(jobEvent, "some error logs here", pod, job, cronjob)
	elapsed := time.Since(start)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// The function should return quickly (well under 200ms) because log shipping is async
	if elapsed > 100*time.Millisecond {
		t.Errorf("MakeAndSendTelemetryJobEventAndLogs took %v, expected <100ms (async log shipping)", elapsed)
	}

	// Wait for the async goroutine to fully complete before cleaning up viper
	select {
	case <-goroutineDone:
		// good — goroutine finished all work
	case <-time.After(3 * time.Second):
		t.Error("async log shipping goroutine did not complete within timeout")
	}

	// Clean up viper AFTER goroutine is done to avoid races
	viper.Set("hostname-override", "")
	viper.Set("ship-logs", false)
}

// TestMakeAndSendTelemetryPodEventAndLogs_AsyncLogShipping verifies async behavior for pod events.
func TestMakeAndSendTelemetryPodEventAndLogs_AsyncLogShipping(t *testing.T) {
	goroutineDone := make(chan struct{}, 1)
	var presignHit int32

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "/logs/presign") {
			time.Sleep(200 * time.Millisecond)
			atomic.StoreInt32(&presignHit, 1)
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"url": ""}`))
			return
		}
		if strings.Contains(r.URL.Path, "/ping/") {
			if atomic.LoadInt32(&presignHit) == 1 {
				select {
				case goroutineDone <- struct{}{}:
				default:
				}
			}
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	}))
	defer server.Close()

	viper.Set("hostname-override", server.URL)
	viper.Set("ship-logs", true)

	cronjob := &v1.CronJob{
		ObjectMeta: metav1.ObjectMeta{
			Name: "async-pod-test", Namespace: "default", UID: "async-pod-uid",
		},
	}
	job := &v1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name: "async-pod-job", Namespace: "default", UID: "async-pod-job-uid",
		},
	}
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{Name: "async-pod-pod", Namespace: "default"},
		Spec:       corev1.PodSpec{NodeName: "node-1"},
	}

	podEvent := &pkg.PodEvent{}
	podEvent.Reason = "Started"
	podEvent.Message = "Container started"

	cronitorApi := CronitorApi{ApiKey: "test-key", UserAgent: "test-agent"}

	start := time.Now()
	err := cronitorApi.MakeAndSendTelemetryPodEventAndLogs(podEvent, "some pod logs here", pod, job, cronjob)
	elapsed := time.Since(start)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if elapsed > 100*time.Millisecond {
		t.Errorf("MakeAndSendTelemetryPodEventAndLogs took %v, expected <100ms (async log shipping)", elapsed)
	}

	select {
	case <-goroutineDone:
	case <-time.After(3 * time.Second):
		t.Error("async log shipping goroutine did not complete within timeout")
	}

	viper.Set("hostname-override", "")
	viper.Set("ship-logs", false)
}

// TestMakeAndSendTelemetry_NoLogs_SkipsLogShipping verifies that when there are no logs,
// only the telemetry ping is sent and no log shipping request is made.
func TestMakeAndSendTelemetry_NoLogs_SkipsLogShipping(t *testing.T) {
	var pingCount int32
	var presignCount int32

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "/logs/presign") {
			atomic.AddInt32(&presignCount, 1)
		} else if strings.Contains(r.URL.Path, "/ping/") {
			atomic.AddInt32(&pingCount, 1)
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	}))
	defer server.Close()

	viper.Set("hostname-override", server.URL)
	viper.Set("ship-logs", true)
	defer func() {
		viper.Set("hostname-override", "")
		viper.Set("ship-logs", false)
	}()

	cronjob := &v1.CronJob{
		ObjectMeta: metav1.ObjectMeta{
			Name: "no-logs-job", Namespace: "default", UID: "no-logs-uid",
		},
	}
	job := &v1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name: "no-logs-job-123", Namespace: "default", UID: "no-logs-job-uid",
		},
	}
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{Name: "no-logs-pod", Namespace: "default"},
		Spec:       corev1.PodSpec{NodeName: "node-1"},
	}

	jobEvent := &pkg.JobEvent{}
	jobEvent.Reason = "Completed"

	cronitorApi := CronitorApi{ApiKey: "test-key", UserAgent: "test-agent"}

	// Empty logs — should NOT trigger log shipping
	err := cronitorApi.MakeAndSendTelemetryJobEventAndLogs(jobEvent, "", pod, job, cronjob)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Give the goroutine time to run (if it would)
	time.Sleep(100 * time.Millisecond)

	// Only 1 ping request should be made (the telemetry ping), no presign requests
	pings := atomic.LoadInt32(&pingCount)
	presigns := atomic.LoadInt32(&presignCount)
	if pings != 1 {
		t.Errorf("expected 1 telemetry ping request, got %d", pings)
	}
	if presigns != 0 {
		t.Errorf("expected 0 log presign requests, got %d", presigns)
	}
}

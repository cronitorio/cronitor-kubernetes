package api

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

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

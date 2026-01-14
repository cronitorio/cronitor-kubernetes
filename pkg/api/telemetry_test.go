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
	// Create a test cronjob
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

	tests := []struct {
		name          string
		event         TelemetryEventStatus
		expectedPath  string
	}{
		{
			name:         "run event",
			event:        Run,
			expectedPath: "/ping/test-api-key/test-uid-123/run",
		},
		{
			name:         "complete event",
			event:        Complete,
			expectedPath: "/ping/test-api-key/test-uid-123/complete",
		},
		{
			name:         "fail event",
			event:        Fail,
			expectedPath: "/ping/test-api-key/test-uid-123/fail",
		},
		{
			name:         "ok event",
			event:        Ok,
			expectedPath: "/ping/test-api-key/test-uid-123/ok",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			telemetryEvent := &TelemetryEvent{
				CronJob: cronjob,
				Event:   tc.event,
			}

			url := api.telemetryUrl(telemetryEvent)

			if !strings.HasSuffix(url, tc.expectedPath) {
				t.Errorf("expected URL to end with '%s', got '%s'", tc.expectedPath, url)
			}

			if !strings.HasPrefix(url, "https://cronitor.link") {
				t.Errorf("expected URL to start with 'https://cronitor.link', got '%s'", url)
			}
		})
	}
}

func TestTelemetryUrlWithCustomCronitorID(t *testing.T) {
	// Create a cronjob with custom cronitor-id annotation
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

	url := api.telemetryUrl(telemetryEvent)

	expectedPath := "/ping/test-api-key/custom-monitor-id/run"
	if !strings.HasSuffix(url, expectedPath) {
		t.Errorf("expected URL to end with '%s', got '%s'", expectedPath, url)
	}
}

func TestTelemetryEventEncode(t *testing.T) {
	series := types.UID("job-uid-456")
	exitCode := 1

	tests := []struct {
		name           string
		event          TelemetryEvent
		expectedParams map[string]string
	}{
		{
			name: "basic event with env",
			event: TelemetryEvent{
				Env:       "production",
				Message:   "Job started",
				Host:      "node-1",
				Timestamp: "1234567890",
				Series:    &series,
			},
			expectedParams: map[string]string{
				"env":     "production",
				"message": "Job started",
				"host":    "node-1",
				"stamp":   "1234567890",
				"series":  "job-uid-456",
			},
		},
		{
			name: "event without env (default)",
			event: TelemetryEvent{
				Message:   "Job completed",
				Host:      "node-2",
				Timestamp: "1234567891",
			},
			expectedParams: map[string]string{
				"message": "Job completed",
				"host":    "node-2",
				"stamp":   "1234567891",
			},
		},
		{
			name: "event with exit code",
			event: TelemetryEvent{
				Message:   "Job failed",
				ExitCode:  &exitCode,
				Timestamp: "1234567892",
			},
			expectedParams: map[string]string{
				"message":   "Job failed",
				"exit_code": "1",
				"stamp":     "1234567892",
			},
		},
		{
			name: "event with metric",
			event: TelemetryEvent{
				Message:   "Logs",
				Metric:    "length:5000",
				Timestamp: "1234567893",
			},
			expectedParams: map[string]string{
				"message": "Logs",
				"metric":  "length:5000",
				"stamp":   "1234567893",
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			encoded := tc.event.Encode()
			params, err := url.ParseQuery(encoded)
			if err != nil {
				t.Fatalf("failed to parse encoded query: %v", err)
			}

			for key, expectedValue := range tc.expectedParams {
				if params.Get(key) != expectedValue {
					t.Errorf("expected param '%s' to be '%s', got '%s'", key, expectedValue, params.Get(key))
				}
			}

			// Verify env is NOT present when not set
			if tc.event.Env == "" && params.Get("env") != "" {
				t.Errorf("expected 'env' param to be absent, got '%s'", params.Get("env"))
			}
		})
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
			expectError:    false,
		},
		{
			name:           "BackOff maps to fail",
			reason:         "BackOff",
			expectedStatus: Fail,
			expectError:    false,
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
			expectError:    false,
		},
		{
			name:           "Completed maps to complete",
			reason:         "Completed",
			expectedStatus: Complete,
			expectError:    false,
		},
		{
			name:           "BackoffLimitExceeded maps to fail",
			reason:         "BackoffLimitExceeded",
			expectedStatus: Fail,
			expectError:    false,
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
	// This test verifies that the env annotation on a CronJob
	// is correctly included in the telemetry event
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

	// Test with job event
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

	// Verify the env is encoded in query params
	encoded := telemetryEvent.Encode()
	params, _ := url.ParseQuery(encoded)
	if params.Get("env") != "staging" {
		t.Errorf("expected encoded env param 'staging', got '%s'", params.Get("env"))
	}
}

func TestTelemetryEventWithoutEnvironmentAnnotation(t *testing.T) {
	// Verify that when no env annotation is present, the env field is empty
	cronjob := &v1.CronJob{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-cronjob",
			Namespace: "default",
			UID:       "test-uid-123",
			// No env annotation
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

	// Verify env is NOT in query params
	encoded := telemetryEvent.Encode()
	params, _ := url.ParseQuery(encoded)
	if params.Get("env") != "" {
		t.Errorf("expected no env param, got '%s'", params.Get("env"))
	}
}

func TestSendTelemetryPostRequest(t *testing.T) {
	// Test that the telemetry request is sent with correct URL and params
	var capturedRequest *http.Request

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedRequest = r
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	}))
	defer server.Close()

	// Override hostname for test
	viper.Set("hostname-override", server.URL)
	defer viper.Set("hostname-override", "")

	cronjob := &v1.CronJob{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-cronjob",
			Namespace: "default",
			UID:       "test-uid-123",
			Annotations: map[string]string{
				"k8s.cronitor.io/env": "test-environment",
			},
		},
	}

	series := types.UID("job-series-789")
	telemetryEvent := &TelemetryEvent{
		CronJob:   cronjob,
		Event:     Complete,
		Message:   "Job completed",
		Env:       "test-environment",
		Host:      "test-node",
		Timestamp: "1234567890",
		Series:    &series,
	}

	api := CronitorApi{
		ApiKey:    "test-api-key",
		UserAgent: "test-agent",
	}

	_, err := api.sendTelemetryPostRequest(telemetryEvent)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify request method
	if capturedRequest.Method != "POST" {
		t.Errorf("expected POST method, got %s", capturedRequest.Method)
	}

	// Verify URL path contains correct components
	expectedPathParts := []string{"/ping", "test-api-key", "test-uid-123", "complete"}
	for _, part := range expectedPathParts {
		if !strings.Contains(capturedRequest.URL.Path, part) {
			t.Errorf("expected URL path to contain '%s', got '%s'", part, capturedRequest.URL.Path)
		}
	}

	// Verify query params
	queryParams := capturedRequest.URL.Query()
	if queryParams.Get("env") != "test-environment" {
		t.Errorf("expected env param 'test-environment', got '%s'", queryParams.Get("env"))
	}
	if queryParams.Get("host") != "test-node" {
		t.Errorf("expected host param 'test-node', got '%s'", queryParams.Get("host"))
	}
	if queryParams.Get("series") != "job-series-789" {
		t.Errorf("expected series param 'job-series-789', got '%s'", queryParams.Get("series"))
	}

	// Verify User-Agent header
	if capturedRequest.Header.Get("User-Agent") != "test-agent" {
		t.Errorf("expected User-Agent 'test-agent', got '%s'", capturedRequest.Header.Get("User-Agent"))
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

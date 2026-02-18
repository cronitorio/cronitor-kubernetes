package pkg

import (
	"encoding/json"
	"os"
	"strconv"
	"testing"

	v1 "k8s.io/api/batch/v1"
)

func TestCronJobInclusion(t *testing.T) {
	var jsonBlob v1.CronJobList
	os.Setenv("DEFAULT_BEHAVIOR", "include")
	err := json.Unmarshal([]byte(`{"metadata":{"selfLink":"/apis/batch/v1beta1/cronjobs","resourceVersion":"41530"},"items":[{"metadata":{"name":"eventrouter-test-croonjob","namespace":"cronitor","selfLink":"/apis/batch/v1beta1/namespaces/cronitor/cronjobs/eventrouter-test-croonjob","uid":"a4892036-090f-4019-8bd1-98bfe0a9034c","resourceVersion":"41467","creationTimestamp":"2020-11-13T06:06:44Z","labels":{"app.kubernetes.io/managed-by":"skaffold","skaffold.dev/run-id":"a592b4e3-dd8e-4b25-a69f-7abe35e264f0"},"managedFields":[{"manager":"Go-http-client","operation":"Update","ApiVersion":"batch/v1beta1","time":"2020-11-13T06:06:44Z","fieldsType":"FieldsV1","fieldsV1":{"f:spec":{"f:concurrencyPolicy":{},"f:failedJobsHistoryLimit":{},"f:jobTemplate":{"f:spec":{"f:template":{"f:spec":{"f:containers":{"k:{\"name\":\"hello\"}":{".":{},"f:args":{},"f:image":{},"f:imagePullPolicy":{},"f:name":{},"f:resources":{},"f:terminationMessagePath":{},"f:terminationMessagePolicy":{}}},"f:dnsPolicy":{},"f:restartPolicy":{},"f:schedulerName":{},"f:securityContext":{},"f:terminationGracePeriodSeconds":{}}}}},"f:schedule":{},"f:successfulJobsHistoryLimit":{},"f:suspend":{}}}},{"manager":"skaffold","operation":"Update","ApiVersion":"batch/v1beta1","time":"2020-11-13T06:06:45Z","fieldsType":"FieldsV1","fieldsV1":{"f:metadata":{"f:labels":{".":{},"f:app.kubernetes.io/managed-by":{},"f:skaffold.dev/run-id":{}}}}},{"manager":"kube-controller-manager","operation":"Update","ApiVersion":"batch/v1beta1","time":"2020-11-13T07:57:06Z","fieldsType":"FieldsV1","fieldsV1":{"f:status":{"f:active":{},"f:lastScheduleTime":{}}}}]},"spec":{"schedule":"*/1 * * * *","concurrencyPolicy":"Forbid","suspend":false,"jobTemplate":{"metadata":{"creationTimestamp":null},"spec":{"template":{"metadata":{"creationTimestamp":null},"spec":{"containers":[{"name":"hello","image":"busybox","args":["/bin/sh","-c","date ; echo Hello from k8s"],"resources":{},"terminationMessagePath":"/dev/termination-log","terminationMessagePolicy":"File","imagePullPolicy":"Always"}],"restartPolicy":"OnFailure","terminationGracePeriodSeconds":30,"dnsPolicy":"ClusterFirst","securityContext":{},"schedulerName":"default-scheduler"}}}},"successfulJobsHistoryLimit":3,"failedJobsHistoryLimit":1},"status":{"active":[{"kind":"Job","namespace":"cronitor","name":"eventrouter-test-croonjob-1605254220","uid":"697df5f5-6366-42fe-a20e-19ec2fefd826","ApiVersion":"batch/v1","resourceVersion":"41465"}],"lastScheduleTime":"2020-11-13T07:57:00Z"}}]}`), &jsonBlob)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	got := len(jsonBlob.Items)
	if got != 1 {
		t.Errorf("len(CronJobs) = %d; want 1", got)
	}

	parser := NewCronitorConfigParser(&jsonBlob.Items[0])
	if got, _ := parser.IsCronJobIncluded(); !got {
		t.Errorf("cronjob.IsCronJobIncluded() = %s; wanted true", strconv.FormatBool(got))
	}
}

func TestGetSchedule(t *testing.T) {
	var jsonBlob v1.CronJobList
	err := json.Unmarshal([]byte(`{"metadata":{"selfLink":"/apis/batch/v1beta1/cronjobs","resourceVersion":"41530"},"items":[{"metadata":{"name":"eventrouter-test-croonjob","namespace":"cronitor","selfLink":"/apis/batch/v1beta1/namespaces/cronitor/cronjobs/eventrouter-test-croonjob","uid":"a4892036-090f-4019-8bd1-98bfe0a9034c","resourceVersion":"41467","creationTimestamp":"2020-11-13T06:06:44Z","annotations":{"k8s.cronitor.io/env": "test-env"},"labels":{"app.kubernetes.io/managed-by":"skaffold","skaffold.dev/run-id":"a592b4e3-dd8e-4b25-a69f-7abe35e264f0"},"managedFields":[{"manager":"Go-http-client","operation":"Update","ApiVersion":"batch/v1beta1","time":"2020-11-13T06:06:44Z","fieldsType":"FieldsV1","fieldsV1":{"f:spec":{"f:concurrencyPolicy":{},"f:failedJobsHistoryLimit":{},"f:jobTemplate":{"f:spec":{"f:template":{"f:spec":{"f:containers":{"k:{\"name\":\"hello\"}":{".":{},"f:args":{},"f:image":{},"f:imagePullPolicy":{},"f:name":{},"f:resources":{},"f:terminationMessagePath":{},"f:terminationMessagePolicy":{}}},"f:dnsPolicy":{},"f:restartPolicy":{},"f:schedulerName":{},"f:securityContext":{},"f:terminationGracePeriodSeconds":{}}}}},"f:schedule":{},"f:successfulJobsHistoryLimit":{},"f:suspend":{}}}},{"manager":"skaffold","operation":"Update","ApiVersion":"batch/v1beta1","time":"2020-11-13T06:06:45Z","fieldsType":"FieldsV1","fieldsV1":{"f:metadata":{"f:labels":{".":{},"f:app.kubernetes.io/managed-by":{},"f:skaffold.dev/run-id":{}}}}},{"manager":"kube-controller-manager","operation":"Update","ApiVersion":"batch/v1beta1","time":"2020-11-13T07:57:06Z","fieldsType":"FieldsV1","fieldsV1":{"f:status":{"f:active":{},"f:lastScheduleTime":{}}}}]},"spec":{"schedule":"*/1 * * * *","concurrencyPolicy":"Forbid","suspend":false,"jobTemplate":{"metadata":{"creationTimestamp":null},"spec":{"template":{"metadata":{"creationTimestamp":null},"spec":{"containers":[{"name":"hello","image":"busybox","args":["/bin/sh","-c","date ; echo Hello from k8s"],"resources":{},"terminationMessagePath":"/dev/termination-log","terminationMessagePolicy":"File","imagePullPolicy":"Always"}],"restartPolicy":"OnFailure","terminationGracePeriodSeconds":30,"dnsPolicy":"ClusterFirst","securityContext":{},"schedulerName":"default-scheduler"}}}},"successfulJobsHistoryLimit":3,"failedJobsHistoryLimit":1},"status":{"active":[{"kind":"Job","namespace":"cronitor","name":"eventrouter-test-croonjob-1605254220","uid":"697df5f5-6366-42fe-a20e-19ec2fefd826","ApiVersion":"batch/v1","resourceVersion":"41465"}],"lastScheduleTime":"2020-11-13T07:57:00Z"}}]}`), &jsonBlob)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	got := len(jsonBlob.Items)
	if got != 1 {
		t.Errorf("len(CronJobs) = %d; want 1", got)
	}

	parser := NewCronitorConfigParser(&jsonBlob.Items[0])
	if result := parser.GetSchedule(); result != "*/1 * * * *" {
		t.Errorf("expected schedule \"*/1 * * * *\", got %s", result)
	}
}

func TestGetCronitorID(t *testing.T) {
	tests := []struct {
		name                   string
		annotationKeyInference string
		annotationCronitorID   string
		expectedID             string
	}{
		{
			name:                   "hashed name as ID",
			annotationKeyInference: "name",
			annotationCronitorID:   "",
			expectedID:             "3278d16696a89a92d297b7c46bfd286b20dc3896",
		},
		{
			name:                   "specific cronitor id",
			annotationKeyInference: "",
			annotationCronitorID:   "1234",
			expectedID:             "1234",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			annotations := []Annotation{
				{Key: "k8s.cronitor.io/key-inference", Value: tc.annotationKeyInference},
				{Key: "k8s.cronitor.io/cronitor-id", Value: tc.annotationCronitorID},
			}

			cronJob, err := CronJobFromAnnotations(annotations)
			if err != nil {
				t.Fatalf("unexpected error unmarshalling json: %v", err)
			}

			parser := NewCronitorConfigParser(&cronJob)
			if id := parser.GetCronitorID(); id != tc.expectedID {
				t.Errorf("expected ID %s, got %s", tc.expectedID, id)
			}
		})
	}
}

func TestGetCronitorName(t *testing.T) {
	tests := []struct {
		name                   string
		annotationNamePrefix   string
		annotationCronitorName string
		expectedName           string
	}{
		{
			name:                   "default behavior",
			annotationNamePrefix:   "",
			annotationCronitorName: "",
			expectedName:           "default/test-cronjob",
		},
		{
			name:                   "no prefix for name",
			annotationNamePrefix:   "none",
			annotationCronitorName: "",
			expectedName:           "test-cronjob",
		},
		{
			name:                   "explicit prefix of namespace",
			annotationNamePrefix:   "namespace",
			annotationCronitorName: "",
			expectedName:           "default/test-cronjob",
		},
		{
			name:                   "specific cronitor name",
			annotationNamePrefix:   "",
			annotationCronitorName: "foo",
			expectedName:           "foo",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			annotations := []Annotation{
				{Key: "k8s.cronitor.io/name-prefix", Value: tc.annotationNamePrefix},
				{Key: "k8s.cronitor.io/cronitor-name", Value: tc.annotationCronitorName},
			}

			cronJob, err := CronJobFromAnnotations(annotations)
			if err != nil {
				t.Fatalf("unexpected error unmarshalling json: %v", err)
			}

			parser := NewCronitorConfigParser(&cronJob)
			if name := parser.GetCronitorName(); name != tc.expectedName {
				t.Errorf("expected Name %s, got %s", tc.expectedName, name)
			}
		})
	}
}

func TestLogCompleteEvent(t *testing.T) {
	tests := []struct {
		name          string
		annotation    string
		expectedValue bool
		expectError   bool
	}{
		{
			name:          "Valid true annotation",
			annotation:    "true",
			expectedValue: true,
			expectError:   false,
		},
		{
			name:          "Valid false annotation",
			annotation:    "false",
			expectedValue: false,
			expectError:   false,
		},
		{
			name:          "Invalid annotation",
			annotation:    "invalid",
			expectedValue: false,
			expectError:   true,
		},
		{
			name:          "No annotation",
			annotation:    "",
			expectedValue: false,
			expectError:   false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var annotations []Annotation
			if tc.annotation != "" {
				annotations = []Annotation{
					{Key: "k8s.cronitor.io/log-complete-event", Value: tc.annotation},
				}
			}

			cronJob, err := CronJobFromAnnotations(annotations)
			if err != nil {
				t.Fatalf("failed to create CronJob from annotations: %v", err)
			}

			parser := NewCronitorConfigParser(&cronJob)
			got, err := parser.LogCompleteEvent()

			if tc.expectError {
				if err == nil {
					t.Error("expected error but got none")
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if got != tc.expectedValue {
				t.Errorf("LogCompleteEvent() = %v, want %v", got, tc.expectedValue)
			}
		})
	}
}

// TestAnnotationBackwardsCompatibility verifies that both the new (preferred) annotation names
// and the legacy (cronitor- prefixed) annotation names work correctly, and that the new
// annotation takes precedence when both are present.
func TestAnnotationBackwardsCompatibility(t *testing.T) {
	t.Run("Key annotation", func(t *testing.T) {
		tests := []struct {
			name        string
			annotations []Annotation
			expectedID  string
		}{
			{
				name: "new annotation (k8s.cronitor.io/key)",
				annotations: []Annotation{
					{Key: "k8s.cronitor.io/key", Value: "new-key"},
				},
				expectedID: "new-key",
			},
			{
				name: "legacy annotation (k8s.cronitor.io/cronitor-id)",
				annotations: []Annotation{
					{Key: "k8s.cronitor.io/cronitor-id", Value: "legacy-id"},
				},
				expectedID: "legacy-id",
			},
			{
				name: "new annotation takes precedence",
				annotations: []Annotation{
					{Key: "k8s.cronitor.io/key", Value: "new-key"},
					{Key: "k8s.cronitor.io/cronitor-id", Value: "legacy-id"},
				},
				expectedID: "new-key",
			},
		}

		for _, tc := range tests {
			t.Run(tc.name, func(t *testing.T) {
				cronJob, err := CronJobFromAnnotations(tc.annotations)
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				parser := NewCronitorConfigParser(&cronJob)
				if id := parser.GetSpecifiedCronitorID(); id != tc.expectedID {
					t.Errorf("expected ID %s, got %s", tc.expectedID, id)
				}
			})
		}
	})

	t.Run("Name annotation", func(t *testing.T) {
		tests := []struct {
			name         string
			annotations  []Annotation
			expectedName string
		}{
			{
				name: "new annotation (k8s.cronitor.io/name)",
				annotations: []Annotation{
					{Key: "k8s.cronitor.io/name", Value: "new-name"},
				},
				expectedName: "new-name",
			},
			{
				name: "legacy annotation (k8s.cronitor.io/cronitor-name)",
				annotations: []Annotation{
					{Key: "k8s.cronitor.io/cronitor-name", Value: "legacy-name"},
				},
				expectedName: "legacy-name",
			},
			{
				name: "new annotation takes precedence",
				annotations: []Annotation{
					{Key: "k8s.cronitor.io/name", Value: "new-name"},
					{Key: "k8s.cronitor.io/cronitor-name", Value: "legacy-name"},
				},
				expectedName: "new-name",
			},
		}

		for _, tc := range tests {
			t.Run(tc.name, func(t *testing.T) {
				cronJob, err := CronJobFromAnnotations(tc.annotations)
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				parser := NewCronitorConfigParser(&cronJob)
				if name := parser.GetCronitorName(); name != tc.expectedName {
					t.Errorf("expected Name %s, got %s", tc.expectedName, name)
				}
			})
		}
	})

	t.Run("Group annotation", func(t *testing.T) {
		tests := []struct {
			name          string
			annotations   []Annotation
			expectedGroup string
		}{
			{
				name: "new annotation (k8s.cronitor.io/group)",
				annotations: []Annotation{
					{Key: "k8s.cronitor.io/group", Value: "new-group"},
				},
				expectedGroup: "new-group",
			},
			{
				name: "legacy annotation (k8s.cronitor.io/cronitor-group)",
				annotations: []Annotation{
					{Key: "k8s.cronitor.io/cronitor-group", Value: "legacy-group"},
				},
				expectedGroup: "legacy-group",
			},
			{
				name: "new annotation takes precedence",
				annotations: []Annotation{
					{Key: "k8s.cronitor.io/group", Value: "new-group"},
					{Key: "k8s.cronitor.io/cronitor-group", Value: "legacy-group"},
				},
				expectedGroup: "new-group",
			},
		}

		for _, tc := range tests {
			t.Run(tc.name, func(t *testing.T) {
				cronJob, err := CronJobFromAnnotations(tc.annotations)
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				parser := NewCronitorConfigParser(&cronJob)
				if group := parser.GetGroup(); group != tc.expectedGroup {
					t.Errorf("expected Group %s, got %s", tc.expectedGroup, group)
				}
			})
		}
	})

	t.Run("Notify annotation", func(t *testing.T) {
		tests := []struct {
			name           string
			annotations    []Annotation
			expectedNotify []string
		}{
			{
				name: "new annotation (k8s.cronitor.io/notify)",
				annotations: []Annotation{
					{Key: "k8s.cronitor.io/notify", Value: "new-notify"},
				},
				expectedNotify: []string{"new-notify"},
			},
			{
				name: "legacy annotation (k8s.cronitor.io/cronitor-notify)",
				annotations: []Annotation{
					{Key: "k8s.cronitor.io/cronitor-notify", Value: "legacy-notify"},
				},
				expectedNotify: []string{"legacy-notify"},
			},
			{
				name: "new annotation takes precedence",
				annotations: []Annotation{
					{Key: "k8s.cronitor.io/notify", Value: "new-notify"},
					{Key: "k8s.cronitor.io/cronitor-notify", Value: "legacy-notify"},
				},
				expectedNotify: []string{"new-notify"},
			},
			{
				name: "comma-separated values work with new annotation",
				annotations: []Annotation{
					{Key: "k8s.cronitor.io/notify", Value: "notify1, notify2, notify3"},
				},
				expectedNotify: []string{"notify1", "notify2", "notify3"},
			},
		}

		for _, tc := range tests {
			t.Run(tc.name, func(t *testing.T) {
				cronJob, err := CronJobFromAnnotations(tc.annotations)
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				parser := NewCronitorConfigParser(&cronJob)
				notify := parser.GetNotify()
				if len(notify) != len(tc.expectedNotify) {
					t.Errorf("expected %d notify entries, got %d", len(tc.expectedNotify), len(notify))
					return
				}
				for i, n := range notify {
					if n != tc.expectedNotify[i] {
						t.Errorf("expected Notify[%d] %s, got %s", i, tc.expectedNotify[i], n)
					}
				}
			})
		}
	})

	t.Run("GraceSeconds annotation", func(t *testing.T) {
		tests := []struct {
			name                 string
			annotations          []Annotation
			expectedGraceSeconds int
		}{
			{
				name: "new annotation (k8s.cronitor.io/grace-seconds)",
				annotations: []Annotation{
					{Key: "k8s.cronitor.io/grace-seconds", Value: "120"},
				},
				expectedGraceSeconds: 120,
			},
			{
				name: "legacy annotation (k8s.cronitor.io/cronitor-grace-seconds)",
				annotations: []Annotation{
					{Key: "k8s.cronitor.io/cronitor-grace-seconds", Value: "60"},
				},
				expectedGraceSeconds: 60,
			},
			{
				name: "new annotation takes precedence",
				annotations: []Annotation{
					{Key: "k8s.cronitor.io/grace-seconds", Value: "120"},
					{Key: "k8s.cronitor.io/cronitor-grace-seconds", Value: "60"},
				},
				expectedGraceSeconds: 120,
			},
			{
				name:                 "no annotation returns -1",
				annotations:          []Annotation{},
				expectedGraceSeconds: -1,
			},
		}

		for _, tc := range tests {
			t.Run(tc.name, func(t *testing.T) {
				cronJob, err := CronJobFromAnnotations(tc.annotations)
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				parser := NewCronitorConfigParser(&cronJob)
				if graceSeconds := parser.GetGraceSeconds(); graceSeconds != tc.expectedGraceSeconds {
					t.Errorf("expected GraceSeconds %d, got %d", tc.expectedGraceSeconds, graceSeconds)
				}
			})
		}
	})

	t.Run("KeyInference annotation", func(t *testing.T) {
		// Expected hash when using "name" inference with default name "default/test-cronjob"
		expectedHashedID := "3278d16696a89a92d297b7c46bfd286b20dc3896"

		tests := []struct {
			name       string
			annotations []Annotation
			expectedID string
		}{
			{
				name: "new annotation (k8s.cronitor.io/key-inference)",
				annotations: []Annotation{
					{Key: "k8s.cronitor.io/key-inference", Value: "name"},
				},
				expectedID: expectedHashedID,
			},
			{
				name: "legacy annotation (k8s.cronitor.io/id-inference)",
				annotations: []Annotation{
					{Key: "k8s.cronitor.io/id-inference", Value: "name"},
				},
				expectedID: expectedHashedID,
			},
			{
				name: "new annotation takes precedence",
				annotations: []Annotation{
					{Key: "k8s.cronitor.io/key-inference", Value: "name"},
					{Key: "k8s.cronitor.io/id-inference", Value: "k8s"},
				},
				expectedID: expectedHashedID,
			},
		}

		for _, tc := range tests {
			t.Run(tc.name, func(t *testing.T) {
				cronJob, err := CronJobFromAnnotations(tc.annotations)
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				parser := NewCronitorConfigParser(&cronJob)
				if id := parser.GetCronitorID(); id != tc.expectedID {
					t.Errorf("expected ID %s, got %s", tc.expectedID, id)
				}
			})
		}
	})
}

func TestGetMetricDuration(t *testing.T) {
	tests := []struct {
		name        string
		annotations []Annotation
		expected    string
	}{
		{
			name: "less than duration",
			annotations: []Annotation{
				{Key: "k8s.cronitor.io/metric.duration", Value: "< 5 seconds"},
			},
			expected: "< 5 seconds",
		},
		{
			name: "greater than duration",
			annotations: []Annotation{
				{Key: "k8s.cronitor.io/metric.duration", Value: "> 1 minute"},
			},
			expected: "> 1 minute",
		},
		{
			name: "comma-separated values",
			annotations: []Annotation{
				{Key: "k8s.cronitor.io/metric.duration", Value: "< 5 seconds, > 1 second"},
			},
			expected: "< 5 seconds, > 1 second",
		},
		{
			name:        "no annotation",
			annotations: []Annotation{},
			expected:    "",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			cronJob, err := CronJobFromAnnotations(tc.annotations)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			parser := NewCronitorConfigParser(&cronJob)
			if got := parser.GetMetricDuration(); got != tc.expected {
				t.Errorf("GetMetricDuration() = %q, want %q", got, tc.expected)
			}
		})
	}
}

func TestGetNote(t *testing.T) {
	tests := []struct {
		name         string
		annotations  []Annotation
		expectedNote string
	}{
		{
			name: "note annotation present",
			annotations: []Annotation{
				{Key: "k8s.cronitor.io/note", Value: "This is my job description"},
			},
			expectedNote: "This is my job description",
		},
		{
			name:         "no note annotation",
			annotations:  []Annotation{},
			expectedNote: "",
		},
		{
			name: "empty note annotation",
			annotations: []Annotation{
				{Key: "k8s.cronitor.io/note", Value: ""},
			},
			expectedNote: "",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			cronJob, err := CronJobFromAnnotations(tc.annotations)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			parser := NewCronitorConfigParser(&cronJob)
			if note := parser.GetNote(); note != tc.expectedNote {
				t.Errorf("expected Note %q, got %q", tc.expectedNote, note)
			}
		})
	}
}

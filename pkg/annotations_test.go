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
		name                  string
		annotationIDInference string
		annotationCronitorID  string
		expectedID            string
	}{
		{
			name:                  "hashed name as ID",
			annotationIDInference: "name",
			annotationCronitorID:  "",
			expectedID:            "3278d16696a89a92d297b7c46bfd286b20dc3896",
		},
		{
			name:                  "specific cronitor id",
			annotationIDInference: "",
			annotationCronitorID:  "1234",
			expectedID:            "1234",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			annotations := []Annotation{
				{Key: "k8s.cronitor.io/id-inference", Value: tc.annotationIDInference},
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

func TestEnabledAutoComplete(t *testing.T) {
	tests := []struct {
		name          string
		annotation    string
		expectedValue bool
		expectError   bool
	}{
		{
			name:          "explicitly disabled",
			annotation:    "false",
			expectedValue: false,
			expectError:   false,
		},
		{
			name:          "explicitly enabled",
			annotation:    "true",
			expectedValue: true,
			expectError:   false,
		},
		{
			name:          "invalid value",
			annotation:    "invalid",
			expectedValue: false,
			expectError:   true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			annotations := []Annotation{
				{Key: "k8s.cronitor.io/auto-complete", Value: tc.annotation},
			}

			cronJob, err := CronJobFromAnnotations(annotations)
			if err != nil {
				t.Fatalf("failed to create CronJob from annotations: %v", err)
			}

			parser := NewCronitorConfigParser(&cronJob)
			got, err := parser.EnabledAutoComplete()

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
				t.Errorf("EnabledAutoComplete() = %v, want %v", got, tc.expectedValue)
			}
		})
	}
}

package pkg

import (
	"encoding/json"
	"k8s.io/api/batch/v1beta1"
	"os"
	"strconv"
	"testing"
)

func TestCronJobInclusion(t *testing.T) {
	var jsonBlob v1beta1.CronJobList
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

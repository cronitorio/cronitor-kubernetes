package api

import (
	"encoding/json"
	"fmt"
	"testing"

	v1 "k8s.io/api/batch/v1"
)

func TestNamespaceTag(t *testing.T) {
	var jsonBlob v1.CronJobList
	// Namespace in this example CronJob is "cronitor"
	err := json.Unmarshal([]byte(`{"metadata":{"selfLink":"/apis/batch/v1beta1/cronjobs","resourceVersion":"41530"},"items":[{"metadata":{"name":"eventrouter-test-croonjob","namespace":"cronitor","selfLink":"/apis/batch/v1beta1/namespaces/cronitor/cronjobs/eventrouter-test-croonjob","uid":"a4892036-090f-4019-8bd1-98bfe0a9034c","resourceVersion":"41467","creationTimestamp":"2020-11-13T06:06:44Z","labels":{"app.kubernetes.io/managed-by":"skaffold","skaffold.dev/run-id":"a592b4e3-dd8e-4b25-a69f-7abe35e264f0"},"managedFields":[{"manager":"Go-http-client","operation":"Update","apiVersion":"batch/v1beta1","time":"2020-11-13T06:06:44Z","fieldsType":"FieldsV1","fieldsV1":{"f:spec":{"f:concurrencyPolicy":{},"f:failedJobsHistoryLimit":{},"f:jobTemplate":{"f:spec":{"f:template":{"f:spec":{"f:containers":{"k:{\"name\":\"hello\"}":{".":{},"f:args":{},"f:image":{},"f:imagePullPolicy":{},"f:name":{},"f:resources":{},"f:terminationMessagePath":{},"f:terminationMessagePolicy":{}}},"f:dnsPolicy":{},"f:restartPolicy":{},"f:schedulerName":{},"f:securityContext":{},"f:terminationGracePeriodSeconds":{}}}}},"f:schedule":{},"f:successfulJobsHistoryLimit":{},"f:suspend":{}}}},{"manager":"skaffold","operation":"Update","apiVersion":"batch/v1beta1","time":"2020-11-13T06:06:45Z","fieldsType":"FieldsV1","fieldsV1":{"f:metadata":{"f:labels":{".":{},"f:app.kubernetes.io/managed-by":{},"f:skaffold.dev/run-id":{}}}}},{"manager":"kube-controller-manager","operation":"Update","apiVersion":"batch/v1beta1","time":"2020-11-13T07:57:06Z","fieldsType":"FieldsV1","fieldsV1":{"f:status":{"f:active":{},"f:lastScheduleTime":{}}}}]},"spec":{"schedule":"*/1 * * * *","concurrencyPolicy":"Forbid","suspend":false,"jobTemplate":{"metadata":{"creationTimestamp":null},"spec":{"template":{"metadata":{"creationTimestamp":null},"spec":{"containers":[{"name":"hello","image":"busybox","args":["/bin/sh","-c","date ; echo Hello from k8s"],"resources":{},"terminationMessagePath":"/dev/termination-log","terminationMessagePolicy":"File","imagePullPolicy":"Always"}],"restartPolicy":"OnFailure","terminationGracePeriodSeconds":30,"dnsPolicy":"ClusterFirst","securityContext":{},"schedulerName":"default-scheduler"}}}},"successfulJobsHistoryLimit":3,"failedJobsHistoryLimit":1},"status":{"active":[{"kind":"Job","namespace":"cronitor","name":"eventrouter-test-croonjob-1605254220","uid":"697df5f5-6366-42fe-a20e-19ec2fefd826","apiVersion":"batch/v1","resourceVersion":"41465"}],"lastScheduleTime":"2020-11-13T07:57:00Z"}}]}`), &jsonBlob)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	cronJob := convertCronJobToCronitorJob(&jsonBlob.Items[0])

	for _, tag := range cronJob.Tags {
		if tag == "kubernetes-namespace:cronitor" {
			return
		}
	}
	t.Errorf("no tag `%s` found on CronitorJob object", "kubernetes-namespace:cronitor")
}

func TestEnvironmentTag(t *testing.T) {
	t.Skip("Skipping for now, we've removed cluster-env tag")
	var jsonBlob v1.CronJob
	// Provided environment is 'staging'
	err := json.Unmarshal([]byte(`{"apiVersion":"batch/v1beta1","kind":"CronJob","metadata":{"annotations":{"k8s.cronitor.io/env":"staging","k8s.cronitor.io/tags":"tag1,tagname:tagvalue","kubectl.kubernetes.io/last-applied-configuration":"{\"apiVersion\":\"batch/v1beta1\",\"kind\":\"CronJob\",\"metadata\":{\"annotations\":{\"k8s.cronitor.io/env\":\"staging\",\"k8s.cronitor.io/tags\":\"tag1,tagname:tagvalue\"},\"name\":\"eventrouter-test-cronjob\",\"namespace\":\"default\"},\"spec\":{\"concurrencyPolicy\":\"Forbid\",\"jobTemplate\":{\"spec\":{\"backoffLimit\":3,\"template\":{\"spec\":{\"containers\":[{\"args\":[\"/bin/sh\",\"-c\",\"date ; sleep 5 ; echo Hello from k8s\"],\"image\":\"busybox\",\"name\":\"hello\"}],\"restartPolicy\":\"OnFailure\"}}}},\"schedule\":\"*/1 * * * *\"}}\n"},"name":"eventrouter-test-cronjob","namespace":"default"},"spec":{"concurrencyPolicy":"Forbid","jobTemplate":{"spec":{"backoffLimit":3,"template":{"spec":{"containers":[{"args":["/bin/sh","-c","date ; sleep 5 ; echo Hello from k8s"],"image":"busybox","name":"hello"}],"restartPolicy":"OnFailure"}}}},"schedule":"*/1 * * * *"}}`), &jsonBlob)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	cronJob := convertCronJobToCronitorJob(&jsonBlob)

	for _, tag := range cronJob.Tags {
		if tag == "cluster-env:staging" {
			return
		}
	}
	t.Errorf("no environment tag `%s` found on CronitorJob object", "cluster-env:staging")
}

func TestTagList(t *testing.T) {
	var jsonBlob v1.CronJob
	// Provided taglist is 'tag1,tagname:tagvalue'
	err := json.Unmarshal([]byte(`{"apiVersion":"batch/v1beta1","kind":"CronJob","metadata":{"annotations":{"k8s.cronitor.io/env":"staging","k8s.cronitor.io/tags":"tag1,tagname:tagvalue","kubectl.kubernetes.io/last-applied-configuration":"{\"apiVersion\":\"batch/v1beta1\",\"kind\":\"CronJob\",\"metadata\":{\"annotations\":{\"k8s.cronitor.io/env\":\"staging\",\"k8s.cronitor.io/tags\":\"tag1,tagname:tagvalue\"},\"name\":\"eventrouter-test-cronjob\",\"namespace\":\"default\"},\"spec\":{\"concurrencyPolicy\":\"Forbid\",\"jobTemplate\":{\"spec\":{\"backoffLimit\":3,\"template\":{\"spec\":{\"containers\":[{\"args\":[\"/bin/sh\",\"-c\",\"date ; sleep 5 ; echo Hello from k8s\"],\"image\":\"busybox\",\"name\":\"hello\"}],\"restartPolicy\":\"OnFailure\"}}}},\"schedule\":\"*/1 * * * *\"}}\n"},"name":"eventrouter-test-cronjob","namespace":"default"},"spec":{"concurrencyPolicy":"Forbid","jobTemplate":{"spec":{"backoffLimit":3,"template":{"spec":{"containers":[{"args":["/bin/sh","-c","date ; sleep 5 ; echo Hello from k8s"],"image":"busybox","name":"hello"}],"restartPolicy":"OnFailure"}}}},"schedule":"*/1 * * * *"}}`), &jsonBlob)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	cronJob := convertCronJobToCronitorJob(&jsonBlob)

	expectedTagList := []string{"tag1", "tagname:tagvalue"}
	for _, value := range expectedTagList {
		t.Run(fmt.Sprintf("check for presence of `%s`", value), func(t *testing.T) {
			for _, tag := range cronJob.Tags {
				if tag == value {
					return
				}
			}
			t.Errorf("no tag `%s` found in tag list", value)
		})
	}
}

func TestExistingCronitorID(t *testing.T) {
	var jsonBlob v1.CronJob
	// provided cronitor-id is 'uv93823'
	err := json.Unmarshal([]byte(`{"apiVersion":"batch/v1beta1","kind":"CronJob","metadata":{"annotations":{"k8s.cronitor.io/cronitor-id":"uv93823","k8s.cronitor.io/env":"staging","kubectl.kubernetes.io/last-applied-configuration":"{\"apiVersion\":\"batch/v1beta1\",\"kind\":\"CronJob\",\"metadata\":{\"annotations\":{\"k8s.cronitor.io/cronitor-id\":\"uv93823\",\"k8s.cronitor.io/env\":\"staging\"},\"name\":\"eventrouter-test-cronjob\",\"namespace\":\"default\"},\"spec\":{\"concurrencyPolicy\":\"Forbid\",\"jobTemplate\":{\"spec\":{\"backoffLimit\":3,\"template\":{\"spec\":{\"containers\":[{\"args\":[\"/bin/sh\",\"-c\",\"date ; sleep 5 ; echo Hello from k8s\"],\"image\":\"busybox\",\"name\":\"hello\"}],\"restartPolicy\":\"OnFailure\"}}}},\"schedule\":\"*/1 * * * *\"}}\n"},"name":"eventrouter-test-cronjob","namespace":"default"},"spec":{"concurrencyPolicy":"Forbid","jobTemplate":{"spec":{"backoffLimit":3,"template":{"spec":{"containers":[{"args":["/bin/sh","-c","date ; sleep 5 ; echo Hello from k8s"],"image":"busybox","name":"hello"}],"restartPolicy":"OnFailure"}}}},"schedule":"*/1 * * * *"}}`), &jsonBlob)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	cronJob := convertCronJobToCronitorJob(&jsonBlob)

	if cronJob.Key != "uv93823" {
		t.Errorf("expected cronitorJob key of `uv93823`, got `%s`", cronJob.Key)
	}
}

func TestEmptyCronitorIDAnnotation(t *testing.T) {
	var jsonBlob v1.CronJob
	// provided cronitor-id is ''
	err := json.Unmarshal([]byte(`{"apiVersion":"batch/v1beta1","kind":"CronJob","metadata":{"annotations":{"k8s.cronitor.io/cronitor-id":"","k8s.cronitor.io/env":"staging","kubectl.kubernetes.io/last-applied-configuration":"{\"apiVersion\":\"batch/v1beta1\",\"kind\":\"CronJob\",\"metadata\":{\"annotations\":{\"k8s.cronitor.io/cronitor-id\":\"uv93823\",\"k8s.cronitor.io/env\":\"staging\"},\"name\":\"eventrouter-test-cronjob\",\"namespace\":\"default\"},\"spec\":{\"concurrencyPolicy\":\"Forbid\",\"jobTemplate\":{\"spec\":{\"backoffLimit\":3,\"template\":{\"spec\":{\"containers\":[{\"args\":[\"/bin/sh\",\"-c\",\"date ; sleep 5 ; echo Hello from k8s\"],\"image\":\"busybox\",\"name\":\"hello\"}],\"restartPolicy\":\"OnFailure\"}}}},\"schedule\":\"*/1 * * * *\"}}\n"},"name":"eventrouter-test-cronjob","namespace":"default"},"spec":{"concurrencyPolicy":"Forbid","jobTemplate":{"spec":{"backoffLimit":3,"template":{"spec":{"containers":[{"args":["/bin/sh","-c","date ; sleep 5 ; echo Hello from k8s"],"image":"busybox","name":"hello"}],"restartPolicy":"OnFailure"}}}},"schedule":"*/1 * * * *"}}`), &jsonBlob)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	cronJob := convertCronJobToCronitorJob(&jsonBlob)

	if cronJob.Key != string(jsonBlob.GetUID()) {
		t.Errorf("expected cronitorJob key of default `%s`, got `%s`", jsonBlob.GetUID(), cronJob.Key)
	}
}

func TestCronitorGroupAnnotation(t *testing.T) {
	var jsonBlob v1.CronJob
	err := json.Unmarshal([]byte(`{"apiVersion":"batch/v1beta1","kind":"CronJob","metadata":{"annotations":{"k8s.cronitor.io/cronitor-group":"test-group","k8s.cronitor.io/env":"staging","kubectl.kubernetes.io/last-applied-configuration":"{\"apiVersion\":\"batch/v1beta1\",\"kind\":\"CronJob\",\"metadata\":{\"annotations\":{\"k8s.cronitor.io/cronitor-id\":\"uv93823\",\"k8s.cronitor.io/env\":\"staging\"},\"name\":\"eventrouter-test-cronjob\",\"namespace\":\"default\"},\"spec\":{\"concurrencyPolicy\":\"Forbid\",\"jobTemplate\":{\"spec\":{\"backoffLimit\":3,\"template\":{\"spec\":{\"containers\":[{\"args\":[\"/bin/sh\",\"-c\",\"date ; sleep 5 ; echo Hello from k8s\"],\"image\":\"busybox\",\"name\":\"hello\"}],\"restartPolicy\":\"OnFailure\"}}}},\"schedule\":\"*/1 * * * *\"}}\n"},"name":"eventrouter-test-cronjob","namespace":"default"},"spec":{"concurrencyPolicy":"Forbid","jobTemplate":{"spec":{"backoffLimit":3,"template":{"spec":{"containers":[{"args":["/bin/sh","-c","date ; sleep 5 ; echo Hello from k8s"],"image":"busybox","name":"hello"}],"restartPolicy":"OnFailure"}}}},"schedule":"*/1 * * * *"}}`), &jsonBlob)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	cronJob := convertCronJobToCronitorJob(&jsonBlob)

	if cronJob.Group != string(jsonBlob.Annotations["k8s.cronitor.io/cronitor-group"]) {
		t.Errorf("expected cronitor-group `%s`, got `%s`", jsonBlob.Annotations["k8s.cronitor.io/cronitor-group"], cronJob.Group)
	}
}

func TestCronitorNotifyAnnotation(t *testing.T) {
	var jsonBlob v1.CronJob
	err := json.Unmarshal([]byte(`{"apiVersion":"batch/v1beta1","kind":"CronJob","metadata":{"annotations":{"k8s.cronitor.io/cronitor-notify":"devops-slack, infra-teams","k8s.cronitor.io/env":"staging","kubectl.kubernetes.io/last-applied-configuration":"{\"apiVersion\":\"batch/v1beta1\",\"kind\":\"CronJob\",\"metadata\":{\"annotations\":{\"k8s.cronitor.io/cronitor-id\":\"uv93823\",\"k8s.cronitor.io/env\":\"staging\"},\"name\":\"eventrouter-test-cronjob\",\"namespace\":\"default\"},\"spec\":{\"concurrencyPolicy\":\"Forbid\",\"jobTemplate\":{\"spec\":{\"backoffLimit\":3,\"template\":{\"spec\":{\"containers\":[{\"args\":[\"/bin/sh\",\"-c\",\"date ; sleep 5 ; echo Hello from k8s\"],\"image\":\"busybox\",\"name\":\"hello\"}],\"restartPolicy\":\"OnFailure\"}}}},\"schedule\":\"*/1 * * * *\"}}\n"},"name":"eventrouter-test-cronjob","namespace":"default"},"spec":{"concurrencyPolicy":"Forbid","jobTemplate":{"spec":{"backoffLimit":3,"template":{"spec":{"containers":[{"args":["/bin/sh","-c","date ; sleep 5 ; echo Hello from k8s"],"image":"busybox","name":"hello"}],"restartPolicy":"OnFailure"}}}},"schedule":"*/1 * * * *"}}`), &jsonBlob)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	cronJob := convertCronJobToCronitorJob(&jsonBlob)

	var expected = []string{"devops-slack", "infra-teams"}

	if cronJob.Notify[0] != expected[0] {
		t.Errorf("expected cronitor-notify `%s`, got `%s`", expected, cronJob.Notify)
	}
}

func TestCronitorGraceSecondsAnnotation(t *testing.T) {
	var jsonBlob v1.CronJob
	err := json.Unmarshal([]byte(`{"apiVersion":"batch/v1beta1","kind":"CronJob","metadata":{"annotations":{"k8s.cronitor.io/cronitor-grace-seconds":"120","k8s.cronitor.io/env":"staging","kubectl.kubernetes.io/last-applied-configuration":"{\"apiVersion\":\"batch/v1beta1\",\"kind\":\"CronJob\",\"metadata\":{\"annotations\":{\"k8s.cronitor.io/cronitor-id\":\"uv93823\",\"k8s.cronitor.io/env\":\"staging\"},\"name\":\"eventrouter-test-cronjob\",\"namespace\":\"default\"},\"spec\":{\"concurrencyPolicy\":\"Forbid\",\"jobTemplate\":{\"spec\":{\"backoffLimit\":3,\"template\":{\"spec\":{\"containers\":[{\"args\":[\"/bin/sh\",\"-c\",\"date ; sleep 5 ; echo Hello from k8s\"],\"image\":\"busybox\",\"name\":\"hello\"}],\"restartPolicy\":\"OnFailure\"}}}},\"schedule\":\"*/1 * * * *\"}}\n"},"name":"eventrouter-test-cronjob","namespace":"default"},"spec":{"concurrencyPolicy":"Forbid","jobTemplate":{"spec":{"backoffLimit":3,"template":{"spec":{"containers":[{"args":["/bin/sh","-c","date ; sleep 5 ; echo Hello from k8s"],"image":"busybox","name":"hello"}],"restartPolicy":"OnFailure"}}}},"schedule":"*/1 * * * *"}}`), &jsonBlob)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	cronJob := convertCronJobToCronitorJob(&jsonBlob)

	if cronJob.GraceSeconds != 120 {
		t.Errorf("expected cronitor-grace-seconds `%d`, got `%d`", 120, cronJob.GraceSeconds)
	}
}

func TestTruncateDefaultName(t *testing.T) {
	shortName := "abcefgh"
	if newName := truncateDefaultName(shortName); newName != shortName {
		t.Errorf("expected truncated name for '%s' to be '%s', got '%s'", shortName, shortName, newName)
	}

	longName := "this-is-a-very-long-namespace-name-lets-make-it-really-freaking-long/and-a-very-long-job-name-very-very-long-abcdef12345"
	expectedNewName := "this-is-a-very-long-namespace-name-lets-make-it-re…d-a-very-long-job-name-very-very-long-abcdef12345"
	if newName := truncateDefaultName(longName); newName != expectedNewName {
		t.Errorf("expected truncated name for '%s' to be '%s', got '%s'", longName, expectedNewName, newName)
	}
}

func TestValidateTagName(t *testing.T) {
	shortTag := "env:short-tag"
	if newTag := ValidateTagName(shortTag); newTag != shortTag {
		t.Errorf("expected validated tag for '%s' to be '%s', got '%s'", shortTag, shortTag, newTag)
	}

	longTag := "kubernetes-namespace:this-is-a-very-long-namespace-name-lets-make-it-really-freaking-long-not-long-enough-lets-keep-going"
	expectedNewTag := "kubernetes-namespace:this-is-a-very-long-namespace-name-lets-make-it-really-freaking-long-not-long-…"
	if newTag := ValidateTagName(longTag); newTag != expectedNewTag {
		t.Errorf("expected validated tag for '%s' to be '%s', got '%s'", longTag, expectedNewTag, newTag)
	}
}

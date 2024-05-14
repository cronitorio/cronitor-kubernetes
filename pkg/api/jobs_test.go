package api

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/cronitorio/cronitor-kubernetes/pkg"
	v1 "k8s.io/api/batch/v1"
)

func TestNamespaceTag(t *testing.T) {
	var cronJobList v1.CronJobList
	// Namespace in this example CronJob is "cronitor"
	err := json.Unmarshal([]byte(`{"metadata":{"selfLink":"/apis/batch/v1beta1/cronjobs","resourceVersion":"41530"},"items":[{"metadata":{"name":"eventrouter-test-croonjob","namespace":"cronitor","selfLink":"/apis/batch/v1beta1/namespaces/cronitor/cronjobs/eventrouter-test-croonjob","uid":"a4892036-090f-4019-8bd1-98bfe0a9034c","resourceVersion":"41467","creationTimestamp":"2020-11-13T06:06:44Z","labels":{"app.kubernetes.io/managed-by":"skaffold","skaffold.dev/run-id":"a592b4e3-dd8e-4b25-a69f-7abe35e264f0"},"managedFields":[{"manager":"Go-http-client","operation":"Update","apiVersion":"batch/v1beta1","time":"2020-11-13T06:06:44Z","fieldsType":"FieldsV1","fieldsV1":{"f:spec":{"f:concurrencyPolicy":{},"f:failedJobsHistoryLimit":{},"f:jobTemplate":{"f:spec":{"f:template":{"f:spec":{"f:containers":{"k:{\"name\":\"hello\"}":{".":{},"f:args":{},"f:image":{},"f:imagePullPolicy":{},"f:name":{},"f:resources":{},"f:terminationMessagePath":{},"f:terminationMessagePolicy":{}}},"f:dnsPolicy":{},"f:restartPolicy":{},"f:schedulerName":{},"f:securityContext":{},"f:terminationGracePeriodSeconds":{}}}}},"f:schedule":{},"f:successfulJobsHistoryLimit":{},"f:suspend":{}}}},{"manager":"skaffold","operation":"Update","apiVersion":"batch/v1beta1","time":"2020-11-13T06:06:45Z","fieldsType":"FieldsV1","fieldsV1":{"f:metadata":{"f:labels":{".":{},"f:app.kubernetes.io/managed-by":{},"f:skaffold.dev/run-id":{}}}}},{"manager":"kube-controller-manager","operation":"Update","apiVersion":"batch/v1beta1","time":"2020-11-13T07:57:06Z","fieldsType":"FieldsV1","fieldsV1":{"f:status":{"f:active":{},"f:lastScheduleTime":{}}}}]},"spec":{"schedule":"*/1 * * * *","concurrencyPolicy":"Forbid","suspend":false,"jobTemplate":{"metadata":{"creationTimestamp":null},"spec":{"template":{"metadata":{"creationTimestamp":null},"spec":{"containers":[{"name":"hello","image":"busybox","args":["/bin/sh","-c","date ; echo Hello from k8s"],"resources":{},"terminationMessagePath":"/dev/termination-log","terminationMessagePolicy":"File","imagePullPolicy":"Always"}],"restartPolicy":"OnFailure","terminationGracePeriodSeconds":30,"dnsPolicy":"ClusterFirst","securityContext":{},"schedulerName":"default-scheduler"}}}},"successfulJobsHistoryLimit":3,"failedJobsHistoryLimit":1},"status":{"active":[{"kind":"Job","namespace":"cronitor","name":"eventrouter-test-croonjob-1605254220","uid":"697df5f5-6366-42fe-a20e-19ec2fefd826","apiVersion":"batch/v1","resourceVersion":"41465"}],"lastScheduleTime":"2020-11-13T07:57:00Z"}}]}`), &cronJobList)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	cronitorJob := convertCronJobToCronitorJob(&cronJobList.Items[0])

	for _, tag := range cronitorJob.Tags {
		if tag == "kubernetes-namespace:cronitor" {
			return
		}
	}
	t.Errorf("no tag `%s` found on CronitorJob object", "kubernetes-namespace:cronitor")
}

func TestEnvironmentTag(t *testing.T) {
	t.Skip("Skipping for now, we've removed cluster-env tag")

	annotations := []pkg.Annotation{
		{Key: "k8s.cronitor.io/env", Value: "staging"},
	}
	cronJob, err := pkg.CronJobFromAnnotations(annotations)
	if err != nil {
		t.Fatalf("unexpected error unmarshalling json: %v", err)
	}
	cronitorJob := convertCronJobToCronitorJob(&cronJob)

	for _, tag := range cronitorJob.Tags {
		if tag == "cluster-env:staging" {
			return
		}
	}
	t.Errorf("no environment tag `%s` found on CronitorJob object", "cluster-env:staging")
}

func TestTagList(t *testing.T) {
	annotations := []pkg.Annotation{
		{Key: "k8s.cronitor.io/tags", Value: "tag1,tagname:tagvalue"},
	}
	cronJob, err := pkg.CronJobFromAnnotations(annotations)
	if err != nil {
		t.Fatalf("unexpected error unmarshalling json: %v", err)
	}
	cronitorJob := convertCronJobToCronitorJob(&cronJob)

	expectedTagList := []string{"tag1", "tagname:tagvalue"}
	for _, value := range expectedTagList {
		t.Run(fmt.Sprintf("check for presence of `%s`", value), func(t *testing.T) {
			for _, tag := range cronitorJob.Tags {
				if tag == value {
					return
				}
			}
			t.Errorf("no tag `%s` found in tag list", value)
		})
	}
}

func TestExistingCronitorID(t *testing.T) {
	annotations := []pkg.Annotation{
		{Key: "k8s.cronitor.io/cronitor-id", Value: "uv93823"},
	}
	cronJob, err := pkg.CronJobFromAnnotations(annotations)
	if err != nil {
		t.Fatalf("unexpected error unmarshalling json: %v", err)
	}
	cronitorJob := convertCronJobToCronitorJob(&cronJob)

	if cronitorJob.Key != string(cronJob.Annotations["k8s.cronitor.io/cronitor-id"]) {
		t.Errorf("expected cronitorJob key of `uv93823`, got `%s`", cronitorJob.Key)
	}
}

func TestEmptyCronitorIDAnnotation(t *testing.T) {
	annotations := []pkg.Annotation{
		{Key: "k8s.cronitor.io/cronitor-id", Value: ""},
	}
	cronJob, err := pkg.CronJobFromAnnotations(annotations)
	if err != nil {
		t.Fatalf("unexpected error unmarshalling json: %v", err)
	}
	cronitorJob := convertCronJobToCronitorJob(&cronJob)

	if cronitorJob.Key != string(cronJob.GetUID()) {
		t.Errorf("expected cronitorJob key of default `%s`, got `%s`", cronJob.GetUID(), cronitorJob.Key)
	}
}

func TestCronitorGroupAnnotation(t *testing.T) {
	annotations := []pkg.Annotation{
		{Key: "k8s.cronitor.io/cronitor-group", Value: "test-group"},
	}
	cronJob, err := pkg.CronJobFromAnnotations(annotations)
	if err != nil {
		t.Fatalf("unexpected error unmarshalling json: %v", err)
	}
	cronitorJob := convertCronJobToCronitorJob(&cronJob)

	if cronitorJob.Group != string(cronJob.Annotations["k8s.cronitor.io/cronitor-group"]) {
		t.Errorf("expected cronitor-group `%s`, got `%s`", cronJob.Annotations["k8s.cronitor.io/cronitor-group"], cronitorJob.Group)
	}
}

func TestCronitorNotifyAnnotation(t *testing.T) {
	annotations := []pkg.Annotation{
		{Key: "k8s.cronitor.io/cronitor-notify", Value: "devops-slack, infra-teams"},
	}
	cronJob, err := pkg.CronJobFromAnnotations(annotations)
	if err != nil {
		t.Fatalf("unexpected error unmarshalling json: %v", err)
	}
	cronitorJob := convertCronJobToCronitorJob(&cronJob)

	var expected = []string{"devops-slack", "infra-teams"}

	if cronitorJob.Notify[0] != expected[0] {
		t.Errorf("expected cronitor-notify `%s`, got `%s`", expected, cronitorJob.Notify)
	}
}

func TestCronitorGraceSecondsAnnotation(t *testing.T) {
	annotations := []pkg.Annotation{
		{Key: "k8s.cronitor.io/cronitor-grace-seconds", Value: "120"},
	}
	cronJob, err := pkg.CronJobFromAnnotations(annotations)
	if err != nil {
		t.Fatalf("unexpected error unmarshalling json: %v", err)
	}
	cronitorJob := convertCronJobToCronitorJob(&cronJob)

	if cronitorJob.GraceSeconds != 120 {
		t.Errorf("expected cronitor-grace-seconds `%d`, got `%d`", 120, cronitorJob.GraceSeconds)
	}
}

func TestTruncateName(t *testing.T) {
	shortName := "abcefgh"
	if newName := truncateName(shortName); newName != shortName {
		t.Errorf("expected truncated name for '%s' to be '%s', got '%s'", shortName, shortName, newName)
	}

	longName := "this-is-a-very-long-namespace-name-lets-make-it-really-freaking-long/and-a-very-long-job-name-very-very-long-abcdef12345"
	expectedNewName := "this-is-a-very-long-namespace-name-lets-make-it-re…d-a-very-long-job-name-very-very-long-abcdef12345"
	if newName := truncateName(longName); newName != expectedNewName {
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

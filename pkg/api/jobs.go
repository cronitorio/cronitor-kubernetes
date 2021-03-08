package api

import (
	"encoding/json"
	"fmt"
	"github.com/cronitorio/cronitor-kubernetes/pkg"
	"k8s.io/api/batch/v1beta1"
	"strconv"
	"strings"
)

/*
  Currently using pieces of the Cronitor CLI as POC, writing extra code as needed.
  Longer term: combine with Cronitor CLI?
*/

// https://docs.google.com/document/d/1erh-fvTkF14jyJGv3DYuN2UalWe6AN49XOUsWHJccso/edit#heading=h.ylm2gai335jy
type CronitorJob struct {
	Key         string   `json:"key"`
	DefaultName string   `json:"defaultName"`
	Name        string   `json:"name,omitempty"`
	DefaultNote string   `json:"defaultNote"`
	Metadata    string   `json:"metadata"` // This is actually a string rather than a map
	Type_       string   `json:"type"`     // 'job'
	Schedule    string   `json:"schedule"`
	Tags        []string `json:"tags,omitempty"`
	Rules       []string `json:"rules"`
}

func (cronitorJob CronitorJob) GetEnvironment() string {
	for _, tag := range cronitorJob.Tags {
		// This is a bit naive; what if somehow more than one "env" tag is present?
		// That shouldn't really be the case, but in future versions we may want to add some
		// structuring around tags, or at least around environments to ensure that only one can
		// be added.
		if strings.HasPrefix(tag, "env:") {
			return strings.TrimPrefix(tag, "env:")
		}
	}

	return ""
}

func convertCronJobToCronitorJob(job *v1beta1.CronJob) CronitorJob {
	configParser := pkg.NewCronitorConfigParser(job)

	name := fmt.Sprintf("%s/%s", job.Namespace, job.Name)
	metadata := make(map[string]string)
	if job.Spec.ConcurrencyPolicy != "" {
		metadata["concurrencyPolicy"] = string(job.Spec.ConcurrencyPolicy)
	}
	if job.Spec.StartingDeadlineSeconds != nil {
		metadata["startingDeadlineSeconds"] = strconv.FormatInt(*job.Spec.StartingDeadlineSeconds, 10)
	}
	metadataJson, _ := json.Marshal(metadata)

	allTags := []string{
		"kubernetes",
		fmt.Sprintf("kubernetes-namespace:%s", job.Namespace),
	}
	for _, tag := range configParser.GetTags() {
		allTags = append(allTags, tag)
	}
	if environment := configParser.GetEnvironment(); environment != "" {
		allTags = append(allTags, fmt.Sprintf("env:%s", environment))
	}

	var key string
	if existingCronitorID := configParser.GetCronitorID(); existingCronitorID != "" {
		key = existingCronitorID
	} else {
		key = string(job.UID)
	}

	cronitorJob := CronitorJob{
		Key:         key,
		DefaultName: name,
		DefaultNote: fmt.Sprintf("created by cronitor-kubernetes, monitors %s in cluster %s", name, job.ObjectMeta.GetClusterName()),
		Schedule:    job.Spec.Schedule,
		Metadata:    string(metadataJson),
		Type_:       "job",
		Tags:        allTags,
		// An empty rules array is required
		Rules: []string{},
	}

	return cronitorJob
}

func convertCronJobsToCronitorJobs(jobs []*v1beta1.CronJob) []CronitorJob {
	outputList := make([]CronitorJob, len(jobs))
	for _, job := range jobs {
		outputList = append(outputList, convertCronJobToCronitorJob(job))
	}
	return outputList
}

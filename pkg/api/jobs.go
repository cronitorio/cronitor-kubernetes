package api

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/aquilax/truncate"
	"github.com/cronitorio/cronitor-kubernetes/pkg"
	v1 "k8s.io/api/batch/v1"
)

/*
  Currently using pieces of the Cronitor CLI as POC, writing extra code as needed.
  Longer term: combine with Cronitor CLI?
*/

// https://docs.google.com/document/d/1erh-fvTkF14jyJGv3DYuN2UalWe6AN49XOUsWHJccso/edit#heading=h.ylm2gai335jy
type CronitorJob struct {
	Key          string   `json:"key"`
	Name         string   `json:"name,omitempty"`
	Note         string   `json:"note,omitempty"`
	Metadata     string   `json:"metadata"` // This is actually a string rather than a map
	Type_        string   `json:"type"`     // 'job'
	Schedule     string   `json:"schedule"`
	Timezone     string   `json:"timezone,omitempty"`
	Tags         []string `json:"tags,omitempty"`
	Notify       []string `json:"notify,omitempty"`
	Group        string   `json:"group,omitempty"`
	GraceSeconds int      `json:"grace_seconds,omitempty"`
	Assertions   []string `json:"assertions,omitempty"`
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

func truncateName(name string) string {
	if len(name) > 100 {
		name = truncate.Truncator(name, 100, truncate.EllipsisMiddleStrategy{})
	}

	return name
}

func ValidateTagName(tagName string) string {
	name := tagName
	if len(tagName) > 100 {
		name = truncate.Truncator(name, 100, truncate.CutEllipsisStrategy{})
	}

	return name
}

func convertCronJobToCronitorJob(cronJob *v1.CronJob) CronitorJob {
	configParser := pkg.NewCronitorConfigParser(cronJob)

	metadata := make(map[string]string)
	if cronJob.Spec.ConcurrencyPolicy != "" {
		metadata["concurrencyPolicy"] = string(cronJob.Spec.ConcurrencyPolicy)
	}
	if cronJob.Spec.StartingDeadlineSeconds != nil {
		metadata["startingDeadlineSeconds"] = strconv.FormatInt(*cronJob.Spec.StartingDeadlineSeconds, 10)
	}
	metadataJson, _ := json.Marshal(metadata)

	allTags := []string{
		"kubernetes",
		ValidateTagName(fmt.Sprintf("kubernetes-namespace:%s", cronJob.Namespace)),
	}
	for _, tag := range configParser.GetTags() {
		allTags = append(allTags, ValidateTagName(tag))
	}

	cronitorJob := CronitorJob{
		Key:      configParser.GetCronitorID(),
		Name:     truncateName(configParser.GetCronitorName()),
		Note:     configParser.GetNote(),
		Schedule: cronJob.Spec.Schedule,
		Timezone: configParser.GetTimezone(),
		Metadata: string(metadataJson),
		Type_:    "job",
		Tags:     allTags,
		Notify:   configParser.GetNotify(),
		Group:    configParser.GetGroup(),
	}

	if graceSeconds := configParser.GetGraceSeconds(); graceSeconds != -1 {
		cronitorJob.GraceSeconds = graceSeconds
	}

	if metricDuration := configParser.GetMetricDuration(); metricDuration != "" {
		cronitorJob.Assertions = parseMetricDurationAssertions(metricDuration)
	}

	return cronitorJob
}

func parseMetricDurationAssertions(value string) []string {
	var assertions []string
	for _, part := range strings.Split(value, ",") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		assertions = append(assertions, "metric.duration "+part)
	}
	return assertions
}

func convertCronJobsToCronitorJobs(jobs []*v1.CronJob) []CronitorJob {
	outputList := make([]CronitorJob, 0, len(jobs))
	for _, job := range jobs {
		outputList = append(outputList, convertCronJobToCronitorJob(job))
	}
	return outputList
}

package api

import (
	"fmt"
	"k8s.io/api/batch/v1beta1"
	"strconv"
)

/*
  Currently using pieces of the Cronitor CLI as POC, writing extra code as needed.
  Longer term: combine with Cronitor CLI?
*/

// https://docs.google.com/document/d/1erh-fvTkF14jyJGv3DYuN2UalWe6AN49XOUsWHJccso/edit#heading=h.ylm2gai335jy
type CronitorJob struct {
	Key         string            `json:"key"`
	DefaultName string            `json:"defaultName"`
	Name        string            `json:"name,omitempty"`
	DefaultNote string            `json:"defaultNote"`
	Metadata    map[string]string `json:"metadata"`
	Type_       string            `json:"type"` // 'job'
	Schedule    string            `json:"schedule"`
	Tags        []string          `json:"tags,omitempty"`
	Rules       []string          `json:"rules"`
}

func convertCronJobToCronitorJob(job *v1beta1.CronJob) CronitorJob {
	name := fmt.Sprintf(`%s/%s`, job.Namespace, job.Name)
	metadata := make(map[string]string)
	if job.Spec.ConcurrencyPolicy != "" {
		metadata["concurrencyPolicy"] = string(job.Spec.ConcurrencyPolicy)
	}
	if job.Spec.StartingDeadlineSeconds != nil {
		metadata["startingDeadlineSeconds"] = strconv.FormatInt(*job.Spec.StartingDeadlineSeconds, 10)
	}
	cronitorJob := CronitorJob{
		Key: string(job.UID),
		// Providing defaultName isn't sufficient; "name" is required as well
		Name:        name,
		DefaultName: name,
		Schedule:    job.Spec.Schedule,
		Metadata:    metadata,
		Type_:       "job",
		Tags:        []string{"kubernetes"},
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

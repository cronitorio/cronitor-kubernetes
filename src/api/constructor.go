package api

import (
	//"github.com/cronitorio/cronitor-cli/lib"
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
	key         string
	defaultName string
	defaultNote string
	metadata    map[string]string
	type_       string // 'job'
	schedule    string
	tags        []string
}

func convertCronJobToMonitors(job *v1beta1.CronJob) CronitorJob {
	name := fmt.Sprintf(`%s/%s`, job.Namespace, job.Name)
	cronitorJob := CronitorJob{
		key:         string(job.UID),
		defaultName: name,
		schedule:    job.Spec.Schedule,
		metadata: map[string]string{
			"concurrencyPolicy":       string(job.Spec.ConcurrencyPolicy),
			"startingDeadlineSeconds": strconv.FormatInt(*job.Spec.StartingDeadlineSeconds, 10),
		},
		type_: "job",
		tags:  []string{"kubernetes"},
	}

	return cronitorJob
}

func convertCronJobsToMonitors(jobs []*v1beta1.CronJob) []CronitorJob {
	outputList := make([]CronitorJob, len(jobs))
	for _, job := range jobs {
		outputList = append(outputList, convertCronJobToMonitors(job))
	}
	return outputList
}

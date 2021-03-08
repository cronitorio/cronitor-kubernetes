package pkg

import (
	"fmt"
	"k8s.io/api/batch/v1beta1"
	"os"
	"strconv"
	"strings"
)

type defaultBehaviorValue string

const (
	defaultBehaviorInclude      defaultBehaviorValue = "include"
	defaultBehaviorExclude      defaultBehaviorValue = "exclude"
	defaultBehaviorNoneProvided defaultBehaviorValue = ""
)

type CronitorAnnotation string

const (
	// AnnotationInclude is the key of the annotation that explicitly
	// controls whether a CronJob will be monitored by Cronitor.
	// Only required when the Cronitor agent is not set to automatically monitor
	// all CronJobs.
	// The only valid values are "true" and "false". Default is "false".
	AnnotationInclude CronitorAnnotation = "k8s.cronitor.io/include"

	// AnnotationExclude is the key of the annotation that explicitly
	// controls whether a CronJob will be monitored by Cronitor.
	// Only required when the Cronitor agent is set to require manual
	// selection of CronJobs to monitor.
	// The only valid values are "true" and "false". Default is "false".
	AnnotationExclude CronitorAnnotation = "k8s.cronitor.io/exclude"

	// AnnotationEnvironment is the environment name that should be sent to Cronitor
	// for the CronJob.
	// Optional. Overrides the default chart-wide environment if present.
	AnnotationEnvironment CronitorAnnotation = "k8s.cronitor.io/env"

	// AnnotationTags is a comma-separated list of Cronitor tags for the CronJob.
	// Optional. Appends to any chart-wide tags.
	AnnotationTags CronitorAnnotation = "k8s.cronitor.io/tags"

	// AnnotationCronitorID is a pre-existing Cronitor monitor ID, for use if you are
	// having the Cronitor agent watch some CronJobs that are already present in Cronitor
	// via manual instrumentation, and you'd like to use the same Monitor object.
	AnnotationCronitorID CronitorAnnotation = "k8s.cronitor.io/cronitor-id"
)

type CronitorConfigParser struct {
	cronjob *v1beta1.CronJob
}

func NewCronitorConfigParser(cronjob *v1beta1.CronJob) CronitorConfigParser {
	return CronitorConfigParser{
		cronjob: cronjob,
	}
}

func (cronitorParser CronitorConfigParser) GetEnvironment() string {
	if env, ok := cronitorParser.cronjob.Annotations[string(AnnotationEnvironment)]; ok && env != "" {
		return env
	}
	if defaultEnvironment := os.Getenv("DEFAULT_ENV"); defaultEnvironment != "" {
		return defaultEnvironment
	}
	return ""
}

func (cronitorParser CronitorConfigParser) GetTags() []string {
	var tagList []string

	// Get tags from Helm chart (via the environment)
	if stringEnvTagList := os.Getenv("TAGS"); stringEnvTagList != "" {
		for _, value := range strings.Split(stringEnvTagList, ",") {
			tagList = append(tagList, value)
		}
	}

	// Get tags from CronJob annotations
	if stringTagList, ok := cronitorParser.cronjob.Annotations[string(AnnotationTags)]; ok && stringTagList != "" {
		for _, value := range strings.Split(stringTagList, ",") {
			tagList = append(tagList, value)
		}
	}

	return tagList
}

func (cronitorParser CronitorConfigParser) GetCronitorID() string {
	if assignedId, ok := cronitorParser.cronjob.Annotations[string(AnnotationCronitorID)]; ok && assignedId != "" {
		return assignedId
	}

	return ""
}

// Inclusion/exclusion behavior

func (cronitorParser CronitorConfigParser) getDefaultBehavior() defaultBehaviorValue {
	defaultBehavior := defaultBehaviorValue(os.Getenv("DEFAULT_BEHAVIOR"))
	if defaultBehavior == defaultBehaviorNoneProvided {
		defaultBehavior = defaultBehaviorInclude
	}
	return defaultBehavior
}

func (cronitorParser CronitorConfigParser) IsCronJobIncluded() (bool, error) {
	cronjob := cronitorParser.cronjob
	defaultBehavior := cronitorParser.getDefaultBehavior()

	switch defaultBehavior {
	case defaultBehaviorExclude:
		raw, ok := cronjob.Annotations[string(AnnotationInclude)]
		// Default if not present in this scenario is to exclude
		if !ok {
			return false, nil
		}
		return strconv.ParseBool(raw)
	case defaultBehaviorInclude:
		raw, ok := cronjob.Annotations[string(AnnotationExclude)]
		// Default if not present in this scenario is to include
		if !ok {
			return true, nil
		}
		returnBool, err := strconv.ParseBool(raw)
		return !returnBool, err
	default:
		return false, fmt.Errorf("invalid DEFAULT_BEHAVIOR value of \"%s\" provided", defaultBehavior)
	}
}

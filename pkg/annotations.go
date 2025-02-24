package pkg

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	v1 "k8s.io/api/batch/v1"
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
	// Overrides the default chart-wide environment if present.
	AnnotationEnvironment CronitorAnnotation = "k8s.cronitor.io/env"

	// AnnotationTags is a comma-separated list of Cronitor tags for the CronJob.
	// Appends to any chart-wide tags.
	AnnotationTags CronitorAnnotation = "k8s.cronitor.io/tags"

	// AnnotationCronitorID is a pre-existing Cronitor monitor key, for use if you are
	// having the Cronitor agent watch some CronJobs that are already present in Cronitor
	// via manual instrumentation, and you'd like to use the same Monitor object.
	AnnotationCronitorID CronitorAnnotation = "k8s.cronitor.io/cronitor-id"

	// AnnotationCronitorName lets you override the Name created by the agent to
	// create the monitor in Cronitor with a custom specified name. This is especially useful
	// if you are attaching the same CronJob across multiple namespaces/clusters to a single
	// Cronitor Monitor across multiple environments.
	AnnotationCronitorName CronitorAnnotation = "k8s.cronitor.io/cronitor-name"

	// AnnotationCronitorGroup lets you provide the key for a Group within the Cronitor application.
	// This is useful if you want to organize monitors within the Cronitor application as they are first created.
	AnnotationCronitorGroup CronitorAnnotation = "k8s.cronitor.io/cronitor-group"

	// AnnotationCronitorNotify lets you provide a comma-separated list of Notification List
	// keys (https://cronitor.io/app/settings/alerts) to be used for dispatching alerts when a job fails/recovers.
	AnnotationCronitorNotify CronitorAnnotation = "k8s.cronitor.io/cronitor-notify"

	// AnnotationCronitorGraceSeconds lets you provide the number of seconds to wait before sending a failure alert.
	AnnotationCronitorGraceSeconds CronitorAnnotation = "k8s.cronitor.io/cronitor-grace-seconds"

	// AnnotationIDInference lets you decide how the Cronitor ID is determined.
	// The only valid values are "k8s" and "name". Default is "k8s".
	AnnotationIDInference CronitorAnnotation = "k8s.cronitor.io/id-inference"

	// AnnotationNamePrefix lets you control the prefix of the name.
	// Valid options include "none", "namespace" (to prepend namespace/), or any other string, which will be prepended as-is.
	AnnotationNamePrefix CronitorAnnotation = "k8s.cronitor.io/name-prefix"

	// AnnotationAutoComplete lets you control the automatic completion telemetry for a CronJob.
	// The only valid values are "true" and "false".
	AnnotationAutoComplete CronitorAnnotation = "k8s.cronitor.io/auto-complete"
)

type CronitorConfigParser struct {
	cronjob *v1.CronJob
}

func NewCronitorConfigParser(cronjob *v1.CronJob) CronitorConfigParser {
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

func (cronitorParser CronitorConfigParser) GetSchedule() string {
	return cronitorParser.cronjob.Spec.Schedule
}

func (cronitorParser CronitorConfigParser) GetTags() []string {
	var tagList []string

	// Get tags from Helm chart (via the environment)
	if stringEnvTagList := os.Getenv("TAGS"); stringEnvTagList != "" {
		for _, value := range strings.Split(stringEnvTagList, ",") {
			tagList = append(tagList, strings.TrimSpace(value))
		}
	}

	// Get tags from CronJob annotations
	if stringTagList, ok := cronitorParser.cronjob.Annotations[string(AnnotationTags)]; ok && stringTagList != "" {
		for _, value := range strings.Split(stringTagList, ",") {
			tagList = append(tagList, strings.TrimSpace(value))
		}
	}

	return tagList
}

// GetSpecifiedCronitorID returns the pre-specified Cronitor monitor ID, if provided as an annotation
// on the CronJob object. If not provided, returns an empty string
func (cronitorParser CronitorConfigParser) GetSpecifiedCronitorID() string {
	if assignedId, ok := cronitorParser.cronjob.Annotations[string(AnnotationCronitorID)]; ok && assignedId != "" {
		return assignedId
	}

	return ""
}

// GetCronitorID returns the correct Cronitor monitor ID for the CronJob.
// Defaults to the CronJob's Kubernetes UID if no pre-specified monitor ID is provided by the user.
func (cronitorParser CronitorConfigParser) GetCronitorID() string {
	// Default behavior
	inference := "k8s"

	// Check if a specific Cronitor ID is assigned and return it if present
	if specifiedId := cronitorParser.GetSpecifiedCronitorID(); specifiedId != "" {
		return specifiedId
	}

	// Override default inference if an annotation is provided
	if annotation, ok := cronitorParser.cronjob.Annotations[string(AnnotationIDInference)]; ok && annotation != "" {
		inference = annotation
	}

	// Return the appropriate ID based on the inference
	switch inference {
	case "name":
		return generateHashFromName(cronitorParser.GetCronitorName())
	default:
		return string(cronitorParser.cronjob.GetUID())
	}
}

// GetSpecifiedCronitorName returns the pre-specified Cronitor monitor name, if provided as an annotation
// on the CronJob object. If not provided, returns an empty string
func (cronitorParser CronitorConfigParser) GetSpecifiedCronitorName() string {
	if assignedName, ok := cronitorParser.cronjob.Annotations[string(AnnotationCronitorName)]; ok && assignedName != "" {
		return assignedName
	}

	return ""
}

// GetCronitorName returns the name to be used by Cronitor monitor.
// Allows the namespace to be prepended or not, and allows arbitrary strings as a prefix.
// Defaults to prepending the namespace if no pre-specified monitor name is provided by the user.
func (cronitorParser CronitorConfigParser) GetCronitorName() string {
	// Default behavior
	prefix := "namespace"

	// Check if a specific Cronitor Name is assigned and return it if present
	if specifiedName := cronitorParser.GetSpecifiedCronitorName(); specifiedName != "" {
		return specifiedName
	}

	// Check if a prefix annotation exists and override the default if present
	if annotation, ok := cronitorParser.cronjob.Annotations[string(AnnotationNamePrefix)]; ok && annotation != "" {
		prefix = annotation
	}

	// Construct the name based on the prefix
	switch prefix {
	case "namespace":
		return fmt.Sprintf("%s/%s", cronitorParser.cronjob.Namespace, cronitorParser.cronjob.Name)
	case "none":
		return cronitorParser.cronjob.Name
	default:
		return fmt.Sprintf("%s%s", prefix, cronitorParser.cronjob.Name)
	}
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

func (cronitorParser CronitorConfigParser) GetNotify() []string {
	var notifications []string

	if stringNotificationList, ok := cronitorParser.cronjob.Annotations[string(AnnotationCronitorNotify)]; ok {
		for _, value := range strings.Split(stringNotificationList, ",") {
			notifications = append(notifications, strings.TrimSpace(value))
		}
	}
	return notifications
}

func (cronitorParser CronitorConfigParser) GetGroup() string {
	if group, ok := cronitorParser.cronjob.Annotations[string(AnnotationCronitorGroup)]; ok {
		return group
	}
	return ""
}

func (cronitorParser CronitorConfigParser) GetGraceSeconds() int {
	if graceSeconds, ok := cronitorParser.cronjob.Annotations[string(AnnotationCronitorGraceSeconds)]; ok {
		graceSecondsInt, err := strconv.Atoi(graceSeconds)
		if err != nil {
			return -1
		}
		return graceSecondsInt
	}
	return -1
}

func (cronitorParser CronitorConfigParser) EnabledAutoComplete() (bool, error) {
	cronjob := cronitorParser.cronjob
	if raw, ok := cronjob.Annotations[string(AnnotationAutoComplete)]; ok {
		return strconv.ParseBool(raw)
	}

	return false, nil
}

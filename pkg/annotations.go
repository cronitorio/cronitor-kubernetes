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

	// AnnotationKey is a pre-existing Cronitor monitor key, for use if you are
	// having the Cronitor agent watch some CronJobs that are already present in Cronitor
	// via manual instrumentation, and you'd like to use the same Monitor object.
	AnnotationKey CronitorAnnotation = "k8s.cronitor.io/key"

	// AnnotationCronitorID is the legacy annotation for AnnotationKey.
	// Deprecated: Use AnnotationKey instead.
	AnnotationCronitorID CronitorAnnotation = "k8s.cronitor.io/cronitor-id"

	// AnnotationName lets you override the Name created by the agent to
	// create the monitor in Cronitor with a custom specified name. This is especially useful
	// if you are attaching the same CronJob across multiple namespaces/clusters to a single
	// Cronitor Monitor across multiple environments.
	AnnotationName CronitorAnnotation = "k8s.cronitor.io/name"

	// AnnotationCronitorName is the legacy annotation for AnnotationName.
	// Deprecated: Use AnnotationName instead.
	AnnotationCronitorName CronitorAnnotation = "k8s.cronitor.io/cronitor-name"

	// AnnotationGroup lets you provide the key for a Group within the Cronitor application.
	// This is useful if you want to organize monitors within the Cronitor application as they are first created.
	AnnotationGroup CronitorAnnotation = "k8s.cronitor.io/group"

	// AnnotationCronitorGroup is the legacy annotation for AnnotationGroup.
	// Deprecated: Use AnnotationGroup instead.
	AnnotationCronitorGroup CronitorAnnotation = "k8s.cronitor.io/cronitor-group"

	// AnnotationNotify lets you provide a comma-separated list of Notification List
	// keys (https://cronitor.io/app/settings/alerts) to be used for dispatching alerts when a job fails/recovers.
	AnnotationNotify CronitorAnnotation = "k8s.cronitor.io/notify"

	// AnnotationCronitorNotify is the legacy annotation for AnnotationNotify.
	// Deprecated: Use AnnotationNotify instead.
	AnnotationCronitorNotify CronitorAnnotation = "k8s.cronitor.io/cronitor-notify"

	// AnnotationGraceSeconds lets you provide the number of seconds to wait before sending a failure alert.
	AnnotationGraceSeconds CronitorAnnotation = "k8s.cronitor.io/grace-seconds"

	// AnnotationCronitorGraceSeconds is the legacy annotation for AnnotationGraceSeconds.
	// Deprecated: Use AnnotationGraceSeconds instead.
	AnnotationCronitorGraceSeconds CronitorAnnotation = "k8s.cronitor.io/cronitor-grace-seconds"

	// AnnotationIDInference lets you decide how the Cronitor ID is determined.
	// The only valid values are "k8s" and "name". Default is "k8s".
	AnnotationIDInference CronitorAnnotation = "k8s.cronitor.io/id-inference"

	// AnnotationNamePrefix lets you control the prefix of the name.
	// Valid options include "none", "namespace" (to prepend namespace/), or any other string, which will be prepended as-is.
	AnnotationNamePrefix CronitorAnnotation = "k8s.cronitor.io/name-prefix"

	// AnnotationLogCompleteEvent lets you control whether job completion events are sent as log records instead of state changes.
	// When set to "true", the agent will not send telemetry events with state=complete, but will send a log event recording the completion.
	// This supports async workflows where the actual task completion occurs outside the Kubernetes job.
	// The only valid values are "true" and "false". Default is "false".
	AnnotationLogCompleteEvent CronitorAnnotation = "k8s.cronitor.io/log-complete-event"
)

type CronitorConfigParser struct {
	cronjob *v1.CronJob
}

func NewCronitorConfigParser(cronjob *v1.CronJob) CronitorConfigParser {
	return CronitorConfigParser{
		cronjob: cronjob,
	}
}

// getAnnotationWithFallback retrieves an annotation value, checking the preferred annotation first
// and falling back to the legacy annotation if not found. This provides backwards compatibility
// for users who have the older cronitor- prefixed annotations.
func (cronitorParser CronitorConfigParser) getAnnotationWithFallback(preferred, legacy CronitorAnnotation) (string, bool) {
	// Check preferred annotation first
	if value, ok := cronitorParser.cronjob.Annotations[string(preferred)]; ok {
		return value, true
	}
	// Fall back to legacy annotation
	if value, ok := cronitorParser.cronjob.Annotations[string(legacy)]; ok {
		return value, true
	}
	return "", false
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

// GetTimezone returns the timezone from the CronJob spec if set.
// Returns empty string if no timezone is specified.
func (cronitorParser CronitorConfigParser) GetTimezone() string {
	if cronitorParser.cronjob.Spec.TimeZone != nil {
		return *cronitorParser.cronjob.Spec.TimeZone
	}
	return ""
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

// GetSpecifiedCronitorID returns the pre-specified Cronitor monitor key, if provided as an annotation
// on the CronJob object. If not provided, returns an empty string.
// Supports both k8s.cronitor.io/key (preferred) and k8s.cronitor.io/cronitor-id (legacy).
func (cronitorParser CronitorConfigParser) GetSpecifiedCronitorID() string {
	if assignedId, ok := cronitorParser.getAnnotationWithFallback(AnnotationKey, AnnotationCronitorID); ok && assignedId != "" {
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
// on the CronJob object. If not provided, returns an empty string.
// Supports both k8s.cronitor.io/name (preferred) and k8s.cronitor.io/cronitor-name (legacy).
func (cronitorParser CronitorConfigParser) GetSpecifiedCronitorName() string {
	if assignedName, ok := cronitorParser.getAnnotationWithFallback(AnnotationName, AnnotationCronitorName); ok && assignedName != "" {
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

// GetNotify returns the notification list keys for this CronJob.
// Supports both k8s.cronitor.io/notify (preferred) and k8s.cronitor.io/cronitor-notify (legacy).
func (cronitorParser CronitorConfigParser) GetNotify() []string {
	var notifications []string

	if stringNotificationList, ok := cronitorParser.getAnnotationWithFallback(AnnotationNotify, AnnotationCronitorNotify); ok {
		for _, value := range strings.Split(stringNotificationList, ",") {
			notifications = append(notifications, strings.TrimSpace(value))
		}
	}
	return notifications
}

// GetGroup returns the group key for this CronJob.
// Supports both k8s.cronitor.io/group (preferred) and k8s.cronitor.io/cronitor-group (legacy).
func (cronitorParser CronitorConfigParser) GetGroup() string {
	if group, ok := cronitorParser.getAnnotationWithFallback(AnnotationGroup, AnnotationCronitorGroup); ok {
		return group
	}
	return ""
}

// GetGraceSeconds returns the grace seconds for this CronJob.
// Supports both k8s.cronitor.io/grace-seconds (preferred) and k8s.cronitor.io/cronitor-grace-seconds (legacy).
func (cronitorParser CronitorConfigParser) GetGraceSeconds() int {
	if graceSeconds, ok := cronitorParser.getAnnotationWithFallback(AnnotationGraceSeconds, AnnotationCronitorGraceSeconds); ok {
		graceSecondsInt, err := strconv.Atoi(graceSeconds)
		if err != nil {
			return -1
		}
		return graceSecondsInt
	}
	return -1
}

// LogCompleteEvent determines whether job completion events should be sent as log events (true)
// or as state=complete telemetry (false).
// When log-complete-event annotation is set to "true", returns true to indicate
// completion should be sent as a log event rather than a complete state change.
func (cronitorParser CronitorConfigParser) LogCompleteEvent() (bool, error) {
	cronjob := cronitorParser.cronjob
	if raw, ok := cronjob.Annotations[string(AnnotationLogCompleteEvent)]; ok {
		logCompleteEvent, err := strconv.ParseBool(raw)
		if err != nil {
			return false, err
		}
		return logCompleteEvent, nil
	}

	return false, nil
}

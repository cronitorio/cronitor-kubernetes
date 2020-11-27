package collector

import (
	"fmt"
	"k8s.io/api/batch/v1beta1"
	"os"
	"strconv"
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
)

type CronitorConfigParser struct {
	cronjob *v1beta1.CronJob
}

func NewCronitorConfigParser(cronjob *v1beta1.CronJob) CronitorConfigParser {
	return CronitorConfigParser{
		cronjob: cronjob,
	}
}

func (cronitorParser CronitorConfigParser) getDefaultBehavior() defaultBehaviorValue {
	defaultBehavior := defaultBehaviorValue(os.Getenv("DEFAULT_BEHAVIOR"))
	if defaultBehavior == defaultBehaviorNoneProvided {
		defaultBehavior = defaultBehaviorInclude
	}
	return defaultBehavior
}

func (cronitorParser CronitorConfigParser) included() (bool, error) {
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

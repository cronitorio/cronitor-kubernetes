package normalizer

import (
	v1 "k8s.io/api/batch/v1"
	"k8s.io/api/batch/v1beta1"
)

type VersionedCronJobWrapper struct {
	BatchV1Beta1CronJob *v1beta1.CronJob
	BatchV1CronJob      *v1.CronJob
}

func CronJobConvertV1Beta1ToV1(v1beta1CJ *v1beta1.CronJob) *v1.CronJob {
	newCronJob := new(v1.CronJob)
	newCronJob.TypeMeta = v1beta1CJ.TypeMeta
	newCronJob.ObjectMeta = v1beta1CJ.ObjectMeta
	newCronJob.Spec = v1.CronJobSpec{
		Schedule:                v1beta1CJ.Spec.Schedule,
		TimeZone:                v1beta1CJ.Spec.TimeZone,
		StartingDeadlineSeconds: v1beta1CJ.Spec.StartingDeadlineSeconds,
		ConcurrencyPolicy:       v1.ConcurrencyPolicy(v1beta1CJ.Spec.ConcurrencyPolicy),
		Suspend:                 v1beta1CJ.Spec.Suspend,
		// We can get away with ignoring JobTemplate because every time we want to get Job information,
		// we always fetch the Job itself from Kubernetes instead of using the JobTemplate spec in any way
		// JobTemplate
		SuccessfulJobsHistoryLimit: v1beta1CJ.Spec.SuccessfulJobsHistoryLimit,
		FailedJobsHistoryLimit:     v1beta1CJ.Spec.FailedJobsHistoryLimit,
	}
	newCronJob.Status = v1.CronJobStatus(v1beta1CJ.Status)

	return newCronJob
}

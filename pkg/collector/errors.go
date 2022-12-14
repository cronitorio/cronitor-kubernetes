package collector

type PodNotFoundError struct {
	podNamespace string
	podName      string
	Err          error
}

func (e PodNotFoundError) Error() string {
	return "pod " + e.podNamespace + "/" + e.podName + ": " + e.Err.Error()
}

func (e PodNotFoundError) Unwrap() error {
	return e.Err
}

type JobNotFoundError struct {
	jobNamespace string
	jobName      string
	Err          error
}

func (e JobNotFoundError) Error() string {
	return "job " + e.jobNamespace + "/" + e.jobName + ": " + e.Err.Error()
}

func (e JobNotFoundError) Unwrap() error {
	return e.Err
}

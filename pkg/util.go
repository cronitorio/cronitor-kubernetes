package pkg

import (
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"

	v1 "k8s.io/api/batch/v1"
)

type Annotation struct {
	Key   string
	Value string
}

func CronJobFromAnnotations(annotations []Annotation) (v1.CronJob, error) {
	var annotationParts []string
	for _, a := range annotations {
		annotationParts = append(annotationParts, fmt.Sprintf("\"%s\": \"%s\"", a.Key, a.Value))
	}
	annotationsStr := strings.Join(annotationParts, ", ")

	jsonBlob := fmt.Sprintf(`{
		"apiVersion": "batch/v1beta1",
		"kind": "CronJob",
		"metadata": {
			"name": "test-cronjob",
			"namespace": "default",
			"annotations": {%s}
		},
		"spec": {
			"concurrencyPolicy": "Forbid",
			"jobTemplate": {
				"spec": {
					"backoffLimit": 3,
					"template": {
						"spec": {
							"containers": [
								{
									"args": [
										"/bin/sh",
										"-c",
										"date ; sleep 5 ; echo Hello from k8s"
									],
									"image": "busybox",
									"name": "hello"
								}
							],
							"restartPolicy": "OnFailure"
						}
					}
				}
			},
			"schedule": "*/1 * * * *"
		}
	}`, annotationsStr)

	var cronJob v1.CronJob
	err := json.Unmarshal([]byte(jsonBlob), &cronJob)

	return cronJob, err
}

func generateHashFromName(name string) string {
	h := sha1.New()
	h.Write([]byte(name))
	bs := h.Sum(nil)
	return hex.EncodeToString(bs)
}

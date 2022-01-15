package pkg

import (
	"github.com/Masterminds/semver"
	"testing"
)

func TestChartYamlParsing(t *testing.T) {
	const ChartData = `ApiVersion: v1
Entries:
  cronitor-kubernetes:
  - ApiVersion: v1
    AppVersion: 0.1.11
    created: "2021-12-31T05:43:35.418290744Z"
    description: 'Helm chart to deploy the Cronitor Kubernetes agent to automatically
      monitor your CronJobs '
    digest: 1cfc9bfcd763f6070bc2cc652dcf9dee0bf1e73792eae402a8f4ec506a0c0723
    name: cronitor-kubernetes
    urls:
    - https://github.com/cronitorio/cronitor-kubernetes/releases/download/cronitor-kubernetes-0.1.15/cronitor-kubernetes-0.1.15.tgz
    version: 0.1.15
  - ApiVersion: v1
    AppVersion: 0.1.10
    created: "2021-10-26T05:03:25.162018558Z"
    description: 'Helm chart to deploy the Cronitor Kubernetes agent to automatically
      monitor your CronJobs '
    digest: 813b8dbb0c0d532d89e3a195b345151ba1a8c286292ba53c6cb71c3b950df933
    name: cronitor-kubernetes
    urls:
    - https://github.com/cronitorio/cronitor-kubernetes/releases/download/cronitor-kubernetes-0.1.14/cronitor-kubernetes-0.1.14.tgz
    version: 0.1.14
  - ApiVersion: v1
    AppVersion: 0.1.9
    created: "2021-10-24T03:58:29.222208132Z"
    description: 'Helm chart to deploy the Cronitor Kubernetes agent to automatically
      monitor your CronJobs '
    digest: d61288ec71995e669c4ecef9718ff1c4d9ca6170f4d63dc7a65bfba36a196a64
    name: cronitor-kubernetes
    urls:
    - https://github.com/cronitorio/cronitor-kubernetes/releases/download/cronitor-kubernetes-0.1.13/cronitor-kubernetes-0.1.13.tgz
    version: 0.1.13
  - ApiVersion: v1
    AppVersion: 0.1.8
    created: "2021-10-24T01:03:56.999932784Z"
    description: 'Helm chart to deploy the Cronitor Kubernetes agent to automatically
      monitor your CronJobs '
    digest: 0fdd1750ce82969ae53b71fd5eb0cde7877f002998c483e2fdf31e8757d2a786
    name: cronitor-kubernetes
    urls:
    - https://github.com/cronitorio/cronitor-kubernetes/releases/download/cronitor-kubernetes-0.1.12/cronitor-kubernetes-0.1.12.tgz
    version: 0.1.12
  - ApiVersion: v1
    AppVersion: 0.1.7
    created: "2021-10-24T00:47:35.39004398Z"
    description: 'Helm chart to deploy the Cronitor Kubernetes agent to automatically
      monitor your CronJobs '
    digest: ada05ea52193b5dbfe682902bade8f922409349fc7b949d5ab3cbb2e94f1ddc5
    name: cronitor-kubernetes
    urls:
    - https://github.com/cronitorio/cronitor-kubernetes/releases/download/cronitor-kubernetes-0.1.11/cronitor-kubernetes-0.1.11.tgz
    version: 0.1.11
  - ApiVersion: v1
    AppVersion: 0.1.6
    created: "2021-10-24T00:34:59.937086271Z"
    description: 'Helm chart to deploy the Cronitor Kubernetes agent to automatically
      monitor your CronJobs '
    digest: ec54d1b154ff584e7b2650f5ec93540ec16a8aab4407f6c5299b13a0b52e41d7
    name: cronitor-kubernetes
    urls:
    - https://github.com/cronitorio/cronitor-kubernetes/releases/download/cronitor-kubernetes-0.1.10/cronitor-kubernetes-0.1.10.tgz
    version: 0.1.10
  - ApiVersion: v1
    AppVersion: 0.1.6
    created: "2021-10-24T00:22:55.32821691Z"
    description: 'Helm chart to deploy the Cronitor Kubernetes agent to automatically
      monitor your CronJobs '
    digest: 72452bb0bc1653781cf21de6c9b09a03691740b01dca146cbbd86c862ca1bc55
    name: cronitor-kubernetes
    urls:
    - https://github.com/cronitorio/cronitor-kubernetes/releases/download/cronitor-kubernetes-0.1.9/cronitor-kubernetes-0.1.9.tgz
    version: 0.1.9
  - ApiVersion: v1
    AppVersion: 0.1.5
    created: "2021-10-22T18:59:50.21764758Z"
    description: 'Helm chart to deploy the Cronitor Kubernetes agent to automatically
      monitor your CronJobs '
    digest: dfba6f0a90d475f6a99a9843bffe7eb350a8b1088b27ce7e825b5abb046f0939
    name: cronitor-kubernetes
    urls:
    - https://github.com/cronitorio/cronitor-kubernetes/releases/download/cronitor-kubernetes-0.1.8/cronitor-kubernetes-0.1.8.tgz
    version: 0.1.8
  - ApiVersion: v1
    AppVersion: 0.1.3
    created: "2021-10-05T01:42:48.941378093Z"
    description: 'Helm chart to deploy the Cronitor Kubernetes agent to automatically
      monitor your CronJobs '
    digest: b7c974df715e5ef1eae6cccedeb079c4d3e0f0fd2f78fd72444a094776c35fde
    name: cronitor-kubernetes
    urls:
    - https://github.com/cronitorio/cronitor-kubernetes/releases/download/cronitor-kubernetes-0.1.6/cronitor-kubernetes-0.1.6.tgz
    version: 0.1.6
  - ApiVersion: v1
    AppVersion: 0.1.2
    created: "2021-08-12T21:31:57.551246314Z"
    description: 'Helm chart to deploy the Cronitor Kubernetes agent to automatically
      monitor your CronJobs '
    digest: 834c88677ba353bb712e0354156087f43cd0eb660c1c16adb1e2b773caf5a8d9
    name: cronitor-kubernetes
    urls:
    - https://github.com/cronitorio/cronitor-kubernetes/releases/download/cronitor-kubernetes-0.1.5/cronitor-kubernetes-0.1.5.tgz
    version: 0.1.5
  - ApiVersion: v1
    AppVersion: 0.1.1
    created: "2021-08-12T21:21:33.800390915Z"
    description: 'Helm chart to deploy the Cronitor Kubernetes agent to automatically
      monitor your CronJobs '
    digest: d83a3af6582f0f7e8deb05b83229de2547f5bb709928bfbd4a0464ea8fc8566e
    name: cronitor-kubernetes
    urls:
    - https://github.com/cronitorio/cronitor-kubernetes/releases/download/cronitor-kubernetes-0.1.4/cronitor-kubernetes-0.1.4.tgz
    version: 0.1.4
  - ApiVersion: v1
    AppVersion: 0.1-alpha
    created: "2021-03-19T22:43:37.251440411Z"
    description: 'Helm chart to deploy the Cronitor Kubernetes agent to automatically
      monitor your CronJobs '
    digest: 2dc6808cc26da00c0a84d7f089e5d103011e7018ee48a013a62e8a6e63aa85fb
    name: cronitor-kubernetes
    urls:
    - https://github.com/cronitorio/cronitor-kubernetes/releases/download/cronitor-kubernetes-0.1.3/cronitor-kubernetes-0.1.3.tgz
    version: 0.1.3
  - ApiVersion: v1
    AppVersion: 0.1-alpha
    created: "2021-03-19T17:39:20.598709125Z"
    description: 'Helm chart to deploy the Cronitor Kubernetes agent to automatically
      monitor your CronJobs '
    digest: 47831e3f019053623733b5d1b9719edc2edf256189e6eae7605785a81a056801
    name: cronitor-kubernetes
    urls:
    - https://github.com/cronitorio/cronitor-kubernetes/releases/download/cronitor-kubernetes-0.1.2/cronitor-kubernetes-0.1.2.tgz
    version: 0.1.2
  - ApiVersion: v1
    AppVersion: 0.1-alpha
    created: "2021-03-09T06:28:46.24198092Z"
    description: 'Helm chart to deploy the Cronitor Kubernetes agent to automatically
      monitor your CronJobs '
    digest: d84a693ad56297a6bdde5419551d95c6676bc51efac911b88f5f9d865c6b0591
    name: cronitor-kubernetes
    urls:
    - https://github.com/cronitorio/cronitor-kubernetes/releases/download/cronitor-kubernetes-0.1.1/cronitor-kubernetes-0.1.1.tgz
    version: 0.1.1
generated: "2021-12-31T05:43:35.418728161Z"`

	chartBytes := []byte(ChartData)
	versions, err := extractVersionsFromChart(chartBytes)
	if err != nil {
		t.Error(err)
	}
	latestVersion := extractLatestVersionFromList(versions)
	if latestVersion != "0.1.11" {
		t.Errorf(`Unexpected latest version, received "%s"`, latestVersion)
	}
}

func TestVersionConstraintPreRelease(t *testing.T) {
	currentVersionString := "0.1.12"
	latestVersionInHelmString := "0.1.11"

	latestVersion, err := semver.NewVersion(latestVersionInHelmString)
	if err != nil {
		t.Error(err)
	}
	constraint, err := semver.NewConstraint("> " + currentVersionString)
	if constraint.Check(latestVersion) {
		t.Errorf("Version checking mismatch")
	}
}

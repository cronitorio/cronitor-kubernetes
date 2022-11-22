module github.com/cronitorio/cronitor-kubernetes

go 1.15

replace github.com/cronitorio/cronitor-cli => github.com/jdotjdot/cronitor-cli v0.0.0-20201122001207-ff47a8dfbadf

require (
	github.com/Masterminds/semver v1.5.0
	github.com/aquilax/truncate v1.0.0
	github.com/cronitorio/cronitor-cli v0.0.0-00010101000000-000000000000
	github.com/getsentry/sentry-go v0.12.0
	github.com/ghodss/yaml v1.0.0
	github.com/google/gofuzz v1.2.0 // indirect
	github.com/googleapis/gnostic v0.5.3 // indirect
	github.com/hashicorp/golang-lru v0.5.4 // indirect
	github.com/imdario/mergo v0.3.8 // indirect
	github.com/pkg/errors v0.9.1
	github.com/sirupsen/logrus v1.4.2
	github.com/spf13/cobra v0.0.6
	github.com/spf13/viper v1.6.2
	google.golang.org/appengine v1.6.7 // indirect
	k8s.io/api v0.25.3
	k8s.io/apimachinery v0.25.3
	k8s.io/client-go v0.25.3
)

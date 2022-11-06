package collector

import (
	"fmt"
	"k8s.io/apimachinery/pkg/version"
	"testing"
)

func TestServerVersionCompare(t *testing.T) {
	serverVersionInfo := version.Info{
		Major: "1",
		Minor: "25",
	}

	coll := CronJobCollection{serverVersion: &serverVersionInfo}
	result, err := coll.CompareServerVersion(1, 24)
	if err != nil {
		t.Error(err)
		t.Fail()
	}

	if result != 1 {
		t.Errorf("With server version 1.25 and compared version 1.24, result was not 1 but %d", result)
	}
}

func TestRightBatchApiVersion(t *testing.T) {
	tests := []struct {
		Version            version.Info
		ExpectedApiVersion string
	}{
		{version.Info{Major: "1", Minor: "25"}, "v1"},
		{version.Info{Major: "1", Minor: "24"}, "v1"},
		{version.Info{Major: "1", Minor: "23"}, "v1beta1"},
		{version.Info{Major: "1", Minor: "22"}, "v1beta1"},
	}

	for _, s := range tests {
		t.Run(fmt.Sprintf("Server version %s.%s", s.Version.Major, s.Version.Minor), func(t *testing.T) {
			coll := CronJobCollection{serverVersion: &s.Version}
			version, err := coll.GetPreferredBatchApiVersion()
			if err != nil {
				t.Error(err)
			}
			if version != s.ExpectedApiVersion {
				t.Errorf("for server %s.%s expecting %s got %s", s.Version.Major, s.Version.Major, s.ExpectedApiVersion, version)
			}
		})
	}
}

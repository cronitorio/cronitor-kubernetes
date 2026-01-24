package collector

import (
	"fmt"
	"testing"

	"k8s.io/apimachinery/pkg/version"
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
		{version.Info{Major: "1", Minor: "25+"}, "v1"},
		{version.Info{Major: "1", Minor: "24"}, "v1"},
		{version.Info{Major: "1", Minor: "23"}, "v1beta1"},
		{version.Info{Major: "1", Minor: "22"}, "v1beta1"},
		{version.Info{Major: "1", Minor: "20+"}, "v1beta1"},
		{version.Info{Major: "1", Minor: "19.alpha-2"}, "v1beta1"},
	}

	for _, s := range tests {
		t.Run(fmt.Sprintf("Server version %s.%s", s.Version.Major, s.Version.Minor), func(t *testing.T) {
			coll := CronJobCollection{serverVersion: &s.Version}
			version, err := coll.GetPreferredBatchApiVersion()
			if err != nil {
				t.Error(err)
			}
			if version != s.ExpectedApiVersion {
				t.Errorf("for server %s.%s expecting %s got %s", s.Version.Major, s.Version.Minor, s.ExpectedApiVersion, version)
			}
		})
	}
}

func TestStopWatchingAll_WhenNotStarted(t *testing.T) {
	// StopWatchingAll should not panic when stopper is nil
	// This tests the nil check we added
	coll := &CronJobCollection{
		stopper: nil,
	}

	// This should not panic - just log a warning and return
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("StopWatchingAll panicked when stopper was nil: %v", r)
		}
	}()

	coll.StopWatchingAll()

	// Verify stopper is still nil (wasn't set to something else)
	if coll.stopper != nil {
		t.Error("stopper should remain nil after StopWatchingAll on unstarted collection")
	}
}

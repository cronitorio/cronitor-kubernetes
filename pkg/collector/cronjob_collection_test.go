package collector

import (
	"fmt"
	"testing"

	"k8s.io/apimachinery/pkg/version"
	apiWatch "k8s.io/apimachinery/pkg/watch"
)

// stopCountWatch records Stop calls and panics if Stop closes its channel twice.
type stopCountWatch struct {
	stops int
	ch    chan apiWatch.Event
}

func (c *stopCountWatch) Stop() {
	c.stops++
	close(c.ch)
}

func (c *stopCountWatch) ResultChan() <-chan apiWatch.Event {
	return c.ch
}

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

func TestStopWatchingAllNilNeverStartedOrStoppedDoesNotPanic(t *testing.T) {
	mustNotPanic := func(name string, fn func()) {
		t.Helper()
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("%s panicked: %v", name, r)
			}
		}()
		fn()
	}

	mustNotPanic("nil collection", func() {
		var coll *CronJobCollection
		coll.StopWatchingAll()
	})

	mustNotPanic("never started", func() {
		coll := &CronJobCollection{}
		coll.StopWatchingAll()
		coll.StopWatchingAll()
		if coll.stopper != nil {
			t.Error("stopper should remain nil after StopWatchingAll on an unstarted collection")
		}
	})

	mustNotPanic("nil watcher fields", func() {
		// Watcher was never given a stop channel, informer, or jobs watcher.
		watcher := &CronJobWatcher{}
		coll := &CronJobCollection{stopper: func() { watcher.StopWatching() }}
		coll.StopWatchingAll()
		coll.StopWatchingAll()
		watcher.StopWatching()
	})

	mustNotPanic("already stopped", func() {
		events := make(chan apiWatch.Event)
		counter := &stopCountWatch{ch: events}
		watcher := &CronJobWatcher{
			stopper:     make(chan struct{}),
			jobsWatcher: &WatchWrapper{watcher: counter},
		}
		coll := &CronJobCollection{stopper: func() { watcher.StopWatching() }}
		coll.StopWatchingAll()
		// Collection stopper is cleared, and the watcher channel is already closed.
		coll.StopWatchingAll()
		watcher.StopWatching()
		select {
		case <-watcher.stopper:
		default:
			t.Error("stop channel should be closed after the first StopWatchingAll")
		}
		if counter.stops != 1 {
			t.Errorf("watch interface Stop called %d times, want 1", counter.stops)
		}
	})
}

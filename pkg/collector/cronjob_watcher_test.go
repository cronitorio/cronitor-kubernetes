package collector

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/cronitorio/cronitor-kubernetes/pkg/api"
	"github.com/spf13/viper"
	v1 "k8s.io/api/batch/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

// mockAPIServer creates a test server that tracks API calls
type mockAPIServer struct {
	server       *httptest.Server
	requestCount int
	lastBody     string
	mu           sync.Mutex
}

func newMockAPIServer() *mockAPIServer {
	m := &mockAPIServer{}
	m.server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		m.mu.Lock()
		defer m.mu.Unlock()
		m.requestCount++

		// Read and store the body
		var body []byte
		if r.Body != nil {
			body = make([]byte, r.ContentLength)
			r.Body.Read(body)
			m.lastBody = string(body)
		}

		w.WriteHeader(http.StatusOK)
		// Return empty array for monitors response
		w.Write([]byte("[]"))
	}))
	return m
}

func (m *mockAPIServer) getRequestCount() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.requestCount
}

func (m *mockAPIServer) close() {
	m.server.Close()
}

// createTestCollection creates a minimal CronJobCollection for testing
func createTestCollection(mockServer *mockAPIServer) CronJobCollection {
	// Set the hostname override to use our mock server
	viper.Set("hostname-override", mockServer.server.URL)

	cronitorApi := &api.CronitorApi{
		ApiKey:    "test-key",
		UserAgent: "test-agent",
	}

	return CronJobCollection{
		cronitorApi: cronitorApi,
		cronjobs:    make(map[types.UID]*v1.CronJob),
	}
}

// createTestCronJob creates a test CronJob with the given parameters
func createTestCronJob(name, namespace, uid, schedule string) *v1.CronJob {
	return &v1.CronJob{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			UID:       types.UID(uid),
		},
		Spec: v1.CronJobSpec{
			Schedule: schedule,
		},
	}
}

func TestOnAdd_SkipsWhenAlreadyTracked(t *testing.T) {
	mockServer := newMockAPIServer()
	defer mockServer.close()

	coll := createTestCollection(mockServer)

	// Create a cronjob and pre-add it to the collection (simulating LoadAllExistingCronJobs)
	cronjob := createTestCronJob("test-job", "default", "uid-123", "*/5 * * * *")
	coll.cronjobs[cronjob.GetUID()] = cronjob

	// Call onAdd - should skip because job is already tracked
	onAdd(coll, cronjob)

	// Verify no API call was made
	if mockServer.getRequestCount() != 0 {
		t.Errorf("expected 0 API calls when job is already tracked, got %d", mockServer.getRequestCount())
	}
}

func TestOnAdd_SyncsWhenNotTracked(t *testing.T) {
	mockServer := newMockAPIServer()
	defer mockServer.close()

	coll := createTestCollection(mockServer)

	// Create a new cronjob (not pre-added to collection)
	cronjob := createTestCronJob("new-job", "default", "uid-456", "*/10 * * * *")

	// Call onAdd - should sync because job is not tracked
	onAdd(coll, cronjob)

	// Verify API call was made
	if mockServer.getRequestCount() != 1 {
		t.Errorf("expected 1 API call when job is not tracked, got %d", mockServer.getRequestCount())
	}

	// Verify job is now tracked
	if !coll.IsTracked(cronjob.GetUID()) {
		t.Error("expected job to be tracked after onAdd")
	}
}

func TestOnUpdate_SyncsWhenScheduleChanges(t *testing.T) {
	mockServer := newMockAPIServer()
	defer mockServer.close()

	coll := createTestCollection(mockServer)

	// Create old and new versions of the cronjob
	oldCronjob := createTestCronJob("my-job", "default", "uid-789", "*/5 * * * *")
	newCronjob := createTestCronJob("my-job", "default", "uid-789", "*/10 * * * *") // Schedule changed

	// Pre-add the old cronjob to simulate it being tracked
	coll.cronjobs[oldCronjob.GetUID()] = oldCronjob

	// Call onUpdate with schedule change
	onUpdate(coll, oldCronjob, newCronjob)

	// Verify API call was made to sync the update
	if mockServer.getRequestCount() != 1 {
		t.Errorf("expected 1 API call when schedule changes, got %d", mockServer.getRequestCount())
	}
}

func TestOnUpdate_SkipsWhenScheduleUnchanged(t *testing.T) {
	mockServer := newMockAPIServer()
	defer mockServer.close()

	coll := createTestCollection(mockServer)

	// Create old and new versions with same schedule
	oldCronjob := createTestCronJob("my-job", "default", "uid-789", "*/5 * * * *")
	newCronjob := createTestCronJob("my-job", "default", "uid-789", "*/5 * * * *") // Same schedule

	// Pre-add the old cronjob
	coll.cronjobs[oldCronjob.GetUID()] = oldCronjob

	// Call onUpdate with no schedule change
	onUpdate(coll, oldCronjob, newCronjob)

	// Verify no API call was made
	if mockServer.getRequestCount() != 0 {
		t.Errorf("expected 0 API calls when schedule unchanged, got %d", mockServer.getRequestCount())
	}
}

func TestOnUpdate_SyncsWhenTimezoneChanges(t *testing.T) {
	mockServer := newMockAPIServer()
	defer mockServer.close()

	coll := createTestCollection(mockServer)

	oldTimezone := "America/New_York"
	newTimezone := "Europe/London"

	// Create old cronjob with one timezone
	oldCronjob := &v1.CronJob{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "my-job",
			Namespace: "default",
			UID:       "uid-tz-1",
		},
		Spec: v1.CronJobSpec{
			Schedule: "*/5 * * * *",
			TimeZone: &oldTimezone,
		},
	}

	// Create new cronjob with different timezone (same schedule)
	newCronjob := &v1.CronJob{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "my-job",
			Namespace: "default",
			UID:       "uid-tz-1",
		},
		Spec: v1.CronJobSpec{
			Schedule: "*/5 * * * *",
			TimeZone: &newTimezone,
		},
	}

	// Pre-add the old cronjob to simulate it being tracked
	coll.cronjobs[oldCronjob.GetUID()] = oldCronjob

	// Call onUpdate with timezone change
	onUpdate(coll, oldCronjob, newCronjob)

	// Verify API call was made to sync the update
	if mockServer.getRequestCount() != 1 {
		t.Errorf("expected 1 API call when timezone changes, got %d", mockServer.getRequestCount())
	}
}

func TestOnUpdate_SyncsWhenTimezoneAdded(t *testing.T) {
	mockServer := newMockAPIServer()
	defer mockServer.close()

	coll := createTestCollection(mockServer)

	newTimezone := "America/Los_Angeles"

	// Create old cronjob without timezone
	oldCronjob := &v1.CronJob{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "my-job",
			Namespace: "default",
			UID:       "uid-tz-2",
		},
		Spec: v1.CronJobSpec{
			Schedule: "*/5 * * * *",
		},
	}

	// Create new cronjob with timezone added
	newCronjob := &v1.CronJob{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "my-job",
			Namespace: "default",
			UID:       "uid-tz-2",
		},
		Spec: v1.CronJobSpec{
			Schedule: "*/5 * * * *",
			TimeZone: &newTimezone,
		},
	}

	// Pre-add the old cronjob
	coll.cronjobs[oldCronjob.GetUID()] = oldCronjob

	// Call onUpdate with timezone added
	onUpdate(coll, oldCronjob, newCronjob)

	// Verify API call was made
	if mockServer.getRequestCount() != 1 {
		t.Errorf("expected 1 API call when timezone added, got %d", mockServer.getRequestCount())
	}
}

func TestOnUpdate_SyncsWhenTimezoneRemoved(t *testing.T) {
	mockServer := newMockAPIServer()
	defer mockServer.close()

	coll := createTestCollection(mockServer)

	oldTimezone := "Asia/Tokyo"

	// Create old cronjob with timezone
	oldCronjob := &v1.CronJob{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "my-job",
			Namespace: "default",
			UID:       "uid-tz-3",
		},
		Spec: v1.CronJobSpec{
			Schedule: "*/5 * * * *",
			TimeZone: &oldTimezone,
		},
	}

	// Create new cronjob with timezone removed
	newCronjob := &v1.CronJob{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "my-job",
			Namespace: "default",
			UID:       "uid-tz-3",
		},
		Spec: v1.CronJobSpec{
			Schedule: "*/5 * * * *",
		},
	}

	// Pre-add the old cronjob
	coll.cronjobs[oldCronjob.GetUID()] = oldCronjob

	// Call onUpdate with timezone removed
	onUpdate(coll, oldCronjob, newCronjob)

	// Verify API call was made
	if mockServer.getRequestCount() != 1 {
		t.Errorf("expected 1 API call when timezone removed, got %d", mockServer.getRequestCount())
	}
}

func TestOnUpdate_SkipsWhenScheduleAndTimezoneUnchanged(t *testing.T) {
	mockServer := newMockAPIServer()
	defer mockServer.close()

	coll := createTestCollection(mockServer)

	timezone := "America/Chicago"

	// Create old and new versions with same schedule and timezone
	oldCronjob := &v1.CronJob{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "my-job",
			Namespace: "default",
			UID:       "uid-tz-4",
		},
		Spec: v1.CronJobSpec{
			Schedule: "*/5 * * * *",
			TimeZone: &timezone,
		},
	}

	newCronjob := &v1.CronJob{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "my-job",
			Namespace: "default",
			UID:       "uid-tz-4",
		},
		Spec: v1.CronJobSpec{
			Schedule: "*/5 * * * *",
			TimeZone: &timezone,
		},
	}

	// Pre-add the old cronjob
	coll.cronjobs[oldCronjob.GetUID()] = oldCronjob

	// Call onUpdate with no changes
	onUpdate(coll, oldCronjob, newCronjob)

	// Verify no API call was made
	if mockServer.getRequestCount() != 0 {
		t.Errorf("expected 0 API calls when schedule and timezone unchanged, got %d", mockServer.getRequestCount())
	}
}

func TestOnUpdate_SyncsWhenJobBecomesIncluded(t *testing.T) {
	mockServer := newMockAPIServer()
	defer mockServer.close()

	coll := createTestCollection(mockServer)

	// Create old cronjob with exclude annotation
	oldCronjob := &v1.CronJob{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "my-job",
			Namespace: "default",
			UID:       "uid-abc",
			Annotations: map[string]string{
				"k8s.cronitor.io/exclude": "true",
			},
		},
		Spec: v1.CronJobSpec{
			Schedule: "*/5 * * * *",
		},
	}

	// Create new cronjob without exclude annotation (now included)
	newCronjob := &v1.CronJob{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "my-job",
			Namespace: "default",
			UID:       "uid-abc",
			// No exclude annotation
		},
		Spec: v1.CronJobSpec{
			Schedule: "*/5 * * * *",
		},
	}

	// Call onUpdate - job is now included
	onUpdate(coll, oldCronjob, newCronjob)

	// Verify API call was made to sync the newly included job
	if mockServer.getRequestCount() != 1 {
		t.Errorf("expected 1 API call when job becomes included, got %d", mockServer.getRequestCount())
	}

	// Verify job is now tracked
	if !coll.IsTracked(newCronjob.GetUID()) {
		t.Error("expected job to be tracked after becoming included")
	}
}

func TestOnUpdate_RemovesWhenJobBecomesExcluded(t *testing.T) {
	mockServer := newMockAPIServer()
	defer mockServer.close()

	coll := createTestCollection(mockServer)

	// Create old cronjob without exclude annotation (was included)
	oldCronjob := &v1.CronJob{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "my-job",
			Namespace: "default",
			UID:       "uid-def",
		},
		Spec: v1.CronJobSpec{
			Schedule: "*/5 * * * *",
		},
	}

	// Create new cronjob with exclude annotation
	newCronjob := &v1.CronJob{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "my-job",
			Namespace: "default",
			UID:       "uid-def",
			Annotations: map[string]string{
				"k8s.cronitor.io/exclude": "true",
			},
		},
		Spec: v1.CronJobSpec{
			Schedule: "*/5 * * * *",
		},
	}

	// Pre-add the old cronjob to simulate it being tracked
	coll.cronjobs[oldCronjob.GetUID()] = oldCronjob

	// Call onUpdate - job is now excluded
	onUpdate(coll, oldCronjob, newCronjob)

	// Verify job is no longer tracked
	if coll.IsTracked(oldCronjob.GetUID()) {
		t.Error("expected job to be untracked after becoming excluded")
	}
}

func TestBulkSync_AllJobsInSingleRequest(t *testing.T) {
	requestCount := 0
	var capturedBodies []string
	var mu sync.Mutex

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()
		requestCount++

		body := make([]byte, r.ContentLength)
		r.Body.Read(body)
		capturedBodies = append(capturedBodies, string(body))

		w.WriteHeader(http.StatusOK)
		w.Write([]byte("[]"))
	}))
	defer server.Close()

	viper.Set("hostname-override", server.URL)

	cronitorApi := &api.CronitorApi{
		ApiKey:    "test-key",
		UserAgent: "test-agent",
	}

	// Simulate bulk sync by calling PutCronJobs directly
	cronjobs := []*v1.CronJob{
		createTestCronJob("job-1", "ns1", "uid-1", "*/5 * * * *"),
		createTestCronJob("job-2", "ns2", "uid-2", "0 * * * *"),
		createTestCronJob("job-3", "ns3", "uid-3", "0 0 * * *"),
	}

	_, err := cronitorApi.PutCronJobs(cronjobs)
	if err != nil {
		t.Fatalf("PutCronJobs failed: %v", err)
	}

	// Verify only 1 request was made
	if requestCount != 1 {
		t.Errorf("expected 1 bulk request, got %d", requestCount)
	}

	// Verify all 3 jobs were in the request
	if len(capturedBodies) > 0 {
		var monitors []map[string]interface{}
		if err := json.Unmarshal([]byte(capturedBodies[0]), &monitors); err != nil {
			t.Fatalf("failed to parse request body: %v", err)
		}
		if len(monitors) != 3 {
			t.Errorf("expected 3 monitors in bulk request, got %d", len(monitors))
		}
	}
}

func TestCoerceObjToV1CronJob_ReturnsNilForUnknownVersion(t *testing.T) {
	// coerceObjToV1CronJob should return nil for unknown versions
	// This tests that event handlers properly check for nil

	cronjob := &v1.CronJob{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-job",
			Namespace: "default",
			UID:       "uid-123",
		},
	}

	// Test with valid v1 version - should return cronjob
	result := coerceObjToV1CronJob("v1", cronjob)
	if result == nil {
		t.Error("expected non-nil result for v1 version")
	}

	// Test with unknown version - should return nil
	result = coerceObjToV1CronJob("v2", cronjob)
	if result != nil {
		t.Error("expected nil result for unknown version")
	}

	result = coerceObjToV1CronJob("", cronjob)
	if result != nil {
		t.Error("expected nil result for empty version")
	}
}

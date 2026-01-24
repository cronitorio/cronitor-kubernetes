#!/bin/bash
# E2E verification script
# Checks that the mock Cronitor API received the expected requests from the agent

set -e

MOCK_POD="deployment/mock-cronitor-api"
MOCK_NS="cronitor-mock"

# Expected number of cronjobs deployed in e2e test (from e2e-tests.yml)
# Note: 8 total CronJobs are deployed, but 1 has exclude annotation, so expect 7
EXPECTED_CRONJOB_COUNT=7

echo "=== E2E Verification ==="
echo ""

# Function to query mock server
query_mock() {
    kubectl exec -n "$MOCK_NS" "$MOCK_POD" -- wget -q -O- "http://localhost:8080$1"
}

# Get monitor sync requests
echo "Checking monitor sync requests..."
MONITOR_RESPONSE=$(query_mock "/debug/monitors")
MONITOR_COUNT=$(echo "$MONITOR_RESPONSE" | jq -r '.count')

echo "  Monitor requests received: $MONITOR_COUNT"

if [ "$MONITOR_COUNT" -lt 1 ]; then
    echo "FAIL: Expected at least 1 monitor sync request, got $MONITOR_COUNT"
    exit 1
fi

# CRITICAL: Verify all cronjobs are batched in a SINGLE request
if [ "$MONITOR_COUNT" -ne 1 ]; then
    echo "FAIL: Expected exactly 1 bulk PUT request, got $MONITOR_COUNT"
    echo "      All cronjobs should be synced in a single batch API call"
    exit 1
fi
echo "  ✓ All cronjobs batched in single request"

# Check that the request was a PUT to /api/monitors
MONITOR_METHOD=$(echo "$MONITOR_RESPONSE" | jq -r '.requests[0].method')
MONITOR_PATH=$(echo "$MONITOR_RESPONSE" | jq -r '.requests[0].path')

echo "  First request: $MONITOR_METHOD $MONITOR_PATH"

if [ "$MONITOR_METHOD" != "PUT" ]; then
    echo "FAIL: Expected PUT method, got $MONITOR_METHOD"
    exit 1
fi

if [ "$MONITOR_PATH" != "/api/monitors" ]; then
    echo "FAIL: Expected path /api/monitors, got $MONITOR_PATH"
    exit 1
fi
echo "  ✓ Correct HTTP method and path"

# Verify Authorization header is present (Basic auth)
AUTH_HEADER=$(echo "$MONITOR_RESPONSE" | jq -r '.requests[0].headers.Authorization // empty')
if [ -z "$AUTH_HEADER" ]; then
    echo "FAIL: Missing Authorization header"
    exit 1
fi
if [[ ! "$AUTH_HEADER" =~ ^Basic ]]; then
    echo "FAIL: Expected Basic auth, got: $AUTH_HEADER"
    exit 1
fi
echo "  ✓ Authorization header present (Basic auth)"

# Check the body contains monitors
MONITOR_BODY=$(echo "$MONITOR_RESPONSE" | jq -r '.requests[0].body')
if [ -z "$MONITOR_BODY" ] || [ "$MONITOR_BODY" == "null" ]; then
    echo "FAIL: Expected request body with monitors"
    exit 1
fi

# Verify the body is valid JSON array
MONITORS_SENT=$(echo "$MONITOR_BODY" | jq -r 'length')
echo "  Monitors in request: $MONITORS_SENT"

if [ "$MONITORS_SENT" -lt 1 ]; then
    echo "FAIL: Expected at least 1 monitor in request body"
    exit 1
fi

# Verify ALL expected cronjobs are present in the single request
if [ "$MONITORS_SENT" -ne "$EXPECTED_CRONJOB_COUNT" ]; then
    echo "FAIL: Expected $EXPECTED_CRONJOB_COUNT monitors in bulk request, got $MONITORS_SENT"
    echo "      All cronjobs should be synced together in a single API call"
    echo "      Monitors received:"
    echo "$MONITOR_BODY" | jq -r '.[].name'
    exit 1
fi
echo "  ✓ All $EXPECTED_CRONJOB_COUNT cronjobs present in bulk request"

# Check first monitor has required fields
FIRST_MONITOR_KEY=$(echo "$MONITOR_BODY" | jq -r '.[0].key // empty')
FIRST_MONITOR_NAME=$(echo "$MONITOR_BODY" | jq -r '.[0].name // empty')
FIRST_MONITOR_SCHEDULE=$(echo "$MONITOR_BODY" | jq -r '.[0].schedule // empty')
FIRST_MONITOR_TYPE=$(echo "$MONITOR_BODY" | jq -r '.[0].type // empty')

echo "  First monitor:"
echo "    key: $FIRST_MONITOR_KEY"
echo "    name: $FIRST_MONITOR_NAME"
echo "    schedule: $FIRST_MONITOR_SCHEDULE"
echo "    type: $FIRST_MONITOR_TYPE"

if [ -z "$FIRST_MONITOR_KEY" ]; then
    echo "FAIL: Monitor missing 'key' field"
    exit 1
fi

if [ -z "$FIRST_MONITOR_NAME" ]; then
    echo "FAIL: Monitor missing 'name' field"
    exit 1
fi

if [ -z "$FIRST_MONITOR_SCHEDULE" ]; then
    echo "FAIL: Monitor missing 'schedule' field"
    exit 1
fi

if [ "$FIRST_MONITOR_TYPE" != "job" ]; then
    echo "FAIL: Expected type 'job', got '$FIRST_MONITOR_TYPE'"
    exit 1
fi

# Verify name is not a UUID (regression test)
if [[ "$FIRST_MONITOR_NAME" =~ ^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$ ]]; then
    echo "FAIL: Monitor name appears to be a UUID: $FIRST_MONITOR_NAME"
    echo "      Names should be in format 'namespace/name'"
    exit 1
fi

# Verify name contains a slash (namespace/name format)
if [[ ! "$FIRST_MONITOR_NAME" =~ "/" ]]; then
    echo "WARN: Monitor name doesn't contain '/': $FIRST_MONITOR_NAME"
    echo "      Expected format 'namespace/name'"
fi

# Check for kubernetes tags
FIRST_MONITOR_TAGS=$(echo "$MONITOR_BODY" | jq -r '.[0].tags // []')
HAS_K8S_TAG=$(echo "$FIRST_MONITOR_TAGS" | jq 'map(select(. == "kubernetes")) | length')

if [ "$HAS_K8S_TAG" -lt 1 ]; then
    echo "FAIL: Monitor missing 'kubernetes' tag"
    exit 1
fi
echo "  ✓ Required fields and tags present"

# =====================================================
# TEST: Exclusion Annotation
# Verify that CronJobs with k8s.cronitor.io/exclude: "true" are NOT synced
# =====================================================
echo ""
echo "Checking exclusion annotation..."

# The excluded CronJob is named "eventrouter-test-croonjob-excluder"
EXCLUDED_MONITOR=$(echo "$MONITOR_BODY" | jq -r '.[] | select(.name | contains("eventrouter-test-croonjob-excluder")) | .name // empty')
if [ -n "$EXCLUDED_MONITOR" ]; then
    echo "FAIL: Excluded CronJob was synced to Cronitor (found: $EXCLUDED_MONITOR)"
    echo "      CronJobs with 'k8s.cronitor.io/exclude: true' should NOT be synced"
    exit 1
fi

# Also check by looking for any monitor key containing the excluded job name
EXCLUDED_KEY=$(echo "$MONITOR_BODY" | jq -r '.[] | select(.key | contains("excluder")) | .key // empty')
if [ -n "$EXCLUDED_KEY" ]; then
    echo "FAIL: Excluded CronJob was synced to Cronitor (found key: $EXCLUDED_KEY)"
    exit 1
fi
echo "  ✓ Exclusion annotation works (excluded job NOT synced)"

# =====================================================
# Verify specific annotation-based monitors
# =====================================================
echo ""
echo "Checking annotation-based monitors..."

# Find monitor with custom cronitor-id
CUSTOM_ID_MONITOR=$(echo "$MONITOR_BODY" | jq -r '.[] | select(.key == "my-custom-id") | .name // empty')
if [ -z "$CUSTOM_ID_MONITOR" ]; then
    echo "FAIL: Expected monitor with key 'my-custom-id' (from cronitor-id annotation)"
    echo "      Available keys:"
    echo "$MONITOR_BODY" | jq -r '.[].key'
    exit 1
fi
echo "  ✓ cronitor-id annotation works (key: my-custom-id)"

# Find monitor with custom name
CUSTOM_NAME_MONITOR=$(echo "$MONITOR_BODY" | jq -r '.[] | select(.name == "my-custom-monitor-name") | .key // empty')
if [ -z "$CUSTOM_NAME_MONITOR" ]; then
    echo "FAIL: Expected monitor with name 'my-custom-monitor-name' (from cronitor-name annotation)"
    echo "      Available names:"
    echo "$MONITOR_BODY" | jq -r '.[].name'
    exit 1
fi
echo "  ✓ cronitor-name annotation works (name: my-custom-monitor-name)"

# Check for environment annotation (staging env)
ENV_MONITOR=$(echo "$MONITOR_BODY" | jq -r '.[] | select(.tags | index("env:staging")) | .name // empty')
if [ -z "$ENV_MONITOR" ]; then
    echo "WARN: No monitor with 'env:staging' tag found (from env annotation)"
    echo "      Tags in first monitor: $FIRST_MONITOR_TAGS"
fi

# Check for group annotation
GROUP_MONITOR=$(echo "$MONITOR_BODY" | jq -r '.[] | select(.group == "my-test-group") | .name // empty')
if [ -z "$GROUP_MONITOR" ]; then
    echo "FAIL: Expected monitor with group 'my-test-group' (from cronitor-group annotation)"
    echo "      Available groups:"
    echo "$MONITOR_BODY" | jq -r '.[].group // "null"'
    exit 1
fi
echo "  ✓ cronitor-group annotation works (group: my-test-group)"

# Check for notify annotation
NOTIFY_MONITOR=$(echo "$MONITOR_BODY" | jq -r '.[] | select(.notify | length > 0) | .name' | head -1)
if [ -z "$NOTIFY_MONITOR" ]; then
    echo "WARN: No monitor with notify list found (from cronitor-notify annotation)"
fi

# Check for grace-seconds annotation
GRACE_MONITOR=$(echo "$MONITOR_BODY" | jq -r '.[] | select(.grace_seconds != null and .grace_seconds > 0) | .name' | head -1)
if [ -z "$GRACE_MONITOR" ]; then
    echo "WARN: No monitor with grace_seconds found (from cronitor-grace-seconds annotation)"
fi

echo ""
echo "=== All E2E checks passed ==="
echo ""
echo "Summary:"
echo "  - Agent successfully synced $MONITORS_SENT monitors to mock server"
echo "  - All $EXPECTED_CRONJOB_COUNT cronjobs batched in SINGLE PUT request"
echo "  - Request format is correct (PUT /api/monitors with Basic auth)"
echo "  - Monitor data structure is valid"
echo "  - Names are human-readable (not UUIDs)"
echo "  - Required tags are present"
echo "  - Exclusion annotation works (excluded jobs NOT synced)"
echo "  - Annotation-based customizations work correctly"

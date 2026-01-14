#!/bin/bash
# E2E verification script
# Checks that the mock Cronitor API received the expected requests from the agent

set -e

MOCK_POD="deployment/mock-cronitor-api"
MOCK_NS="cronitor-mock"

# Expected number of cronjobs deployed in e2e test (from e2e-tests.yml)
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

echo ""
echo "=== All E2E checks passed ==="
echo ""
echo "Summary:"
echo "  - Agent successfully synced $MONITORS_SENT monitors to mock server"
echo "  - All $EXPECTED_CRONJOB_COUNT cronjobs batched in SINGLE PUT request"
echo "  - Request format is correct (PUT /api/monitors)"
echo "  - Monitor data structure is valid"
echo "  - Names are human-readable (not UUIDs)"
echo "  - Required tags are present"

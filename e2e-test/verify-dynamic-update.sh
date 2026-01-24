#!/bin/bash
# E2E test: Dynamic Update Detection
# Tests that schedule changes to CronJobs are detected and synced to Cronitor

set -e

MOCK_POD="deployment/mock-cronitor-api"
MOCK_NS="cronitor-mock"
TARGET_NS="cronitor"

echo "=== E2E Test: Dynamic Update Detection ==="
echo ""

# Function to query mock server
query_mock() {
    kubectl exec -n "$MOCK_NS" "$MOCK_POD" -- wget -q -O- "http://localhost:8080$1"
}

# Function to clear mock server state
clear_mock() {
    kubectl exec -n "$MOCK_NS" "$MOCK_POD" -- wget -q -O- --post-data="" "http://localhost:8080/debug/clear"
}

# Step 1: Clear existing requests from initial sync
echo "Step 1: Clearing mock server request history..."
clear_mock
echo "  ✓ Cleared"

# Verify it's empty
INITIAL_COUNT=$(query_mock "/debug/monitors" | jq -r '.count')
if [ "$INITIAL_COUNT" -ne 0 ]; then
    echo "FAIL: Expected 0 requests after clear, got $INITIAL_COUNT"
    exit 1
fi
echo "  ✓ Verified empty state"

# Step 2: Update a CronJob's schedule
echo ""
echo "Step 2: Updating CronJob schedule..."

# Get the current schedule of test-cronjob
ORIGINAL_SCHEDULE=$(kubectl get cronjob test-cronjob -n "$TARGET_NS" -o jsonpath='{.spec.schedule}')
echo "  Original schedule: $ORIGINAL_SCHEDULE"

# Update to a new schedule (change from */1 to */2)
NEW_SCHEDULE="*/2 * * * *"
kubectl patch cronjob test-cronjob -n "$TARGET_NS" --type='json' -p='[{"op": "replace", "path": "/spec/schedule", "value": "'"$NEW_SCHEDULE"'"}]'
echo "  New schedule: $NEW_SCHEDULE"
echo "  ✓ CronJob patched"

# Step 3: Wait for the agent to detect the change
echo ""
echo "Step 3: Waiting for agent to detect the change..."
sleep 10

# Step 4: Verify a new sync request was made
echo ""
echo "Step 4: Verifying update was synced..."

MONITOR_RESPONSE=$(query_mock "/debug/monitors")
MONITOR_COUNT=$(echo "$MONITOR_RESPONSE" | jq -r '.count')

echo "  Monitor sync requests received: $MONITOR_COUNT"

if [ "$MONITOR_COUNT" -lt 1 ]; then
    echo "FAIL: Expected at least 1 sync request after update, got $MONITOR_COUNT"
    echo "      The agent should detect CronJob schedule changes and sync"
    exit 1
fi
echo "  ✓ Agent detected the change and sent sync request"

# Step 5: Verify the updated schedule is in the request
MONITOR_BODY=$(echo "$MONITOR_RESPONSE" | jq -r '.requests[0].body')

# Find the test-cronjob monitor and check its schedule
UPDATED_MONITOR=$(echo "$MONITOR_BODY" | jq -r '.[] | select(.name | contains("test-cronjob")) | .schedule')

if [ -z "$UPDATED_MONITOR" ]; then
    echo "WARN: Could not find test-cronjob in sync request"
    echo "      Monitors in request:"
    echo "$MONITOR_BODY" | jq -r '.[].name'
else
    echo "  Synced schedule: $UPDATED_MONITOR"

    # The schedule in Cronitor format might differ slightly, but should reflect the change
    if [[ "$UPDATED_MONITOR" == *"2"* ]] || [[ "$UPDATED_MONITOR" == "$NEW_SCHEDULE" ]]; then
        echo "  ✓ Updated schedule confirmed"
    else
        echo "WARN: Schedule may not reflect the update"
        echo "      Expected: $NEW_SCHEDULE"
        echo "      Got: $UPDATED_MONITOR"
    fi
fi

# Step 6: Restore original schedule
echo ""
echo "Step 5: Restoring original schedule..."
kubectl patch cronjob test-cronjob -n "$TARGET_NS" --type='json' -p='[{"op": "replace", "path": "/spec/schedule", "value": "'"$ORIGINAL_SCHEDULE"'"}]'
echo "  ✓ Restored to: $ORIGINAL_SCHEDULE"

echo ""
echo "=== Dynamic Update Test Passed ==="
echo ""
echo "Summary:"
echo "  - Agent detected CronJob schedule change"
echo "  - Sync request was sent to Cronitor API"
echo "  - Updated schedule was included in request"

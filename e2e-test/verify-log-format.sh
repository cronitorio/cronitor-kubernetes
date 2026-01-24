#!/bin/bash
# E2E verification script for log format configuration
# Tests that the agent correctly outputs logs in the configured format (text or json)

set -e

AGENT_NS="${AGENT_NS:-cronitor}"
LOG_FORMAT="${LOG_FORMAT:-}"  # Expected format: "json" or "text"

echo "=== Log Format E2E Verification ==="
echo ""

# Find agent pod
AGENT_POD=$(kubectl get pods -n "$AGENT_NS" -l app.kubernetes.io/name=cronitor-kubernetes-agent -o jsonpath='{.items[0].metadata.name}' 2>/dev/null || true)

if [ -z "$AGENT_POD" ]; then
    echo "FAIL: Agent pod not found in namespace '$AGENT_NS'"
    exit 1
fi

echo "Agent pod: $AGENT_POD"
echo "Expected log format: ${LOG_FORMAT:-auto-detect}"
echo ""

# Get agent logs (wait a bit for logs to be generated)
echo "Retrieving agent logs..."
sleep 5

AGENT_LOGS=$(kubectl logs -n "$AGENT_NS" "$AGENT_POD" --tail=20 2>/dev/null || true)

if [ -z "$AGENT_LOGS" ]; then
    echo "FAIL: Could not retrieve agent logs"
    exit 1
fi

echo "Sample logs:"
echo "$AGENT_LOGS" | head -5
echo "..."
echo ""

# Get first non-empty log line
FIRST_LOG_LINE=$(echo "$AGENT_LOGS" | grep -v '^$' | head -1)

if [ -z "$FIRST_LOG_LINE" ]; then
    echo "FAIL: No log lines found"
    exit 1
fi

# Determine if logs are JSON
IS_JSON="false"
if echo "$FIRST_LOG_LINE" | jq . >/dev/null 2>&1; then
    IS_JSON="true"
fi

echo "Log analysis:"
echo "  First line: $FIRST_LOG_LINE"
echo "  Is JSON: $IS_JSON"
echo ""

# Verify expected format
if [ "$LOG_FORMAT" = "json" ]; then
    if [ "$IS_JSON" != "true" ]; then
        echo "FAIL: Expected JSON format but logs are not valid JSON"
        exit 1
    fi
    echo "✓ Logs are in JSON format as expected"

    # Verify JSON structure
    echo ""
    echo "Verifying JSON structure..."

    HAS_TIME=$(echo "$FIRST_LOG_LINE" | jq 'has("time")' 2>/dev/null || echo "false")
    HAS_LEVEL=$(echo "$FIRST_LOG_LINE" | jq 'has("level")' 2>/dev/null || echo "false")
    HAS_MSG=$(echo "$FIRST_LOG_LINE" | jq 'has("msg")' 2>/dev/null || echo "false")

    if [ "$HAS_TIME" != "true" ]; then
        echo "FAIL: JSON log missing 'time' field"
        exit 1
    fi
    echo "  ✓ Has 'time' field"

    if [ "$HAS_LEVEL" != "true" ]; then
        echo "FAIL: JSON log missing 'level' field"
        exit 1
    fi
    echo "  ✓ Has 'level' field"

    if [ "$HAS_MSG" != "true" ]; then
        echo "FAIL: JSON log missing 'msg' field"
        exit 1
    fi
    echo "  ✓ Has 'msg' field"

    # Extract and display log level
    LOG_LEVEL=$(echo "$FIRST_LOG_LINE" | jq -r '.level' 2>/dev/null || echo "unknown")
    echo "  Log level: $LOG_LEVEL"

elif [ "$LOG_FORMAT" = "text" ]; then
    if [ "$IS_JSON" = "true" ]; then
        echo "FAIL: Expected text format but logs are JSON"
        exit 1
    fi
    echo "✓ Logs are in text format as expected"

    # Verify text format has expected structure (key=value pairs)
    if echo "$FIRST_LOG_LINE" | grep -qE 'level=[A-Z]+'; then
        echo "  ✓ Has level=LEVEL format"
    fi
    if echo "$FIRST_LOG_LINE" | grep -qE 'msg='; then
        echo "  ✓ Has msg= format"
    fi

else
    # Auto-detect mode - just report what we found
    if [ "$IS_JSON" = "true" ]; then
        echo "Detected: JSON format"
        echo ""
        echo "JSON structure:"
        echo "$FIRST_LOG_LINE" | jq -r 'keys[]' 2>/dev/null | while read key; do
            echo "  - $key"
        done
    else
        echo "Detected: Text format"
    fi
fi

echo ""
echo "=== Log format verification passed ==="

"""
Telemetry Tests - Moved to Go Unit Tests

The telemetry tests that were previously in this file have been moved to Go unit tests
in pkg/api/telemetry_test.go. These tests verify:

1. Telemetry URL construction (TestTelemetryUrl, TestTelemetryUrlWithCustomCronitorID)
2. Environment parameter encoding (TestTelemetryEventIncludesEnvironmentFromAnnotation)
3. Event status translation (TestTranslatePodEventReasonToTelemetryEventStatus,
   TestTranslateJobEventReasonToTelemetryEventStatus)
4. Full request verification with httptest mock (TestSendTelemetryPostRequest)

The Go tests verify that:
- The correct URL is constructed: /ping/{api_key}/{monitor_key}/{state}
- The env query parameter is included when the annotation is present
- Pod events (Started, BackOff) map to correct states (run, fail)
- Job events (SuccessfulCreate, Completed, BackoffLimitExceeded) map to correct states

We trust the Cronitor API to correctly process telemetry when we send the right
URL and parameters. The Go unit tests verify our code sends the correct data.

Previous Python e2e tests that are now covered:
- test_telemetry_sent_to_correct_environment -> TestTelemetryEventIncludesEnvironmentFromAnnotation
- test_failing_monitor_should_fail -> TestTranslateJobEventReasonToTelemetryEventStatus
- test_successful_monitor_should_succeed -> TestTranslateJobEventReasonToTelemetryEventStatus
"""

# No e2e tests remain in this file - all telemetry logic is tested via Go unit tests

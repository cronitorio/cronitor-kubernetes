"""
E2E Smoke Tests for Monitor Sync

These tests verify the full end-to-end flow of syncing CronJobs to Cronitor monitors.
They require a real Kubernetes cluster and Cronitor API access.

Note: Many tests have been moved to Go unit tests:

pkg/api/jobs_test.go:
- TestExistingCronitorID (cronitor-id annotation)
- TestCronitorNameAnnotation (cronitor-name annotation)
- TestCronitorGroupAnnotation (cronitor-group annotation)
- TestCronitorNotifyAnnotation (cronitor-notify annotation)
- TestCronitorGraceSecondsAnnotation (cronitor-grace-seconds annotation)
- TestMonitorNameIsNeverUUID (name is never a UUID)
- TestDefaultMonitorName (default namespace/name format)

pkg/api/telemetry_test.go:
- TestTelemetryUrl, TestTelemetryEventEncode (telemetry URL and params)
- TestTranslate*EventReasonToTelemetryEventStatus (event status mapping)
- TestTelemetryEventIncludesEnvironmentFromAnnotation (env routing)

Removed tests that verify server-side behavior we should trust:
- test_same_id_should_result_one_monitor (Cronitor API deduplication)

The Go unit tests verify our code sends correct data. We trust Cronitor's API
to handle that data correctly.
"""
import pytest
from typing import Optional
import os
import time
from ..kubernetes_wrapper import get_cronjob_by_name, patch_cronjob_by_name
from ..cronitor_wrapper import cronitor_wrapper_from_environment

cronitor_wrapper = cronitor_wrapper_from_environment()


@pytest.mark.parametrize("name,namespace", [
    ['test-cronjob', None],
    pytest.param('test-cronjob-namespace', os.getenv('KUBERNETES_EXTRA_NAMESPACE'),
                 marks=pytest.mark.xfail(os.getenv("TEST_CONFIGURATION") == 'single_namespace_rbac',
                                         raises=StopIteration,
                                         reason="The specially namespaced job should not be present in "
                                                "the 'single_namespace_rbac' test configuration.")),
    ['test-env-annotation', None],
    ['test-env-annotation-home', None],
    ['eventrouter-test-cronjob-fail', None],
])
def test_included_cronjobs_present(name: str, namespace: Optional[str]):
    """
    Smoke test: Verify that CronJobs are synced to Cronitor with correct names.

    This tests the full flow: K8s CronJob -> Agent -> Cronitor API -> Monitor exists.
    """
    cronjob = get_cronjob_by_name(name, namespace)
    key = cronjob['metadata']['uid']
    monitor = next(m for m in cronitor_wrapper.get_all_ci_monitors()
                   if m['key'] == key)

    # Ensure the name is correct
    assert monitor['name'] == (cronjob['metadata']['namespace'] + '/' + cronjob['metadata']['name'])


def test_monitor_schedule_gets_updated():
    """
    Smoke test: Verify that schedule changes in K8s propagate to Cronitor.

    This tests the watcher functionality: when a CronJob schedule is updated,
    the agent should detect this and update the monitor in Cronitor.
    """
    random_id = os.getenv("RANDOM_ID")
    monitor_key = "test-schedule-change-{RANDOM_ID}".format(RANDOM_ID=random_id)
    monitor = cronitor_wrapper.get_ci_monitor_by_key(monitor_key)
    assert monitor is not None, f"no monitor with key {monitor_key} exists"
    assert monitor['schedule'] == "*/5 */10 * * *", f"expected monitor schedule '*/5 */10 * * *', got '{monitor['schedule']}'"

    new_schedule = "*/10 */50 * * *"
    patch_cronjob_by_name("test-schedule-change", None, {"spec": {"schedule": new_schedule}})
    cronitor_wrapper.bust_monitor_cache()
    time.sleep(3)
    monitor = cronitor_wrapper.get_ci_monitor_by_key(monitor_key)
    assert monitor['schedule'] == new_schedule, f"expected monitor schedule '{new_schedule}', got '{monitor['schedule']}'"

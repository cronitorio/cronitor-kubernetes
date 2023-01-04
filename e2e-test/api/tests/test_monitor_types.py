import pytest
from pytest import assume
from typing import Optional
import os
import uuid
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
    """Ensure that each CronJob properly exists in Cronitor by name, with key"""
    cronjob = get_cronjob_by_name(name, namespace)
    key = cronjob['metadata']['uid']
    monitor = next(m for m in cronitor_wrapper.get_all_ci_monitors()
                   if m['key'] == key)

    # Ensure the name is correct
    assert monitor['name'] == (cronjob['metadata']['namespace'] + '/' + cronjob['metadata']['name'])


EXCLUDED = ['eventrouter-test-croonjob-excluder', ]


@pytest.mark.parametrize("name", EXCLUDED)
def test_expected_cronjobs_missing(name: str):
    """Ensure excluded/non-existing cron jobs are _not_ in Cronitor"""
    cronjob = get_cronjob_by_name(name)
    key = cronjob['metadata']['uid']
    with pytest.raises(StopIteration):
        monitor = next(m for m in cronitor_wrapper.get_all_ci_monitors()
                       if m['key'] == key)


def test_no_monitors_with_uid_names():
    # Ensure no unexpected monitors or monitors with UID names
    # We may need to do further testing outside the tag scope, depending on
    # if this issue happens in such a way that monitors are auto-created without tags
    monitors = cronitor_wrapper.get_all_ci_monitors()
    for monitor in monitors:
        # pytest-assume allows multiple failures per test
        with assume:
            # We want the monitor names NOT to be UUIDs. If they are a UUID,
            # that means we encountered a bug with the monitor creation somehow
            # and defaultName was not set.
            with pytest.raises(ValueError):
                uuid.UUID(monitor['name'])


def test_monitor_created_with_new_id():
    random_id = os.getenv("RANDOM_ID")
    monitor_key = "annotation-test-id-{RANDOM_ID}".format(RANDOM_ID=random_id)

    monitor = cronitor_wrapper.get_ci_monitor_by_key(monitor_key)
    assert monitor is not None, f"no monitor with key {monitor_key} exists but one should"


def test_monitor_schedule_gets_updated():
    random_id = os.getenv("RANDOM_ID")
    monitor_key = "test-schedule-change-{RANDOM_ID}".format(RANDOM_ID=random_id)
    monitor = cronitor_wrapper.get_ci_monitor_by_key(monitor_key)
    assert monitor is not None, f"no monitor with key {monitor_key} exists"
    assert monitor['schedule'] == "*/5 */10 * * *", f"expected monitor schedule '*/5 */10 * * *', got '{monitor['schedule']}'"

    new_schedule = "*/10 */50 * * *"
    patch_cronjob_by_name("test-schedule-change", None, {"spec": {"schedule": new_schedule}})
    monitor = cronitor_wrapper.get_ci_monitor_by_key(monitor_key)
    assert monitor['schedule'] == new_schedule, f"expected monitor schedule '{new_schedule}', got '{monitor['schedule']}'"


def test_monitor_created_with_specified_name():
    random_id = os.getenv("RANDOM_ID")
    monitor_name = "annotation-test-name-{RANDOM_ID}".format(RANDOM_ID=random_id)
    monitors = cronitor_wrapper.get_all_ci_monitors()

    _ = next(monitor for monitor in monitors if monitor["name"] == monitor_name)

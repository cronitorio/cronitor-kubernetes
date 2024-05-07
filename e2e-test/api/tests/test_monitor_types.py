import pytest
from pytest import assume
from typing import Optional
import os
import uuid
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
            # and Name was not set.
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
    cronitor_wrapper.bust_monitor_cache()
    time.sleep(3)
    monitor = cronitor_wrapper.get_ci_monitor_by_key(monitor_key)
    assert monitor['schedule'] == new_schedule, f"expected monitor schedule '{new_schedule}', got '{monitor['schedule']}'"


def test_monitor_created_with_specified_name():
    random_id = os.getenv("RANDOM_ID")
    monitor_name = "annotation-test-name-{RANDOM_ID}".format(RANDOM_ID=random_id)
    monitors = cronitor_wrapper.get_all_ci_monitors()

    _ = next(monitor for monitor in monitors if monitor["name"] == monitor_name)


def test_monitor_created_with_group():
    random_id = os.getenv("RANDOM_ID")
    monitor_key = "test-group-annotation-{RANDOM_ID}".format(RANDOM_ID=random_id)
    monitor = cronitor_wrapper.get_ci_monitor_by_key(monitor_key)
    assert monitor is not None, f"no monitor with key {monitor_key} exists"
    assert monitor['group'] == "test-group", f"expected monitor group 'test-group', got '{monitor['group']}'"


def test_monitor_created_with_notify():
    random_id = os.getenv("RANDOM_ID")
    monitor_key = "test-notify-annotation-{RANDOM_ID}".format(RANDOM_ID=random_id)
    monitor = cronitor_wrapper.get_ci_monitor_by_key(monitor_key)
    assert monitor is not None, f"no monitor with key {monitor_key} exists"
    assert monitor['notify'] == ["devops-slack", "infra-teams"], f"expected ['devops-slack', 'infra-teams'] got '{monitor['notify']}'"


def test_monitor_created_with_grace_seconds():
    random_id = os.getenv("RANDOM_ID")
    monitor_key = "test-grace-seconds-annotation-{RANDOM_ID}".format(RANDOM_ID=random_id)
    monitor = cronitor_wrapper.get_ci_monitor_by_key(monitor_key)
    assert monitor is not None, f"no monitor with key {monitor_key} exists"
    assert monitor['grace_seconds'] == 305, f"expected monitor grace_seconds '305', got '{monitor['grace_seconds']}'"


def test_same_id_should_result_one_monitor():
    random_id = os.getenv("RANDOM_ID")
    monitor_key = "test-id-annotation-multiple-{RANDOM_ID}".format(RANDOM_ID=random_id)
    monitor = cronitor_wrapper.get_ci_monitor_by_key(monitor_key)
    assert monitor is not None, f"no monitor with {monitor_key} exists"

    ci_monitors = cronitor_wrapper.get_all_ci_monitors()
    monitors_with_relevant_name = [
        monitor for monitor in ci_monitors
        if 'multiple' in monitor['key']
    ]
    how_many = len(monitors_with_relevant_name)
    names = ', '.join([monitor['name'] for monitor in monitors_with_relevant_name])
    assert how_many == 1, f"There isn't 1 monitor with 'multiple' in the key, there are {how_many}: {names}"

    pings = cronitor_wrapper.get_ping_history_by_monitor(monitor_key=monitor_key, env='env1')
    assert len(pings[monitor_key]) > 0
    pings = cronitor_wrapper.get_ping_history_by_monitor(monitor_key=monitor_key, env='env2')
    assert len(pings[monitor_key]) > 0

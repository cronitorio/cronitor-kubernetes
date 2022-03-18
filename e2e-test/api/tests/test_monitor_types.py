import pytest
from pytest import assume
from typing import Optional
import uuid
from ..kubernetes_wrapper import get_cronjob_by_name
from ..cronitor_wrapper import cronitor_wrapper_from_environment

cronitor_wrapper = cronitor_wrapper_from_environment()


@pytest.mark.parametrize("name,namespace", [
    ['test-cronjob', None],
    ['test-cronjob-namespace', 'extra-namespace'],
    ['test-env-annotation', None],
    ['test-env-annotation-home', None],
    ['eventrouter-test-cronjob-fail', None],
])
def test_included_cronjobs_present(name: str, namespace: Optional[str] = None):
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
    # We may need to do further testing outside of the tag scope, depending on
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

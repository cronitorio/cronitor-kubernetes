import pytest
from ..kubernetes_wrapper import get_cronjob_by_name
from ..cronitor_wrapper import cronitor_wrapper_from_environment
import os

cronitor_wrapper = cronitor_wrapper_from_environment(ci_tag=os.getenv('CI_TAG'))


@pytest.mark.parametrize("name", [
    'eventrouter-test-cronjob-2',
    'environment-test-telemetry',
    'test-env-annotation-home',
    'eventrouter-test-cronjob-fail',
])
def test_included_cronjobs_present(name: str):
    """Ensure that each CronJob properly exists in Cronitor by name, with key"""
    cronjob = get_cronjob_by_name(name)
    key = cronjob['metadata']['uid']
    monitor = next(m for m in cronitor_wrapper.get_all_ci_monitors()
                   if m['key'] == key)

    # Ensure the name is correct
    assert monitor['name'] == (cronjob['metadata']['namespace'] + '/' + cronjob['metadata']['name'])


EXCLUDED = ['eventrouter-test-croonjob-excluder', ]
NONEXISTENT = ['this-does-not-exit']


@pytest.mark.parametrize("name", EXCLUDED + NONEXISTENT)
def test_expected_cronjobs_missing(name: str):
    """Ensure excluded/non-existing cron jobs are _not_ in Cronitor"""
    cronjob = get_cronjob_by_name(name)
    key = cronjob['metadata']['uid']
    with pytest.raises(StopIteration):
        monitor = next(m for m in cronitor_wrapper.get_all_ci_monitors()
                       if m['key'] == key)

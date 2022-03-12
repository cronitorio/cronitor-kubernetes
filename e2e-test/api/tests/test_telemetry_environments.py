import pytest
from ..cronitor_wrapper import cronitor_wrapper_from_environment
from ..kubernetes_wrapper import get_cronjob_by_name
cronitor_wrapper = cronitor_wrapper_from_environment()


def test_telemetry_sent_to_correct_environment():
    cronjob = get_cronjob_by_name('test-env-annotation')
    key = cronjob['metadata']['uid']

    # Ensure no pings in CI
    pings = cronitor_wrapper.get_ping_history_by_monitor(monitor_key=key, env='CI')
    assert pings[key][0] == 'No ping history for this monitor'

    # Ensure there are pings in correct env (by annotation)
    pings = cronitor_wrapper.get_ping_history_by_monitor(monitor_key=key, env='environment-test-telemetry')
    assert len(pings[key]) > 0

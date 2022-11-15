import os
from functools import partialmethod, cache
import requests
import logging
logger = logging.getLogger(__name__)
logging.basicConfig(level=logging.INFO)


class CronitorWrapper:
    def __init__(self, api_key, ci_tag):
        self.api_key = api_key
        self.ci_tag = ci_tag

    def _request(self, method, *args, **kwargs):
        return requests.request(method, *args, auth=(self.api_key, ''), **kwargs)

    get = partialmethod(_request, 'GET')
    delete = partialmethod(_request, 'DELETE')

    def delete_all_monitors(self):
        monitors = self.get_all_monitors()
        for monitor in monitors:
            self.delete_monitor_by_key(monitor['key'])

    @cache
    def get_all_monitors(self, *, page: int = 1):
        PAGE_SIZE = 50
        response = self.get('https://cronitor.io/api/monitors', params={'page': page}).json()
        results = response.get('monitors', [])
        # Previously, if we had _exactly_ 50 monitors, we'd hit an infinite loop
        if len(results) == PAGE_SIZE and response['total_monitor_count'] != PAGE_SIZE:
            additional_results = self.get_all_monitors(page=page+1)
            results += additional_results
        return results

    def get_all_ci_monitors(self):
        results = self.get_all_monitors()
        monitors = [
            monitor for monitor in results if self.ci_tag in monitor['tags']
        ]
        logger.info("Monitors found: %s", [m['key'] for m in monitors])
        return monitors

    def get_ci_monitor_by_key(self, key: str):
        monitors = self.get_all_ci_monitors()
        try:
            return next(monitor for monitor in monitors if monitor['key'] == key)
        except StopIteration:
            return None

    def delete_monitor_by_key(self, key: str):
        response = self.delete(f'https://cronitor.io/api/monitors/{key}')
        response.raise_for_status()

    def get_ping_history_by_monitor(self, monitor_key: str, env: str):
        response = self.get(f'https://cronitor.io/api/monitors/{monitor_key}/pings',
                            params={'env': env}).json()
        return response

    def get_monitor_with_events_and_invocations(self, monitor_key: str, env: str):
        response = self.get(f'https://cronitor.io/api/monitors/{monitor_key}',
                            params={'env': env,
                                    'withEvents': 'true',
                                    'withInvocations': 'true',
                                    'withStatus': 'true'}).json()
        return response


def cronitor_wrapper_from_environment(ci_tag=None):
    CRONITOR_API_KEY = os.getenv('CRONITOR_API_KEY')
    if not CRONITOR_API_KEY:
        raise ValueError("An API key must be supplied.")
    if not ci_tag:
        ci_tag = os.getenv('CI_TAG')
    cronitor_wrapper = CronitorWrapper(api_key=CRONITOR_API_KEY,
                                       ci_tag=ci_tag)
    return cronitor_wrapper

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

    def get_all_monitors(self):
        # Deal with pagination?
        results = self.get('https://cronitor.io/api/monitors').json().get('monitors', [])
        return results

    @cache
    def get_all_ci_monitors(self):
        results = self.get_all_monitors()
        monitors = [
            monitor for monitor in results if self.ci_tag in monitor['tags']
        ]
        logger.info("Monitors found: %s", [m['key'] for m in monitors])
        return monitors

    def delete_monitor_by_key(self, key: str):
        response = self.delete(f'https://cronitor.io/api/monitors/{key}')
        response.raise_for_status()


def cronitor_wrapper_from_environment(ci_tag: str = 'CI'):
    CRONITOR_API_KEY = os.getenv('CRONITOR_API_KEY')
    if not CRONITOR_API_KEY:
        raise ValueError("An API key must be supplied.")
    cronitor_wrapper = CronitorWrapper(api_key=CRONITOR_API_KEY,
                                       ci_tag=ci_tag)
    return cronitor_wrapper

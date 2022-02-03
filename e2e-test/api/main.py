from multiprocessing.sharedctypes import Value
import requests
import os
from functools import partialmethod
import click
import logging
logger = logging.getLogger(__name__)
logging.basicConfig(level=logging.INFO)

CRONITOR_API_KEY = os.getenv('CRONITOR_API_KEY')
if not CRONITOR_API_KEY:
    raise ValueError("An API key must be supplied.")

class CronitorWrapper: 
    def __init__(self, api_key, ci_tag):
        self.api_key = api_key
        self.ci_tag = ci_tag
    
    def _request(self, method, *args, **kwargs):
        return requests.request(method, *args, auth=(self.api_key, ''), **kwargs)

    get = partialmethod(_request, 'GET')
    delete = partialmethod(_request, 'DELETE')


    def get_all_ci_monitors(self):
        # Deal with pagination?
        monitors = []
        results = self.get('https://cronitor.io/api/monitors').json().get('monitors', [])
        for result in results:
            if self.ci_tag in result['tags']:
                monitors.append(result)

        logger.info("Monitors found: %s", [m['key'] for m in monitors])
        return monitors

    def delete_monitor_by_key(self, key: str):
        response = self.delete(f'https://cronitor.io/api/monitors/{key}')
        response.raise_for_status()



@click.command()
@click.option('--ci-tag')
def cleanup(ci_tag):
    cronitor_wrapper = CronitorWrapper(api_key=CRONITOR_API_KEY,
                                   ci_tag=ci_tag)
    monitors = cronitor_wrapper.get_all_ci_monitors()
    logger.info("Deleting %d monitors.", len(monitors))
    for monitor in monitors:
        logger.info("Deleting: %s", monitor['key'])
        cronitor_wrapper.delete_monitor_by_key(monitor['key'])


if __name__ == '__main__':
    cleanup()
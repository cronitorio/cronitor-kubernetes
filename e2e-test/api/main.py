import click
import logging
import os
from cronitor_wrapper import CronitorWrapper

logger = logging.getLogger(__name__)
logging.basicConfig(level=logging.INFO)

CRONITOR_API_KEY = os.getenv('CRONITOR_API_KEY')
if not CRONITOR_API_KEY:
    raise ValueError("An API key must be supplied.")


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

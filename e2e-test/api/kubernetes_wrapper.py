"""
Python wrapper around kubectl to make operations to retrieve and check
CronJobs in the local kind cluster easier.

Why use subprocess instead of the official Python Kubernetes client?
The Python client is hard to use and has a lot quirks--particularly with
forcing you to namespace by API version. This can be particularly frustrating
given that the CronJob API namespace has changed over the years, and it is easier
for our purposes to simply run `kubectl get cronjob` than have to know the particular
API version for each individual CronJob.
"""

import subprocess
from typing import Optional, Dict
import json


class CronJobNotFound(BaseException):
    pass


def get_cronjob_by_name(name: str, namespace: Optional[str] = None) -> Dict:
    try:
        response = subprocess.check_output([
            'kubectl', 'get',
            *(['-n', namespace] if namespace else []),
            'cronjob', name,
            '-o', 'json'
        ],
            stderr=subprocess.STDOUT)
    except subprocess.CalledProcessError as err:
        # Generally if we get an error here with kubectl exiting 1,
        # it's because the CronJob we're fetching doesn't exist.
        if b'not found' in err.output:
            raise CronJobNotFound(err.output)
        else:
            raise
    return json.loads(response)

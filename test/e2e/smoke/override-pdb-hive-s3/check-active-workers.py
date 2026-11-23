#!/usr/bin/env python
import argparse
import sys
import time

import trino
from trino import exceptions

if not sys.warnoptions:
    import warnings
warnings.simplefilter("ignore")

ACTIVE_WORKER_RETRY_TIMEOUT_SECONDS = 40
RETRY_INTERVAL_SECONDS = 5
REQUEST_TIMEOUT_SECONDS = 10
RETRYABLE_CONNECTION_ERRORS = (
    exceptions.TrinoConnectionError,
    exceptions.Http502Error,
    exceptions.Http503Error,
    exceptions.Http504Error,
)
RETRYABLE_QUERY_ERRORS = {
    "NO_NODES_AVAILABLE",
    "SERVER_STARTING_UP",
}


class ActiveWorkerCountMismatch(Exception):
    def __init__(self, expected, actual):
        self.expected = expected
        self.actual = actual
        super().__init__(
            f"Mismatch: [expected/active] workers [{expected}/{actual}]"
        )


def is_retryable_error(error):
    if isinstance(error, ActiveWorkerCountMismatch):
        return True
    if isinstance(error, RETRYABLE_CONNECTION_ERRORS):
        return True
    if isinstance(error, exceptions.HttpError):
        return str(error).startswith("error 429")
    return (
        isinstance(error, exceptions.TrinoQueryError)
        and error.error_name in RETRYABLE_QUERY_ERRORS
    )


def retry_on_transient(
    operation,
    description,
    timeout_seconds=ACTIVE_WORKER_RETRY_TIMEOUT_SECONDS,
):
    deadline = time.monotonic() + timeout_seconds
    attempt = 1

    while True:
        try:
            return operation()
        except Exception as error:
            if not is_retryable_error(error):
                raise

            remaining = deadline - time.monotonic()
            if remaining <= RETRY_INTERVAL_SECONDS:
                print(
                    f"[ERROR] {description} did not recover after {attempt} attempts: {error}",
                    file=sys.stderr,
                    flush=True,
                )
                raise

            print(
                f"[WARN] {description} attempt {attempt} hit a transient error: {error}; "
                f"retrying in {RETRY_INTERVAL_SECONDS}s",
                file=sys.stderr,
                flush=True,
            )
            time.sleep(RETRY_INTERVAL_SECONDS)
            attempt += 1


def get_connection(username, password, namespace):
    host = (
        'test-trino-coordinator-default-0.test-trino-coordinator-default-headless.'
        + namespace
        + '.svc.cluster.local'
    )
    conn = trino.dbapi.connect(
        host=host,
        port=8080,
        user=username,
        http_scheme='http',
        # auth=trino.auth.BasicAuthentication(username, password),
        session_properties={"query_max_execution_time": "60s"},
        request_timeout=REQUEST_TIMEOUT_SECONDS,
        max_attempts=1,
    )
    conn._http_session.verify = False
    return conn


def query_active_workers(username, password, namespace):
    conn = get_connection(username, password, namespace)
    cursor = conn.cursor()

    cursor.execute("USE \"hive\".\"default\"")
    print("USE command executed successfully.")

    cursor.execute("SHOW SCHEMAS")
    schemas = cursor.fetchall()
    print("Schemas:", schemas)

    cursor.execute("SHOW TABLES from \"hive\".\"default\"")
    tables = cursor.fetchall()
    print("Tables:", tables)

    cursor.execute(
        "SELECT COUNT(*) as nodes FROM system.runtime.nodes "
        "WHERE coordinator=false AND state='active'"
    )
    (active_workers,) = cursor.fetchone()
    return active_workers


def require_active_workers(username, password, namespace, expected_workers):
    active_workers = query_active_workers(username, password, namespace)
    if int(active_workers) != expected_workers:
        raise ActiveWorkerCountMismatch(expected_workers, active_workers)
    return active_workers


if __name__ == '__main__':
    all_args = argparse.ArgumentParser()
    all_args.add_argument("-u", "--user", required=True,
                          help="Username to connect as")
    all_args.add_argument("-p", "--password", required=True,
                          help="Password for the user")
    all_args.add_argument("-n", "--namespace", required=True,
                          help="Namespace the test is running in")
    all_args.add_argument("-w", "--workers", required=True,
                          help="Expected amount of workers to be present")

    args = vars(all_args.parse_args())

    expected_workers = int(args['workers'])
    retry_on_transient(
        lambda: require_active_workers(
            args['user'],
            args['password'],
            args['namespace'],
            expected_workers,
        ),
        "Trino readiness check",
    )

    print("Test check-active-workers.py succeeded!")

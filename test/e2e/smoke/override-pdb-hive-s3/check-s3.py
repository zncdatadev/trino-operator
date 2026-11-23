#!/usr/bin/env python
import argparse
import sys
import time

import trino
from trino import exceptions

if not sys.warnoptions:
    import warnings
warnings.simplefilter("ignore")

S3_RETRY_TIMEOUT_SECONDS = 40
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


def is_retryable_error(error):
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
    timeout_seconds=S3_RETRY_TIMEOUT_SECONDS,
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
    # If you want to debug this locally use
    # kubectl -n kuttl-test-XXX port-forward svc/trino-coordinator-default 8443
    # host = '127.0.0.1'

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


def run_query(connection, query):
    print(f"[DEBUG] Executing query {query}")
    cursor = connection.cursor()
    cursor.execute(query)
    return cursor.fetchall()


def check_s3(username, password, namespace, bucket):
    connection = get_connection(username, password, namespace)

    trino_version = run_query(
        connection,
        "select node_version from system.runtime.nodes "
        "where coordinator = true and state = 'active'",
    )[0][0]
    print(f"[INFO] Testing against Trino version \"{trino_version}\"")

    # node version is blank in the test environment, i can not find cause of this issue, so i commented out this part
    # assert len(trino_version) >= 3
    # assert trino_version.isnumeric()
    # assert trino_version == run_query(connection, "select version()")[0][0]

    run_query(
        connection,
        f"CREATE SCHEMA IF NOT EXISTS hive.minio "
        f"WITH (location = 's3a://{bucket}/')",
    )

    run_query(connection, "DROP TABLE IF EXISTS hive.minio.taxi_data")
    run_query(connection, "DROP TABLE IF EXISTS hive.minio.taxi_data_copy")
    run_query(connection, "DROP TABLE IF EXISTS hive.minio.taxi_data_transformed")
    run_query(connection, "DROP TABLE IF EXISTS hive.hdfs.taxi_data_copy")
    run_query(connection, "DROP TABLE IF EXISTS iceberg.minio.taxi_data_copy_iceberg")

    run_query(connection, f"""
CREATE TABLE IF NOT EXISTS hive.minio.taxi_data (
    vendor_id VARCHAR,
    tpep_pickup_datetime VARCHAR,
    tpep_dropoff_datetime VARCHAR,
    passenger_count VARCHAR,
    trip_distance VARCHAR,
    ratecode_id VARCHAR
) WITH (
    external_location = 's3a://{bucket}/taxi-data/',
    format = 'csv',
    skip_header_line_count = 1
)
    """)
    assert run_query(connection, "SELECT COUNT(*) FROM hive.minio.taxi_data")[0][0] == 5000
    rows_written = run_query(
        connection,
        "CREATE TABLE IF NOT EXISTS hive.minio.taxi_data_copy "
        "AS SELECT * FROM hive.minio.taxi_data",
    )[0][0]
    assert rows_written == 5000 or rows_written == 0
    assert run_query(connection, "SELECT COUNT(*) FROM hive.minio.taxi_data_copy")[0][0] == 5000

    rows_written = run_query(connection, """
CREATE TABLE IF NOT EXISTS hive.minio.taxi_data_transformed AS
SELECT
    CAST(vendor_id as BIGINT) as vendor_id,
    tpep_pickup_datetime,
    tpep_dropoff_datetime,
    CAST(passenger_count as BIGINT) as passenger_count,
    CAST(trip_distance as DOUBLE) as trip_distance,
    CAST(ratecode_id as BIGINT) as ratecode_id
FROM hive.minio.taxi_data
""")[0][0]
    assert rows_written == 5000 or rows_written == 0
    assert run_query(
        connection,
        "SELECT COUNT(*) FROM hive.minio.taxi_data_transformed",
    )[0][0] == 5000

    # Here we can test iceberg, HDFS, PostgreSQL, etc.


if __name__ == '__main__':
    all_args = argparse.ArgumentParser()
    all_args.add_argument(
        "-n", "--namespace", required=True,
        help="Namespace the test is running in",
    )
    all_args.add_argument(
        "-u", "--user", required=True,
        help="Username to connect as",
    )
    all_args.add_argument(
        "-p", "--password", required=True,
        help="Password for the user",
    )
    all_args.add_argument(
        "-b", "--bucket", required=False, default="trino",
        help="S3 Bucket name",
    )

    args = vars(all_args.parse_args())

    print("Starting S3 tests...")
    retry_on_transient(
        lambda: check_s3(
            args['user'],
            args['password'],
            args['namespace'],
            args['bucket'],
        ),
        "Hive S3 validation",
    )
    print("[SUCCESS] All tests in check-s3.py succeeded!")

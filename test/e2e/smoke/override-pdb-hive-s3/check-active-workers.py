#!/usr/bin/env python
import trino
import argparse
import sys

if not sys.warnoptions:
    import warnings
warnings.simplefilter("ignore")


def get_connection(username, password, namespace):
    host = 'test-trino-coordinator-default-0.test-trino-coordinator-default.' + namespace + '.svc.cluster.local'
    conn = trino.dbapi.connect(
        host=host,
        port=8080,
        user=username,
        http_scheme='http',
        # auth=trino.auth.BasicAuthentication(username, password),
        session_properties={"query_max_execution_time": "60s"},
    )
    conn._http_session.verify = False
    return conn


if __name__ == '__main__':
    # Construct an argument parser
    all_args = argparse.ArgumentParser()

    # Add arguments to the parser
    all_args.add_argument("-u", "--user", required=True,
                          help="Username to connect as")
    all_args.add_argument("-p", "--password", required=True,
                          help="Password for the user")
    all_args.add_argument("-n", "--namespace", required=True,
                          help="Namespace the test is running in")
    all_args.add_argument("-w", "--workers", required=True,
                          help="Expected amount of workers to be present")

    args = vars(all_args.parse_args())

    expected_workers = args['workers']
    conn = get_connection(args['user'], args['password'], args['namespace'])

    cursor = conn.cursor()

    cursor.execute("USE \"hive\".\"default\"")
    print("USE command executed successfully.")

    cursor.execute("SHOW SCHEMAS")
    schemas = cursor.fetchall()
    print("Schemas:", schemas)

    cursor.execute("SHOW TABLES from \"hive\".\"default\"")
    tables = cursor.fetchall()
    print("Tables:", tables)

    cursor.execute("SELECT COUNT(*) as nodes FROM system.runtime.nodes WHERE coordinator=false AND state='active'")

    (active_workers,) = cursor.fetchone()

    if int(active_workers) != int(expected_workers):
        print("Mismatch: [expected/active] workers [" + str(expected_workers) + "/" + str(active_workers) + "]")
        exit(-1)

    print("Test check-active-workers.py succeeded!")

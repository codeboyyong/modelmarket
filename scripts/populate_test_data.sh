#!/usr/bin/env sh
set -eu

MM_APP_ENV_ARG="${1:-dev}"

# shellcheck disable=SC1091
. scripts/load-db-env.sh "$MM_APP_ENV_ARG"

echo "Populating test data for MM_APP_ENV=$MM_APP_ENV MM_DB_DRIVER=$MM_DB_DRIVER MM_DB_NAME=$MM_DB_NAME"
run_sql_file "db/populate_test_data.sql"

#!/usr/bin/env sh
set -eu

MM_APP_ENV_ARG="${1:-dev}"

# shellcheck disable=SC1091
. scripts/load-db-env.sh "$MM_APP_ENV_ARG"
echo "--------------------------------------------------"
echo "--Initializing database for MM_APP_ENV=$MM_APP_ENV"
echo "--MM_DB_HOST=$MM_DB_HOST MM_DB_DRIVER=$MM_DB_DRIVER MM_DB_NAME=$MM_DB_NAME"
case "$MM_APP_ENV" in
  dev|test|local)
    echo "--Resetting dev/test schema before init"
    run_sql_command "DROP SCHEMA IF EXISTS public CASCADE; CREATE SCHEMA public;"
    ;;
  *)
    echo "--Non-dev environment: running create-if-missing schema SQL without reset"
    ;;
esac
run_sql_file "db/init_db.sql"

echo "--------------------------------------------------"

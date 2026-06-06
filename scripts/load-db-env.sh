#!/usr/bin/env sh
set -eu

APP_ENV_ARG="${1:-dev}"
export APP_ENV="$APP_ENV_ARG"

load_env_file() {
  file="$1"
  if [ -f "$file" ]; then
    set -a
    # shellcheck disable=SC1090
    . "$file"
    set +a
  fi
}

load_env_file ".env"
load_env_file ".env.${APP_ENV}"
load_env_file "deploy/env/${APP_ENV}.env"

export MM_DB_DRIVER="${MM_DB_DRIVER:-postgres}"
export MM_DB_HOST="${MM_DB_HOST:-localhost}"
export MM_DB_PORT="${MM_DB_PORT:-5432}"
export MM_DB_NAME="${MM_DB_NAME:-model_market}"
export MM_DB_USER="${MM_DB_USER:-model_market}"
if [ -z "${MM_DB_PASSWORD+x}" ]; then
  export MM_DB_PASSWORD="model_market"
else
  export MM_DB_PASSWORD
fi

if [ -z "${MM_DB_SSL_MODE:-}" ]; then
  case "$APP_ENV" in
    prod|production|qa|staging)
      export MM_DB_SSL_MODE="require"
      ;;
    *)
      export MM_DB_SSL_MODE="disable"
      ;;
  esac
fi

run_sql_file() {
  sql_file="$1"
  case "$MM_DB_DRIVER" in
    postgres|postgresql|pgx)
      if ! command -v psql >/dev/null 2>&1; then
        echo "psql is required to run $sql_file for MM_DB_DRIVER=$MM_DB_DRIVER." >&2
        exit 1
      fi
      database_url="${MM_DATABASE_URL:-}"
      if [ -n "$database_url" ]; then
        PGSSLMODE="$MM_DB_SSL_MODE" psql "$database_url" -v ON_ERROR_STOP=1 -f "$sql_file"
      else
        PGPASSWORD="$MM_DB_PASSWORD" PGSSLMODE="$MM_DB_SSL_MODE" psql \
          -h "$MM_DB_HOST" \
          -p "$MM_DB_PORT" \
          -U "$MM_DB_USER" \
          -d "$MM_DB_NAME" \
          -v ON_ERROR_STOP=1 \
          -f "$sql_file"
      fi
      ;;
    *)
      echo "MM_DB_DRIVER=$MM_DB_DRIVER is not supported by this script yet." >&2
      echo "The SQL files are intentionally generic, but execution adapters must be added per database driver." >&2
      exit 1
      ;;
  esac
}

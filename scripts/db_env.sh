#!/usr/bin/env sh
set -eu

ENV_NAME="${1:-dev}"
ENV_FILE="config/env/${ENV_NAME}.env"

if [ ! -f "$ENV_FILE" ]; then
  echo "Missing environment file: $ENV_FILE" >&2
  echo "Usage: $0 dev|qa|prod" >&2
  exit 1
fi

set -a
# shellcheck disable=SC1090
. "$ENV_FILE"
set +a

MM_DB_DRIVER="${MM_DB_DRIVER:-postgres}"
MM_DB_HOST="${MM_DB_HOST:-localhost}"
MM_DB_PORT="${MM_DB_PORT:-5432}"
MM_DB_NAME="${MM_DB_NAME:-model_market}"
MM_DB_USER="${MM_DB_USER:-model_market}"
MM_DB_PASSWORD="${MM_DB_PASSWORD:-}"
MM_DB_SSL_MODE="${MM_DB_SSL_MODE:-disable}"
MM_DATABASE_URL="${MM_DATABASE_URL:-}"

if [ -z "$MM_DATABASE_URL" ] && [ "$MM_DB_DRIVER" = "postgres" ]; then
  if [ -n "$MM_DB_PASSWORD" ]; then
    MM_DATABASE_URL="postgres://${MM_DB_USER}:${MM_DB_PASSWORD}@${MM_DB_HOST}:${MM_DB_PORT}/${MM_DB_NAME}?sslmode=${MM_DB_SSL_MODE}"
  else
    MM_DATABASE_URL="postgres://${MM_DB_USER}@${MM_DB_HOST}:${MM_DB_PORT}/${MM_DB_NAME}?sslmode=${MM_DB_SSL_MODE}"
  fi
fi

export ENV_NAME MM_DB_DRIVER MM_DB_HOST MM_DB_PORT MM_DB_NAME MM_DB_USER MM_DB_PASSWORD MM_DB_SSL_MODE MM_DATABASE_URL

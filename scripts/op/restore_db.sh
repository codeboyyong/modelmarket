#!/usr/bin/env sh
set -eu

usage() {
  echo "Usage: $0 [dev|qa|prod] backup.dump [--yes]" >&2
  echo "Example: $0 dev backups/model_market-dev-20260829T010000Z.dump" >&2
  echo "WARNING: restore deletes and replaces objects in the target database." >&2
}

SCRIPT_DIR=$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)
REPO_ROOT=$(CDPATH= cd -- "$SCRIPT_DIR/../.." && pwd)
ENV_NAME_ARG="${1:-}"
BACKUP_ARG="${2:-}"
CONFIRM_FLAG="${3:-}"

if [ "$ENV_NAME_ARG" = "-h" ] || [ "$ENV_NAME_ARG" = "--help" ]; then
  usage
  exit 0
fi
case "$ENV_NAME_ARG" in
  dev|qa|prod|test|local) ;;
  *)
    usage
    exit 2
    ;;
esac
if [ -z "$BACKUP_ARG" ]; then
  usage
  exit 2
fi
if [ -n "$CONFIRM_FLAG" ] && [ "$CONFIRM_FLAG" != "--yes" ]; then
  usage
  exit 2
fi
if ! command -v pg_restore >/dev/null 2>&1; then
  echo "pg_restore is required. Install the PostgreSQL client tools first." >&2
  exit 1
fi

cd "$REPO_ROOT"
case "$BACKUP_ARG" in
  /*) BACKUP_PATH="$BACKUP_ARG" ;;
  *) BACKUP_PATH="$REPO_ROOT/$BACKUP_ARG" ;;
esac
if [ ! -f "$BACKUP_PATH" ]; then
  echo "Backup archive not found: $BACKUP_PATH" >&2
  exit 1
fi

# Validate the archive before asking for confirmation or touching the database.
if ! pg_restore --list "$BACKUP_PATH" >/dev/null; then
  echo "Invalid or unreadable pg_dump custom archive: $BACKUP_PATH" >&2
  exit 1
fi

# shellcheck disable=SC1091
. scripts/load-db-env.sh "$ENV_NAME_ARG"

echo "DANGER: this will replace database objects and data."
echo "Environment: $ENV_NAME_ARG"
echo "Target:      $MM_DB_USER@$MM_DB_HOST:$MM_DB_PORT/$MM_DB_NAME"
echo "Archive:     $BACKUP_PATH"

if [ "$CONFIRM_FLAG" != "--yes" ]; then
  printf "Type the target database name '%s' to continue: " "$MM_DB_NAME"
  IFS= read -r CONFIRMATION
  if [ "$CONFIRMATION" != "$MM_DB_NAME" ]; then
    echo "Restore cancelled." >&2
    exit 1
  fi
fi

echo "Restoring database..."
if [ -n "${MM_DATABASE_URL:-}" ]; then
  pg_restore \
    --dbname "$MM_DATABASE_URL" \
    --clean \
    --if-exists \
    --no-owner \
    --no-privileges \
    --exit-on-error \
    --single-transaction \
    "$BACKUP_PATH"
else
  PGPASSWORD="${MM_DB_PASSWORD:-}" PGSSLMODE="${MM_DB_SSL_MODE:-disable}" \
    pg_restore \
      --host "$MM_DB_HOST" \
      --port "$MM_DB_PORT" \
      --username "$MM_DB_USER" \
      --dbname "$MM_DB_NAME" \
      --clean \
      --if-exists \
      --no-owner \
      --no-privileges \
      --exit-on-error \
      --single-transaction \
      "$BACKUP_PATH"
fi

echo "Restore completed successfully."
echo "Target database: $MM_DB_NAME"


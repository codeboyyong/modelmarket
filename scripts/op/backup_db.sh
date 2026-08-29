#!/usr/bin/env sh
set -eu

usage() {
  echo "Usage: $0 [dev|qa|prod] [output.dump]" >&2
  echo "Example: $0 dev" >&2
  echo "Example: $0 prod /secure/backups/model-market-prod.dump" >&2
}

SCRIPT_DIR=$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)
REPO_ROOT=$(CDPATH= cd -- "$SCRIPT_DIR/../.." && pwd)
ENV_NAME_ARG="${1:-dev}"

case "$ENV_NAME_ARG" in
  dev|qa|prod|test|local) ;;
  -h|--help)
    usage
    exit 0
    ;;
  *)
    usage
    exit 2
    ;;
esac

if ! command -v pg_dump >/dev/null 2>&1; then
  echo "pg_dump is required. Install the PostgreSQL client tools first." >&2
  exit 1
fi
if ! command -v pg_restore >/dev/null 2>&1; then
  echo "pg_restore is required to validate the completed archive." >&2
  exit 1
fi

cd "$REPO_ROOT"
# shellcheck disable=SC1091
. scripts/load-db-env.sh "$ENV_NAME_ARG"

TIMESTAMP=$(date -u +%Y%m%dT%H%M%SZ)
DEFAULT_OUTPUT="$REPO_ROOT/backups/${MM_DB_NAME}-${ENV_NAME_ARG}-${TIMESTAMP}.dump"
OUTPUT_PATH="${2:-$DEFAULT_OUTPUT}"
case "$OUTPUT_PATH" in
  /*) ;;
  *) OUTPUT_PATH="$REPO_ROOT/$OUTPUT_PATH" ;;
esac

if [ -e "$OUTPUT_PATH" ]; then
  echo "Refusing to overwrite existing backup: $OUTPUT_PATH" >&2
  exit 1
fi

OUTPUT_DIR=$(dirname -- "$OUTPUT_PATH")
mkdir -p "$OUTPUT_DIR"
PARTIAL_PATH="${OUTPUT_PATH}.partial"
cleanup_partial() {
  rm -f -- "$PARTIAL_PATH"
}
trap cleanup_partial EXIT HUP INT TERM

echo "Backing up PostgreSQL database '$MM_DB_NAME' for environment '$ENV_NAME_ARG'..."
echo "Destination: $OUTPUT_PATH"

if [ -n "${MM_DATABASE_URL:-}" ]; then
  pg_dump \
    --dbname "$MM_DATABASE_URL" \
    --format=custom \
    --compress=6 \
    --no-owner \
    --no-privileges \
    --file "$PARTIAL_PATH"
else
  PGPASSWORD="${MM_DB_PASSWORD:-}" PGSSLMODE="${MM_DB_SSL_MODE:-disable}" \
    pg_dump \
      --host "$MM_DB_HOST" \
      --port "$MM_DB_PORT" \
      --username "$MM_DB_USER" \
      --dbname "$MM_DB_NAME" \
      --format=custom \
      --compress=6 \
      --no-owner \
      --no-privileges \
      --file "$PARTIAL_PATH"
fi

# Reading the table of contents catches truncated or invalid custom archives.
pg_restore --list "$PARTIAL_PATH" >/dev/null
mv -- "$PARTIAL_PATH" "$OUTPUT_PATH"
trap - EXIT HUP INT TERM

echo "Backup completed successfully."
echo "Archive: $OUTPUT_PATH"
echo "Size: $(du -h "$OUTPUT_PATH" | awk '{print $1}')"


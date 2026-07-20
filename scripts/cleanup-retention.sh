#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "$0")/.." && pwd)"
ENVIRONMENT="${1:-dev}"
MODE="${2:---dry-run}"

# shellcheck source=scripts/load-db-env.sh
source "$ROOT_DIR/scripts/load-db-env.sh" "$ENVIRONMENT"

ARGS=()
if [[ "$MODE" == "--apply" ]]; then
  ARGS+=(--apply)
elif [[ "$MODE" != "--dry-run" ]]; then
  echo "Usage: scripts/cleanup-retention.sh [environment] [--dry-run|--apply]" >&2
  exit 2
fi

cd "$ROOT_DIR/backend"
GOCACHE="$ROOT_DIR/.cache/go-build" GOMODCACHE="$ROOT_DIR/.cache/go-mod" go run ./cmd/retention-cleanup "${ARGS[@]}"

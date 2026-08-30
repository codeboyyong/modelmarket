#!/usr/bin/env sh
set -eu

MM_APP_ENV_ARG="${1:-dev}"
API_BASE="${API_BASE:-http://localhost:8080}"

# shellcheck disable=SC1091
. scripts/load-db-env.sh "$MM_APP_ENV_ARG"

echo "Checking backend health and readiness..."
curl -fsS "$API_BASE/healthz" >/dev/null
curl -fsS "$API_BASE/readyz" >/dev/null

if [ -n "${FACEBOOK_CLIENT_ID:-}" ] && [ -n "${FACEBOOK_CLIENT_SECRET:-}" ] && [ -n "${FACEBOOK_REDIRECT_URI:-}" ]; then
  status="$(curl -sS -o /dev/null -w '%{http_code}' "$API_BASE/api/v1/auth/oauth/facebook/start")"
  [ "$status" = "302" ] || { echo "Facebook OAuth start returned HTTP $status" >&2; exit 1; }
  echo "Facebook OAuth start passed. Complete the provider consent in a browser."
else
  echo "Facebook credentials absent; configuration check skipped."
fi

if [ "${PAYMENT_PROVIDER_MODE:-mock}" = "stripe" ]; then
  [ -n "${STRIPE_SECRET_KEY:-}" ] && [ -n "${STRIPE_WEBHOOK_SECRET:-}" ] || { echo "Stripe mode requires STRIPE_SECRET_KEY and STRIPE_WEBHOOK_SECRET." >&2; exit 1; }
  echo "Stripe configuration is present. Use Stripe CLI to exercise the signed webhook."
else
  echo "Stripe remains in mock mode."
fi

case "${OBJECT_STORAGE_PROVIDER:-local}" in
  local)
    scripts/demo-smoke.sh "$MM_APP_ENV_ARG"
    ;;
  s3)
    [ -n "${MM_ASSET_BUCKET:-}" ] && [ -n "${AWS_REGION:-}" ] || { echo "S3 mode requires MM_ASSET_BUCKET and AWS_REGION." >&2; exit 1; }
    echo "S3 configuration is present. Create an upload from the Workbench to verify bucket permissions."
    ;;
  *) echo "OBJECT_STORAGE_PROVIDER must be local or s3." >&2; exit 1 ;;
esac

echo "Integration verification passed."

#!/usr/bin/env sh
set -eu

MM_APP_ENV_ARG="${1:-dev}"
API_BASE="${API_BASE:-http://localhost:8080}"
OPENAI_SMOKE_MODEL="${OPENAI_SMOKE_MODEL:-openai/gpt-5.6-luna}"

# shellcheck disable=SC1091
. scripts/load-db-env.sh "$MM_APP_ENV_ARG"

if [ -z "${OPENAI_API_KEY:-}" ]; then
  echo "OPENAI_API_KEY is not set in the loaded environment." >&2
  exit 1
fi

json_field() {
  node -e "const fs=require('fs'); const data=JSON.parse(fs.readFileSync(0,'utf8')); const path=process.argv[1].split('.'); let v=data; for (const p of path) v=v?.[p]; if (v === undefined || v === null) process.exit(1); console.log(typeof v === 'object' ? JSON.stringify(v) : v);" "$1"
}

echo "Checking the Model Market backend..."
curl -fsS "$API_BASE/readyz" >/dev/null

echo "Creating a temporary project API key..."
key_response="$(curl -fsS -X POST "$API_BASE/api/v1/api-keys" \
  -H "Content-Type: application/json" \
  -d '{"project_id":"project-demo","name":"OpenAI smoke key"}')"
api_key="$(printf '%s' "$key_response" | json_field api_key)"

echo "Requesting a small completion through Model Market..."
chat_response="$(curl -fsS -X POST "$API_BASE/api/v1/chat/completions" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $api_key" \
  -d "{\"model\":\"$OPENAI_SMOKE_MODEL\",\"messages\":[{\"role\":\"user\",\"content\":\"Reply with exactly OPENAI_OK\"}],\"parameters\":{\"max_completion_tokens\":16}}")"

content="$(printf '%s' "$chat_response" | json_field choices.0.message.content)"
request_id="$(printf '%s' "$chat_response" | json_field id)"

if [ -z "$content" ]; then
  echo "OpenAI returned an empty completion." >&2
  exit 1
fi

echo "OpenAI smoke passed (request $request_id)."

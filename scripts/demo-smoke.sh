#!/usr/bin/env sh
set -eu

MM_APP_ENV_ARG="${1:-dev}"
API_BASE="${API_BASE:-http://localhost:8080}"

# shellcheck disable=SC1091
. scripts/load-db-env.sh "$MM_APP_ENV_ARG"

json_field() {
  node -e "const fs=require('fs'); const data=JSON.parse(fs.readFileSync(0,'utf8')); const path=process.argv[1].split('.'); let v=data; for (const p of path) v=v?.[p]; if (v === undefined || v === null) process.exit(1); if (typeof v === 'object') console.log(JSON.stringify(v)); else console.log(v);" "$1"
}

echo "Checking backend health..."
curl -fsS "$API_BASE/healthz" >/dev/null
curl -fsS "$API_BASE/readyz" >/dev/null

echo "Running mock payment purchase..."
curl -fsS -X POST "$API_BASE/api/v1/credits/purchase" \
  -H "Content-Type: application/json" \
  -d '{"user_id":"user-yong-zhao","amount_cents":100,"payment_method":"credit_card"}' >/dev/null

echo "Creating a demo API key..."
key_response="$(curl -fsS -X POST "$API_BASE/api/v1/api-keys" \
  -H "Content-Type: application/json" \
  -d '{"project_id":"project-demo","name":"Demo smoke key"}')"
api_key="$(printf '%s' "$key_response" | json_field api_key)"

generate_media() {
  label="$1"
  model="$2"
  prompt="$3"
  parameters="$4"
  output_file="$5"

  echo "Generating a mock $label artifact..."
  chat_response="$(curl -fsS -X POST "$API_BASE/api/v1/chat/completions" \
    -H "Content-Type: application/json" \
    -H "Authorization: Bearer $api_key" \
    -d "{\"model\":\"$model\",\"conversation_id\":\"conversation-demo\",\"messages\":[{\"role\":\"user\",\"content\":\"$prompt\"}],\"parameters\":$parameters}")"
  artifact_url="$(printf '%s' "$chat_response" | json_field artifacts.0.download_url)"
  artifact_id="$(printf '%s' "$chat_response" | json_field artifacts.0.id)"

  echo "Verifying $label artifact download..."
  curl -fsS "$artifact_url" >"$output_file"

  echo "Verifying $label database record..."
  run_sql_command "select id, asset_type, asset_origin, download_url from user_workbench_assets where id = '$artifact_id';"
  echo "$label artifact: $artifact_url"
}

generate_media "image" "mock-creative-default" "Create a clean local E2E demo image" '{"size":"1024x1024","aspect_ratio":"1:1","output_count":1}' "/tmp/model-market-demo-image.svg"
generate_media "video" "x-ai/grok-imagine-video" "Create a five second local E2E demo video" '{"resolution":"720p","duration_seconds":5,"aspect_ratio":"16:9"}' "/tmp/model-market-demo-video.json"
generate_media "audio" "google/lyria-3-clip-preview" "Create a short local E2E demo audio clip" '{"duration_seconds":30,"format":"mp3"}' "/tmp/model-market-demo-audio.json"

echo "Demo smoke passed."

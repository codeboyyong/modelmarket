#!/usr/bin/env sh
set -eu

IMAGE_REGISTRY="${IMAGE_REGISTRY:-}"
IMAGE_NAMESPACE="${IMAGE_NAMESPACE:-model-market}"
IMAGE_TAG="${IMAGE_TAG:-phase1}"
PLATFORMS="${PLATFORMS:-linux/amd64,linux/arm64}"
BUILDER_NAME="${BUILDER_NAME:-model-market-builder}"
OUTPUT_MODE="${OUTPUT_MODE:-push}"

if [ "$OUTPUT_MODE" != "push" ] && [ "$OUTPUT_MODE" != "load" ]; then
  echo "OUTPUT_MODE must be 'push' or 'load'." >&2
  exit 1
fi

if [ "$OUTPUT_MODE" = "load" ] && printf '%s' "$PLATFORMS" | grep -q ','; then
  echo "OUTPUT_MODE=load supports only one platform. Set PLATFORMS=linux/arm64 or PLATFORMS=linux/amd64." >&2
  exit 1
fi

if ! docker buildx version >/dev/null 2>&1; then
  echo "Docker buildx is required for multi-architecture builds." >&2
  echo "Install Docker Buildx or use Docker Desktop/Colima with buildx enabled." >&2
  exit 1
fi

if ! docker buildx inspect "$BUILDER_NAME" >/dev/null 2>&1; then
  docker buildx create --name "$BUILDER_NAME" --use >/dev/null
else
  docker buildx use "$BUILDER_NAME" >/dev/null
fi

docker buildx inspect --bootstrap >/dev/null

prefix=""
if [ -n "$IMAGE_REGISTRY" ]; then
  prefix="${IMAGE_REGISTRY}/"
fi

backend_image="${prefix}${IMAGE_NAMESPACE}/backend:${IMAGE_TAG}"
frontend_image="${prefix}${IMAGE_NAMESPACE}/frontend:${IMAGE_TAG}"

output_flag="--push"
if [ "$OUTPUT_MODE" = "load" ]; then
  output_flag="--load"
fi

echo "Building backend image: $backend_image"
docker buildx build \
  --platform "$PLATFORMS" \
  -f backend/Dockerfile \
  -t "$backend_image" \
  "$output_flag" \
  .

echo "Building frontend image: $frontend_image"
docker buildx build \
  --platform "$PLATFORMS" \
  -f frontend/Dockerfile \
  -t "$frontend_image" \
  "$output_flag" \
  .

echo "Done."

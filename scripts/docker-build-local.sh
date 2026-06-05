#!/usr/bin/env sh
set -eu

PLATFORM="${PLATFORM:-$(uname -m)}"

case "$PLATFORM" in
  arm64|aarch64)
    PLATFORM="linux/arm64"
    ;;
  x86_64|amd64)
    PLATFORM="linux/amd64"
    ;;
esac

IMAGE_TAG="${IMAGE_TAG:-local}" \
PLATFORMS="$PLATFORM" \
OUTPUT_MODE=load \
scripts/docker-build-multiarch.sh

#!/usr/bin/env sh
set -eu

if docker compose version >/dev/null 2>&1; then
  docker compose "$@"
elif command -v docker-compose >/dev/null 2>&1; then
  docker-compose "$@"
else
  echo "Docker is installed, but Docker Compose is not available." >&2
  echo "Install the Docker Compose plugin or legacy docker-compose, then retry." >&2
  exit 1
fi

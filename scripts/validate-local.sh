#!/usr/bin/env sh
set -eu

# Validate the local codebase without Docker, PostgreSQL, or Redis:
# backend unit tests, frontend TypeScript typecheck, and frontend production build.
mkdir -p .cache/go-build .cache/go-mod
(cd backend && GOCACHE="$(pwd)/../.cache/go-build" GOMODCACHE="$(pwd)/../.cache/go-mod" go test ./...)
(cd frontend && npm run typecheck && npm run build)

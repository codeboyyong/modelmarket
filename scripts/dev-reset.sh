#!/usr/bin/env sh
set -eu

scripts/compose.sh down -v
scripts/compose.sh up --build

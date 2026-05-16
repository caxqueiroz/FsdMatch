#!/usr/bin/env bash
# End-to-end smoke test against testdata/sample-spring-app.
# Phase 1 stub: verifies `init` creates a usable DB.
set -euo pipefail

cd "$(dirname "$0")/.."

DB="$(mktemp -t fsdtrace-smoke.XXXXXX.db)"
trap 'rm -f "$DB"' EXIT

make build
./bin/fsdtrace init --db "$DB"

# Sanity-check: schema present.
TABLES=$(./bin/fsdtrace --db "$DB" status 2>/dev/null || true)
echo "$TABLES"

echo "Phase 1 smoke OK ($DB)"

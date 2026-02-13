#!/usr/bin/env sh
# run-smoke.sh — Run E2E smoke tests against a running API server.
# Usage: ./scripts/run-smoke.sh
#
# Environment variables (all optional, have sensible defaults):
#   API_BASE_URL     — base URL of the API (default: http://localhost:8080)
#   SMOKE_TOKEN      — Bearer token for authenticated requests
#   SMOKE_PROFILE_ID — profile ID to use (auto-detected if empty)
#
# Examples:
#   ./scripts/run-smoke.sh
#   API_BASE_URL=https://my-app.onrender.com SMOKE_TOKEN=eyJ... ./scripts/run-smoke.sh

set -e

ROOT="$(cd "$(dirname "$0")/.." && pwd)"

export API_BASE_URL="${API_BASE_URL:-http://localhost:8080}"
export SMOKE_TOKEN="${SMOKE_TOKEN:-}"
export SMOKE_PROFILE_ID="${SMOKE_PROFILE_ID:-}"

echo "=== Health Hub Smoke Tests ==="
echo "  API_BASE_URL=$API_BASE_URL"
if [ -n "$SMOKE_TOKEN" ]; then
    echo "  SMOKE_TOKEN=(set)"
else
    echo "  SMOKE_TOKEN=(not set)"
fi
if [ -n "$SMOKE_PROFILE_ID" ]; then
    echo "  SMOKE_PROFILE_ID=$SMOKE_PROFILE_ID"
else
    echo "  SMOKE_PROFILE_ID=(auto-detect)"
fi
echo ""

cd "$ROOT/server"
exec go run ./cmd/smoke

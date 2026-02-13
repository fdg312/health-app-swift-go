#!/usr/bin/env sh
# dev-server.sh — Start the API server with safe local defaults.
# Usage: ./scripts/dev-server.sh
#
# Override any variable by exporting it before running, e.g.:
#   DATABASE_URL=postgres://... ./scripts/dev-server.sh

set -e

ROOT="$(cd "$(dirname "$0")/.." && pwd)"

# ---- Safe local defaults (no S3, no SMTP, no strict auth) ----
export APP_ENV="${APP_ENV:-local}"
export PORT="${PORT:-8080}"
export BLOB_MODE="${BLOB_MODE:-local}"
export REPORTS_MODE="${REPORTS_MODE:-local}"
export EMAIL_SENDER_MODE="${EMAIL_SENDER_MODE:-local}"
export AUTH_MODE="${AUTH_MODE:-none}"
export AUTH_REQUIRED="${AUTH_REQUIRED:-0}"
export AI_MODE="${AI_MODE:-mock}"

echo "=== Health Hub Dev Server ==="
echo "  APP_ENV=$APP_ENV  PORT=$PORT"
echo "  AUTH_MODE=$AUTH_MODE  AUTH_REQUIRED=$AUTH_REQUIRED"
echo "  BLOB_MODE=$BLOB_MODE  REPORTS_MODE=$REPORTS_MODE"
echo "  EMAIL_SENDER_MODE=$EMAIL_SENDER_MODE  (OTP codes printed to console)"
echo "  DATABASE_URL=${DATABASE_URL:-(not set, using in-memory storage)}"
echo ""

cd "$ROOT/server"
exec go run ./cmd/api

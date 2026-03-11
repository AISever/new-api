#!/usr/bin/env bash
set -euo pipefail

REPO_DIR="${REPO_DIR:-$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)}"
DEFAULT_BRANCH="${DEFAULT_BRANCH:-codex/prod-live}"

usage() {
  cat <<'EOF'
Deploy the tencent test environment from a git branch.

Usage:
  ./ops/scripts/tencent-test-deploy.sh [branch]

Defaults:
  branch              codex/prod-live
  REPO_DIR            current repo root
  PROJECT_NAME        new-api-test
  APP_PORT            3001
  PG_DB               new-api-test
  IMAGE_NAME          new-api:tencent-test
  COMPOSE_FILE        docker-compose.tencent-test.yml

Recommended:
  Deploy a candidate `codex/release-*` branch to the tencent test environment first.
EOF
}

if [ "${1:-}" = "-h" ] || [ "${1:-}" = "--help" ]; then
  usage
  exit 0
fi

export REPO_DIR
export PROJECT_NAME="${PROJECT_NAME:-new-api-test}"
export APP_PORT="${APP_PORT:-3001}"
export PG_DB="${PG_DB:-new-api-test}"
export IMAGE_NAME="${IMAGE_NAME:-new-api:tencent-test}"
export COMPOSE_FILE="${COMPOSE_FILE:-docker-compose.tencent-test.yml}"
export APP_CONTAINER="${APP_CONTAINER:-new-api-test}"
export REDIS_CONTAINER="${REDIS_CONTAINER:-redis-test}"
export POSTGRES_CONTAINER="${POSTGRES_CONTAINER:-postgres-test}"
export VERSION_SUFFIX="${VERSION_SUFFIX:-+tencent-test}"
export DEPLOY_LABEL="tencent test environment"

exec "$REPO_DIR/ops/scripts/deploy-from-branch.sh" "${1:-$DEFAULT_BRANCH}"

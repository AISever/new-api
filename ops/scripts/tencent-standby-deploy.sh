#!/usr/bin/env bash
set -euo pipefail

REPO_DIR="${REPO_DIR:-$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)}"
DEFAULT_BRANCH="${DEFAULT_BRANCH:-codex/prod-live}"

usage() {
  cat <<'EOF'
Deploy the tencent standby environment from a git branch.

Usage:
  ./ops/scripts/tencent-standby-deploy.sh [branch]

Defaults:
  branch              codex/prod-live
  REPO_DIR            current repo root
  PROJECT_NAME        new-api
  BASE_COMPOSE_FILE   docker-compose.yml
  COMPOSE_FILE        docker-compose.override.yml
  IMAGE_NAME          new-api:codex-prod-live

Notes:
  This keeps the real tencent standby topology aligned with the current server:
  official `docker-compose.yml` as base, plus a generated override file for build/VERSION.
EOF
}

if [ "${1:-}" = "-h" ] || [ "${1:-}" = "--help" ]; then
  usage
  exit 0
fi

export REPO_DIR
export PROJECT_NAME="${PROJECT_NAME:-new-api}"
export IMAGE_NAME="${IMAGE_NAME:-new-api:codex-prod-live}"
export COMPOSE_MODE="override"
export BASE_COMPOSE_FILE="${BASE_COMPOSE_FILE:-docker-compose.yml}"
export COMPOSE_FILE="${COMPOSE_FILE:-docker-compose.override.yml}"
export DEPLOY_LABEL="tencent standby environment"

exec "$REPO_DIR/ops/scripts/deploy-from-branch.sh" "${1:-$DEFAULT_BRANCH}"

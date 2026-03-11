#!/usr/bin/env bash
set -euo pipefail

REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
BACKUP_DIR="${1:-}"

usage() {
  cat <<'EOF'
Restore the tencent test environment from a backup directory already present on the server.

Usage:
  ./ops/scripts/tencent-test-restore-from-backup.sh /opt/new-api-src/backups/origin/<timestamp>

Optional env:
  REPO_DIR             default: current repo root
  COMPOSE_PROJECT      default: new-api-test
  COMPOSE_FILE         default: <repo>/docker-compose.tencent-test.yml
  POSTGRES_CONTAINER   default: postgres-test
  PG_DB                default: new-api-test
  APP_PORT             default: 3001
  PRE_RESTORE_BACKUP   default: true
  SERVER_ADDRESS       optional
EOF
}

if [ "${1:-}" = "-h" ] || [ "${1:-}" = "--help" ]; then
  usage
  exit 0
fi

if [ -z "$BACKUP_DIR" ]; then
  echo "ERROR: missing BACKUP_DIR" >&2
  usage >&2
  exit 2
fi

REPO_DIR="${REPO_DIR:-$REPO_ROOT}"
COMPOSE_PROJECT="${COMPOSE_PROJECT:-new-api-test}"
COMPOSE_FILE="${COMPOSE_FILE:-$REPO_DIR/docker-compose.tencent-test.yml}"
POSTGRES_CONTAINER="${POSTGRES_CONTAINER:-postgres-test}"
PG_DB="${PG_DB:-new-api-test}"
APP_PORT="${APP_PORT:-3001}"

if [ ! -f "$COMPOSE_FILE" ]; then
  echo "ERROR: missing compose file: $COMPOSE_FILE" >&2
  echo "Hint: run ./ops/scripts/tencent-test-deploy.sh once first." >&2
  exit 2
fi

export REPO_DIR COMPOSE_PROJECT COMPOSE_FILE POSTGRES_CONTAINER PG_DB APP_PORT
export PG_SUPERUSER="${PG_SUPERUSER:-newapi}"
exec "$REPO_ROOT/ops/scripts/tencent-restore-from-backup.sh" "$BACKUP_DIR"

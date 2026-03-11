#!/usr/bin/env bash
set -euo pipefail

REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"

usage() {
  cat <<'EOF'
Backup kkidc production data and upload to the tencent standby server.

Optional env file:
  ENV_FILE              default: .tencent/.env.lighthouse

Required env:
  ORIGIN_SSH_HOST        e.g. 202.140.142.149
  LIGHTHOUSE_SSH_HOST    e.g. 123.206.229.105 (or set IP_1 in ENV_FILE)
  LIGHTHOUSE_SSH_KEY     e.g. .tencent/lighthouse-shanghai.pem (auto-detects common paths)

Origin SSH auth (choose one):
  ORIGIN_SSH_PASS        SSH password (uses sshpass)
  ORIGIN_SSH_KEY         SSH private key path

Optional env:
  ORIGIN_SSH_USER        default: root
  ORIGIN_APP_DIR         default: /opt/new-api
  ORIGIN_PG_CONTAINER    default: new-api-postgres
  ORIGIN_PG_DB           default: new-api
  ORIGIN_PG_USER         default: newapi

  LIGHTHOUSE_SSH_USER    default: root
  LIGHTHOUSE_DEST_BASE   default: /opt/new-api-src/backups/origin
  LOCAL_OUTPUT_DIR       default: /tmp/new-api-origin-backup-<timestamp>
  KEEP_LOCAL             default: true (set false to cleanup LOCAL_OUTPUT_DIR)
  ORIGIN_KEEP_REMOTE     default: true (set false to cleanup origin backup dir)

Outputs:
  Prints the tencent backup directory path.
EOF
}

require_env() {
  local name="$1"
  if [ -z "${!name:-}" ]; then
    echo "ERROR: missing env $name" >&2
    usage >&2
    exit 2
  fi
}

set_if_unset() {
  local name="$1"
  local value="$2"
  if [ -z "${!name:-}" ] && [ -n "$value" ]; then
    printf -v "$name" '%s' "$value"
    export "$name"
  fi
}

load_env_file() {
  local file="$1"
  [ -f "$file" ] || return 0

  while IFS= read -r line || [ -n "$line" ]; do
    line="${line%%#*}"
    line="${line%"${line##*[![:space:]]}"}"
    line="${line#"${line%%[![:space:]]*}"}"
    [ -z "$line" ] && continue

    if [[ "$line" =~ ^([A-Za-z_][A-Za-z0-9_]*)[[:space:]]*=[[:space:]]*(.*)$ ]]; then
      local key="${BASH_REMATCH[1]}"
      local value="${BASH_REMATCH[2]}"
      value="${value%\"}"
      value="${value#\"}"
      set_if_unset "$key" "$value"
    fi
  done < "$file"
}

ENV_FILE="${ENV_FILE:-$REPO_ROOT/.tencent/.env.lighthouse}"
load_env_file "$ENV_FILE"

set_if_unset LIGHTHOUSE_SSH_HOST "${IP_1:-}"
if [ -z "${LIGHTHOUSE_SSH_KEY:-}" ]; then
  if [ -f "$REPO_ROOT/.tencent/lighthouse-shanghai.pem" ]; then
    LIGHTHOUSE_SSH_KEY="$REPO_ROOT/.tencent/lighthouse-shanghai.pem"
  elif [ -f "$REPO_ROOT/.tencent/lighthouse.pem" ]; then
    LIGHTHOUSE_SSH_KEY="$REPO_ROOT/.tencent/lighthouse.pem"
  fi
fi

if [ -n "${LIGHTHOUSE_SSH_KEY:-}" ] && [ ! -f "$LIGHTHOUSE_SSH_KEY" ] && [ -f "$REPO_ROOT/$LIGHTHOUSE_SSH_KEY" ]; then
  LIGHTHOUSE_SSH_KEY="$REPO_ROOT/$LIGHTHOUSE_SSH_KEY"
fi
if [ -n "${ORIGIN_SSH_KEY:-}" ] && [ ! -f "$ORIGIN_SSH_KEY" ] && [ -f "$REPO_ROOT/$ORIGIN_SSH_KEY" ]; then
  ORIGIN_SSH_KEY="$REPO_ROOT/$ORIGIN_SSH_KEY"
fi

require_env ORIGIN_SSH_HOST
require_env LIGHTHOUSE_SSH_HOST
require_env LIGHTHOUSE_SSH_KEY

ORIGIN_SSH_USER="${ORIGIN_SSH_USER:-root}"
ORIGIN_APP_DIR="${ORIGIN_APP_DIR:-/opt/new-api}"
ORIGIN_PG_CONTAINER="${ORIGIN_PG_CONTAINER:-new-api-postgres}"
ORIGIN_PG_DB="${ORIGIN_PG_DB:-new-api}"
ORIGIN_PG_USER="${ORIGIN_PG_USER:-newapi}"

LIGHTHOUSE_SSH_USER="${LIGHTHOUSE_SSH_USER:-root}"
LIGHTHOUSE_DEST_BASE="${LIGHTHOUSE_DEST_BASE:-/opt/new-api-src/backups/origin}"
KEEP_LOCAL="${KEEP_LOCAL:-true}"
ORIGIN_KEEP_REMOTE="${ORIGIN_KEEP_REMOTE:-true}"

TS="$(date +%Y%m%d-%H%M%S)"
LOCAL_OUTPUT_DIR="${LOCAL_OUTPUT_DIR:-/tmp/new-api-origin-backup-${TS}}"
mkdir -p "$LOCAL_OUTPUT_DIR"

origin_ssh() {
  local cmd=("$@")
  if [ -n "${ORIGIN_SSH_PASS:-}" ]; then
    if ! command -v sshpass >/dev/null 2>&1; then
      echo "ERROR: sshpass not installed, but ORIGIN_SSH_PASS is set" >&2
      exit 2
    fi
    SSHPASS="$ORIGIN_SSH_PASS" sshpass -e ssh -o StrictHostKeyChecking=no -o ConnectTimeout=15 \
      "${ORIGIN_SSH_USER}@${ORIGIN_SSH_HOST}" "${cmd[@]}"
  else
    if [ -z "${ORIGIN_SSH_KEY:-}" ]; then
      echo "ERROR: set ORIGIN_SSH_PASS or ORIGIN_SSH_KEY" >&2
      exit 2
    fi
    ssh -i "$ORIGIN_SSH_KEY" -o StrictHostKeyChecking=no -o ConnectTimeout=15 \
      "${ORIGIN_SSH_USER}@${ORIGIN_SSH_HOST}" "${cmd[@]}"
  fi
}

origin_scp_get() {
  local remote_path="$1"
  local local_path="$2"
  if [ -n "${ORIGIN_SSH_PASS:-}" ]; then
    if ! command -v sshpass >/dev/null 2>&1; then
      echo "ERROR: sshpass not installed, but ORIGIN_SSH_PASS is set" >&2
      exit 2
    fi
    SSHPASS="$ORIGIN_SSH_PASS" sshpass -e scp -o StrictHostKeyChecking=no -o ConnectTimeout=15 \
      "${ORIGIN_SSH_USER}@${ORIGIN_SSH_HOST}:${remote_path}" "$local_path"
  else
    scp -i "$ORIGIN_SSH_KEY" -o StrictHostKeyChecking=no -o ConnectTimeout=15 \
      "${ORIGIN_SSH_USER}@${ORIGIN_SSH_HOST}:${remote_path}" "$local_path"
  fi
}

lighthouse_ssh() {
  ssh -i "$LIGHTHOUSE_SSH_KEY" -o StrictHostKeyChecking=no -o ConnectTimeout=15 \
    "${LIGHTHOUSE_SSH_USER}@${LIGHTHOUSE_SSH_HOST}" "$@"
}

lighthouse_scp_put() {
  local local_path="$1"
  local remote_path="$2"
  scp -i "$LIGHTHOUSE_SSH_KEY" -o StrictHostKeyChecking=no -o ConnectTimeout=15 \
    "$local_path" "${LIGHTHOUSE_SSH_USER}@${LIGHTHOUSE_SSH_HOST}:${remote_path}"
}

echo "==[1/3] Create backup bundle on origin ==" >&2
REMOTE_BACKUP_DIR="${ORIGIN_APP_DIR}/backups/${TS}"
REMOTE_BUNDLE="${REMOTE_BACKUP_DIR}/bundle.tgz"

origin_ssh bash -s <<EOF
set -euo pipefail

APP_DIR="${ORIGIN_APP_DIR}"
BACKUP_DIR="${REMOTE_BACKUP_DIR}"
WORK_DIR="\${BACKUP_DIR}/work"
PG_CONTAINER="${ORIGIN_PG_CONTAINER}"
PG_DB="${ORIGIN_PG_DB}"
PG_USER="${ORIGIN_PG_USER}"

mkdir -p "\$WORK_DIR"

docker ps -a --format "table {{.Names}}\t{{.Image}}\t{{.Status}}\t{{.Ports}}" > "\$WORK_DIR/docker-ps.txt" || true
docker volume ls > "\$WORK_DIR/docker-volumes.txt" || true

docker exec "\$PG_CONTAINER" pg_dumpall -U "\$PG_USER" --globals-only > "\$WORK_DIR/postgres-globals.sql"
docker exec "\$PG_CONTAINER" pg_dump -U "\$PG_USER" -d "\$PG_DB" -F c > "\$WORK_DIR/new-api.dump"

tar -czf "\$WORK_DIR/app-data.tgz" -C "\$APP_DIR" data
tar -czf "\$WORK_DIR/app-logs.tgz" -C "\$APP_DIR" logs
cp "\$APP_DIR/.env" "\$WORK_DIR/app.env"
[ -f "\$APP_DIR/docker-compose.yml" ] && cp "\$APP_DIR/docker-compose.yml" "\$WORK_DIR/docker-compose.yml" || true

(
  cd "\$WORK_DIR"
  if command -v sha256sum >/dev/null 2>&1; then
    sha256sum * > SHA256SUMS
  else
    shasum -a 256 * > SHA256SUMS
  fi
)

tar -czf "\$BACKUP_DIR/bundle.tgz" -C "\$WORK_DIR" .
rm -rf "\$WORK_DIR"

echo "\$BACKUP_DIR/bundle.tgz"
EOF

echo "==[2/3] Download bundle to local ==" >&2
origin_scp_get "$REMOTE_BUNDLE" "$LOCAL_OUTPUT_DIR/bundle.tgz"

echo "==[3/3] Upload bundle to Lighthouse and extract ==" >&2
LIGHTHOUSE_DEST_DIR="${LIGHTHOUSE_DEST_BASE}/${TS}"
lighthouse_ssh "mkdir -p '$LIGHTHOUSE_DEST_DIR'"
lighthouse_scp_put "$LOCAL_OUTPUT_DIR/bundle.tgz" "$LIGHTHOUSE_DEST_DIR/bundle.tgz"
lighthouse_ssh bash -lc "cd '$LIGHTHOUSE_DEST_DIR' && tar -xzf bundle.tgz && if command -v sha256sum >/dev/null 2>&1; then sha256sum -c SHA256SUMS >/dev/null; elif command -v shasum >/dev/null 2>&1; then shasum -a 256 -c SHA256SUMS >/dev/null; else echo 'ERROR: missing sha256sum/shasum on lighthouse' >&2; exit 2; fi"

if [ "$ORIGIN_KEEP_REMOTE" != "true" ]; then
  echo "== Cleanup origin backup dir ==" >&2
  origin_ssh "rm -rf '$REMOTE_BACKUP_DIR'"
fi

if [ "$KEEP_LOCAL" != "true" ]; then
  rm -rf "$LOCAL_OUTPUT_DIR"
fi

echo "$LIGHTHOUSE_DEST_DIR"

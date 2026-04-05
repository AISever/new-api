#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"

resolve_shared_repo_root() {
  local common_dir=""
  common_dir="$(git -C "$PROJECT_ROOT" rev-parse --git-common-dir 2>/dev/null || true)"
  if [ -z "$common_dir" ]; then
    printf '%s\n' "$PROJECT_ROOT"
    return
  fi
  cd "$common_dir/.." && pwd
}

resolve_default_repo_file() {
  local relative_path="$1"
  local shared_root="$2"

  if [ -e "$PROJECT_ROOT/$relative_path" ]; then
    printf '%s\n' "$PROJECT_ROOT/$relative_path"
    return
  fi
  printf '%s\n' "$shared_root/$relative_path"
}

SHARED_REPO_ROOT="$(resolve_shared_repo_root)"
CONFIG_FILE="${CONFIG_FILE:-$(resolve_default_repo_file ".kkidc/.env.lighthouse" "$SHARED_REPO_ROOT")}"

TARGET_ENV=""
DRY_RUN="false"
INCLUDE_DATA="true"
INCLUDE_LOGS="true"

REMOTE_HOST=""
REMOTE_USER=""
REMOTE_PASSWORD=""

POSTGRES_CONTAINER="new-api-postgres"
PG_DB=""
DATA_DIR=""
LOG_DIR=""
BACKUP_ROOT=""
BACKUP_DIR=""
BACKUP_STAMP=""
DB_DUMP_PATH=""
GLOBALS_DUMP_PATH=""
DATA_ARCHIVE_PATH=""
LOG_ARCHIVE_PATH=""

usage() {
  cat <<EOF
Backup kkidc host data for production, enterprise, or test.

Usage:
  $(basename "$0") production|enterprise|test [options]

Options:
  --config PATH   host config file (default: .kkidc/.env.lighthouse)
  --dry-run       print resolved settings without performing remote backup
  --no-data       skip application data archive
  --no-logs       skip log archive
  --help          show this help
EOF
}

die() {
  printf '[ERROR] %s\n' "$1" >&2
  exit 1
}

parse_args() {
  if [ $# -lt 1 ]; then
    usage
    exit 1
  fi

  case "$1" in
    production|prod)
      TARGET_ENV="production"
      ;;
    enterprise)
      TARGET_ENV="enterprise"
      ;;
    test)
      TARGET_ENV="test"
      ;;
    --help|-h)
      usage
      exit 0
      ;;
    *)
      die "unsupported target environment: $1"
      ;;
  esac
  shift

  while [ $# -gt 0 ]; do
    case "$1" in
      --config)
        CONFIG_FILE="$2"
        shift 2
        ;;
      --dry-run)
        DRY_RUN="true"
        shift
        ;;
      --no-data)
        INCLUDE_DATA="false"
        shift
        ;;
      --no-logs)
        INCLUDE_LOGS="false"
        shift
        ;;
      --help|-h)
        usage
        exit 0
        ;;
      *)
        die "unknown option: $1"
        ;;
    esac
  done
}

load_host_config() {
  [ -f "$CONFIG_FILE" ] || die "missing host config: $CONFIG_FILE"
  set -a
  # shellcheck disable=SC1090
  source "$CONFIG_FILE"
  set +a

  REMOTE_HOST="${IP_1:-${SERVER_IP:-}}"
  REMOTE_USER="${SSH_user:-${SERVER_USER:-}}"
  REMOTE_PASSWORD="${SSH_password:-${SERVER_PASSWORD:-}}"

  [ -n "$REMOTE_HOST" ] || die "missing IP_1 in $CONFIG_FILE"
  [ -n "$REMOTE_USER" ] || die "missing SSH_user in $CONFIG_FILE"
  [ -n "$REMOTE_PASSWORD" ] || die "missing SSH_password in $CONFIG_FILE"
}

resolve_target_settings() {
  BACKUP_STAMP="$(date +%Y%m%d%H%M%S)"

  if [ "$TARGET_ENV" = "production" ]; then
    PG_DB="new-api"
    DATA_DIR="/opt/new-api/data"
    LOG_DIR="/opt/new-api/logs"
    BACKUP_ROOT="/opt/new-api-backups"
  elif [ "$TARGET_ENV" = "enterprise" ]; then
    PG_DB="new-api-enterprise"
    DATA_DIR="/opt/new-api-enterprise/data"
    LOG_DIR="/opt/new-api-enterprise/logs"
    BACKUP_ROOT="/opt/new-api-enterprise-backups"
  else
    PG_DB="new-api-test"
    DATA_DIR="/opt/new-api-test/data"
    LOG_DIR="/opt/new-api-test/logs"
    BACKUP_ROOT="/opt/new-api-test-backups"
  fi

  BACKUP_DIR="${BACKUP_ROOT}/${BACKUP_STAMP}"
  DB_DUMP_PATH="${BACKUP_DIR}/db.dump"
  GLOBALS_DUMP_PATH="${BACKUP_DIR}/postgres-globals.sql"
  DATA_ARCHIVE_PATH="${BACKUP_DIR}/data.tgz"
  LOG_ARCHIVE_PATH="${BACKUP_DIR}/logs.tgz"
}

check_dependencies() {
  if [ "$DRY_RUN" = "true" ]; then
    return
  fi
  command -v sshpass >/dev/null 2>&1 || die "sshpass is required"
}

remote_cmd() {
  sshpass -p "$REMOTE_PASSWORD" ssh \
    -o StrictHostKeyChecking=no \
    -o ServerAliveInterval=30 \
    -o ServerAliveCountMax=10 \
    -o PreferredAuthentications=password \
    -o PubkeyAuthentication=no \
    "${REMOTE_USER}@${REMOTE_HOST}" "$1"
}

print_output() {
  local action="$1"
  echo "action=${action}"
  echo "target_env=${TARGET_ENV}"
  echo "remote_host=${REMOTE_HOST}"
  echo "remote_user=${REMOTE_USER}"
  echo "postgres_container=${POSTGRES_CONTAINER}"
  echo "pg_db=${PG_DB}"
  echo "backup_root=${BACKUP_ROOT}"
  echo "backup_dir=${BACKUP_DIR}"
  echo "db_dump=${DB_DUMP_PATH}"
  echo "globals_dump=${GLOBALS_DUMP_PATH}"
  echo "include_data=${INCLUDE_DATA}"
  if [ "$INCLUDE_DATA" = "true" ]; then
    echo "data_archive=${DATA_ARCHIVE_PATH}"
  else
    echo "data_archive=disabled"
  fi
  echo "include_logs=${INCLUDE_LOGS}"
  if [ "$INCLUDE_LOGS" = "true" ]; then
    echo "log_archive=${LOG_ARCHIVE_PATH}"
  else
    echo "log_archive=disabled"
  fi
}

run_backup() {
  local remote_script=""

  remote_script+="set -euo pipefail; "
  remote_script+="mkdir -p '${BACKUP_DIR}'; "
  remote_script+="docker exec '${POSTGRES_CONTAINER}' pg_dump -U newapi -d '${PG_DB}' --clean --if-exists --no-owner --no-privileges -Fc > '${DB_DUMP_PATH}'; "
  remote_script+="docker exec '${POSTGRES_CONTAINER}' pg_dumpall -U newapi --globals-only > '${GLOBALS_DUMP_PATH}'; "

  if [ "$INCLUDE_DATA" = "true" ]; then
    remote_script+="if [ -d '${DATA_DIR}' ]; then tar -czf '${DATA_ARCHIVE_PATH}' -C '$(dirname "$DATA_DIR")' '$(basename "$DATA_DIR")'; fi; "
  fi

  if [ "$INCLUDE_LOGS" = "true" ]; then
    remote_script+="if [ -d '${LOG_DIR}' ]; then tar -czf '${LOG_ARCHIVE_PATH}' -C '$(dirname "$LOG_DIR")' '$(basename "$LOG_DIR")'; fi; "
  fi

  remote_cmd "$remote_script"
}

main() {
  parse_args "$@"
  load_host_config
  resolve_target_settings
  check_dependencies

  if [ "$DRY_RUN" = "true" ]; then
    print_output "backup-dry-run"
    return 0
  fi

  run_backup
  print_output "backup"
}

main "$@"

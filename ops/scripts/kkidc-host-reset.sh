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
APP_ENV_FILE="${APP_ENV_FILE:-$(resolve_default_repo_file ".env.local" "$SHARED_REPO_ROOT")}"

TARGET_ENV=""
DRY_RUN="false"

REMOTE_HOST=""
REMOTE_USER=""
REMOTE_PASSWORD=""

APP_CONTAINER=""
PG_DB=""
DATA_DIR=""
LOG_DIR=""
REDIS_CONTAINER="new-api-redis"
REDIS_DB_INDEX=""
TARGET_BACKUP_DIR=""

usage() {
  cat <<EOF
Reset a kkidc test or enterprise environment to a clean state.

Usage:
  $(basename "$0") test|enterprise [options]

Options:
  --config PATH            host config file (default: .kkidc/.env.lighthouse)
  --app-env-file PATH      app env file used when restarting target app (default: .env.local)
  --dry-run                print resolved settings without performing reset
  --help                   show this help
EOF
}

die() {
  printf '[ERROR] %s\n' "$1" >&2
  exit 1
}

log() {
  printf '[%s] %s\n' "$1" "$2"
}

parse_args() {
  if [ $# -lt 1 ]; then
    usage
    exit 1
  fi

  case "$1" in
    test)
      TARGET_ENV="test"
      ;;
    enterprise)
      TARGET_ENV="enterprise"
      ;;
    production|prod)
      die "reset does not support production target"
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
      --app-env-file)
        APP_ENV_FILE="$2"
        shift 2
        ;;
      --dry-run)
        DRY_RUN="true"
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
  case "$TARGET_ENV" in
    test)
      APP_CONTAINER="new-api-test"
      PG_DB="new-api-test"
      DATA_DIR="/opt/new-api-test/data"
      LOG_DIR="/opt/new-api-test/logs"
      REDIS_DB_INDEX="1"
      ;;
    enterprise)
      APP_CONTAINER="new-api-enterprise"
      PG_DB="new-api-enterprise"
      DATA_DIR="/opt/new-api-enterprise/data"
      LOG_DIR="/opt/new-api-enterprise/logs"
      REDIS_DB_INDEX="2"
      ;;
    *)
      die "reset only supports test or enterprise target"
      ;;
  esac
}

check_dependencies() {
  if [ "$DRY_RUN" = "true" ]; then
    return
  fi

  command -v sshpass >/dev/null 2>&1 || die "sshpass is required"
  [ -f "$APP_ENV_FILE" ] || die "missing app env file: $APP_ENV_FILE"
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

remote_bash() {
  sshpass -p "$REMOTE_PASSWORD" ssh \
    -o StrictHostKeyChecking=no \
    -o ServerAliveInterval=30 \
    -o ServerAliveCountMax=10 \
    -o PreferredAuthentications=password \
    -o PubkeyAuthentication=no \
    "${REMOTE_USER}@${REMOTE_HOST}" 'bash -s'
}

print_output() {
  local action="$1"

  echo "action=${action}"
  echo "target_env=${TARGET_ENV}"
  echo "remote_host=${REMOTE_HOST}"
  echo "remote_user=${REMOTE_USER}"
  echo "app_container=${APP_CONTAINER}"
  echo "pg_db=${PG_DB}"
  echo "redis_container=${REDIS_CONTAINER}"
  echo "redis_db_index=${REDIS_DB_INDEX}"
  echo "data_dir=${DATA_DIR}"
  echo "log_dir=${LOG_DIR}"
  if [ -n "$TARGET_BACKUP_DIR" ]; then
    echo "target_backup_dir=${TARGET_BACKUP_DIR}"
  fi
  echo "restart_via=ops/scripts/kkidc-host-deploy.sh ${TARGET_ENV}"
}

backup_current_target() {
  local backup_output

  log INFO "backing up current ${TARGET_ENV} environment before reset"
  backup_output="$(bash "$SCRIPT_DIR/kkidc-host-backup.sh" "$TARGET_ENV" --config "$CONFIG_FILE")"
  printf '%s\n' "$backup_output"
  TARGET_BACKUP_DIR="$(printf '%s\n' "$backup_output" | awk -F= '/^backup_dir=/{print $2}')"
  [ -n "$TARGET_BACKUP_DIR" ] || die "failed to parse target backup dir from kkidc-host-backup.sh output"
}

reset_target() {
  log INFO "stopping ${TARGET_ENV} app before reset"
  remote_cmd "docker rm -f '${APP_CONTAINER}' >/dev/null 2>&1 || true"

  log INFO "dropping target data, database, and cache for ${TARGET_ENV}"
  remote_bash <<EOF
set -euo pipefail
docker exec '${REDIS_CONTAINER}' redis-cli -n '${REDIS_DB_INDEX}' FLUSHDB >/dev/null
docker exec 'new-api-postgres' psql -U newapi -d postgres -c "DROP DATABASE IF EXISTS \"${PG_DB}\";"
docker exec 'new-api-postgres' psql -U newapi -d postgres -c "CREATE DATABASE \"${PG_DB}\";"
rm -rf '${DATA_DIR}' '${LOG_DIR}'
mkdir -p '$(dirname "$DATA_DIR")' '$(dirname "$LOG_DIR")'
docker exec '${REDIS_CONTAINER}' redis-cli -n '${REDIS_DB_INDEX}' FLUSHDB >/dev/null
EOF
}

restart_target_app() {
  log INFO "restarting ${TARGET_ENV} app via formal ${TARGET_ENV} deploy flow"
  bash "$SCRIPT_DIR/kkidc-host-deploy.sh" "$TARGET_ENV" \
    --config "$CONFIG_FILE" \
    --app-env-file "$APP_ENV_FILE"
}

main() {
  parse_args "$@"
  load_host_config
  resolve_target_settings
  check_dependencies

  if [ "$DRY_RUN" = "true" ]; then
    print_output "reset-dry-run"
    exit 0
  fi

  backup_current_target
  reset_target
  restart_target_app
  print_output "reset"
}

main "$@"

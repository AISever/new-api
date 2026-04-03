#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"

CONFIG_FILE="${CONFIG_FILE:-$PROJECT_ROOT/.kkidc/.env.lighthouse}"
APP_ENV_FILE="${APP_ENV_FILE:-$PROJECT_ROOT/.env.local}"

TARGET_ENV=""
DRY_RUN="false"
RESTORE_LOGS="true"
SOURCE_BACKUP_DIR=""

REMOTE_HOST=""
REMOTE_USER=""
REMOTE_PASSWORD=""

APP_CONTAINER=""
PG_DB=""
DATA_DIR=""
LOG_DIR=""
REDIS_CONTAINER="new-api-redis"
REDIS_DB_INDEX=""

SOURCE_DB_DUMP=""
SOURCE_DATA_ARCHIVE=""
SOURCE_LOG_ARCHIVE=""
TARGET_BACKUP_DIR=""

usage() {
  cat <<EOF
Restore a kkidc backup directory into the kkidc test environment.

Usage:
  $(basename "$0") test [options]

Options:
  --config PATH            host config file (default: .kkidc/.env.lighthouse)
  --app-env-file PATH      app env file used when restarting test app (default: .env.local)
  --source-backup-dir PATH remote backup directory created by kkidc-host-backup.sh
  --no-logs                skip restoring log archive
  --dry-run                print resolved settings without performing restore
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
    production|prod|enterprise)
      die "restore currently only supports test target"
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
      --source-backup-dir)
        SOURCE_BACKUP_DIR="$2"
        shift 2
        ;;
      --no-logs)
        RESTORE_LOGS="false"
        shift
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
  if [ "$TARGET_ENV" != "test" ]; then
    die "restore currently only supports test target"
  fi

  APP_CONTAINER="new-api-test"
  PG_DB="new-api-test"
  DATA_DIR="/opt/new-api-test/data"
  LOG_DIR="/opt/new-api-test/logs"
  REDIS_DB_INDEX="1"
}

resolve_source_paths() {
  [ -n "$SOURCE_BACKUP_DIR" ] || die "--source-backup-dir is required"
  SOURCE_DB_DUMP="${SOURCE_BACKUP_DIR}/db.dump"
  SOURCE_DATA_ARCHIVE="${SOURCE_BACKUP_DIR}/data.tgz"
  SOURCE_LOG_ARCHIVE="${SOURCE_BACKUP_DIR}/logs.tgz"
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
  echo "data_dir=${DATA_DIR}"
  echo "log_dir=${LOG_DIR}"
  echo "source_backup_dir=${SOURCE_BACKUP_DIR}"
  echo "source_db_dump=${SOURCE_DB_DUMP}"
  echo "source_data_archive=${SOURCE_DATA_ARCHIVE}"
  if [ "$RESTORE_LOGS" = "true" ]; then
    echo "restore_logs=true"
    echo "source_log_archive=${SOURCE_LOG_ARCHIVE}"
  else
    echo "restore_logs=false"
    echo "source_log_archive=disabled"
  fi
  if [ -n "$TARGET_BACKUP_DIR" ]; then
    echo "target_backup_dir=${TARGET_BACKUP_DIR}"
  fi
  echo "restart_via=ops/scripts/kkidc-host-deploy.sh test"
}

verify_source_backup() {
  local remote_script=""

  remote_script+="set -euo pipefail; "
  remote_script+="[ -f '${SOURCE_DB_DUMP}' ] || { echo 'missing ${SOURCE_DB_DUMP}' >&2; exit 1; }; "
  remote_script+="[ -f '${SOURCE_DATA_ARCHIVE}' ] || { echo 'missing ${SOURCE_DATA_ARCHIVE}' >&2; exit 1; }; "
  if [ "$RESTORE_LOGS" = "true" ]; then
    remote_script+="[ -f '${SOURCE_LOG_ARCHIVE}' ] || { echo 'missing ${SOURCE_LOG_ARCHIVE}' >&2; exit 1; }; "
  fi

  remote_cmd "$remote_script" >/dev/null
}

backup_current_target() {
  local backup_output

  log INFO "backing up current test environment before restore"
  backup_output="$(bash "$SCRIPT_DIR/kkidc-host-backup.sh" test --config "$CONFIG_FILE")"
  printf '%s\n' "$backup_output"
  TARGET_BACKUP_DIR="$(printf '%s\n' "$backup_output" | awk -F= '/^backup_dir=/{print $2}')"
  [ -n "$TARGET_BACKUP_DIR" ] || die "failed to parse target backup dir from kkidc-host-backup.sh output"
}

restore_target_from_backup() {
  local restore_logs_flag="$RESTORE_LOGS"

  log INFO "stopping test app before restore"
  remote_cmd "docker rm -f '${APP_CONTAINER}' >/dev/null 2>&1 || true"

  log INFO "restoring database, data, and cache from ${SOURCE_BACKUP_DIR}"
  remote_bash <<EOF
set -euo pipefail
docker exec '${REDIS_CONTAINER}' redis-cli -n '${REDIS_DB_INDEX}' FLUSHDB >/dev/null
docker exec '${REDIS_CONTAINER}' redis-cli -n '${REDIS_DB_INDEX}' FLUSHDB >/dev/null
docker exec 'new-api-postgres' psql -U newapi -d postgres -tc "SELECT 1 FROM pg_database WHERE datname = '${PG_DB}'" | grep -q 1 || \
docker exec 'new-api-postgres' psql -U newapi -d postgres -c "CREATE DATABASE \"${PG_DB}\";"
rm -rf '${DATA_DIR}' '${LOG_DIR}'
mkdir -p '$(dirname "$DATA_DIR")' '$(dirname "$LOG_DIR")'
tar -xzf '${SOURCE_DATA_ARCHIVE}' -C '$(dirname "$DATA_DIR")'
if [ "$restore_logs_flag" = "true" ]; then
  tar -xzf '${SOURCE_LOG_ARCHIVE}' -C '$(dirname "$LOG_DIR")'
fi
docker exec -i 'new-api-postgres' pg_restore -U newapi -d '${PG_DB}' --clean --if-exists --no-owner --no-privileges < '${SOURCE_DB_DUMP}' >/dev/null
docker exec '${REDIS_CONTAINER}' redis-cli -n '${REDIS_DB_INDEX}' FLUSHDB >/dev/null
EOF
}

restart_test_app() {
  log INFO "restarting test app via formal test deploy flow"
  bash "$SCRIPT_DIR/kkidc-host-deploy.sh" test \
    --config "$CONFIG_FILE" \
    --app-env-file "$APP_ENV_FILE"
}

main() {
  parse_args "$@"
  load_host_config
  resolve_target_settings
  resolve_source_paths
  check_dependencies

  if [ "$DRY_RUN" = "true" ]; then
    print_output "restore-dry-run"
    exit 0
  fi

  verify_source_backup
  backup_current_target
  restore_target_from_backup
  restart_test_app
  print_output "restore"
}

main "$@"

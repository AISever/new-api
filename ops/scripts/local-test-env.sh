#!/usr/bin/env bash
set -euo pipefail

REPO_DIR="${REPO_DIR:-$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)}"
PROJECT_NAME="${PROJECT_NAME:-local-test}"
COMPOSE_FILE="${COMPOSE_FILE:-$REPO_DIR/ops/compose/local-test.yml}"
DOCKERFILE_PATH="${DOCKERFILE_PATH:-$REPO_DIR/ops/docker/Dockerfile.local-test}"
BASELINE_DIR="${BASELINE_DIR:-$REPO_DIR/ops/local-test/baseline}"
BACKUP_ROOT="${BACKUP_ROOT:-$REPO_DIR/ops/local-test/backups}"

APP_SERVICE="${APP_SERVICE:-new-api}"
POSTGRES_SERVICE="${POSTGRES_SERVICE:-postgres}"
REDIS_SERVICE="${REDIS_SERVICE:-redis}"

APP_CONTAINER="${APP_CONTAINER:-new-api-test}"
POSTGRES_CONTAINER="${POSTGRES_CONTAINER:-postgres-test}"
REDIS_CONTAINER="${REDIS_CONTAINER:-redis-test}"

APP_PORT="${APP_PORT:-3001}"
DB_USER="${DB_USER:-root}"
DB_PASSWORD="${DB_PASSWORD:-123456}"
DB_NAME="${DB_NAME:-new-api}"
ADMIN_USERNAME="${ADMIN_USERNAME:-admin}"
ADMIN_PASSWORD="${ADMIN_PASSWORD:-Admin123456!}"
ADMIN_PASSWORD_HASH="${ADMIN_PASSWORD_HASH:-\$2b\$12\$JElXTf0Q2PcHIG3RiH9cLOGKnRXTXdAUmtMRfrbEiBfFCECPxdy8C}"

HEALTH_TIMEOUT_SECONDS="${HEALTH_TIMEOUT_SECONDS:-120}"
HTTP_TIMEOUT_SECONDS="${HTTP_TIMEOUT_SECONDS:-60}"
LOG_TAIL_LINES="${LOG_TAIL_LINES:-80}"

APP_URL="http://127.0.0.1:${APP_PORT}"
STATUS_URL="${APP_URL}/api/status"

usage() {
  cat <<EOF
Manage the local Docker test environment.

Usage:
  $(basename "$0") <command>

Commands:
  deploy              Build and update the local test environment.
  status              Show container, health, image, and URL.
  logs                Show app container logs.
  stop                Stop the local test environment while keeping data.
  reset-to-baseline   Recreate PostgreSQL/Redis and restore the fixed baseline.
  reseed-passwords    Reset fixed test credentials in the current database.
  backup-current      Export the current local environment into a timestamped backup.
EOF
}

require_cmd() {
  command -v "$1" >/dev/null 2>&1 || {
    echo "ERROR: missing required command: $1" >&2
    exit 2
  }
}

ensure_common_layout() {
  mkdir -p "$REPO_DIR/data" "$REPO_DIR/logs" "$BACKUP_ROOT"
}

ensure_assets() {
  [ -f "$COMPOSE_FILE" ] || {
    echo "ERROR: missing compose file: $COMPOSE_FILE" >&2
    exit 2
  }
  [ -f "$DOCKERFILE_PATH" ] || {
    echo "ERROR: missing dockerfile: $DOCKERFILE_PATH" >&2
    exit 2
  }
}

ensure_baseline_assets() {
  ensure_assets
  [ -f "$BASELINE_DIR/manifest.json" ] || {
    echo "ERROR: missing baseline manifest: $BASELINE_DIR/manifest.json" >&2
    exit 2
  }
  [ -f "$BASELINE_DIR/new-api.dump" ] || {
    echo "ERROR: missing baseline dump: $BASELINE_DIR/new-api.dump" >&2
    exit 2
  }
  [ -f "$BASELINE_DIR/postgres-globals.sql" ] || {
    echo "ERROR: missing baseline globals: $BASELINE_DIR/postgres-globals.sql" >&2
    exit 2
  }
}

compose_cmd() {
  LOCAL_TEST_IMAGE="${LOCAL_TEST_IMAGE:-codex/new-api-local:latest}" \
    docker compose -f "$COMPOSE_FILE" -p "$PROJECT_NAME" "$@"
}

remove_container_if_exists() {
  local container_name="$1"
  if docker container inspect "$container_name" >/dev/null 2>&1; then
    docker rm -f "$container_name" >/dev/null
  fi
}

prepare_runtime_containers() {
  remove_container_if_exists "$APP_CONTAINER"
  remove_container_if_exists "$POSTGRES_CONTAINER"
  remove_container_if_exists "$REDIS_CONTAINER"
}

make_image_ref() {
  local short_sha
  short_sha="$(git -C "$REPO_DIR" rev-parse --short HEAD 2>/dev/null || echo nogit)"
  echo "codex/new-api-local:local-${short_sha}-$(date +%Y%m%d%H%M%S)"
}

current_image() {
  docker inspect "$APP_CONTAINER" --format '{{.Config.Image}}' 2>/dev/null || true
}

inspect_status_triplet() {
  docker inspect "$APP_CONTAINER" --format '{{.State.Status}} {{if .State.Health}}{{.State.Health.Status}}{{else}}none{{end}} {{.RestartCount}}' 2>/dev/null || true
}

status_field() {
  inspect_status_triplet | awk '{print $1}'
}

health_field() {
  inspect_status_triplet | awk '{print $2}'
}

restart_count_field() {
  inspect_status_triplet | awk '{print $3}'
}

wait_for_postgres() {
  local deadline now
  deadline=$((SECONDS + HEALTH_TIMEOUT_SECONDS))
  while true; do
    if docker exec "$POSTGRES_CONTAINER" pg_isready -U "$DB_USER" -d postgres >/dev/null 2>&1; then
      return 0
    fi
    now=$SECONDS
    if [ "$now" -ge "$deadline" ]; then
      echo "ERROR: postgres did not become ready within ${HEALTH_TIMEOUT_SECONDS}s" >&2
      docker logs --tail "$LOG_TAIL_LINES" "$POSTGRES_CONTAINER" >&2 || true
      exit 1
    fi
    sleep 2
  done
}

wait_for_app_health() {
  local deadline state health
  deadline=$((SECONDS + HEALTH_TIMEOUT_SECONDS))
  while true; do
    state="$(status_field)"
    health="$(health_field)"
    if [ "$state" = "running" ] && [ "$health" = "healthy" ]; then
      return 0
    fi
    if [ "$SECONDS" -ge "$deadline" ]; then
      echo "ERROR: app did not become healthy within ${HEALTH_TIMEOUT_SECONDS}s" >&2
      docker logs --tail "$LOG_TAIL_LINES" "$APP_CONTAINER" >&2 || true
      exit 1
    fi
    sleep 2
  done
}

verify_status_endpoint() {
  local deadline payload
  deadline=$((SECONDS + HTTP_TIMEOUT_SECONDS))
  while true; do
    payload="$(curl -fsS "$STATUS_URL" 2>/dev/null || true)"
    if printf '%s' "$payload" | grep -Eq '"success":[[:space:]]*true' &&
      printf '%s' "$payload" | grep -Eq '"setup":[[:space:]]*true'; then
      return 0
    fi
    if [ "$SECONDS" -ge "$deadline" ]; then
      echo "ERROR: status endpoint did not report success/setup within ${HTTP_TIMEOUT_SECONDS}s" >&2
      docker logs --tail "$LOG_TAIL_LINES" "$APP_CONTAINER" >&2 || true
      exit 1
    fi
    sleep 2
  done
}

build_frontend() {
  (
    cd "$REPO_DIR/web"
    npm run build
  )
}

restore_app_data_if_present() {
  rm -rf "$REPO_DIR/data"/* "$REPO_DIR/logs"/*
  if [ -f "$BASELINE_DIR/app-data.tgz" ]; then
    tar -xzf "$BASELINE_DIR/app-data.tgz" -C "$REPO_DIR/data"
  fi
}

do_reseed_passwords() {
  docker exec -i "$POSTGRES_CONTAINER" psql -U "$DB_USER" -d postgres -v ON_ERROR_STOP=1 <<SQL
ALTER ROLE ${DB_USER} WITH PASSWORD '${DB_PASSWORD}';
SQL

  docker exec -i "$POSTGRES_CONTAINER" psql -U "$DB_USER" -d "$DB_NAME" -v ON_ERROR_STOP=1 <<SQL
UPDATE users
SET password = '${ADMIN_PASSWORD_HASH}'
WHERE username = '${ADMIN_USERNAME}';
SQL
}

print_status() {
  echo "action=status"
  echo "container=${APP_CONTAINER}"
  echo "status=$(status_field)"
  echo "health_status=$(health_field)"
  echo "restart_count=$(restart_count_field)"
  echo "image=$(current_image)"
  echo "app_url=${APP_URL}"
}

deploy() {
  require_cmd docker
  require_cmd node
  require_cmd npm
  require_cmd curl
  require_cmd git
  ensure_common_layout
  ensure_assets

  export LOCAL_TEST_IMAGE
  LOCAL_TEST_IMAGE="${LOCAL_TEST_IMAGE:-$(make_image_ref)}"

  compose_cmd config >/dev/null
  build_frontend
  compose_cmd build "$APP_SERVICE"
  prepare_runtime_containers
  compose_cmd up -d "$POSTGRES_SERVICE" "$REDIS_SERVICE" "$APP_SERVICE"
  wait_for_app_health
  verify_status_endpoint

  echo "action=deploy"
  echo "image=$(current_image)"
  echo "health_status=$(health_field)"
  echo "app_url=${APP_URL}"
}

status_cmd() {
  require_cmd docker
  ensure_assets
  print_status
}

logs_cmd() {
  require_cmd docker
  docker logs --tail "$LOG_TAIL_LINES" "$APP_CONTAINER"
}

stop_cmd() {
  require_cmd docker
  ensure_assets
  compose_cmd stop "$APP_SERVICE" "$POSTGRES_SERVICE" "$REDIS_SERVICE"
  echo "action=stop"
  echo "app_container=${APP_CONTAINER}"
  echo "db_container=${POSTGRES_CONTAINER}"
  echo "redis_container=${REDIS_CONTAINER}"
}

reset_to_baseline() {
  require_cmd docker
  require_cmd curl
  ensure_common_layout
  ensure_baseline_assets

  compose_cmd down -v
  prepare_runtime_containers
  restore_app_data_if_present
  compose_cmd up -d "$POSTGRES_SERVICE" "$REDIS_SERVICE"
  wait_for_postgres
  docker exec -i "$POSTGRES_CONTAINER" psql -U "$DB_USER" -d postgres -v ON_ERROR_STOP=1 < "$BASELINE_DIR/postgres-globals.sql"
  docker exec -i "$POSTGRES_CONTAINER" pg_restore -U "$DB_USER" -d "$DB_NAME" --clean --if-exists --no-owner --no-privileges < "$BASELINE_DIR/new-api.dump"
  do_reseed_passwords
  compose_cmd up -d "$APP_SERVICE"
  wait_for_app_health
  verify_status_endpoint

  echo "action=reset-to-baseline"
  echo "admin_username=${ADMIN_USERNAME}"
  echo "health_status=$(health_field)"
  echo "app_url=${APP_URL}"
}

reseed_passwords_cmd() {
  require_cmd docker
  ensure_assets
  do_reseed_passwords
  echo "action=reseed-passwords"
  echo "admin_username=${ADMIN_USERNAME}"
  echo "admin_password=${ADMIN_PASSWORD}"
  echo "db_user=${DB_USER}"
  echo "db_password=${DB_PASSWORD}"
}

backup_current() {
  require_cmd docker
  ensure_common_layout
  ensure_assets

  local stamp backup_dir image_ref
  stamp="$(date +%Y%m%d-%H%M%S)"
  backup_dir="${BACKUP_ROOT}/${stamp}"
  image_ref="$(current_image)"

  mkdir -p "$backup_dir"
  docker exec "$POSTGRES_CONTAINER" pg_dump -U "$DB_USER" -d "$DB_NAME" --clean --if-exists --no-owner --no-privileges -Fc > "$backup_dir/new-api.dump"
  docker exec "$POSTGRES_CONTAINER" pg_dumpall -U "$DB_USER" --globals-only > "$backup_dir/postgres-globals.sql"
  tar -czf "$backup_dir/app-data.tgz" -C "$REPO_DIR" data >/dev/null 2>&1 || true

  cat > "$backup_dir/manifest.json" <<EOF
{
  "version": "${stamp}",
  "generated_at": "${stamp}",
  "app": {
    "url": "${APP_URL}",
    "container": "${APP_CONTAINER}",
    "image": "${image_ref}"
  },
  "database": {
    "container": "${POSTGRES_CONTAINER}",
    "user": "${DB_USER}",
    "password": "${DB_PASSWORD}",
    "database": "${DB_NAME}"
  },
  "redis": {
    "container": "${REDIS_CONTAINER}"
  },
  "accounts": {
    "admin": {
      "username": "${ADMIN_USERNAME}",
      "password": "${ADMIN_PASSWORD}"
    }
  }
}
EOF

  echo "action=backup-current"
  echo "backup_dir=${backup_dir}"
}

COMMAND="${1:-}"

case "$COMMAND" in
  deploy)
    deploy
    ;;
  status)
    status_cmd
    ;;
  logs)
    logs_cmd
    ;;
  stop)
    stop_cmd
    ;;
  reset-to-baseline)
    reset_to_baseline
    ;;
  reseed-passwords)
    reseed_passwords_cmd
    ;;
  backup-current)
    backup_current
    ;;
  -h|--help|help|'')
    usage
    ;;
  *)
    echo "ERROR: unsupported command: $COMMAND" >&2
    usage >&2
    exit 2
    ;;
esac

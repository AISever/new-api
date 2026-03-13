#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"
REPO_DIR="${REPO_DIR:-$PROJECT_ROOT}"

CONFIG_FILE="$PROJECT_ROOT/.kkidc/.env.lighthouse"
APP_ENV_FILE="${APP_ENV_FILE:-$PROJECT_ROOT/.env.local}"
FRONTEND_BUILD_NODE_OPTIONS="${FRONTEND_BUILD_NODE_OPTIONS:---max-old-space-size=2048}"
BUILD_STRATEGY_REQUESTED="${BUILD_STRATEGY:-auto}"

TARGET_ENV=""
DRY_RUN="false"
KEEP_STAGE_DIR="false"
SKIP_BUILD="false"
CLEAR_CACHE="false"
SYNC_PROD_DATA_FROM_LEGACY="false"

REMOTE_HOST=""
REMOTE_USER=""
REMOTE_PASSWORD=""
PUBLIC_HOSTNAME=""
PUBLIC_HOSTNAME_ALIASES=""
PUBLIC_HOSTNAME_LIST=""
PUBLIC_CADDY_SITE_ADDRESSES=""
LEGACY_BUILD_HOST=""
LEGACY_BUILD_USER=""
LEGACY_BUILD_PASSWORD=""

DB_PASSWORD=""
SESSION_SECRET=""
CRYPTO_SECRET=""

BRANCH_NAME=""
SHA=""
VERSION_VALUE=""
STAGE_DIR=""
REMOTE_BUILD_DIR=""
IMAGE_NAME=""
APP_CONTAINER=""
APP_PORT=""
PG_DB=""
REDIS_DB_INDEX=""
REDIS_CONN_STRING=""
DATA_DIR=""
LOG_DIR=""
SERVER_ADDRESS=""
REMOTE_NETWORK="new-api_default"
POSTGRES_CONTAINER="new-api-postgres"
REDIS_CONTAINER="new-api-redis"
CADDY_CONTAINER="snowlight-caddy"
BUILD_STRATEGY_RESOLVED=""
LEGACY_BUILD_DIR=""

usage() {
  cat <<EOF
Deploy the new kkidc host using clean committed source only.

Usage:
  $(basename "$0") production|test [options]

Options:
  --config PATH         host config file (default: .kkidc/.env.lighthouse)
  --app-env-file PATH   app secret config file (default: .env.local)
  --dry-run             print resolved settings and staged source path
  --keep-stage-dir      keep staged archive directory after exit
  --skip-build          reuse existing remote image tag for current HEAD
  --build-strategy MODE auto | local | legacy-remote | remote (default: auto)
  --clear-cache         flush Redis DB for the target environment after deploy
  --sync-prod-data-from-legacy
                        restore old kkidc production data onto the new host before deploy
  --help                show this help
EOF
}

log() {
  printf '[%s] %s\n' "$1" "$2"
}

die() {
  printf '[ERROR] %s\n' "$1" >&2
  exit 1
}

cleanup() {
  if [ "$KEEP_STAGE_DIR" != "true" ] && [ -n "$STAGE_DIR" ] && [ -d "$STAGE_DIR" ]; then
    rm -rf "$STAGE_DIR"
  fi
}

trap cleanup EXIT

parse_args() {
  if [ $# -lt 1 ]; then
    usage
    exit 1
  fi

  case "$1" in
    production|prod)
      TARGET_ENV="production"
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
      --app-env-file)
        APP_ENV_FILE="$2"
        shift 2
        ;;
      --dry-run)
        DRY_RUN="true"
        shift
        ;;
      --keep-stage-dir)
        KEEP_STAGE_DIR="true"
        shift
        ;;
      --skip-build)
        SKIP_BUILD="true"
        shift
        ;;
      --build-strategy)
        BUILD_STRATEGY_REQUESTED="$2"
        shift 2
        ;;
      --clear-cache)
        CLEAR_CACHE="true"
        shift
        ;;
      --sync-prod-data-from-legacy)
        SYNC_PROD_DATA_FROM_LEGACY="true"
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
  PUBLIC_HOSTNAME="${HOSTNAME:-}"
  PUBLIC_HOSTNAME_ALIASES="${HOSTNAME_ALIASES:-}"

  [ -n "$REMOTE_HOST" ] || die "missing IP_1 in $CONFIG_FILE"
  [ -n "$REMOTE_USER" ] || die "missing SSH_user in $CONFIG_FILE"
  [ -n "$REMOTE_PASSWORD" ] || die "missing SSH_password in $CONFIG_FILE"
}

load_app_env() {
  [ -f "$APP_ENV_FILE" ] || die "missing app env file: $APP_ENV_FILE"
  set -a
  # shellcheck disable=SC1090
  source "$APP_ENV_FILE"
  set +a

  DB_PASSWORD="${DB_PASSWORD:-NewApi2024Secure}"
  SESSION_SECRET="${SESSION_SECRET:-}"
  CRYPTO_SECRET="${CRYPTO_SECRET:-}"
  LEGACY_BUILD_HOST="${SERVER_IP:-}"
  LEGACY_BUILD_USER="${SERVER_USER:-}"
  LEGACY_BUILD_PASSWORD="${SERVER_PASSWORD:-}"

  [ -n "$SESSION_SECRET" ] || die "SESSION_SECRET is required in $APP_ENV_FILE"
  [ -n "$CRYPTO_SECRET" ] || die "CRYPTO_SECRET is required in $APP_ENV_FILE"
}

resolve_git_version() {
  git -C "$REPO_DIR" rev-parse --verify HEAD >/dev/null 2>&1 || die "REPO_DIR is not a git repository: $REPO_DIR"
  SHA="$(git -C "$REPO_DIR" rev-parse --short HEAD)"
  BRANCH_NAME="$(git -C "$REPO_DIR" branch --show-current || true)"
  if [ -z "$BRANCH_NAME" ]; then
    BRANCH_NAME="detached-head"
  fi
  VERSION_VALUE="${BRANCH_NAME}+${SHA}+kkidc-${TARGET_ENV}"
}

resolve_target_settings() {
  if [ -n "$PUBLIC_HOSTNAME" ] && [ -n "$PUBLIC_HOSTNAME_ALIASES" ]; then
    PUBLIC_HOSTNAME_LIST="${PUBLIC_HOSTNAME},${PUBLIC_HOSTNAME_ALIASES}"
  else
    PUBLIC_HOSTNAME_LIST="${PUBLIC_HOSTNAME}"
  fi
  PUBLIC_CADDY_SITE_ADDRESSES="${PUBLIC_HOSTNAME_LIST//,/, }"

  if [ "$TARGET_ENV" = "production" ]; then
    APP_CONTAINER="new-api-local"
    APP_PORT="3000"
    PG_DB="new-api"
    REDIS_DB_INDEX="0"
    DATA_DIR="/opt/new-api/data"
    LOG_DIR="/opt/new-api/logs"
    REMOTE_BUILD_DIR="/opt/new-api-build-production"
    IMAGE_NAME="new-api:kkidc-production-${SHA}"
    if [ -n "$PUBLIC_HOSTNAME" ]; then
      SERVER_ADDRESS="https://${PUBLIC_HOSTNAME}"
    else
      SERVER_ADDRESS="http://${REMOTE_HOST}:${APP_PORT}"
    fi
  else
    APP_CONTAINER="new-api-test"
    APP_PORT="3001"
    PG_DB="new-api-test"
    REDIS_DB_INDEX="1"
    DATA_DIR="/opt/new-api-test/data"
    LOG_DIR="/opt/new-api-test/logs"
    REMOTE_BUILD_DIR="/opt/new-api-build-test"
    IMAGE_NAME="new-api:kkidc-test-${SHA}"
    if [ -n "$PUBLIC_HOSTNAME" ]; then
      SERVER_ADDRESS="http://${PUBLIC_HOSTNAME}:${APP_PORT}"
    else
      SERVER_ADDRESS="http://${REMOTE_HOST}:${APP_PORT}"
    fi
  fi

  if [ "$REDIS_DB_INDEX" = "0" ]; then
    REDIS_CONN_STRING="redis://${REDIS_CONTAINER}"
  else
    REDIS_CONN_STRING="redis://${REDIS_CONTAINER}/${REDIS_DB_INDEX}"
  fi
}

local_docker_available() {
  command -v docker >/dev/null 2>&1 && docker info >/dev/null 2>&1
}

resolve_build_strategy() {
  if [ "$SKIP_BUILD" = "true" ]; then
    BUILD_STRATEGY_RESOLVED="skip"
    return
  fi

  case "$BUILD_STRATEGY_REQUESTED" in
    auto)
      if local_docker_available; then
        BUILD_STRATEGY_RESOLVED="local"
      elif [ -n "$LEGACY_BUILD_HOST" ] && [ -n "$LEGACY_BUILD_USER" ] && [ -n "$LEGACY_BUILD_PASSWORD" ] && [ "$LEGACY_BUILD_HOST" != "$REMOTE_HOST" ]; then
        BUILD_STRATEGY_RESOLVED="legacy-remote"
      else
        BUILD_STRATEGY_RESOLVED="remote"
      fi
      ;;
    local|legacy-remote|remote)
      BUILD_STRATEGY_RESOLVED="$BUILD_STRATEGY_REQUESTED"
      ;;
    *)
      die "unsupported build strategy: $BUILD_STRATEGY_REQUESTED"
      ;;
  esac
}

stage_clean_repo() {
  STAGE_DIR="$(mktemp -d "${TMPDIR:-/tmp}/kkidc-${TARGET_ENV}-XXXXXX")"
  git -C "$REPO_DIR" archive --format=tar HEAD | tar -xf - -C "$STAGE_DIR"
  printf '%s\n' "$VERSION_VALUE" > "$STAGE_DIR/VERSION"

  [ -f "$STAGE_DIR/Dockerfile" ] || die "missing Dockerfile in staged source"
  cp "$STAGE_DIR/Dockerfile" "$STAGE_DIR/Dockerfile.deploy"
  sed -i.bak "s|RUN go mod download|RUN go env -w GOPROXY=https://goproxy.cn,direct \\&\\& go mod download|" "$STAGE_DIR/Dockerfile.deploy"
  sed -i.bak "s|RUN DISABLE_ESLINT_PLUGIN='true' VITE_REACT_APP_VERSION=\$(cat VERSION) bun run build|RUN DISABLE_ESLINT_PLUGIN='true' NODE_OPTIONS='${FRONTEND_BUILD_NODE_OPTIONS}' VITE_REACT_APP_VERSION=\$(cat VERSION) bun run build|" "$STAGE_DIR/Dockerfile.deploy"
  rm -f "$STAGE_DIR/Dockerfile.deploy.bak"
}

print_dry_run() {
  cat <<EOF
target_env=${TARGET_ENV}
remote_host=${REMOTE_HOST}
remote_user=${REMOTE_USER}
public_hostname=${PUBLIC_HOSTNAME}
public_hostnames=${PUBLIC_HOSTNAME_LIST}
caddy_site_addresses=${PUBLIC_CADDY_SITE_ADDRESSES}
app_container=${APP_CONTAINER}
app_port=${APP_PORT}
pg_db=${PG_DB}
redis_conn_string=${REDIS_CONN_STRING}
data_dir=${DATA_DIR}
log_dir=${LOG_DIR}
remote_build_dir=${REMOTE_BUILD_DIR}
image_name=${IMAGE_NAME}
version_value=${VERSION_VALUE}
server_address=${SERVER_ADDRESS}
stage_dir=${STAGE_DIR}
build_strategy=${BUILD_STRATEGY_RESOLVED}
sync_prod_data_from_legacy=${SYNC_PROD_DATA_FROM_LEGACY}
EOF
}

check_dependencies() {
  command -v git >/dev/null 2>&1 || die "git is required"
  command -v sshpass >/dev/null 2>&1 || die "sshpass is required"
  command -v tar >/dev/null 2>&1 || die "tar is required"
  if [ "$BUILD_STRATEGY_RESOLVED" = "local" ] && ! local_docker_available; then
    die "local docker is unavailable for build strategy=local"
  fi
}

validate_requested_operation() {
  if [ "$SYNC_PROD_DATA_FROM_LEGACY" = "true" ] && [ "$TARGET_ENV" != "production" ]; then
    die "--sync-prod-data-from-legacy only supports production"
  fi
  if [ "$SYNC_PROD_DATA_FROM_LEGACY" = "true" ] && { [ -z "$LEGACY_BUILD_HOST" ] || [ -z "$LEGACY_BUILD_USER" ] || [ -z "$LEGACY_BUILD_PASSWORD" ]; }; then
    die "--sync-prod-data-from-legacy requires legacy kkidc credentials in $APP_ENV_FILE"
  fi
  if [ "$SYNC_PROD_DATA_FROM_LEGACY" = "true" ] && [ "$LEGACY_BUILD_HOST" = "$REMOTE_HOST" ]; then
    die "--sync-prod-data-from-legacy requires legacy and target hosts to differ"
  fi
}

remote_cmd() {
  sshpass -p "$REMOTE_PASSWORD" ssh \
    -o StrictHostKeyChecking=no \
    -o PreferredAuthentications=password \
    -o PubkeyAuthentication=no \
    "${REMOTE_USER}@${REMOTE_HOST}" "$@"
}

remote_bash() {
  sshpass -p "$REMOTE_PASSWORD" ssh \
    -o StrictHostKeyChecking=no \
    -o PreferredAuthentications=password \
    -o PubkeyAuthentication=no \
    "${REMOTE_USER}@${REMOTE_HOST}" 'bash -s'
}

legacy_cmd() {
  sshpass -p "$LEGACY_BUILD_PASSWORD" ssh \
    -o StrictHostKeyChecking=no \
    -o PreferredAuthentications=password \
    -o PubkeyAuthentication=no \
    "${LEGACY_BUILD_USER}@${LEGACY_BUILD_HOST}" "$@"
}

sync_stage_to_host() {
  local target_host="$1"
  local target_user="$2"
  local target_password="$3"
  local target_dir="$4"

  COPYFILE_DISABLE=1 tar -C "$STAGE_DIR" -cf - . | sshpass -p "$target_password" ssh \
    -o StrictHostKeyChecking=no \
    -o PreferredAuthentications=password \
    -o PubkeyAuthentication=no \
    "${target_user}@${target_host}" "rm -rf '$target_dir' && mkdir -p '$target_dir' && tar -xf - -C '$target_dir'"
}

copy_file_to_remote() {
  local source_file="$1"
  local target_file="$2"

  sshpass -p "$REMOTE_PASSWORD" scp \
    -o StrictHostKeyChecking=no \
    -o PreferredAuthentications=password \
    -o PubkeyAuthentication=no \
    "$source_file" "${REMOTE_USER}@${REMOTE_HOST}:$target_file"
}

prepare_remote_runtime() {
  remote_bash <<EOF
set -euo pipefail
mkdir -p '${DATA_DIR}' '${LOG_DIR}' /opt/snowlight /opt/snowlight/data /opt/snowlight/config
docker network inspect '${REMOTE_NETWORK}' >/dev/null 2>&1 || docker network create '${REMOTE_NETWORK}' >/dev/null
EOF
}

ensure_test_database() {
  if [ "$TARGET_ENV" != "test" ]; then
    return
  fi

  remote_bash <<EOF
set -euo pipefail
docker exec '${POSTGRES_CONTAINER}' psql -U newapi -d postgres -tc "SELECT 1 FROM pg_database WHERE datname = '${PG_DB}'" | grep -q 1 || \
docker exec '${POSTGRES_CONTAINER}' psql -U newapi -d postgres -c "CREATE DATABASE \\"${PG_DB}\\";"
EOF
}

build_remote_image() {
  case "$BUILD_STRATEGY_RESOLVED" in
    skip)
      return
      ;;
    local)
      docker build -t "$IMAGE_NAME" -f "$STAGE_DIR/Dockerfile.deploy" "$STAGE_DIR"
      docker save "$IMAGE_NAME" | remote_cmd "docker load >/dev/null"
      ;;
    legacy-remote)
      LEGACY_BUILD_DIR="/tmp/new-api-build-${TARGET_ENV}-${SHA}"
      sync_stage_to_host "$LEGACY_BUILD_HOST" "$LEGACY_BUILD_USER" "$LEGACY_BUILD_PASSWORD" "$LEGACY_BUILD_DIR"
      legacy_cmd "cd '$LEGACY_BUILD_DIR' && docker build -t '$IMAGE_NAME' -f Dockerfile.deploy ."
      legacy_cmd "docker save '$IMAGE_NAME'" | remote_cmd "docker load >/dev/null"
      ;;
    remote)
      sync_stage_to_host "$REMOTE_HOST" "$REMOTE_USER" "$REMOTE_PASSWORD" "$REMOTE_BUILD_DIR"
      remote_cmd "cd '$REMOTE_BUILD_DIR' && docker build -t '$IMAGE_NAME' -f Dockerfile.deploy ."
      ;;
  esac
}

deploy_remote_app() {
  remote_bash <<EOF
set -euo pipefail
docker rm -f '${APP_CONTAINER}' >/dev/null 2>&1 || true
docker run -d \
  --name '${APP_CONTAINER}' \
  --restart always \
  --network '${REMOTE_NETWORK}' \
  -p '${APP_PORT}:3000' \
  -v '${DATA_DIR}:/data' \
  -v '${LOG_DIR}:/app/logs' \
  -e SQL_DSN='postgresql://newapi:${DB_PASSWORD}@${POSTGRES_CONTAINER}:5432/${PG_DB}' \
  -e REDIS_CONN_STRING='${REDIS_CONN_STRING}' \
  -e TZ='Asia/Shanghai' \
  -e SESSION_SECRET='${SESSION_SECRET}' \
  -e CRYPTO_SECRET='${CRYPTO_SECRET}' \
  -e ERROR_LOG_ENABLED='true' \
  -e BATCH_UPDATE_ENABLED='true' \
  -e MEMORY_CACHE_ENABLED='true' \
  -e VERSION='${VERSION_VALUE}' \
  '${IMAGE_NAME}' --log-dir /app/logs >/dev/null
EOF
}

configure_remote_caddy() {
  if [ "$TARGET_ENV" != "production" ] || [ -z "$PUBLIC_HOSTNAME" ]; then
    return
  fi

  remote_bash <<EOF
set -euo pipefail
cat > /opt/snowlight/Caddyfile <<'CADDY'
${PUBLIC_CADDY_SITE_ADDRESSES} {
    encode gzip
    reverse_proxy 172.17.0.1:3000 {
        header_up X-Real-IP {remote_host}
        header_up Host {host}
    }
}

http://${REMOTE_HOST} {
    encode gzip
    reverse_proxy 172.17.0.1:3000 {
        header_up X-Real-IP {remote_host}
        header_up Host {host}
    }
}
CADDY

docker rm -f '${CADDY_CONTAINER}' >/dev/null 2>&1 || true
docker run -d \
  --name '${CADDY_CONTAINER}' \
  --restart always \
  -p 80:80 -p 443:443 \
  -v /opt/snowlight/Caddyfile:/etc/caddy/Caddyfile:ro \
  -v /opt/snowlight/data:/data \
  -v /opt/snowlight/config:/config \
  registry.cn-beijing.aliyuncs.com/yingxuesec/caddy:2-alpine >/dev/null
EOF
}

verify_remote_app() {
  remote_bash <<EOF
set -euo pipefail
for i in \$(seq 1 90); do
  if curl -fsS "http://127.0.0.1:${APP_PORT}/api/status" >/dev/null 2>&1; then
    exit 0
  fi
  sleep 1
done
exit 1
EOF
}

update_server_address_option() {
  remote_bash <<EOF
set -euo pipefail
docker exec -i '${POSTGRES_CONTAINER}' psql -U newapi -d '${PG_DB}' <<'SQL' >/dev/null 2>&1 || exit 0
INSERT INTO options (key, value)
VALUES ('ServerAddress', '${SERVER_ADDRESS}')
ON CONFLICT (key) DO UPDATE SET value = EXCLUDED.value;
SQL
docker restart '${APP_CONTAINER}' >/dev/null
EOF
}

verify_remote_entrypoint() {
  if [ "$TARGET_ENV" = "production" ] && [ -n "$PUBLIC_HOSTNAME" ]; then
    remote_bash <<EOF
set -euo pipefail
for i in \$(seq 1 90); do
  if curl -kfsS --resolve '${PUBLIC_HOSTNAME}:443:127.0.0.1' 'https://${PUBLIC_HOSTNAME}/api/status' >/dev/null 2>&1; then
    exit 0
  fi
  sleep 1
done
exit 1
EOF
  else
    remote_bash <<EOF
set -euo pipefail
for i in \$(seq 1 90); do
  if curl -fsS 'http://127.0.0.1:${APP_PORT}/api/status' >/dev/null 2>&1; then
    exit 0
  fi
  sleep 1
done
exit 1
EOF
  fi
}

clear_remote_cache() {
  if [ "$CLEAR_CACHE" != "true" ]; then
    return 0
  fi
  remote_cmd "docker exec '${REDIS_CONTAINER}' redis-cli -n '${REDIS_DB_INDEX}' FLUSHDB >/dev/null"
}

sync_production_data_from_legacy() {
  if [ "$SYNC_PROD_DATA_FROM_LEGACY" != "true" ]; then
    return 0
  fi

  local backup_stamp
  backup_stamp="$(date +%Y%m%d%H%M%S)"
  local migration_dir
  migration_dir="${STAGE_DIR}/legacy-production-sync"
  mkdir -p "$migration_dir"

  log INFO "backing up current target production state"
  remote_bash <<EOF
set -euo pipefail
backup_root='/opt/new-api-migration-backups/${backup_stamp}'
mkdir -p "\${backup_root}"
docker exec '${POSTGRES_CONTAINER}' pg_dump -U newapi -d '${PG_DB}' --clean --if-exists --no-owner --no-privileges > "\${backup_root}/db.sql"
if [ -d '${DATA_DIR}' ]; then
  cp -a '${DATA_DIR}' "\${backup_root}/data"
fi
if [ -d '${LOG_DIR}' ]; then
  cp -a '${LOG_DIR}' "\${backup_root}/logs"
fi
EOF

  log INFO "stopping target production app before restore"
  remote_cmd "docker rm -f '${APP_CONTAINER}' >/dev/null 2>&1 || true"

  log INFO "exporting legacy production database snapshot"
  legacy_cmd "docker exec '${POSTGRES_CONTAINER}' pg_dump -U newapi -d '${PG_DB}' --clean --if-exists --no-owner --no-privileges" > "${migration_dir}/db.sql"

  log INFO "exporting legacy production data and logs"
  legacy_cmd "tar --warning=no-file-changed --ignore-failed-read -C /opt -cf - new-api/data new-api/logs" > "${migration_dir}/prod-data.tar"

  log INFO "copying production data bundle to target host"
  remote_cmd "mkdir -p '${REMOTE_BUILD_DIR}'"
  copy_file_to_remote "${migration_dir}/db.sql" "${REMOTE_BUILD_DIR}/legacy-prod-db.sql"
  copy_file_to_remote "${migration_dir}/prod-data.tar" "${REMOTE_BUILD_DIR}/legacy-prod-data.tar"

  log INFO "restoring legacy production data and logs on target host"
  remote_bash <<EOF
set -euo pipefail
rm -rf '${DATA_DIR}' '${LOG_DIR}'
mkdir -p /opt
tar -C /opt -xf '${REMOTE_BUILD_DIR}/legacy-prod-data.tar'
EOF

  log INFO "restoring legacy production database snapshot"
  remote_bash <<EOF
set -euo pipefail
docker exec -i '${POSTGRES_CONTAINER}' psql -U newapi -d '${PG_DB}' < '${REMOTE_BUILD_DIR}/legacy-prod-db.sql' >/dev/null
rm -f '${REMOTE_BUILD_DIR}/legacy-prod-db.sql' '${REMOTE_BUILD_DIR}/legacy-prod-data.tar'
EOF

  log INFO "flushing target production redis cache"
  remote_cmd "docker exec '${REDIS_CONTAINER}' redis-cli -n '${REDIS_DB_INDEX}' FLUSHDB >/dev/null"
}

main() {
  parse_args "$@"
  load_host_config
  load_app_env
  resolve_git_version
  resolve_target_settings
  stage_clean_repo
  resolve_build_strategy
  validate_requested_operation

  if [ "$DRY_RUN" = "true" ]; then
    print_dry_run
    exit 0
  fi

  check_dependencies
  prepare_remote_runtime
  ensure_test_database
  build_remote_image
  sync_production_data_from_legacy
  deploy_remote_app
  verify_remote_app
  update_server_address_option
  configure_remote_caddy
  verify_remote_entrypoint
  clear_remote_cache

  print_dry_run
}

main "$@"

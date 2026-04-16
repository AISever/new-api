#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"
REPO_DIR="${REPO_DIR:-$PROJECT_ROOT}"

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
FRONTEND_BUILD_NODE_OPTIONS="${FRONTEND_BUILD_NODE_OPTIONS:---max-old-space-size=4096}"
BUILD_STRATEGY_REQUESTED="${BUILD_STRATEGY:-auto}"
TARGET_IMAGE_PLATFORM="${TARGET_IMAGE_PLATFORM:-linux/amd64}"
REMOTE_BUILD_MIN_MEM_AVAILABLE_MB="${REMOTE_BUILD_MIN_MEM_AVAILABLE_MB:-2048}"
REMOTE_BUILD_MAX_LOAD1="${REMOTE_BUILD_MAX_LOAD1:-4.00}"

TARGET_ENV=""
ACTION="deploy"
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
TEST_HOSTNAME=""
ENTERPRISE_HOSTNAME=""
ENTERPRISE_HOSTNAME_ALIASES=""
PUBLIC_HOSTNAME_LIST=""
PUBLIC_CADDY_SITE_ADDRESSES=""
ENTERPRISE_HOSTNAME_LIST=""
ENTERPRISE_CADDY_SITE_ADDRESSES=""
TARGET_HOSTNAME=""
TARGET_HOSTNAME_LIST=""
TARGET_CADDY_SITE_ADDRESSES=""
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
DOCS_MANIFEST_PATH=""
REMOTE_NETWORK="new-api_default"
POSTGRES_CONTAINER="new-api-postgres"
REDIS_CONTAINER="new-api-redis"
CADDY_CONTAINER="snowlight-caddy"
CADDY_CONFIG_STATUS="not-run"
BUILD_STRATEGY_RESOLVED=""
LEGACY_BUILD_DIR=""
DURATION_STAGE_SECONDS="0"
DURATION_IMAGE_SECONDS="0"
DURATION_IMAGE_BUILD_SECONDS="0"
DURATION_IMAGE_EXPORT_SECONDS="0"
DURATION_IMAGE_UPLOAD_SECONDS="0"
DURATION_IMAGE_LOAD_SECONDS="0"
DURATION_DATA_SYNC_SECONDS="0"
DURATION_REMOTE_START_SECONDS="0"
DURATION_VERIFY_SECONDS="0"
DURATION_TOTAL_SECONDS="0"
DEPLOY_GIT_REMOTE_NAME="${DEPLOY_GIT_REMOTE:-origin}"
DEPLOY_GIT_REF_NAME="${DEPLOY_GIT_REF:-}"
HEAD_PUSH_VERIFIED="false"

usage() {
  cat <<EOF
Deploy the new kkidc host using committed source already pushed to the remote git ref.

Usage:
  $(basename "$0") production|enterprise|test|test-stop [options]

Options:
  --config PATH         host config file (default: .kkidc/.env.lighthouse)
  --app-env-file PATH   app secret config file (default: .env.local)
  --dry-run             print resolved settings and staged source path
  --keep-stage-dir      keep staged archive directory after exit
  --skip-build          reuse existing remote image tag for current HEAD
  --build-strategy MODE auto | local | legacy-remote | remote (default: auto -> local)
  --clear-cache         flush Redis DB for the target environment after deploy
  --sync-prod-data-from-legacy
                        restore old kkidc production data onto the new host before deploy
  --help                show this help
EOF
}

log() {
  printf '[%s] %s\n' "$1" "$2"
}

elapsed_seconds() {
  local started_at="$1"
  local finished_at="$2"
  echo $((finished_at - started_at))
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
    enterprise)
      TARGET_ENV="enterprise"
      ;;
    test)
      TARGET_ENV="test"
      ;;
    test-stop)
      TARGET_ENV="test"
      ACTION="stop"
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
  TEST_HOSTNAME="${TEST_HOSTNAME:-}"
  ENTERPRISE_HOSTNAME="${ENTERPRISE_HOSTNAME:-}"
  ENTERPRISE_HOSTNAME_ALIASES="${ENTERPRISE_HOSTNAME_ALIASES:-}"

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

build_hostname_list() {
  local primary_hostname="$1"
  local alias_hostnames="$2"

  if [ -n "$primary_hostname" ] && [ -n "$alias_hostnames" ]; then
    printf '%s,%s\n' "$primary_hostname" "$alias_hostnames"
    return
  fi

  printf '%s\n' "$primary_hostname"
}

format_caddy_site_addresses() {
  local hostname_list="$1"

  if [ -z "$hostname_list" ]; then
    printf '\n'
    return
  fi

  printf '%s\n' "${hostname_list//,/, }"
}

resolve_deploy_git_ref() {
  local upstream_ref=""

  if [ -n "$DEPLOY_GIT_REF_NAME" ]; then
    return
  fi

  if [ -n "$BRANCH_NAME" ] && [ "$BRANCH_NAME" != "detached-head" ]; then
    upstream_ref="$(git -C "$REPO_DIR" for-each-ref --format='%(upstream:short)' "refs/heads/${BRANCH_NAME}" | head -n 1)"
  fi

  if [ -n "$upstream_ref" ]; then
    DEPLOY_GIT_REF_NAME="$upstream_ref"
    return
  fi

  [ "$BRANCH_NAME" != "detached-head" ] || die "deploy requires a branch with a pushed remote ref; detached HEAD must set DEPLOY_GIT_REF explicitly"
  DEPLOY_GIT_REF_NAME="${DEPLOY_GIT_REMOTE_NAME}/${BRANCH_NAME}"
}

verify_head_pushed() {
  resolve_deploy_git_ref

  git -C "$REPO_DIR" fetch --quiet "$DEPLOY_GIT_REMOTE_NAME" || die "failed to fetch remote '${DEPLOY_GIT_REMOTE_NAME}' to verify pushed HEAD"
  git -C "$REPO_DIR" rev-parse --verify "$DEPLOY_GIT_REF_NAME" >/dev/null 2>&1 || die "deploy requires HEAD to be pushed; remote ref '${DEPLOY_GIT_REF_NAME}' does not exist"

  if ! git -C "$REPO_DIR" merge-base --is-ancestor HEAD "$DEPLOY_GIT_REF_NAME"; then
    die "deploy requires HEAD to be pushed; local HEAD is not contained in '${DEPLOY_GIT_REF_NAME}'"
  fi

  HEAD_PUSH_VERIFIED="true"
}

resolve_target_settings() {
  PUBLIC_HOSTNAME_LIST="$(build_hostname_list "$PUBLIC_HOSTNAME" "$PUBLIC_HOSTNAME_ALIASES")"
  PUBLIC_CADDY_SITE_ADDRESSES="$(format_caddy_site_addresses "$PUBLIC_HOSTNAME_LIST")"
  ENTERPRISE_HOSTNAME_LIST="$(build_hostname_list "$ENTERPRISE_HOSTNAME" "$ENTERPRISE_HOSTNAME_ALIASES")"
  ENTERPRISE_CADDY_SITE_ADDRESSES="$(format_caddy_site_addresses "$ENTERPRISE_HOSTNAME_LIST")"

  if [ "$TARGET_ENV" = "production" ]; then
    APP_CONTAINER="new-api-local"
    APP_PORT="3000"
    PG_DB="new-api"
    REDIS_DB_INDEX="0"
    DATA_DIR="/opt/new-api/data"
    LOG_DIR="/opt/new-api/logs"
    REMOTE_BUILD_DIR="/opt/new-api-build-production"
    IMAGE_NAME="new-api:kkidc-production-${SHA}"
    TARGET_HOSTNAME="$PUBLIC_HOSTNAME"
    TARGET_HOSTNAME_LIST="$PUBLIC_HOSTNAME_LIST"
    TARGET_CADDY_SITE_ADDRESSES="$PUBLIC_CADDY_SITE_ADDRESSES"
    if [ -n "$TARGET_HOSTNAME" ]; then
      SERVER_ADDRESS="https://${TARGET_HOSTNAME}"
    else
      SERVER_ADDRESS="http://${REMOTE_HOST}:${APP_PORT}"
    fi
    DOCS_MANIFEST_PATH=""
  elif [ "$TARGET_ENV" = "enterprise" ]; then
    APP_CONTAINER="new-api-enterprise"
    APP_PORT="3002"
    PG_DB="new-api-enterprise"
    REDIS_DB_INDEX="2"
    DATA_DIR="/opt/new-api-enterprise/data"
    LOG_DIR="/opt/new-api-enterprise/logs"
    REMOTE_BUILD_DIR="/opt/new-api-build-enterprise"
    IMAGE_NAME="new-api:kkidc-enterprise-${SHA}"
    TARGET_HOSTNAME="$ENTERPRISE_HOSTNAME"
    TARGET_HOSTNAME_LIST="$ENTERPRISE_HOSTNAME_LIST"
    TARGET_CADDY_SITE_ADDRESSES="$ENTERPRISE_CADDY_SITE_ADDRESSES"
    if [ -n "$TARGET_HOSTNAME" ]; then
      SERVER_ADDRESS="https://${TARGET_HOSTNAME}"
    else
      SERVER_ADDRESS="http://${REMOTE_HOST}:${APP_PORT}"
    fi
    DOCS_MANIFEST_PATH="/enterprise-docs/apifox/manifest.json"
  else
    APP_CONTAINER="new-api-test"
    APP_PORT="3001"
    PG_DB="new-api-test"
    REDIS_DB_INDEX="1"
    DATA_DIR="/opt/new-api-test/data"
    LOG_DIR="/opt/new-api-test/logs"
    REMOTE_BUILD_DIR="/opt/new-api-build-test"
    IMAGE_NAME="new-api:kkidc-test-${SHA}"
    TARGET_HOSTNAME="$TEST_HOSTNAME"
    TARGET_HOSTNAME_LIST="$TEST_HOSTNAME"
    TARGET_CADDY_SITE_ADDRESSES=""
    if [ -n "$TEST_HOSTNAME" ]; then
      SERVER_ADDRESS="http://${TEST_HOSTNAME}:${APP_PORT}"
    else
      SERVER_ADDRESS="http://${REMOTE_HOST}:${APP_PORT}"
    fi
    DOCS_MANIFEST_PATH="/enterprise-docs/apifox/manifest.json"
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
      BUILD_STRATEGY_RESOLVED="local"
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
  sed -i.bak "s|RUN DISABLE_ESLINT_PLUGIN='true' VITE_REACT_APP_VERSION=\$(cat VERSION) npm run build|RUN DISABLE_ESLINT_PLUGIN='true' NODE_OPTIONS='${FRONTEND_BUILD_NODE_OPTIONS}' VITE_REACT_APP_VERSION=\$(cat VERSION) npm run build|" "$STAGE_DIR/Dockerfile.deploy"
  python3 - "$STAGE_DIR/Dockerfile.deploy" <<'PY'
from pathlib import Path
import sys

path = Path(sys.argv[1])
content = path.read_text()
needle = "RUN apk add --no-cache ca-certificates tzdata wget \\"
replacement = "RUN sed -i 's|https://dl-cdn.alpinelinux.org/alpine|https://mirrors.aliyun.com/alpine|g' /etc/apk/repositories \\\n    && apk add --no-cache ca-certificates tzdata wget \\"
if needle in content:
    path.write_text(content.replace(needle, replacement, 1))
PY
  rm -f "$STAGE_DIR/Dockerfile.deploy.bak"
}

print_dry_run() {
  cat <<EOF
action=${ACTION}
target_env=${TARGET_ENV}
remote_host=${REMOTE_HOST}
remote_user=${REMOTE_USER}
public_hostname=${TARGET_HOSTNAME}
public_hostnames=${TARGET_HOSTNAME_LIST}
caddy_site_addresses=${TARGET_CADDY_SITE_ADDRESSES}
app_container=${APP_CONTAINER}
app_port=${APP_PORT}
pg_db=${PG_DB}
redis_conn_string=${REDIS_CONN_STRING}
caddy_config_status=${CADDY_CONFIG_STATUS}
data_dir=${DATA_DIR}
log_dir=${LOG_DIR}
remote_build_dir=${REMOTE_BUILD_DIR}
image_name=${IMAGE_NAME}
version_value=${VERSION_VALUE}
server_address=${SERVER_ADDRESS}
docs_manifest_path=${DOCS_MANIFEST_PATH}
deploy_git_remote=${DEPLOY_GIT_REMOTE_NAME}
deploy_git_ref=${DEPLOY_GIT_REF_NAME}
head_pushed_verified=${HEAD_PUSH_VERIFIED}
stage_dir=${STAGE_DIR}
build_strategy=${BUILD_STRATEGY_RESOLVED}
target_image_platform=${TARGET_IMAGE_PLATFORM}
sync_prod_data_from_legacy=${SYNC_PROD_DATA_FROM_LEGACY}
EOF
}

print_deploy_summary() {
  print_dry_run
  cat <<EOF
duration_stage_seconds=${DURATION_STAGE_SECONDS}
duration_image_seconds=${DURATION_IMAGE_SECONDS}
duration_image_build_seconds=${DURATION_IMAGE_BUILD_SECONDS}
duration_image_export_seconds=${DURATION_IMAGE_EXPORT_SECONDS}
duration_image_upload_seconds=${DURATION_IMAGE_UPLOAD_SECONDS}
duration_image_load_seconds=${DURATION_IMAGE_LOAD_SECONDS}
duration_data_sync_seconds=${DURATION_DATA_SYNC_SECONDS}
duration_remote_start_seconds=${DURATION_REMOTE_START_SECONDS}
duration_verify_seconds=${DURATION_VERIFY_SECONDS}
duration_total_seconds=${DURATION_TOTAL_SECONDS}
EOF
}

check_dependencies() {
  command -v sshpass >/dev/null 2>&1 || die "sshpass is required"
  if [ "$ACTION" = "stop" ]; then
    return
  fi

  command -v git >/dev/null 2>&1 || die "git is required"
  command -v tar >/dev/null 2>&1 || die "tar is required"
  if [ "$BUILD_STRATEGY_RESOLVED" = "local" ] && ! local_docker_available; then
    die "local docker is unavailable for build strategy=local"
  fi
}

check_remote_build_guard() {
  local mem_available_mb
  local load1

  if [ "$BUILD_STRATEGY_RESOLVED" != "remote" ]; then
    return
  fi

  mem_available_mb="$(remote_cmd "free -m | awk '/^Mem:/ {print \$7}'")"
  load1="$(remote_cmd "uptime | awk -F'load average: ' '{print \$2}' | cut -d',' -f1 | tr -d ' '")"

  if [ -z "$mem_available_mb" ] || [ -z "$load1" ]; then
    die "remote build guard could not determine host resources"
  fi

  if [ "$mem_available_mb" -lt "$REMOTE_BUILD_MIN_MEM_AVAILABLE_MB" ]; then
    die "remote build guard rejected current host state: MemAvailable=${mem_available_mb}MB < ${REMOTE_BUILD_MIN_MEM_AVAILABLE_MB}MB"
  fi

  if ! awk -v value="$load1" -v max="$REMOTE_BUILD_MAX_LOAD1" 'BEGIN { exit !(value <= max) }'; then
    die "remote build guard rejected current host state: load1=${load1} > ${REMOTE_BUILD_MAX_LOAD1}"
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
    -o ServerAliveInterval=30 \
    -o ServerAliveCountMax=10 \
    -o PreferredAuthentications=password \
    -o PubkeyAuthentication=no \
    "${REMOTE_USER}@${REMOTE_HOST}" "$@"
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

legacy_cmd() {
  sshpass -p "$LEGACY_BUILD_PASSWORD" ssh \
    -o StrictHostKeyChecking=no \
    -o ServerAliveInterval=30 \
    -o ServerAliveCountMax=10 \
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
    -o ServerAliveInterval=30 \
    -o ServerAliveCountMax=10 \
    -o PreferredAuthentications=password \
    -o PubkeyAuthentication=no \
    "${target_user}@${target_host}" "rm -rf '$target_dir' && mkdir -p '$target_dir' && tar -xf - -C '$target_dir'"
}

copy_file_to_remote() {
  local source_file="$1"
  local target_file="$2"

  sshpass -p "$REMOTE_PASSWORD" scp \
    -o StrictHostKeyChecking=no \
    -o ServerAliveInterval=30 \
    -o ServerAliveCountMax=10 \
    -o PreferredAuthentications=password \
    -o PubkeyAuthentication=no \
    "$source_file" "${REMOTE_USER}@${REMOTE_HOST}:$target_file"
}

prepare_remote_runtime() {
  remote_bash <<EOF
set -euo pipefail
mkdir -p '${DATA_DIR}' '${LOG_DIR}' '${REMOTE_BUILD_DIR}' /opt/snowlight /opt/snowlight/data /opt/snowlight/config
docker network inspect '${REMOTE_NETWORK}' >/dev/null 2>&1 || docker network create '${REMOTE_NETWORK}' >/dev/null
EOF
}

ensure_target_database() {
  if [ "$TARGET_ENV" = "production" ]; then
    return
  fi

  remote_bash <<EOF
set -euo pipefail
docker exec '${POSTGRES_CONTAINER}' psql -U newapi -d postgres -tc "SELECT 1 FROM pg_database WHERE datname = '${PG_DB}'" | grep -q 1 || \
docker exec '${POSTGRES_CONTAINER}' psql -U newapi -d postgres -c "CREATE DATABASE \\"${PG_DB}\\";"
EOF
}

build_remote_image() {
  local step_started_at
  local image_archive_path
  local remote_image_archive_path

  case "$BUILD_STRATEGY_RESOLVED" in
    skip)
      return
      ;;
    local)
      step_started_at="$SECONDS"
      docker build --platform "$TARGET_IMAGE_PLATFORM" -t "$IMAGE_NAME" -f "$STAGE_DIR/Dockerfile.deploy" "$STAGE_DIR"
      DURATION_IMAGE_BUILD_SECONDS="$(elapsed_seconds "$step_started_at" "$SECONDS")"

      image_archive_path="$(mktemp "${TMPDIR:-/tmp}/kkidc-image-${TARGET_ENV}-${SHA}-XXXXXX.tar")"
      remote_image_archive_path="${REMOTE_BUILD_DIR}/${IMAGE_NAME//[:\/]/-}.tar"

      step_started_at="$SECONDS"
      docker save -o "$image_archive_path" "$IMAGE_NAME"
      DURATION_IMAGE_EXPORT_SECONDS="$(elapsed_seconds "$step_started_at" "$SECONDS")"

      step_started_at="$SECONDS"
      copy_file_to_remote "$image_archive_path" "$remote_image_archive_path"
      DURATION_IMAGE_UPLOAD_SECONDS="$(elapsed_seconds "$step_started_at" "$SECONDS")"

      step_started_at="$SECONDS"
      remote_cmd "docker load -i '$remote_image_archive_path' >/dev/null && rm -f '$remote_image_archive_path'"
      DURATION_IMAGE_LOAD_SECONDS="$(elapsed_seconds "$step_started_at" "$SECONDS")"

      rm -f "$image_archive_path"
      ;;
    legacy-remote)
      LEGACY_BUILD_DIR="/tmp/new-api-build-${TARGET_ENV}-${SHA}"
      sync_stage_to_host "$LEGACY_BUILD_HOST" "$LEGACY_BUILD_USER" "$LEGACY_BUILD_PASSWORD" "$LEGACY_BUILD_DIR"
      step_started_at="$SECONDS"
      legacy_cmd "cd '$LEGACY_BUILD_DIR' && docker build -t '$IMAGE_NAME' -f Dockerfile.deploy ."
      DURATION_IMAGE_BUILD_SECONDS="$(elapsed_seconds "$step_started_at" "$SECONDS")"
      step_started_at="$SECONDS"
      legacy_cmd "docker save '$IMAGE_NAME'" | remote_cmd "docker load >/dev/null"
      DURATION_IMAGE_UPLOAD_SECONDS="$(elapsed_seconds "$step_started_at" "$SECONDS")"
      ;;
    remote)
      sync_stage_to_host "$REMOTE_HOST" "$REMOTE_USER" "$REMOTE_PASSWORD" "$REMOTE_BUILD_DIR"
      step_started_at="$SECONDS"
      remote_cmd "cd '$REMOTE_BUILD_DIR' && docker build -t '$IMAGE_NAME' -f Dockerfile.deploy ."
      DURATION_IMAGE_BUILD_SECONDS="$(elapsed_seconds "$step_started_at" "$SECONDS")"
      ;;
  esac
}

build_caddy_site_block() {
  local site_addresses="$1"
  local upstream_port="$2"

  [ -n "$site_addresses" ] || return 0

  cat <<EOF
${site_addresses} {
    encode gzip
    reverse_proxy 172.17.0.1:${upstream_port} {
        header_up X-Real-IP {remote_host}
        header_up Host {host}
    }
}

EOF
}

build_caddy_http_fallback_block() {
  cat <<EOF
http://${REMOTE_HOST} {
    encode gzip
    reverse_proxy 172.17.0.1:3000 {
        header_up X-Real-IP {remote_host}
        header_up Host {host}
    }
}
EOF
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
  local site_block=""
  local caddyfile_content=""
  local existing_caddyfile=""

  site_block="$(build_caddy_site_block "$PUBLIC_CADDY_SITE_ADDRESSES" "3000")"
  if [ -n "$site_block" ]; then
    caddyfile_content+="${site_block}"$'\n'
  fi

  site_block="$(build_caddy_site_block "$ENTERPRISE_CADDY_SITE_ADDRESSES" "3002")"
  if [ -n "$site_block" ]; then
    caddyfile_content+="${site_block}"$'\n'
  fi

  if [ -n "$PUBLIC_HOSTNAME" ]; then
    caddyfile_content+="$(build_caddy_http_fallback_block)"
  fi

  if [ -z "$caddyfile_content" ]; then
    CADDY_CONFIG_STATUS="skipped-empty"
    return
  fi

  existing_caddyfile="$(remote_cmd "cat /opt/snowlight/Caddyfile 2>/dev/null || true")"
  if [ "$existing_caddyfile" = "$caddyfile_content" ] && remote_cmd "docker container inspect '${CADDY_CONTAINER}' >/dev/null 2>&1"; then
    log INFO "caddy config unchanged; skipping caddy restart"
    CADDY_CONFIG_STATUS="unchanged"
    return
  fi

  remote_bash <<EOF
set -euo pipefail
cat > /opt/snowlight/Caddyfile <<'CADDY'
${caddyfile_content}
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
  CADDY_CONFIG_STATUS="restarted"
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

update_runtime_options() {
  remote_bash <<EOF
set -euo pipefail
docker exec -i '${POSTGRES_CONTAINER}' psql -U newapi -d '${PG_DB}' <<'SQL' >/dev/null 2>&1 || exit 0
INSERT INTO options (key, value)
VALUES ('ServerAddress', '${SERVER_ADDRESS}')
ON CONFLICT (key) DO UPDATE SET value = EXCLUDED.value;
$(if [ -n "$DOCS_MANIFEST_PATH" ]; then cat <<EOF2
INSERT INTO options (key, value)
VALUES ('general_setting.docs_manifest_path', '${DOCS_MANIFEST_PATH}')
ON CONFLICT (key) DO UPDATE SET value = EXCLUDED.value;
EOF2
fi)
SQL
docker restart '${APP_CONTAINER}' >/dev/null
EOF
}

verify_remote_entrypoint() {
  if [ -n "$TARGET_HOSTNAME" ]; then
    remote_bash <<EOF
set -euo pipefail
for i in \$(seq 1 90); do
  if curl -kfsS --resolve '${TARGET_HOSTNAME}:443:127.0.0.1' 'https://${TARGET_HOSTNAME}/api/status' >/dev/null 2>&1; then
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

stop_test_app() {
  local already_stopped="false"
  local stopped_container="${APP_CONTAINER}"

  if remote_cmd "docker container inspect '${APP_CONTAINER}' >/dev/null 2>&1"; then
    remote_cmd "docker rm -f '${APP_CONTAINER}' >/dev/null"
  else
    already_stopped="true"
    stopped_container=""
  fi

  cat <<EOF
action=test-stop
target_env=${TARGET_ENV}
remote_host=${REMOTE_HOST}
remote_user=${REMOTE_USER}
app_container=${APP_CONTAINER}
already_stopped=${already_stopped}
stopped_container=${stopped_container}
retained_data_dir=${DATA_DIR}
retained_log_dir=${LOG_DIR}
retained_pg_db=${PG_DB}
retained_redis_db=${REDIS_DB_INDEX}
EOF
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
  local total_started_at="$SECONDS"
  local stage_started_at
  local image_started_at
  local data_sync_started_at
  local remote_start_started_at
  local verify_started_at

  parse_args "$@"
  load_host_config

  if [ "$ACTION" = "stop" ]; then
    resolve_target_settings
    if [ "$DRY_RUN" = "true" ]; then
      print_dry_run
      exit 0
    fi

    check_dependencies
    stop_test_app
    exit 0
  fi

  load_app_env
  resolve_git_version
  resolve_target_settings
  verify_head_pushed
  stage_started_at="$SECONDS"
  stage_clean_repo
  resolve_build_strategy
  validate_requested_operation
  DURATION_STAGE_SECONDS="$(elapsed_seconds "$stage_started_at" "$SECONDS")"

  if [ "$DRY_RUN" = "true" ]; then
    print_dry_run
    exit 0
  fi

  check_dependencies
  check_remote_build_guard

  image_started_at="$SECONDS"
  prepare_remote_runtime
  ensure_target_database
  build_remote_image
  DURATION_IMAGE_SECONDS="$(elapsed_seconds "$image_started_at" "$SECONDS")"

  data_sync_started_at="$SECONDS"
  sync_production_data_from_legacy
  DURATION_DATA_SYNC_SECONDS="$(elapsed_seconds "$data_sync_started_at" "$SECONDS")"

  remote_start_started_at="$SECONDS"
  deploy_remote_app
  DURATION_REMOTE_START_SECONDS="$(elapsed_seconds "$remote_start_started_at" "$SECONDS")"

  verify_started_at="$SECONDS"
  verify_remote_app
  update_runtime_options
  configure_remote_caddy
  verify_remote_entrypoint
  clear_remote_cache
  DURATION_VERIFY_SECONDS="$(elapsed_seconds "$verify_started_at" "$SECONDS")"
  DURATION_TOTAL_SECONDS="$(elapsed_seconds "$total_started_at" "$SECONDS")"

  print_deploy_summary
}

main "$@"

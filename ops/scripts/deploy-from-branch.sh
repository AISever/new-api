#!/usr/bin/env bash
set -euo pipefail

REPO_DIR="${REPO_DIR:-$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)}"
DEFAULT_BRANCH="${DEFAULT_BRANCH:-codex/prod-live}"
BRANCH="${1:-${BRANCH:-$DEFAULT_BRANCH}}"

PROJECT_NAME="${PROJECT_NAME:-}"
IMAGE_NAME="${IMAGE_NAME:-}"
COMPOSE_MODE="${COMPOSE_MODE:-full}"
COMPOSE_FILE="${COMPOSE_FILE:-}"
BASE_COMPOSE_FILE="${BASE_COMPOSE_FILE:-docker-compose.yml}"
APP_CONTAINER="${APP_CONTAINER:-new-api}"
REDIS_CONTAINER="${REDIS_CONTAINER:-redis}"
POSTGRES_CONTAINER="${POSTGRES_CONTAINER:-postgres}"
APP_PORT="${APP_PORT:-3000}"
PG_DB="${PG_DB:-new-api}"
VERSION_SUFFIX="${VERSION_SUFFIX:-}"
DEPLOY_LABEL="${DEPLOY_LABEL:-tencent environment}"
FETCH_TIMEOUT_SECONDS="${FETCH_TIMEOUT_SECONDS:-180}"
ALLOW_STALE_ON_FETCH_TIMEOUT="${ALLOW_STALE_ON_FETCH_TIMEOUT:-false}"
FRONTEND_BUILD_NODE_OPTIONS="${FRONTEND_BUILD_NODE_OPTIONS:---max-old-space-size=2048}"
BUILD_IMAGE="${BUILD_IMAGE:-true}"
COMPOSE_BUILD_MEMORY="${COMPOSE_BUILD_MEMORY:-5g}"

usage() {
  cat <<EOF
Deploy ${DEPLOY_LABEL} from a git branch.

Usage:
  $(basename "$0") [branch]

Runtime env:
  REPO_DIR           default: current repo root
  PROJECT_NAME       required
  IMAGE_NAME         required
  COMPOSE_MODE       full | override (default: full)
  COMPOSE_FILE       required when COMPOSE_MODE=full
  BASE_COMPOSE_FILE  default: docker-compose.yml
  APP_CONTAINER      default: new-api
  REDIS_CONTAINER    default: redis
  POSTGRES_CONTAINER default: postgres
  APP_PORT           default: 3000
  PG_DB              default: new-api
  VERSION_SUFFIX     default: empty
  FETCH_TIMEOUT_SECONDS default: 180
  FRONTEND_BUILD_NODE_OPTIONS default: --max-old-space-size=2048
  BUILD_IMAGE        default: true
  COMPOSE_BUILD_MEMORY default: 5g
EOF
}

if [ "${1:-}" = "-h" ] || [ "${1:-}" = "--help" ]; then
  usage
  exit 0
fi

if [ -z "$PROJECT_NAME" ]; then
  echo "ERROR: PROJECT_NAME is required" >&2
  exit 2
fi

if [ -z "$IMAGE_NAME" ]; then
  echo "ERROR: IMAGE_NAME is required" >&2
  exit 2
fi

if [ "$COMPOSE_MODE" != "full" ] && [ "$COMPOSE_MODE" != "override" ]; then
  echo "ERROR: unsupported COMPOSE_MODE=$COMPOSE_MODE" >&2
  exit 2
fi

if [ "$COMPOSE_MODE" = "full" ] && [ -z "$COMPOSE_FILE" ]; then
  echo "ERROR: COMPOSE_FILE is required when COMPOSE_MODE=full" >&2
  exit 2
fi

cd "$REPO_DIR"
mkdir -p data logs

fetch_and_checkout_branch() {
  local fetch_ok="false"

  if timeout "$FETCH_TIMEOUT_SECONDS" git fetch --depth 1 --no-tags origin "$BRANCH"; then
    fetch_ok="true"
  else
    if [ "$ALLOW_STALE_ON_FETCH_TIMEOUT" = "true" ]; then
      echo "WARN: git fetch timeout, use local branch code" >&2
    else
      echo "ERROR: git fetch timed out for branch $BRANCH; refusing to deploy stale local code" >&2
      exit 1
    fi
  fi

  if git show-ref --verify --quiet "refs/heads/${BRANCH}"; then
    git checkout "$BRANCH"
  else
    git checkout -b "$BRANCH" "origin/$BRANCH"
  fi

  if [ "$fetch_ok" = "true" ] && git rev-parse --verify -q FETCH_HEAD >/dev/null; then
    git reset --hard FETCH_HEAD
  fi
}

prepare_dockerfile() {
  cp Dockerfile Dockerfile.deploy
  sed -i "s|RUN go mod download|RUN go env -w GOPROXY=https://goproxy.cn,direct \\&\\& go mod download|" Dockerfile.deploy
  sed -i "s|RUN DISABLE_ESLINT_PLUGIN='true' VITE_REACT_APP_VERSION=\$(cat VERSION) bun run build|RUN DISABLE_ESLINT_PLUGIN='true' NODE_OPTIONS='${FRONTEND_BUILD_NODE_OPTIONS}' VITE_REACT_APP_VERSION=\$(cat VERSION) bun run build|" Dockerfile.deploy
}

write_full_compose() {
  cat > "$COMPOSE_FILE" <<YAML
version: '3.4'
services:
  new-api:
    image: ${IMAGE_NAME}
    container_name: ${APP_CONTAINER}
    restart: always
    command: --log-dir /app/logs
    build:
      context: .
      dockerfile: Dockerfile.deploy
    ports:
      - "${APP_PORT}:3000"
    volumes:
      - ./data:/data
      - ./logs:/app/logs
    environment:
      SQL_DSN: postgresql://newapi:\${DB_PASSWORD:-NewApi2024Secure}@postgres:5432/${PG_DB}
      REDIS_CONN_STRING: redis://redis
      TZ: Asia/Shanghai
      SESSION_SECRET: \${SESSION_SECRET:-ChangeThisToRandomString2024}
      CRYPTO_SECRET: \${CRYPTO_SECRET:-ChangeThisToAnotherRandomString2024}
      ERROR_LOG_ENABLED: "true"
      BATCH_UPDATE_ENABLED: "true"
      MEMORY_CACHE_ENABLED: "true"
      VERSION: ${VERSION_VALUE}
    depends_on:
      - redis
      - postgres
    healthcheck:
      test: ["CMD-SHELL", "wget -q -O - http://localhost:3000/api/status | grep -Eq '\"success\":[[:space:]]*true' || exit 1"]
      interval: 30s
      timeout: 10s
      retries: 3

  redis:
    image: redis:latest
    container_name: ${REDIS_CONTAINER}
    restart: always

  postgres:
    image: postgres:15
    container_name: ${POSTGRES_CONTAINER}
    restart: always
    environment:
      POSTGRES_USER: newapi
      POSTGRES_PASSWORD: \${DB_PASSWORD:-NewApi2024Secure}
      POSTGRES_DB: ${PG_DB}
    volumes:
      - pg_data:/var/lib/postgresql/data

volumes:
  pg_data:
YAML
}

write_override_compose() {
  cat > "$COMPOSE_FILE" <<YAML
services:
  new-api:
    image: ${IMAGE_NAME}
    build:
      context: .
      dockerfile: Dockerfile.deploy
    environment:
      - VERSION=${VERSION_VALUE}
YAML
}

compose_cmd() {
  if [ "$COMPOSE_MODE" = "override" ]; then
    docker compose -f "$BASE_COMPOSE_FILE" -f "$COMPOSE_FILE" -p "$PROJECT_NAME" "$@"
  else
    docker compose -f "$COMPOSE_FILE" -p "$PROJECT_NAME" "$@"
  fi
}

fetch_and_checkout_branch
SHA="$(git rev-parse --short HEAD)"
VERSION_VALUE="${BRANCH}+${SHA}${VERSION_SUFFIX}"
prepare_dockerfile

if [ "$COMPOSE_MODE" = "override" ]; then
  COMPOSE_FILE="${COMPOSE_FILE:-docker-compose.override.yml}"
  [ -f "$BASE_COMPOSE_FILE" ] || {
    echo "ERROR: missing base compose file: $REPO_DIR/$BASE_COMPOSE_FILE" >&2
    exit 2
  }
  write_override_compose
else
  write_full_compose
fi

if [ "$BUILD_IMAGE" = "true" ]; then
  compose_cmd build --memory "$COMPOSE_BUILD_MEMORY" new-api
  compose_cmd up -d
else
  compose_cmd up -d --no-build
fi

if [[ "$COMPOSE_FILE" = /* ]]; then
  COMPOSE_PATH="$COMPOSE_FILE"
else
  COMPOSE_PATH="${REPO_DIR}/${COMPOSE_FILE}"
fi

echo "deployed_branch=${BRANCH}"
echo "deployed_sha=${SHA}"
echo "deployed_version=${VERSION_VALUE}"
echo "compose_mode=${COMPOSE_MODE}"
echo "compose_file=${COMPOSE_PATH}"
echo "build_image=${BUILD_IMAGE}"

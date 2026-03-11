#!/usr/bin/env bash
set -euo pipefail

REPO_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
ENV_FILE="$REPO_DIR/.tencent/.env.lighthouse"

TARGET="test"
BRANCH="codex/prod-live"
IMAGE_NAME=""
PLATFORM="${PLATFORM:-linux/amd64}"
NO_DEPLOY="false"
FETCH_TIMEOUT_SECONDS="${FETCH_TIMEOUT_SECONDS:-180}"
FRONTEND_BUILD_NODE_OPTIONS="${FRONTEND_BUILD_NODE_OPTIONS:---max-old-space-size=2048}"

LIGHTHOUSE_SSH_HOST="${LIGHTHOUSE_SSH_HOST:-}"
LIGHTHOUSE_SSH_USER="${LIGHTHOUSE_SSH_USER:-root}"
LIGHTHOUSE_SSH_KEY="${LIGHTHOUSE_SSH_KEY:-}"

REMOTE_REPO_DIR=""
REMOTE_DEPLOY_SCRIPT=""

usage() {
  cat <<'EOF'
Build image on local machine and upload to tencent, then deploy without remote build.

Usage:
  ./ops/scripts/tencent-local-build-upload-deploy.sh [options]

Options:
  --target <test|standby>   default: test
  --branch <branch>         default: codex/prod-live
  --image <name:tag>        target-specific default if omitted
  --platform <platform>     default: linux/amd64
  --env-file <path>         default: .tencent/.env.lighthouse
  --repo-dir <path>         default: current repo root
  --no-deploy               upload only, do not run remote deploy
  -h, --help

Environment (optional override):
  LIGHTHOUSE_SSH_HOST
  LIGHTHOUSE_SSH_USER
  LIGHTHOUSE_SSH_KEY
  FETCH_TIMEOUT_SECONDS     default: 5
  FRONTEND_BUILD_NODE_OPTIONS default: --max-old-space-size=2048

Notes:
  - This script keeps existing deployment modes untouched; it adds an image-upload path.
  - Build source is the current local workspace at REPO_DIR.
  - The current checkout must be clean and exactly match origin/<branch>.
EOF
}

verify_local_source_matches_branch() {
  local current_branch
  local worktree_status
  local local_sha
  local remote_sha

  if ! timeout "$FETCH_TIMEOUT_SECONDS" git -C "$REPO_DIR" fetch --depth 1 --no-tags origin "$BRANCH"; then
    echo "ERROR: git fetch timed out for $BRANCH; refusing local image deploy" >&2
    exit 1
  fi

  current_branch="$(git -C "$REPO_DIR" rev-parse --abbrev-ref HEAD)"
  if [ "$current_branch" != "$BRANCH" ]; then
    echo "ERROR: current local branch is $current_branch, expected $BRANCH" >&2
    echo "Hint: switch to $BRANCH before using local image upload deploy" >&2
    exit 2
  fi

  worktree_status="$(git -C "$REPO_DIR" status --porcelain)"
  if [ -n "$worktree_status" ]; then
    echo "ERROR: local workspace is not clean; refusing to build an image with uncommitted changes" >&2
    exit 2
  fi

  local_sha="$(git -C "$REPO_DIR" rev-parse HEAD)"
  remote_sha="$(git -C "$REPO_DIR" rev-parse FETCH_HEAD)"
  if [ "$local_sha" != "$remote_sha" ]; then
    echo "ERROR: local $BRANCH ($local_sha) does not match origin/$BRANCH ($remote_sha)" >&2
    echo "Hint: push or fast-forward the branch before using local image upload deploy" >&2
    exit 2
  fi
}

set_if_unset() {
  local key="$1"
  local value="$2"
  if [ -z "${!key:-}" ] && [ -n "$value" ]; then
    printf -v "$key" '%s' "$value"
    export "$key"
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

while [[ $# -gt 0 ]]; do
  case "$1" in
    --target)
      TARGET="${2:-}"
      shift 2
      ;;
    --branch)
      BRANCH="${2:-}"
      shift 2
      ;;
    --image)
      IMAGE_NAME="${2:-}"
      shift 2
      ;;
    --platform)
      PLATFORM="${2:-}"
      shift 2
      ;;
    --env-file)
      ENV_FILE="${2:-}"
      shift 2
      ;;
    --repo-dir)
      REPO_DIR="${2:-}"
      shift 2
      ;;
    --no-deploy)
      NO_DEPLOY="true"
      shift
      ;;
    -h|--help)
      usage
      exit 0
      ;;
    *)
      echo "ERROR: unknown option $1" >&2
      usage >&2
      exit 2
      ;;
  esac
done

if [ ! -d "$REPO_DIR/.git" ]; then
  echo "ERROR: REPO_DIR is not a git repository: $REPO_DIR" >&2
  exit 2
fi

verify_local_source_matches_branch

load_env_file "$ENV_FILE"
set_if_unset LIGHTHOUSE_SSH_HOST "${IP_1:-}"

if [ -z "${LIGHTHOUSE_SSH_KEY:-}" ]; then
  if [ -f "$REPO_DIR/.tencent/lighthouse-shanghai.pem" ]; then
    LIGHTHOUSE_SSH_KEY="$REPO_DIR/.tencent/lighthouse-shanghai.pem"
  elif [ -f "$REPO_DIR/.tencent/lighthouse.pem" ]; then
    LIGHTHOUSE_SSH_KEY="$REPO_DIR/.tencent/lighthouse.pem"
  fi
fi

if [ -n "${LIGHTHOUSE_SSH_KEY:-}" ] && [ ! -f "$LIGHTHOUSE_SSH_KEY" ] && [ -f "$REPO_DIR/$LIGHTHOUSE_SSH_KEY" ]; then
  LIGHTHOUSE_SSH_KEY="$REPO_DIR/$LIGHTHOUSE_SSH_KEY"
fi

if [ -z "${LIGHTHOUSE_SSH_HOST:-}" ] || [ -z "${LIGHTHOUSE_SSH_KEY:-}" ]; then
  echo "ERROR: missing LIGHTHOUSE_SSH_HOST or LIGHTHOUSE_SSH_KEY" >&2
  echo "Hint: check $ENV_FILE and .tencent key files" >&2
  exit 2
fi

case "$TARGET" in
  test)
    IMAGE_NAME="${IMAGE_NAME:-new-api:tencent-test}"
    REMOTE_REPO_DIR="/opt/new-api-test-src"
    REMOTE_DEPLOY_SCRIPT="./ops/scripts/tencent-test-deploy.sh"
    ;;
  standby)
    IMAGE_NAME="${IMAGE_NAME:-new-api:codex-prod-live}"
    REMOTE_REPO_DIR="/opt/new-api-src"
    REMOTE_DEPLOY_SCRIPT="./ops/scripts/tencent-standby-deploy.sh"
    ;;
  *)
    echo "ERROR: unsupported target: $TARGET (expected test or standby)" >&2
    exit 2
    ;;
esac

if ! command -v docker >/dev/null 2>&1; then
  echo "ERROR: docker is required" >&2
  exit 2
fi
if ! docker info >/dev/null 2>&1; then
  echo "ERROR: docker daemon is not available" >&2
  exit 2
fi
if ! docker buildx version >/dev/null 2>&1; then
  echo "ERROR: docker buildx is required" >&2
  exit 2
fi

TMP_DOCKERFILE="$(mktemp "$REPO_DIR/.dockerfile.deploy.XXXXXX")"
trap 'rm -f "$TMP_DOCKERFILE" "$TMP_DOCKERFILE.bak"' EXIT

cp "$REPO_DIR/Dockerfile" "$TMP_DOCKERFILE"
sed -i.bak "s|RUN go mod download|RUN go env -w GOPROXY=https://goproxy.cn,direct \\&\\& go mod download|" "$TMP_DOCKERFILE"
sed -i.bak "s|RUN DISABLE_ESLINT_PLUGIN='true' VITE_REACT_APP_VERSION=\$(cat VERSION) bun run build|RUN DISABLE_ESLINT_PLUGIN='true' NODE_OPTIONS='${FRONTEND_BUILD_NODE_OPTIONS}' VITE_REACT_APP_VERSION=\$(cat VERSION) bun run build|" "$TMP_DOCKERFILE"
rm -f "$TMP_DOCKERFILE.bak"

echo "==[1/3] Build local image: $IMAGE_NAME ($PLATFORM) ==" >&2
docker buildx build --platform "$PLATFORM" -t "$IMAGE_NAME" --load -f "$TMP_DOCKERFILE" "$REPO_DIR"

echo "==[2/3] Upload image to tencent: $LIGHTHOUSE_SSH_HOST ==" >&2
docker save "$IMAGE_NAME" | ssh -i "$LIGHTHOUSE_SSH_KEY" -o StrictHostKeyChecking=no -o ConnectTimeout=15 \
  "${LIGHTHOUSE_SSH_USER}@${LIGHTHOUSE_SSH_HOST}" "docker load"

if [ "$NO_DEPLOY" = "true" ]; then
  echo "UPLOAD_ONLY image=$IMAGE_NAME target=$TARGET host=$LIGHTHOUSE_SSH_HOST"
  exit 0
fi

echo "==[3/3] Deploy on tencent without build ==" >&2
ssh -i "$LIGHTHOUSE_SSH_KEY" -o StrictHostKeyChecking=no -o ConnectTimeout=15 \
  "${LIGHTHOUSE_SSH_USER}@${LIGHTHOUSE_SSH_HOST}" \
  "cd '$REMOTE_REPO_DIR' && IMAGE_NAME='$IMAGE_NAME' BUILD_IMAGE=false FETCH_TIMEOUT_SECONDS='$FETCH_TIMEOUT_SECONDS' '$REMOTE_DEPLOY_SCRIPT' '$BRANCH'"

echo "DEPLOY_OK image=$IMAGE_NAME target=$TARGET branch=$BRANCH host=$LIGHTHOUSE_SSH_HOST"

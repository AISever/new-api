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
PROFILE_PATH=""
DRY_RUN="false"
TARGET_BASE_URL="${TARGET_BASE_URL:-}"
TARGET_ROOT_USERNAME="${TARGET_ROOT_USERNAME:-}"
TARGET_ROOT_PASSWORD="${TARGET_ROOT_PASSWORD:-}"
PROBE_UPSTREAM="false"
REQUIRE_CHANNEL_KEYS="false"
CONFIRM_PRODUCTION="false"
CHANNEL_KEYS=()

REMOTE_HOST=""
TEST_HOSTNAME=""
ENTERPRISE_HOSTNAME=""
PRODUCTION_HOSTNAME=""

usage() {
  cat <<EOF
Import a manifest/profile into kkidc test, enterprise, or guarded production environments.

Usage:
  $(basename "$0") test|enterprise|production [options]

Options:
  --config PATH                  host config file (default: .kkidc/.env.lighthouse)
  --profile PATH                 onboarding profile path (required)
  --target-base-url URL          override target base url
  --target-root-username USER    root/admin username for target login
  --target-root-password PASS    root/admin password for target login
  --probe-upstream               run family-level upstream probe before import
  --require-channel-keys         require explicit family channel keys
  --channel-key FAMILY=KEY       provide explicit channel key for a family
  --confirm-production           allow production import after separate backup and deploy gates
  --dry-run                      run manifest dry-run only, no target mutation
  --help                         show this help
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
    test)
      TARGET_ENV="test"
      ;;
    enterprise)
      TARGET_ENV="enterprise"
      ;;
    production|prod)
      TARGET_ENV="production"
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
      --profile)
        PROFILE_PATH="$2"
        shift 2
        ;;
      --target-base-url)
        TARGET_BASE_URL="$2"
        shift 2
        ;;
      --target-root-username)
        TARGET_ROOT_USERNAME="$2"
        shift 2
        ;;
      --target-root-password)
        TARGET_ROOT_PASSWORD="$2"
        shift 2
        ;;
      --probe-upstream)
        PROBE_UPSTREAM="true"
        shift
        ;;
      --require-channel-keys)
        REQUIRE_CHANNEL_KEYS="true"
        shift
        ;;
      --channel-key)
        CHANNEL_KEYS+=("$2")
        shift 2
        ;;
      --confirm-production)
        CONFIRM_PRODUCTION="true"
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
  TEST_HOSTNAME="${TEST_HOSTNAME:-}"
  ENTERPRISE_HOSTNAME="${ENTERPRISE_HOSTNAME:-}"
  PRODUCTION_HOSTNAME="${PRODUCTION_HOSTNAME:-api.aisever.cn}"

  [ -n "$REMOTE_HOST" ] || die "missing IP_1 in $CONFIG_FILE"
}

resolve_target_base_url() {
  if [ -n "$TARGET_BASE_URL" ]; then
    return
  fi

  case "$TARGET_ENV" in
    test)
      if [ -n "$TEST_HOSTNAME" ]; then
        TARGET_BASE_URL="http://${TEST_HOSTNAME}:3001"
      else
        TARGET_BASE_URL="http://${REMOTE_HOST}:3001"
      fi
      ;;
    enterprise)
      if [ -n "$ENTERPRISE_HOSTNAME" ]; then
        TARGET_BASE_URL="https://${ENTERPRISE_HOSTNAME}"
      else
        TARGET_BASE_URL="http://${REMOTE_HOST}:3002"
      fi
      ;;
    production)
      [ "$CONFIRM_PRODUCTION" = "true" ] || die "production import requires --confirm-production"
      [ -n "$PRODUCTION_HOSTNAME" ] || die "missing PRODUCTION_HOSTNAME for production import"
      TARGET_BASE_URL="https://${PRODUCTION_HOSTNAME}"
      ;;
    *)
      die "unsupported target environment: $TARGET_ENV"
      ;;
  esac
}

check_dependencies() {
  command -v python3 >/dev/null 2>&1 || die "python3 is required"
  [ -n "$PROFILE_PATH" ] || die "--profile is required"
  [ -f "$PROFILE_PATH" ] || die "profile not found: $PROFILE_PATH"

  if [ "$DRY_RUN" != "true" ]; then
    [ -n "$TARGET_ROOT_USERNAME" ] || die "--target-root-username is required for non-dry-run"
    [ -n "$TARGET_ROOT_PASSWORD" ] || die "--target-root-password is required for non-dry-run"
  fi
}

main() {
  parse_args "$@"
  load_host_config
  resolve_target_base_url
  check_dependencies

  if [ "$DRY_RUN" = "true" ]; then
    exec python3 "$PROJECT_ROOT/ops/channel-onboarding/dry_run.py" \
      --profile "$PROFILE_PATH" \
      --target-environment "$TARGET_ENV"
  fi

  IMPORT_CMD=(
    python3 "$PROJECT_ROOT/ops/channel-onboarding/import_profile.py"
    --profile "$PROFILE_PATH" \
    --target-base-url "$TARGET_BASE_URL" \
    --target-root-username "$TARGET_ROOT_USERNAME" \
    --target-root-password "$TARGET_ROOT_PASSWORD" \
    --target-environment "$TARGET_ENV"
  )

  if [ "$CONFIRM_PRODUCTION" = "true" ]; then
    IMPORT_CMD+=(--confirm-production)
  fi

  if [ "$PROBE_UPSTREAM" = "true" ]; then
    IMPORT_CMD+=(--probe-upstream)
  fi
  if [ "$REQUIRE_CHANNEL_KEYS" = "true" ]; then
    IMPORT_CMD+=(--require-channel-keys)
  fi
  for item in "${CHANNEL_KEYS[@]-}"; do
    [ -n "$item" ] || continue
    IMPORT_CMD+=(--channel-key "$item")
  done

  exec "${IMPORT_CMD[@]}"
}

main "$@"

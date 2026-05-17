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
PROFILE_PATH="${PROFILE_PATH:-$PROJECT_ROOT/ops/channel-onboarding/profiles/yunwu.yaml}"
TARGET_BASE_URL="${TARGET_BASE_URL:-}"
TARGET_ROOT_USERNAME="${TARGET_ROOT_USERNAME:-}"
TARGET_ROOT_PASSWORD="${TARGET_ROOT_PASSWORD:-}"
CHANNEL_KEYS=()

usage() {
  cat <<EOF
Run yunwu onboarding against the kkidc test environment.

Usage:
  $(basename "$0") [options]

Options:
  --config PATH                  host config file (default: .kkidc/.env.lighthouse)
  --profile PATH                 profile path (default: ops/channel-onboarding/profiles/yunwu.yaml)
  --target-base-url URL          override target base url
  --target-root-username USER    root/admin username for target login
  --target-root-password PASS    root/admin password for target login
  --channel-key FAMILY=KEY       explicit family key, required for kling/vidu/doubao-video
  --help                         show this help
EOF
}

die() {
  printf '[ERROR] %s\n' "$1" >&2
  exit 1
}

parse_args() {
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
      --channel-key)
        CHANNEL_KEYS+=("$2")
        shift 2
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

  local remote_host="${IP_1:-${SERVER_IP:-}}"
  local test_hostname="${TEST_HOSTNAME:-}"

  if [ -z "$TARGET_BASE_URL" ]; then
    if [ -n "$test_hostname" ]; then
      TARGET_BASE_URL="http://${test_hostname}:3001"
    else
      [ -n "$remote_host" ] || die "missing IP_1 in $CONFIG_FILE"
      TARGET_BASE_URL="http://${remote_host}:3001"
    fi
  fi
}

check_dependencies() {
  command -v python3 >/dev/null 2>&1 || die "python3 is required"
  command -v curl >/dev/null 2>&1 || die "curl is required"
  [ -f "$PROFILE_PATH" ] || die "profile not found: $PROFILE_PATH"
  [ -n "$TARGET_ROOT_USERNAME" ] || die "--target-root-username is required"
  [ -n "$TARGET_ROOT_PASSWORD" ] || die "--target-root-password is required"
  [ "${#CHANNEL_KEYS[@]}" -gt 0 ] || die "at least one --channel-key is required"
}

test_status_url_from_profile() {
  python3 - "$PROFILE_PATH" <<'PY'
import sys
import yaml

with open(sys.argv[1], "r", encoding="utf-8") as f:
    profile = yaml.safe_load(f)
print((profile.get("verification") or {}).get("test_status_url") or "")
PY
}

main() {
  parse_args "$@"
  load_host_config
  check_dependencies

  IMPORT_CMD=(
    python3 "$PROJECT_ROOT/ops/channel-onboarding/import_profile.py"
    --profile "$PROFILE_PATH"
    --target-base-url "$TARGET_BASE_URL"
    --target-root-username "$TARGET_ROOT_USERNAME"
    --target-root-password "$TARGET_ROOT_PASSWORD"
    --target-environment test
    --probe-upstream
    --require-channel-keys
  )
  for item in "${CHANNEL_KEYS[@]}"; do
    IMPORT_CMD+=(--channel-key "$item")
  done

  IMPORT_OUTPUT="$("${IMPORT_CMD[@]}")"
  STATUS_URL="$(test_status_url_from_profile)"
  [ -n "$STATUS_URL" ] || die "profile verification.test_status_url is required"
  STATUS_BODY="$(curl -fsS "$STATUS_URL")"

  python3 - "$IMPORT_OUTPUT" "$STATUS_URL" "$STATUS_BODY" <<'PY'
import json
import sys

import_output = json.loads(sys.argv[1])
status_url = sys.argv[2]
status_body = json.loads(sys.argv[3])

result = dict(import_output)
result["test_status_check"] = {
    "url": status_url,
    "success": bool(status_body.get("success")),
    "raw": status_body,
}
print(json.dumps(result, ensure_ascii=False, indent=2))
PY
}

main "$@"

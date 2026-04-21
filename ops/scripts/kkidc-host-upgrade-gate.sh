#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"
REPO_DIR="${REPO_DIR:-$PROJECT_ROOT}"

MODE=""
KEEP_TEST_RUNNING="false"
TEST_STATUS_URL="${TEST_STATUS_URL:-http://114.66.47.192:3001/api/status}"
DEPLOY_GIT_REMOTE_NAME="${DEPLOY_GIT_REMOTE:-origin}"
DEPLOY_GIT_REF_NAME="${DEPLOY_GIT_REF:-}"

usage() {
  cat <<EOF
Run the upstream stable-upgrade gate for the new-api kkidc workflow.

Usage:
  $(basename "$0") local|test [options]

Options:
  --keep-test-running   leave the kkidc test environment running after verification
  --test-status-url URL override the test environment status URL (default: ${TEST_STATUS_URL})
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

parse_args() {
  if [ $# -lt 1 ]; then
    usage
    exit 1
  fi

  case "$1" in
    local|test)
      MODE="$1"
      shift
      ;;
    --help|-h)
      usage
      exit 0
      ;;
    *)
      die "unsupported mode: $1"
      ;;
  esac

  while [ $# -gt 0 ]; do
    case "$1" in
      --keep-test-running)
        KEEP_TEST_RUNNING="true"
        shift
        ;;
      --test-status-url)
        TEST_STATUS_URL="$2"
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

check_repo() {
  [ -d "$REPO_DIR/.git" ] || git -C "$REPO_DIR" rev-parse --git-dir >/dev/null 2>&1 || die "REPO_DIR is not a git repository: $REPO_DIR"
  [ -f "$PROJECT_ROOT/ops/scripts/kkidc-host-deploy.sh" ] || die "missing official deploy script: ops/scripts/kkidc-host-deploy.sh"
}

resolve_git_context() {
  git -C "$REPO_DIR" rev-parse --verify HEAD >/dev/null 2>&1 || die "failed to resolve HEAD in $REPO_DIR"
  BRANCH_NAME="$(git -C "$REPO_DIR" branch --show-current || true)"
  SHA="$(git -C "$REPO_DIR" rev-parse --short HEAD)"
  TRACKED_STATUS="$(git -C "$REPO_DIR" status --short --untracked-files=no)"
}

resolve_deploy_git_ref() {
  local upstream_ref=""

  if [ -n "$DEPLOY_GIT_REF_NAME" ]; then
    return
  fi

  if [ -n "$BRANCH_NAME" ]; then
    upstream_ref="$(git -C "$REPO_DIR" for-each-ref --format='%(upstream:short)' "refs/heads/${BRANCH_NAME}" | head -n 1)"
  fi

  if [ -n "$upstream_ref" ]; then
    DEPLOY_GIT_REF_NAME="$upstream_ref"
    return
  fi

  [ -n "$BRANCH_NAME" ] || die "test gate requires a branch with a pushed upstream ref"
  DEPLOY_GIT_REF_NAME="${DEPLOY_GIT_REMOTE_NAME}/${BRANCH_NAME}"
}

verify_head_pushed() {
  resolve_deploy_git_ref
  git -C "$REPO_DIR" fetch --quiet "$DEPLOY_GIT_REMOTE_NAME" || die "failed to fetch remote '${DEPLOY_GIT_REMOTE_NAME}'"
  git -C "$REPO_DIR" rev-parse --verify "$DEPLOY_GIT_REF_NAME" >/dev/null 2>&1 || die "deploy requires HEAD to be pushed; remote ref '${DEPLOY_GIT_REF_NAME}' does not exist"

  if ! git -C "$REPO_DIR" merge-base --is-ancestor HEAD "$DEPLOY_GIT_REF_NAME"; then
    die "deploy requires HEAD to be pushed; local HEAD is not contained in '${DEPLOY_GIT_REF_NAME}'"
  fi
}

require_clean_tracked_state_for_test_mode() {
  if [ -n "$TRACKED_STATUS" ]; then
    die "test gate requires a clean tracked worktree; commit or stash tracked changes first"
  fi
}

run_local_suite() {
  log INFO "running backend regression suite"
  (
    cd "$REPO_DIR"
    go test ./controller ./model -count=1
  )

  log INFO "running kkidc deploy-script smoke tests"
  bash "$PROJECT_ROOT/ops/tests/kkidc-host-deploy-pushed-head-test.sh"
  bash "$PROJECT_ROOT/ops/tests/kkidc-host-deploy-build-strategy-test.sh"
  bash "$PROJECT_ROOT/ops/tests/kkidc-host-deploy-local-build-platform-test.sh"

  log INFO "running frontend production build"
  (
    cd "$REPO_DIR/web"
    bun run build
  )
}

verify_test_status_version() {
  local status_json
  local version

  status_json="$(curl -fsS "$TEST_STATUS_URL")"
  version="$(printf '%s' "$status_json" | python3 -c 'import json, sys; data=json.load(sys.stdin); print(data.get("data", {}).get("version", ""))')"

  if [ -z "$version" ]; then
    die "test gate could not read version from ${TEST_STATUS_URL}"
  fi

  if [[ "$version" != *"+${SHA}+kkidc-test" ]]; then
    die "test gate expected test version to contain +${SHA}+kkidc-test, got: ${version}"
  fi

  log INFO "verified test environment version: ${version}"
}

deploy_and_verify_test_env() {
  log INFO "deploying kkidc test environment via official script"
  (
    cd "$PROJECT_ROOT"
    REPO_DIR="$REPO_DIR" bash "$PROJECT_ROOT/ops/scripts/kkidc-host-deploy.sh" test
  )

  log INFO "checking kkidc test health at ${TEST_STATUS_URL}"
  verify_test_status_version

  if [ "$KEEP_TEST_RUNNING" = "true" ]; then
    log INFO "keeping kkidc test environment running as requested"
    return
  fi

  log INFO "stopping kkidc test environment via official script"
  (
    cd "$PROJECT_ROOT"
    REPO_DIR="$REPO_DIR" bash "$PROJECT_ROOT/ops/scripts/kkidc-host-deploy.sh" test-stop
  )
}

main() {
  parse_args "$@"
  check_repo
  resolve_git_context

  log INFO "upgrade gate mode=${MODE} branch=${BRANCH_NAME:-detached} sha=${SHA}"
  run_local_suite

  if [ "$MODE" = "local" ]; then
    log INFO "local upgrade gate passed"
    return
  fi

  require_clean_tracked_state_for_test_mode
  verify_head_pushed
  deploy_and_verify_test_env
  log INFO "test upgrade gate passed"
}

main "$@"

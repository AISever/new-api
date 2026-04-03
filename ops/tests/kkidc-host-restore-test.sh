#!/usr/bin/env bash
set -euo pipefail

SCRIPT_UNDER_TEST="${1:-$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)/scripts/kkidc-host-restore.sh}"

ROOT_DIR="$(mktemp -d)"
trap 'rm -rf "$ROOT_DIR"' EXIT

CONFIG_FILE="$ROOT_DIR/.env.lighthouse"

cat > "$CONFIG_FILE" <<'EOF'
IP_1=114.66.47.192
SSH_user=root
SSH_password=test-password
HOSTNAME=api.aisever.cn
TEST_HOSTNAME=114.66.47.192
ENTERPRISE_HOSTNAME=corp-api.aisever.cn
EOF

SOURCE_BACKUP_DIR="/opt/new-api-backups/20260404013224"

set +e
TEST_OUTPUT="$(bash "$SCRIPT_UNDER_TEST" test --config "$CONFIG_FILE" --source-backup-dir "$SOURCE_BACKUP_DIR" --dry-run 2>&1)"
TEST_EXIT="$?"
set -e

if [ "$TEST_EXIT" -ne 0 ]; then
  echo "FAIL: test dry-run should succeed" >&2
  echo "$TEST_OUTPUT" >&2
  exit 1
fi

if ! printf '%s\n' "$TEST_OUTPUT" | grep -q '^target_env=test$'; then
  echo "FAIL: restore output missing test target env" >&2
  exit 1
fi

if ! printf '%s\n' "$TEST_OUTPUT" | grep -q '^source_backup_dir=/opt/new-api-backups/20260404013224$'; then
  echo "FAIL: restore should echo source backup dir" >&2
  exit 1
fi

if ! printf '%s\n' "$TEST_OUTPUT" | grep -q '^pg_db=new-api-test$'; then
  echo "FAIL: restore should target new-api-test database" >&2
  exit 1
fi

if ! printf '%s\n' "$TEST_OUTPUT" | grep -q '^data_dir=/opt/new-api-test/data$'; then
  echo "FAIL: restore should target test data dir" >&2
  exit 1
fi

if ! printf '%s\n' "$TEST_OUTPUT" | grep -q '^restore_logs=true$'; then
  echo "FAIL: logs should restore by default" >&2
  exit 1
fi

set +e
PROD_OUTPUT="$(bash "$SCRIPT_UNDER_TEST" production --config "$CONFIG_FILE" --source-backup-dir "$SOURCE_BACKUP_DIR" --dry-run 2>&1)"
PROD_EXIT="$?"
set -e

if [ "$PROD_EXIT" -eq 0 ]; then
  echo "FAIL: production target must be rejected" >&2
  exit 1
fi

if ! printf '%s\n' "$PROD_OUTPUT" | grep -F -q '[ERROR] restore currently only supports test target'; then
  echo "FAIL: expected production target validation message" >&2
  exit 1
fi

set +e
MISSING_OUTPUT="$(bash "$SCRIPT_UNDER_TEST" test --config "$CONFIG_FILE" --dry-run 2>&1)"
MISSING_EXIT="$?"
set -e

if [ "$MISSING_EXIT" -eq 0 ]; then
  echo "FAIL: missing source backup dir must be rejected" >&2
  exit 1
fi

if ! printf '%s\n' "$MISSING_OUTPUT" | grep -F -q '[ERROR] --source-backup-dir is required'; then
  echo "FAIL: expected missing source backup dir validation message" >&2
  exit 1
fi

echo "PASS: kkidc host restore dry-run validates test-only restore settings"

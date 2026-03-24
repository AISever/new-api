#!/usr/bin/env bash
set -euo pipefail

SCRIPT_UNDER_TEST="${1:-$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)/scripts/kkidc-host-backup.sh}"

ROOT_DIR="$(mktemp -d)"
trap 'rm -rf "$ROOT_DIR"' EXIT

CONFIG_FILE="$ROOT_DIR/.env.lighthouse"

cat > "$CONFIG_FILE" <<'EOF'
IP_1=114.66.47.192
SSH_user=root
SSH_password=test-password
HOSTNAME=api.aisever.cn
TEST_HOSTNAME=114.66.47.192
EOF

set +e
PROD_OUTPUT="$(bash "$SCRIPT_UNDER_TEST" production --config "$CONFIG_FILE" --dry-run 2>&1)"
PROD_EXIT="$?"
set -e

if [ "$PROD_EXIT" -ne 0 ]; then
  echo "FAIL: production dry-run should succeed" >&2
  echo "$PROD_OUTPUT" >&2
  exit 1
fi

if ! printf '%s\n' "$PROD_OUTPUT" | grep -q '^target_env=production$'; then
  echo "FAIL: production output missing target env" >&2
  exit 1
fi

if ! printf '%s\n' "$PROD_OUTPUT" | grep -q '^pg_db=new-api$'; then
  echo "FAIL: production backup should target new-api database" >&2
  exit 1
fi

if ! printf '%s\n' "$PROD_OUTPUT" | grep -q '^postgres_container=new-api-postgres$'; then
  echo "FAIL: production postgres container mismatch" >&2
  exit 1
fi

if ! printf '%s\n' "$PROD_OUTPUT" | grep -q '^backup_root=/opt/new-api-backups$'; then
  echo "FAIL: production backup root mismatch" >&2
  exit 1
fi

if ! printf '%s\n' "$PROD_OUTPUT" | grep -Eq '^backup_dir=/opt/new-api-backups/[0-9]{14}$'; then
  echo "FAIL: production backup dir should include timestamp" >&2
  exit 1
fi

if ! printf '%s\n' "$PROD_OUTPUT" | grep -q '^include_data=true$'; then
  echo "FAIL: production should include data by default" >&2
  exit 1
fi

if ! printf '%s\n' "$PROD_OUTPUT" | grep -q '^include_logs=true$'; then
  echo "FAIL: production should include logs by default" >&2
  exit 1
fi

set +e
TEST_OUTPUT="$(bash "$SCRIPT_UNDER_TEST" test --config "$CONFIG_FILE" --dry-run --no-logs 2>&1)"
TEST_EXIT="$?"
set -e

if [ "$TEST_EXIT" -ne 0 ]; then
  echo "FAIL: test dry-run should succeed" >&2
  echo "$TEST_OUTPUT" >&2
  exit 1
fi

if ! printf '%s\n' "$TEST_OUTPUT" | grep -q '^target_env=test$'; then
  echo "FAIL: test output missing target env" >&2
  exit 1
fi

if ! printf '%s\n' "$TEST_OUTPUT" | grep -q '^pg_db=new-api-test$'; then
  echo "FAIL: test backup should target new-api-test database" >&2
  exit 1
fi

if ! printf '%s\n' "$TEST_OUTPUT" | grep -q '^backup_root=/opt/new-api-test-backups$'; then
  echo "FAIL: test backup root mismatch" >&2
  exit 1
fi

if ! printf '%s\n' "$TEST_OUTPUT" | grep -q '^include_logs=false$'; then
  echo "FAIL: --no-logs should disable log backup" >&2
  exit 1
fi

echo "PASS: kkidc host backup dry-run renders production/test backup settings"

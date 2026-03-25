#!/usr/bin/env bash
set -euo pipefail

SCRIPT_UNDER_TEST="${1:-$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)/scripts/kkidc-host-deploy.sh}"

ROOT_DIR="$(mktemp -d)"
trap 'rm -rf "$ROOT_DIR"' EXIT

REPO_DIR="$ROOT_DIR/repo"
CONFIG_FILE="$ROOT_DIR/.env.lighthouse"
APP_ENV_FILE="$ROOT_DIR/.env.local"

git init "$REPO_DIR" >/dev/null
git -C "$REPO_DIR" config user.name test
git -C "$REPO_DIR" config user.email test@example.com
git -C "$REPO_DIR" branch -M codex/release-v0.11.2-patch.2-p1

mkdir -p "$REPO_DIR"
cat > "$REPO_DIR/payload.txt" <<'EOF'
clean
EOF
cat > "$REPO_DIR/Dockerfile" <<'EOF'
FROM scratch
RUN go mod download
RUN DISABLE_ESLINT_PLUGIN='true' VITE_REACT_APP_VERSION=$(cat VERSION) npm run build
RUN apk add --no-cache ca-certificates tzdata wget \
    && update-ca-certificates
EOF
mkdir -p "$REPO_DIR/ops/scripts"

git -C "$REPO_DIR" add payload.txt Dockerfile
git -C "$REPO_DIR" commit -m "seed" >/dev/null

echo "dirty" > "$REPO_DIR/payload.txt"
echo "should-not-sync" > "$REPO_DIR/untracked.txt"

cat > "$CONFIG_FILE" <<'EOF'
IP_1=114.66.47.192
SSH_user=root
SSH_password=test-password
HOSTNAME=api.aisever.cn
HOSTNAME_ALIASES=newapi.aisever.cn
EOF

cat > "$APP_ENV_FILE" <<'EOF'
DB_PASSWORD=NewApi2024Secure
SESSION_SECRET=session-secret
CRYPTO_SECRET=crypto-secret
EOF

set +e
PROD_OUTPUT="$(REPO_DIR="$REPO_DIR" APP_ENV_FILE="$APP_ENV_FILE" bash "$SCRIPT_UNDER_TEST" production --config "$CONFIG_FILE" --dry-run --keep-stage-dir 2>&1)"
PROD_EXIT="$?"
set -e

if [ "$PROD_EXIT" -ne 0 ]; then
  echo "FAIL: production dry-run should succeed" >&2
  echo "$PROD_OUTPUT" >&2
  exit 1
fi

PROD_STAGE_DIR="$(printf '%s\n' "$PROD_OUTPUT" | awk -F= '/^stage_dir=/{print $2}')"
PROD_VERSION="$(printf '%s\n' "$PROD_OUTPUT" | awk -F= '/^version_value=/{print $2}')"

if [ ! -d "$PROD_STAGE_DIR" ]; then
  echo "FAIL: expected production stage dir to exist" >&2
  exit 1
fi

if [ "$(cat "$PROD_STAGE_DIR/payload.txt")" != "clean" ]; then
  echo "FAIL: staged payload should come from committed HEAD, not dirty working tree" >&2
  exit 1
fi

if ! grep -q "NODE_OPTIONS='--max-old-space-size=2048'" "$PROD_STAGE_DIR/Dockerfile.deploy"; then
  echo "FAIL: staged Dockerfile.deploy should inject frontend NODE_OPTIONS" >&2
  exit 1
fi

if ! grep -q "mirrors.aliyun.com/alpine" "$PROD_STAGE_DIR/Dockerfile.deploy"; then
  echo "FAIL: staged Dockerfile.deploy should rewrite Alpine package mirror" >&2
  exit 1
fi

if [ -e "$PROD_STAGE_DIR/untracked.txt" ]; then
  echo "FAIL: untracked files must not be included in staged deploy source" >&2
  exit 1
fi

if ! printf '%s\n' "$PROD_OUTPUT" | grep -q '^target_env=production$'; then
  echo "FAIL: production output missing target env" >&2
  exit 1
fi

if ! printf '%s\n' "$PROD_OUTPUT" | grep -q '^app_container=new-api-local$'; then
  echo "FAIL: production container mismatch" >&2
  exit 1
fi

if ! printf '%s\n' "$PROD_OUTPUT" | grep -q '^app_port=3000$'; then
  echo "FAIL: production port mismatch" >&2
  exit 1
fi

if ! printf '%s\n' "$PROD_OUTPUT" | grep -q '^server_address=https://api.aisever.cn$'; then
  echo "FAIL: production server address mismatch" >&2
  exit 1
fi

if ! printf '%s\n' "$PROD_OUTPUT" | grep -q '^public_hostnames=api.aisever.cn,newapi.aisever.cn$'; then
  echo "FAIL: production public hostnames mismatch" >&2
  exit 1
fi

if ! printf '%s\n' "$PROD_OUTPUT" | grep -q '^caddy_site_addresses=api.aisever.cn, newapi.aisever.cn$'; then
  echo "FAIL: caddy site addresses should be comma+space separated" >&2
  exit 1
fi

if ! printf '%s\n' "$PROD_OUTPUT" | grep -q '^sync_prod_data_from_legacy=false$'; then
  echo "FAIL: production sync flag should default to false" >&2
  exit 1
fi

if ! printf '%s\n' "$PROD_OUTPUT" | grep -q '^build_strategy=local$'; then
  echo "FAIL: production auto build strategy should resolve to local" >&2
  exit 1
fi

if ! printf '%s\n' "$PROD_VERSION" | grep -q '^codex/release-v0.11.2-patch.2-p1+'; then
  echo "FAIL: production version should include branch prefix" >&2
  exit 1
fi

set +e
TEST_OUTPUT="$(REPO_DIR="$REPO_DIR" APP_ENV_FILE="$APP_ENV_FILE" bash "$SCRIPT_UNDER_TEST" test --config "$CONFIG_FILE" --dry-run 2>&1)"
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

if ! printf '%s\n' "$TEST_OUTPUT" | grep -q '^app_container=new-api-test$'; then
  echo "FAIL: test container mismatch" >&2
  exit 1
fi

if ! printf '%s\n' "$TEST_OUTPUT" | grep -q '^app_port=3001$'; then
  echo "FAIL: test port mismatch" >&2
  exit 1
fi

if ! printf '%s\n' "$TEST_OUTPUT" | grep -q '^pg_db=new-api-test$'; then
  echo "FAIL: test database mismatch" >&2
  exit 1
fi

if ! printf '%s\n' "$TEST_OUTPUT" | grep -q '^redis_conn_string=redis://new-api-redis/1$'; then
  echo "FAIL: test redis db mismatch" >&2
  exit 1
fi

if ! printf '%s\n' "$TEST_OUTPUT" | grep -q '^server_address=http://114.66.47.192:3001$'; then
  echo "FAIL: test server address mismatch" >&2
  exit 1
fi

set +e
SYNC_TEST_OUTPUT="$(REPO_DIR="$REPO_DIR" APP_ENV_FILE="$APP_ENV_FILE" bash "$SCRIPT_UNDER_TEST" test --config "$CONFIG_FILE" --dry-run --sync-prod-data-from-legacy 2>&1)"
SYNC_TEST_EXIT="$?"
set -e

if [ "$SYNC_TEST_EXIT" -eq 0 ]; then
  echo "FAIL: sync-prod-data flag must reject test environment" >&2
  exit 1
fi

if ! printf '%s\n' "$SYNC_TEST_OUTPUT" | grep -q -- '--sync-prod-data-from-legacy only supports production'; then
  echo "FAIL: expected sync-prod-data flag validation message" >&2
  exit 1
fi

echo "PASS: kkidc host deploy dry-run renders production/test settings and stages clean committed source"

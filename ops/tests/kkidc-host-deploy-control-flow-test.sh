#!/usr/bin/env bash
set -euo pipefail

SCRIPT_UNDER_TEST="${1:-$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)/scripts/kkidc-host-deploy.sh}"

ROOT_DIR="$(mktemp -d)"
trap 'rm -rf "$ROOT_DIR"' EXIT

REPO_DIR="$ROOT_DIR/repo"
CONFIG_FILE="$ROOT_DIR/.env.lighthouse"
APP_ENV_FILE="$ROOT_DIR/.env.local"
FAKEBIN_DIR="$ROOT_DIR/fakebin"

mkdir -p "$REPO_DIR" "$FAKEBIN_DIR"
git init "$REPO_DIR" >/dev/null
git -C "$REPO_DIR" config user.name test
git -C "$REPO_DIR" config user.email test@example.com
git -C "$REPO_DIR" branch -M codex/release-v0.11.2-patch.2-p1

cat > "$REPO_DIR/Dockerfile" <<'EOF'
FROM scratch
RUN go mod download
RUN DISABLE_ESLINT_PLUGIN='true' VITE_REACT_APP_VERSION=$(cat VERSION) bun run build
EOF
echo "seed" > "$REPO_DIR/payload.txt"
git -C "$REPO_DIR" add Dockerfile payload.txt
git -C "$REPO_DIR" commit -m "seed" >/dev/null

cat > "$CONFIG_FILE" <<'EOF'
IP_1=114.66.47.192
SSH_user=root
SSH_password=test-password
HOSTNAME=newapi.aisever.cn
EOF

cat > "$APP_ENV_FILE" <<'EOF'
DB_PASSWORD=NewApi2024Secure
SESSION_SECRET=session-secret
CRYPTO_SECRET=crypto-secret
SERVER_IP=202.140.142.149
SERVER_USER=root
SERVER_PASSWORD=legacy-password
EOF

cat > "$FAKEBIN_DIR/sshpass" <<'EOF'
#!/usr/bin/env bash
shift 2
cat >/dev/null || true
exit 0
EOF
chmod +x "$FAKEBIN_DIR/sshpass"

OUTPUT="$(PATH="$FAKEBIN_DIR:$PATH" REPO_DIR="$REPO_DIR" APP_ENV_FILE="$APP_ENV_FILE" bash "$SCRIPT_UNDER_TEST" test --config "$CONFIG_FILE" --skip-build)"

if ! printf '%s\n' "$OUTPUT" | grep -q '^build_strategy=skip$'; then
  echo "FAIL: expected skip build strategy in final output" >&2
  exit 1
fi

if ! printf '%s\n' "$OUTPUT" | grep -q '^target_env=test$'; then
  echo "FAIL: expected final output after skip-build run" >&2
  exit 1
fi

for field in duration_stage_seconds duration_image_seconds duration_image_build_seconds duration_image_export_seconds duration_image_upload_seconds duration_image_load_seconds duration_data_sync_seconds duration_remote_start_seconds duration_verify_seconds duration_total_seconds; do
  if ! printf '%s\n' "$OUTPUT" | grep -Eq "^${field}=[0-9]+$"; then
    echo "FAIL: expected ${field} in final output" >&2
    exit 1
  fi
done

echo "PASS: kkidc host deploy completes non-build flow without silent early exit"

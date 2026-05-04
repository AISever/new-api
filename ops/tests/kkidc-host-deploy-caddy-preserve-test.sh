#!/usr/bin/env bash
set -euo pipefail

SCRIPT_UNDER_TEST="${1:-$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)/scripts/kkidc-host-deploy.sh}"

ROOT_DIR="$(mktemp -d)"
trap 'rm -rf "$ROOT_DIR"' EXIT

REPO_DIR="$ROOT_DIR/repo"
REMOTE_REPO_DIR="$ROOT_DIR/remote.git"
CONFIG_FILE="$ROOT_DIR/.env.lighthouse"
APP_ENV_FILE="$ROOT_DIR/.env.local"
FAKEBIN_DIR="$ROOT_DIR/fakebin"
SSH_LOG="$ROOT_DIR/ssh.log"
EXISTING_CADDY_FILE="$ROOT_DIR/existing.Caddyfile"
WRITTEN_CADDY_FILE="$ROOT_DIR/written.Caddyfile"

mkdir -p "$REPO_DIR" "$FAKEBIN_DIR"
git init --bare "$REMOTE_REPO_DIR" >/dev/null
git init "$REPO_DIR" >/dev/null
git -C "$REPO_DIR" config user.name test
git -C "$REPO_DIR" config user.email test@example.com
git -C "$REPO_DIR" branch -M codex/release-v0.13.2-p1
git -C "$REPO_DIR" remote add origin "$REMOTE_REPO_DIR"

cat > "$REPO_DIR/Dockerfile" <<'EOF'
FROM scratch
RUN go mod download
RUN DISABLE_ESLINT_PLUGIN='true' VITE_REACT_APP_VERSION=$(cat VERSION) npm run build
EOF
echo "seed" > "$REPO_DIR/payload.txt"
git -C "$REPO_DIR" add Dockerfile payload.txt
git -C "$REPO_DIR" commit -m "seed" >/dev/null
git -C "$REPO_DIR" push -u origin codex/release-v0.13.2-p1 >/dev/null

cat > "$CONFIG_FILE" <<'EOF'
IP_1=114.66.47.192
SSH_user=root
SSH_password=test-password
HOSTNAME=api.aisever.cn
HOSTNAME_ALIASES=newapi.aisever.cn
ENTERPRISE_HOSTNAME=corp-api.aisever.cn
EOF

cat > "$APP_ENV_FILE" <<'EOF'
DB_PASSWORD=NewApi2024Secure
SESSION_SECRET=session-secret
CRYPTO_SECRET=crypto-secret
EOF

cat > "$EXISTING_CADDY_FILE" <<'EOF'
api.aisever.cn, newapi.aisever.cn {
    reverse_proxy 172.17.0.1:9999
}

image.aisever.cn {
    encode gzip
    reverse_proxy 172.17.0.1:13002 {
        header_up X-Real-IP {remote_host}
        header_up Host {host}
    }
}

corp-api.aisever.cn {
    reverse_proxy 172.17.0.1:3999
}
EOF

cat > "$FAKEBIN_DIR/sshpass" <<'EOF'
#!/usr/bin/env bash
if [ "$1" = "-p" ]; then
  shift 2
fi

last_arg=""
for arg in "$@"; do
  last_arg="$arg"
done
printf '%s\n' "$last_arg" >> "$SSH_LOG"

case "$last_arg" in
  *"cat /opt/snowlight/Caddyfile"*)
    cat "$EXISTING_CADDY_FILE"
    exit 0
    ;;
  "bash -s")
    script="$(cat)"
    if printf '%s\n' "$script" | grep -q "cat > /opt/snowlight/Caddyfile <<'CADDY'"; then
      printf '%s\n' "$script" | awk '
        /^cat > \/opt\/snowlight\/Caddyfile <<'\''CADDY'\''$/ { capture = 1; next }
        capture && /^CADDY$/ { capture = 0; exit }
        capture { print }
      ' > "$WRITTEN_CADDY_FILE"
    fi
    exit 0
    ;;
  *)
    cat >/dev/null || true
    exit 0
    ;;
esac
EOF
chmod +x "$FAKEBIN_DIR/sshpass"

OUTPUT="$(PATH="$FAKEBIN_DIR:$PATH" SSH_LOG="$SSH_LOG" EXISTING_CADDY_FILE="$EXISTING_CADDY_FILE" WRITTEN_CADDY_FILE="$WRITTEN_CADDY_FILE" REPO_DIR="$REPO_DIR" APP_ENV_FILE="$APP_ENV_FILE" bash "$SCRIPT_UNDER_TEST" test --config "$CONFIG_FILE" --skip-build)"

if ! printf '%s\n' "$OUTPUT" | grep -q '^caddy_config_status=restarted$'; then
  echo "FAIL: expected caddy restart after writing merged config" >&2
  echo "$OUTPUT" >&2
  exit 1
fi

if [ ! -s "$WRITTEN_CADDY_FILE" ]; then
  echo "FAIL: expected deploy to write a Caddyfile" >&2
  cat "$SSH_LOG" >&2
  exit 1
fi

if ! grep -q '^image\.aisever\.cn {' "$WRITTEN_CADDY_FILE"; then
  echo "FAIL: unmanaged image service route should be preserved" >&2
  cat "$WRITTEN_CADDY_FILE" >&2
  exit 1
fi

if ! grep -q '172\.17\.0\.1:13002' "$WRITTEN_CADDY_FILE"; then
  echo "FAIL: unmanaged image service upstream should be preserved" >&2
  cat "$WRITTEN_CADDY_FILE" >&2
  exit 1
fi

if ! grep -q '^api\.aisever\.cn, newapi\.aisever\.cn {' "$WRITTEN_CADDY_FILE"; then
  echo "FAIL: managed production route should be present" >&2
  cat "$WRITTEN_CADDY_FILE" >&2
  exit 1
fi

if ! grep -q '172\.17\.0\.1:3000' "$WRITTEN_CADDY_FILE"; then
  echo "FAIL: managed production upstream should be refreshed" >&2
  cat "$WRITTEN_CADDY_FILE" >&2
  exit 1
fi

if ! grep -q '^corp-api\.aisever\.cn {' "$WRITTEN_CADDY_FILE"; then
  echo "FAIL: managed enterprise route should be present" >&2
  cat "$WRITTEN_CADDY_FILE" >&2
  exit 1
fi

if ! grep -q '172\.17\.0\.1:3002' "$WRITTEN_CADDY_FILE"; then
  echo "FAIL: managed enterprise upstream should be refreshed" >&2
  cat "$WRITTEN_CADDY_FILE" >&2
  exit 1
fi

if grep -q '172\.17\.0\.1:9999\|172\.17\.0\.1:3999' "$WRITTEN_CADDY_FILE"; then
  echo "FAIL: stale managed route blocks should be replaced" >&2
  cat "$WRITTEN_CADDY_FILE" >&2
  exit 1
fi

echo "PASS: kkidc host deploy preserves unmanaged Caddy routes while refreshing managed routes"

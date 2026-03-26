#!/usr/bin/env bash
set -euo pipefail

SCRIPT_UNDER_TEST="${1:-$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)/scripts/kkidc-host-deploy.sh}"

ROOT_DIR="$(mktemp -d)"
trap 'rm -rf "$ROOT_DIR"' EXIT

REPO_DIR="$ROOT_DIR/repo"
CONFIG_FILE="$ROOT_DIR/.env.lighthouse"
APP_ENV_FILE="$ROOT_DIR/.env.local"
FAKEBIN_DIR="$ROOT_DIR/fakebin"
SSH_LOG="$ROOT_DIR/ssh.log"

mkdir -p "$REPO_DIR" "$FAKEBIN_DIR"
git init "$REPO_DIR" >/dev/null
git -C "$REPO_DIR" config user.name test
git -C "$REPO_DIR" config user.email test@example.com
git -C "$REPO_DIR" branch -M codex/release-v0.11.7-p1

cat > "$REPO_DIR/Dockerfile" <<'EOF'
FROM scratch
RUN go mod download
RUN DISABLE_ESLINT_PLUGIN='true' VITE_REACT_APP_VERSION=$(cat VERSION) npm run build
EOF
echo "seed" > "$REPO_DIR/payload.txt"
git -C "$REPO_DIR" add Dockerfile payload.txt
git -C "$REPO_DIR" commit -m "seed" >/dev/null

cat > "$CONFIG_FILE" <<'EOF'
IP_1=114.66.47.192
SSH_user=root
SSH_password=test-password
HOSTNAME=api.aisever.cn
EOF

cat > "$APP_ENV_FILE" <<'EOF'
DB_PASSWORD=NewApi2024Secure
SESSION_SECRET=session-secret
CRYPTO_SECRET=crypto-secret
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
  *"free -m"* )
    printf '128\n'
    exit 0
    ;;
  *"uptime"* )
    printf '9.50\n'
    exit 0
    ;;
  * )
    cat >/dev/null || true
    exit 0
    ;;
esac
EOF
chmod +x "$FAKEBIN_DIR/sshpass"

set +e
OUTPUT="$(SSH_LOG="$SSH_LOG" PATH="$FAKEBIN_DIR:$PATH" REPO_DIR="$REPO_DIR" APP_ENV_FILE="$APP_ENV_FILE" BUILD_STRATEGY=remote bash "$SCRIPT_UNDER_TEST" test --config "$CONFIG_FILE" 2>&1)"
EXIT_CODE="$?"
set -e

if [ "$EXIT_CODE" -eq 0 ]; then
  echo "FAIL: remote build should be rejected when remote host resources are insufficient" >&2
  exit 1
fi

if ! printf '%s\n' "$OUTPUT" | grep -q "remote build guard rejected current host state"; then
  echo "FAIL: expected remote build guard rejection message" >&2
  echo "$OUTPUT" >&2
  exit 1
fi

if grep -q "docker build -t 'new-api:kkidc-test-" "$SSH_LOG"; then
  echo "FAIL: remote docker build should not start after guard rejection" >&2
  cat "$SSH_LOG" >&2
  exit 1
fi

echo "PASS: kkidc host deploy rejects remote builds when host resources are below guard threshold"

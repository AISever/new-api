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
DOCKER_LOG="$ROOT_DIR/docker.log"
SSH_LOG="$ROOT_DIR/ssh.log"

mkdir -p "$REPO_DIR" "$FAKEBIN_DIR"
git init --bare "$REMOTE_REPO_DIR" >/dev/null
git init "$REPO_DIR" >/dev/null
git -C "$REPO_DIR" config user.name test
git -C "$REPO_DIR" config user.email test@example.com
git -C "$REPO_DIR" branch -M codex/release-v0.12.12-p1
git -C "$REPO_DIR" remote add origin "$REMOTE_REPO_DIR"

cat > "$REPO_DIR/Dockerfile" <<'EOF'
FROM scratch
RUN go mod download
RUN DISABLE_ESLINT_PLUGIN='true' VITE_REACT_APP_VERSION=$(cat VERSION) npm run build
EOF
echo "seed" > "$REPO_DIR/payload.txt"
git -C "$REPO_DIR" add Dockerfile payload.txt
git -C "$REPO_DIR" commit -m "seed" >/dev/null
git -C "$REPO_DIR" push -u origin codex/release-v0.12.12-p1 >/dev/null

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

cat > "$FAKEBIN_DIR/docker" <<'EOF'
#!/usr/bin/env bash
printf '%s\n' "$*" >> "$DOCKER_LOG"
case "$1" in
  info)
    exit 0
    ;;
  build|save)
    exit 0
    ;;
  *)
    exit 0
    ;;
esac
EOF
chmod +x "$FAKEBIN_DIR/docker"

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
  *"docker image inspect 'new-api:kkidc-test-"*)
    exit 0
    ;;
  *)
    cat >/dev/null || true
    exit 0
    ;;
esac
EOF
chmod +x "$FAKEBIN_DIR/sshpass"

OUTPUT_REUSE="$(DOCKER_LOG="$DOCKER_LOG" SSH_LOG="$SSH_LOG" PATH="$FAKEBIN_DIR:$PATH" REPO_DIR="$REPO_DIR" APP_ENV_FILE="$APP_ENV_FILE" bash "$SCRIPT_UNDER_TEST" test --config "$CONFIG_FILE")"

if grep -q '^build ' "$DOCKER_LOG"; then
  echo "FAIL: deploy should reuse existing remote image instead of rebuilding" >&2
  cat "$DOCKER_LOG" >&2
  exit 1
fi

if grep -q '^save ' "$DOCKER_LOG"; then
  echo "FAIL: deploy should not export image archive when remote image is reused" >&2
  cat "$DOCKER_LOG" >&2
  exit 1
fi

if ! printf '%s\n' "$OUTPUT_REUSE" | grep -q '^image_source=reused-remote$'; then
  echo "FAIL: expected reused-remote image source in final output" >&2
  echo "$OUTPUT_REUSE" >&2
  exit 1
fi

if ! grep -q "docker image inspect 'new-api:kkidc-test-" "$SSH_LOG"; then
  echo "FAIL: deploy should check whether the remote image already exists" >&2
  cat "$SSH_LOG" >&2
  exit 1
fi

: > "$DOCKER_LOG"

OUTPUT_REBUILD="$(DOCKER_LOG="$DOCKER_LOG" SSH_LOG="$SSH_LOG" PATH="$FAKEBIN_DIR:$PATH" REPO_DIR="$REPO_DIR" APP_ENV_FILE="$APP_ENV_FILE" bash "$SCRIPT_UNDER_TEST" test --config "$CONFIG_FILE" --rebuild)"

if ! grep -q '^build --platform linux/amd64 -t new-api:kkidc-test-' "$DOCKER_LOG"; then
  echo "FAIL: --rebuild should force a fresh local image build" >&2
  cat "$DOCKER_LOG" >&2
  exit 1
fi

if ! printf '%s\n' "$OUTPUT_REBUILD" | grep -q '^image_source=built-local$'; then
  echo "FAIL: expected built-local image source after --rebuild" >&2
  echo "$OUTPUT_REBUILD" >&2
  exit 1
fi

echo "PASS: kkidc host deploy reuses same-SHA remote images unless --rebuild is requested"

#!/usr/bin/env bash
set -euo pipefail

SCRIPT_UNDER_TEST="${1:-$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)/scripts/kkidc-host-deploy.sh}"

ROOT_DIR="$(mktemp -d)"
trap 'rm -rf "$ROOT_DIR"' EXIT

REPO_DIR="$ROOT_DIR/repo"
CONFIG_FILE="$ROOT_DIR/.env.lighthouse"
APP_ENV_FILE="$ROOT_DIR/.env.local"
FAKEBIN_DIR="$ROOT_DIR/fakebin"
DOCKER_LOG="$ROOT_DIR/docker.log"
SSH_LOG="$ROOT_DIR/ssh.log"

mkdir -p "$REPO_DIR" "$FAKEBIN_DIR"
git init "$REPO_DIR" >/dev/null
git -C "$REPO_DIR" config user.name test
git -C "$REPO_DIR" config user.email test@example.com
git -C "$REPO_DIR" branch -M codex/release-v0.11.7-p1

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
  build)
    exit 0
    ;;
  save)
    printf 'fake-image-stream'
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
cat >/dev/null || true
exit 0
EOF
chmod +x "$FAKEBIN_DIR/sshpass"

OUTPUT="$(DOCKER_LOG="$DOCKER_LOG" SSH_LOG="$SSH_LOG" PATH="$FAKEBIN_DIR:$PATH" REPO_DIR="$REPO_DIR" APP_ENV_FILE="$APP_ENV_FILE" bash "$SCRIPT_UNDER_TEST" test --config "$CONFIG_FILE")"

if ! grep -q -- "--platform linux/amd64 -t new-api:kkidc-test-" "$DOCKER_LOG"; then
  echo "FAIL: local docker build should target linux/amd64" >&2
  cat "$DOCKER_LOG" >&2
  exit 1
fi

if ! grep -q "^save -o .* new-api:kkidc-test-" "$DOCKER_LOG"; then
  echo "FAIL: expected local image export step" >&2
  cat "$DOCKER_LOG" >&2
  exit 1
fi

if ! grep -q "docker load -i '/opt/new-api-build-test/new-api-kkidc-test-" "$SSH_LOG"; then
  echo "FAIL: expected remote docker load step using transferred archive" >&2
  cat "$SSH_LOG" >&2
  exit 1
fi

if ! printf '%s\n' "$OUTPUT" | grep -q '^build_strategy=local$'; then
  echo "FAIL: expected local build strategy in final output" >&2
  exit 1
fi

for field in duration_image_build_seconds duration_image_export_seconds duration_image_upload_seconds duration_image_load_seconds; do
  if ! printf '%s\n' "$OUTPUT" | grep -Eq "^${field}=[0-9]+$"; then
    echo "FAIL: expected ${field} in final output" >&2
    exit 1
  fi
done

echo "PASS: kkidc host deploy local build targets linux/amd64 before remote load"

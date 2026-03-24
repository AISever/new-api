#!/usr/bin/env bash
set -euo pipefail

SCRIPT_UNDER_TEST="${1:-$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)/scripts/kkidc-host-deploy.sh}"

ROOT_DIR="$(mktemp -d)"
trap 'rm -rf "$ROOT_DIR"' EXIT

REPO_DIR="$ROOT_DIR/repo"
CONFIG_FILE="$ROOT_DIR/.env.lighthouse"
APP_ENV_FILE="$ROOT_DIR/.env.local"

mkdir -p "$REPO_DIR"
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
SERVER_IP=202.140.142.149
SERVER_USER=root
SERVER_PASSWORD=legacy-password
EOF

OUTPUT="$(REPO_DIR="$REPO_DIR" APP_ENV_FILE="$APP_ENV_FILE" bash "$SCRIPT_UNDER_TEST" test --config "$CONFIG_FILE" --dry-run)"

if ! printf '%s\n' "$OUTPUT" | grep -q '^build_strategy=remote$'; then
  echo "FAIL: auto build strategy should prefer target-host remote build over legacy remote build" >&2
  exit 1
fi

echo "PASS: kkidc host deploy auto build strategy prefers remote target host"

#!/usr/bin/env bash
set -euo pipefail

SCRIPT_UNDER_TEST="${1:-$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)/scripts/kkidc-host-deploy.sh}"

ROOT_DIR="$(mktemp -d)"
trap 'rm -rf "$ROOT_DIR"' EXIT

REPO_DIR="$ROOT_DIR/repo"
REMOTE_REPO_DIR="$ROOT_DIR/remote.git"
CONFIG_FILE="$ROOT_DIR/.env.lighthouse"
APP_ENV_FILE="$ROOT_DIR/.env.local"

git init --bare "$REMOTE_REPO_DIR" >/dev/null
git init "$REPO_DIR" >/dev/null
git -C "$REPO_DIR" config user.name test
git -C "$REPO_DIR" config user.email test@example.com
git -C "$REPO_DIR" branch -M codex/release-v0.11.8-p1
git -C "$REPO_DIR" remote add origin "$REMOTE_REPO_DIR"

cat > "$REPO_DIR/Dockerfile" <<'EOF'
FROM scratch
RUN go mod download
RUN DISABLE_ESLINT_PLUGIN='true' VITE_REACT_APP_VERSION=$(cat VERSION) npm run build
EOF
echo "seed" > "$REPO_DIR/payload.txt"
git -C "$REPO_DIR" add Dockerfile payload.txt
git -C "$REPO_DIR" commit -m "seed" >/dev/null
git -C "$REPO_DIR" push -u origin codex/release-v0.11.8-p1 >/dev/null

echo "unpushed" >> "$REPO_DIR/payload.txt"
git -C "$REPO_DIR" add payload.txt
git -C "$REPO_DIR" commit -m "unpushed local change" >/dev/null

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

set +e
OUTPUT="$(REPO_DIR="$REPO_DIR" APP_ENV_FILE="$APP_ENV_FILE" bash "$SCRIPT_UNDER_TEST" production --config "$CONFIG_FILE" --dry-run 2>&1)"
EXIT_CODE="$?"
set -e

if [ "$EXIT_CODE" -eq 0 ]; then
  echo "FAIL: deploy dry-run should reject local HEAD that has not been pushed" >&2
  exit 1
fi

if ! printf '%s\n' "$OUTPUT" | grep -q "deploy requires HEAD to be pushed"; then
  echo "FAIL: expected pushed-head rejection message" >&2
  echo "$OUTPUT" >&2
  exit 1
fi

echo "PASS: kkidc host deploy refuses local HEAD commits that are not pushed"

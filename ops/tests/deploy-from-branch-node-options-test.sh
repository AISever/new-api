#!/usr/bin/env bash
set -euo pipefail

SCRIPT_UNDER_TEST="${1:-$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)/scripts/deploy-from-branch.sh}"

ROOT_DIR="$(mktemp -d)"
trap 'rm -rf "$ROOT_DIR"' EXIT

ORIGIN_DIR="$ROOT_DIR/origin.git"
SEED_DIR="$ROOT_DIR/seed"
WORK_DIR="$ROOT_DIR/work"
FAKEBIN_DIR="$ROOT_DIR/fakebin"
SED_LOG="$ROOT_DIR/sed.log"

mkdir -p "$SEED_DIR" "$FAKEBIN_DIR"

git init --bare "$ORIGIN_DIR" >/dev/null
git init "$SEED_DIR" >/dev/null

git -C "$SEED_DIR" config user.name test
git -C "$SEED_DIR" config user.email test@example.com

cat > "$SEED_DIR/Dockerfile" <<'EOF'
FROM scratch
RUN go mod download
RUN DISABLE_ESLINT_PLUGIN='true' VITE_REACT_APP_VERSION=$(cat VERSION) bun run build
EOF
touch "$SEED_DIR/VERSION"
cat > "$SEED_DIR/docker-compose.yml" <<'EOF'
services:
  new-api:
    image: placeholder
EOF
echo "seed" > "$SEED_DIR/file.txt"

git -C "$SEED_DIR" add Dockerfile VERSION docker-compose.yml file.txt
git -C "$SEED_DIR" commit -m "seed" >/dev/null
git -C "$SEED_DIR" branch -M codex/prod-live
git -C "$SEED_DIR" remote add origin "$ORIGIN_DIR"
git -C "$SEED_DIR" push -u origin codex/prod-live >/dev/null

git clone --branch codex/prod-live "$ORIGIN_DIR" "$WORK_DIR" >/dev/null

cat > "$FAKEBIN_DIR/docker" <<'EOF'
#!/usr/bin/env bash
exit 0
EOF

cat > "$FAKEBIN_DIR/sed" <<'EOF'
#!/usr/bin/env bash
printf '%s\n' "$*" >> "$SED_LOG"
exit 0
EOF

chmod +x "$FAKEBIN_DIR/docker" "$FAKEBIN_DIR/sed"

PATH="$FAKEBIN_DIR:$PATH" \
SED_LOG="$SED_LOG" \
REPO_DIR="$WORK_DIR" \
PROJECT_NAME="dryrun" \
IMAGE_NAME="dryrun" \
COMPOSE_MODE="override" \
ALLOW_STALE_ON_FETCH_TIMEOUT="true" \
FRONTEND_BUILD_NODE_OPTIONS="--max-old-space-size=1024" \
bash "$SCRIPT_UNDER_TEST" codex/prod-live >/dev/null 2>&1

grep -F -- "--max-old-space-size=1024" "$SED_LOG" >/dev/null

echo "PASS: deploy-from-branch forwards FRONTEND_BUILD_NODE_OPTIONS into Dockerfile generation"

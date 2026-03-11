#!/usr/bin/env bash
set -euo pipefail

SCRIPT_UNDER_TEST="${1:-$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)/scripts/deploy-from-branch.sh}"

ROOT_DIR="$(mktemp -d)"
trap 'rm -rf "$ROOT_DIR"' EXIT

ORIGIN_DIR="$ROOT_DIR/origin.git"
SEED_DIR="$ROOT_DIR/seed"
WORK_DIR="$ROOT_DIR/work"
FAKEBIN_DIR="$ROOT_DIR/fakebin"
TIMEOUT_LOG="$ROOT_DIR/timeout.log"

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
git -C "$WORK_DIR" config user.name test
git -C "$WORK_DIR" config user.email test@example.com
git -C "$WORK_DIR" fetch origin codex/prod-live >/dev/null

echo "local-only" >> "$WORK_DIR/file.txt"
git -C "$WORK_DIR" commit -am "local-only" >/dev/null
LOCAL_HEAD="$(git -C "$WORK_DIR" rev-parse HEAD)"

cat > "$FAKEBIN_DIR/timeout" <<'EOF'
#!/usr/bin/env bash
printf '%s' "${1:-}" > "$TIMEOUT_LOG"
exit 124
EOF

cat > "$FAKEBIN_DIR/docker" <<'EOF'
#!/usr/bin/env bash
exit 0
EOF

cat > "$FAKEBIN_DIR/sed" <<'EOF'
#!/usr/bin/env bash
exit 0
EOF

chmod +x "$FAKEBIN_DIR/timeout" "$FAKEBIN_DIR/docker" "$FAKEBIN_DIR/sed"

set +e
PATH="$FAKEBIN_DIR:$PATH" \
TIMEOUT_LOG="$TIMEOUT_LOG" \
REPO_DIR="$WORK_DIR" \
PROJECT_NAME="dryrun" \
IMAGE_NAME="dryrun" \
COMPOSE_MODE="override" \
FETCH_TIMEOUT_SECONDS="7" \
bash "$SCRIPT_UNDER_TEST" codex/prod-live >/dev/null 2>&1
EXIT_CODE="$?"
set -e

CURRENT_HEAD="$(git -C "$WORK_DIR" rev-parse HEAD)"
ACTUAL_TIMEOUT="$(cat "$TIMEOUT_LOG")"

if [ "$EXIT_CODE" -eq 0 ]; then
  echo "FAIL: fetch timeout should fail by default instead of deploying stale code" >&2
  exit 1
fi

if [ "$CURRENT_HEAD" != "$LOCAL_HEAD" ]; then
  echo "FAIL: fetch timeout should not reset HEAD on failure" >&2
  exit 1
fi

if [ "$ACTUAL_TIMEOUT" != "7" ]; then
  echo "FAIL: expected timeout override 7, got $ACTUAL_TIMEOUT" >&2
  exit 1
fi

echo "PASS: deploy-from-branch fails closed on fetch timeout"

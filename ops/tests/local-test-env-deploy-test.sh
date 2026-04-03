#!/usr/bin/env bash
set -euo pipefail

SCRIPT_UNDER_TEST="${1:-$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)/scripts/local-test-env.sh}"

ROOT_DIR="$(mktemp -d)"
trap 'rm -rf "$ROOT_DIR"' EXIT

REPO_DIR="$ROOT_DIR/repo"
FAKEBIN_DIR="$ROOT_DIR/fakebin"
DOCKER_LOG="$ROOT_DIR/docker.log"
NPM_LOG="$ROOT_DIR/npm.log"

mkdir -p \
  "$REPO_DIR/ops/compose" \
  "$REPO_DIR/ops/docker" \
  "$REPO_DIR/ops/local-test/baseline" \
  "$REPO_DIR/web" \
  "$REPO_DIR/data" \
  "$REPO_DIR/logs" \
  "$FAKEBIN_DIR"

cat > "$REPO_DIR/ops/compose/local-test.yml" <<'EOF'
services:
  new-api:
    image: test
EOF

cat > "$REPO_DIR/ops/docker/Dockerfile.local-test" <<'EOF'
FROM scratch
EOF

cat > "$REPO_DIR/ops/local-test/baseline/manifest.json" <<'EOF'
{"version":"2026-03-23","admin":{"username":"admin","password":"Admin123456!"}}
EOF

touch "$REPO_DIR/ops/local-test/baseline/new-api.dump"
touch "$REPO_DIR/ops/local-test/baseline/postgres-globals.sql"

cat > "$FAKEBIN_DIR/docker" <<'EOF'
#!/usr/bin/env bash
printf '%s\n' "$*" >> "$DOCKER_LOG"
if [ "$1" = "inspect" ]; then
  if printf '%s' "$*" | grep -q 'Config.Image'; then
    echo "codex/new-api-local:test"
  else
    echo "running healthy 0"
  fi
  exit 0
fi
exit 0
EOF

cat > "$FAKEBIN_DIR/npm" <<'EOF'
#!/usr/bin/env bash
printf '%s\n' "$*" >> "$NPM_LOG"
exit 0
EOF

cat > "$FAKEBIN_DIR/node" <<'EOF'
#!/usr/bin/env bash
exit 0
EOF

cat > "$FAKEBIN_DIR/curl" <<'EOF'
#!/usr/bin/env bash
cat <<JSON
{"success":true,"data":{"setup":true}}
JSON
EOF

cat > "$FAKEBIN_DIR/git" <<'EOF'
#!/usr/bin/env bash
if [ "$1" = "rev-parse" ] && [ "$2" = "--short" ] && [ "$3" = "HEAD" ]; then
  echo "abc1234"
  exit 0
fi
exit 0
EOF

cat > "$FAKEBIN_DIR/date" <<'EOF'
#!/usr/bin/env bash
echo "20260323210000"
EOF

chmod +x "$FAKEBIN_DIR/docker" "$FAKEBIN_DIR/npm" "$FAKEBIN_DIR/node" \
  "$FAKEBIN_DIR/curl" "$FAKEBIN_DIR/git" "$FAKEBIN_DIR/date"

OUTPUT="$(
  PATH="$FAKEBIN_DIR:$PATH" \
  DOCKER_LOG="$DOCKER_LOG" \
  NPM_LOG="$NPM_LOG" \
  REPO_DIR="$REPO_DIR" \
  bash "$SCRIPT_UNDER_TEST" deploy
)"

printf '%s\n' "$OUTPUT" | grep -q '^action=deploy$'
printf '%s\n' "$OUTPUT" | grep -q '^app_url=http://127.0.0.1:3001$'
printf '%s\n' "$OUTPUT" | grep -q '^health_status=healthy$'
printf '%s\n' "$OUTPUT" | grep -q '^image=codex/new-api-local:test$'

grep -q 'run build' "$NPM_LOG"
grep -q 'compose .* config' "$DOCKER_LOG"
grep -q 'compose .* build new-api' "$DOCKER_LOG"
grep -q 'compose .* up -d postgres redis new-api' "$DOCKER_LOG"

echo "PASS: local-test-env deploy builds frontend, rebuilds containers, and verifies health"

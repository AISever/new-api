#!/usr/bin/env bash
set -euo pipefail

SCRIPT_UNDER_TEST="${1:-$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)/scripts/local-test-env.sh}"

ROOT_DIR="$(mktemp -d)"
trap 'rm -rf "$ROOT_DIR"' EXIT

REPO_DIR="$ROOT_DIR/repo"
FAKEBIN_DIR="$ROOT_DIR/fakebin"
DOCKER_LOG="$ROOT_DIR/docker.log"

mkdir -p \
  "$REPO_DIR/ops/compose" \
  "$REPO_DIR/ops/docker" \
  "$REPO_DIR/ops/local-test/baseline" \
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
  echo "running healthy 0"
  exit 0
fi
exit 0
EOF

cat > "$FAKEBIN_DIR/node" <<'EOF'
#!/usr/bin/env bash
exit 0
EOF

cat > "$FAKEBIN_DIR/npm" <<'EOF'
#!/usr/bin/env bash
exit 0
EOF

cat > "$FAKEBIN_DIR/curl" <<'EOF'
#!/usr/bin/env bash
cat <<JSON
{"success":true,"data":{"setup":true}}
JSON
EOF

chmod +x "$FAKEBIN_DIR/docker" "$FAKEBIN_DIR/node" "$FAKEBIN_DIR/npm" "$FAKEBIN_DIR/curl"

OUTPUT="$(
  PATH="$FAKEBIN_DIR:$PATH" \
  DOCKER_LOG="$DOCKER_LOG" \
  REPO_DIR="$REPO_DIR" \
  bash "$SCRIPT_UNDER_TEST" reset-to-baseline
)"

printf '%s\n' "$OUTPUT" | grep -q '^action=reset-to-baseline$'
printf '%s\n' "$OUTPUT" | grep -q '^health_status=healthy$'
printf '%s\n' "$OUTPUT" | grep -q '^admin_username=admin$'

grep -q 'compose .* down -v' "$DOCKER_LOG"
grep -q 'compose .* up -d postgres redis' "$DOCKER_LOG"
grep -q 'exec -i postgres-test psql -U root -d postgres' "$DOCKER_LOG"
grep -q 'exec -i postgres-test pg_restore -U root -d new-api' "$DOCKER_LOG"
grep -q 'exec -i postgres-test psql -U root -d new-api' "$DOCKER_LOG"
grep -q 'compose .* up -d new-api' "$DOCKER_LOG"

echo "PASS: local-test-env reset-to-baseline recreates services and restores the baseline"

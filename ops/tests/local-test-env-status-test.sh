#!/usr/bin/env bash
set -euo pipefail

SCRIPT_UNDER_TEST="${1:-$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)/scripts/local-test-env.sh}"

ROOT_DIR="$(mktemp -d)"
trap 'rm -rf "$ROOT_DIR"' EXIT

REPO_DIR="$ROOT_DIR/repo"
FAKEBIN_DIR="$ROOT_DIR/fakebin"

mkdir -p \
  "$REPO_DIR/ops/compose" \
  "$REPO_DIR/ops/docker" \
  "$REPO_DIR/ops/local-test/baseline" \
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
if [ "$1" = "inspect" ]; then
  case "$*" in
    *"Config.Image"*)
      echo "codex/new-api-local:abc1234"
      ;;
    *"State.Status"* )
      echo "running healthy 0"
      ;;
    *"NetworkSettings.Ports"* )
      echo "0.0.0.0:3001"
      ;;
    *)
      echo "running"
      ;;
  esac
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

chmod +x "$FAKEBIN_DIR/docker" "$FAKEBIN_DIR/node" "$FAKEBIN_DIR/npm"

OUTPUT="$(
  PATH="$FAKEBIN_DIR:$PATH" \
  REPO_DIR="$REPO_DIR" \
  bash "$SCRIPT_UNDER_TEST" status
)"

printf '%s\n' "$OUTPUT" | grep -q '^action=status$'
printf '%s\n' "$OUTPUT" | grep -q '^container=new-api-test$'
printf '%s\n' "$OUTPUT" | grep -q '^status=running$'
printf '%s\n' "$OUTPUT" | grep -q '^health_status=healthy$'
printf '%s\n' "$OUTPUT" | grep -q '^image=codex/new-api-local:abc1234$'
printf '%s\n' "$OUTPUT" | grep -q '^app_url=http://127.0.0.1:3001$'

echo "PASS: local-test-env status reports the active test environment"

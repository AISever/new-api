#!/usr/bin/env bash
set -euo pipefail

SCRIPT_UNDER_TEST="${1:-$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)/scripts/tencent-test-stop.sh}"

ROOT_DIR="$(mktemp -d)"
trap 'rm -rf "$ROOT_DIR"' EXIT

FAKEBIN_DIR="$ROOT_DIR/fakebin"
DOCKER_LOG="$ROOT_DIR/docker.log"
STATE_FILE="$ROOT_DIR/new-api-test.present"

mkdir -p "$FAKEBIN_DIR"

cat > "$FAKEBIN_DIR/docker" <<'EOF'
#!/usr/bin/env bash
printf '%s\n' "$*" >> "$DOCKER_LOG"
if [ "$1" = "container" ] && [ "$2" = "inspect" ] && [ "$3" = "new-api-test" ]; then
  [ -f "$STATE_FILE" ]
  exit $?
fi
if [ "$1" = "rm" ] && [ "$2" = "-f" ] && [ "$3" = "new-api-test" ]; then
  rm -f "$STATE_FILE"
  exit 0
fi
exit 0
EOF
chmod +x "$FAKEBIN_DIR/docker"

touch "$STATE_FILE"

OUTPUT="$(PATH="$FAKEBIN_DIR:$PATH" DOCKER_LOG="$DOCKER_LOG" STATE_FILE="$STATE_FILE" bash "$SCRIPT_UNDER_TEST")"

printf '%s\n' "$OUTPUT" | grep -q '^action=test-stop$'
printf '%s\n' "$OUTPUT" | grep -q '^target_env=tencent-test$'
printf '%s\n' "$OUTPUT" | grep -q '^already_stopped=false$'
printf '%s\n' "$OUTPUT" | grep -q '^stopped_container=new-api-test$'
printf '%s\n' "$OUTPUT" | grep -q '^retained_postgres_container=postgres-test$'
printf '%s\n' "$OUTPUT" | grep -q '^retained_redis_container=redis-test$'
printf '%s\n' "$OUTPUT" | grep -q '^retained_pg_db=new-api-test$'
printf '%s\n' "$OUTPUT" | grep -q '^retained_app_port=3001$'

if grep -q "postgres-test\\|redis-test" "$DOCKER_LOG"; then
  echo "FAIL: tencent test stop should not operate on postgres-test or redis-test" >&2
  exit 1
fi

OUTPUT2="$(PATH="$FAKEBIN_DIR:$PATH" DOCKER_LOG="$DOCKER_LOG" STATE_FILE="$STATE_FILE" bash "$SCRIPT_UNDER_TEST")"
printf '%s\n' "$OUTPUT2" | grep -q '^already_stopped=true$'
printf '%s\n' "$OUTPUT2" | grep -q '^stopped_container=$'

echo "PASS: tencent test stop removes only the test app and is idempotent"

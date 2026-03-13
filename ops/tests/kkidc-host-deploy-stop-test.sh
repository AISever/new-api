#!/usr/bin/env bash
set -euo pipefail

SCRIPT_UNDER_TEST="${1:-$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)/scripts/kkidc-host-deploy.sh}"

ROOT_DIR="$(mktemp -d)"
trap 'rm -rf "$ROOT_DIR"' EXIT

CONFIG_FILE="$ROOT_DIR/.env.lighthouse"
FAKEBIN_DIR="$ROOT_DIR/fakebin"
SSH_LOG="$ROOT_DIR/ssh.log"
STATE_FILE="$ROOT_DIR/new-api-test.present"

mkdir -p "$FAKEBIN_DIR"

cat > "$CONFIG_FILE" <<'EOF'
IP_1=114.66.47.192
SSH_user=root
SSH_password=test-password
HOSTNAME=api.aisever.cn
HOSTNAME_ALIASES=newapi.aisever.cn
EOF

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
  *"docker container inspect 'new-api-test'"*)
    [ -f "$STATE_FILE" ]
    exit $?
    ;;
  *"docker rm -f 'new-api-test'"*)
    rm -f "$STATE_FILE"
    exit 0
    ;;
  *)
    exit 0
    ;;
esac
EOF
chmod +x "$FAKEBIN_DIR/sshpass"

touch "$STATE_FILE"

OUTPUT="$(PATH="$FAKEBIN_DIR:$PATH" SSH_LOG="$SSH_LOG" STATE_FILE="$STATE_FILE" bash "$SCRIPT_UNDER_TEST" test-stop --config "$CONFIG_FILE")"

printf '%s\n' "$OUTPUT" | grep -q '^action=test-stop$'
printf '%s\n' "$OUTPUT" | grep -q '^target_env=test$'
printf '%s\n' "$OUTPUT" | grep -q '^already_stopped=false$'
printf '%s\n' "$OUTPUT" | grep -q '^stopped_container=new-api-test$'
printf '%s\n' "$OUTPUT" | grep -q '^retained_data_dir=/opt/new-api-test/data$'
printf '%s\n' "$OUTPUT" | grep -q '^retained_log_dir=/opt/new-api-test/logs$'
printf '%s\n' "$OUTPUT" | grep -q '^retained_pg_db=new-api-test$'
printf '%s\n' "$OUTPUT" | grep -q '^retained_redis_db=1$'

if grep -q "new-api-local\\|new-api-postgres\\|new-api-redis\\|snowlight-caddy" "$SSH_LOG"; then
  echo "FAIL: test-stop should not touch production or shared infrastructure containers" >&2
  exit 1
fi

OUTPUT2="$(PATH="$FAKEBIN_DIR:$PATH" SSH_LOG="$SSH_LOG" STATE_FILE="$STATE_FILE" bash "$SCRIPT_UNDER_TEST" test-stop --config "$CONFIG_FILE")"
printf '%s\n' "$OUTPUT2" | grep -q '^already_stopped=true$'
printf '%s\n' "$OUTPUT2" | grep -q '^stopped_container=$'

echo "PASS: kkidc host deploy test-stop removes only the test app and is idempotent"

#!/usr/bin/env bash
set -euo pipefail

SCRIPT_UNDER_TEST="${1:-$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)/scripts/kkidc-host-backup.sh}"

ROOT_DIR="$(mktemp -d)"
trap 'rm -rf "$ROOT_DIR"' EXIT

CONFIG_FILE="$ROOT_DIR/.env.lighthouse"
FAKEBIN_DIR="$ROOT_DIR/fakebin"
REMOTE_LOG="$ROOT_DIR/remote.log"

mkdir -p "$FAKEBIN_DIR"

cat > "$CONFIG_FILE" <<'EOF'
IP_1=114.66.47.192
SSH_user=root
SSH_password=test-password
HOSTNAME=api.aisever.cn
EOF

cat > "$FAKEBIN_DIR/sshpass" <<EOF
#!/usr/bin/env bash
shift 2
printf '%s\n' "\$*" >> "$REMOTE_LOG"
cat >/dev/null || true
exit 0
EOF
chmod +x "$FAKEBIN_DIR/sshpass"

OUTPUT="$(PATH="$FAKEBIN_DIR:$PATH" bash "$SCRIPT_UNDER_TEST" production --config "$CONFIG_FILE" --no-data)"

if ! printf '%s\n' "$OUTPUT" | grep -q '^action=backup$'; then
  echo "FAIL: expected backup action in final output" >&2
  exit 1
fi

if ! printf '%s\n' "$OUTPUT" | grep -q '^include_data=false$'; then
  echo "FAIL: expected include_data=false in final output" >&2
  exit 1
fi

if ! printf '%s\n' "$OUTPUT" | grep -Eq '^backup_dir=/opt/new-api-backups/[0-9]{14}$'; then
  echo "FAIL: expected remote backup dir in final output" >&2
  exit 1
fi

if ! grep -q "docker exec 'new-api-postgres' pg_dump -U newapi -d 'new-api'" "$REMOTE_LOG"; then
  echo "FAIL: expected pg_dump command to be sent to remote host" >&2
  exit 1
fi

echo "PASS: kkidc host backup completes remote backup flow with stubbed ssh"

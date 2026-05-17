#!/usr/bin/env bash
set -euo pipefail

SCRIPT_UNDER_TEST="${1:-$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)/scripts/kkidc-host-import-profile.sh}"

ROOT_DIR="$(mktemp -d)"
TARGET_PID=""

cleanup() {
  if [ -n "$TARGET_PID" ]; then
    kill "$TARGET_PID" 2>/dev/null || true
    wait "$TARGET_PID" 2>/dev/null || true
  fi
  rm -rf "$ROOT_DIR"
}

trap cleanup EXIT

CONFIG_FILE="$ROOT_DIR/.env.lighthouse"
TARGET_PORT_FILE="$ROOT_DIR/target.port"
SERVER_SCRIPT="$ROOT_DIR/fake_target_server.py"
TARGET_LOG="$ROOT_DIR/target.log"
PROFILE_PATH="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)/channel-onboarding/profiles/yunwu.yaml"

cat > "$CONFIG_FILE" <<'EOF'
IP_1=127.0.0.1
SSH_user=root
SSH_password=test-password
TEST_HOSTNAME=127.0.0.1
ENTERPRISE_HOSTNAME=
EOF

cat > "$SERVER_SCRIPT" <<'PY'
#!/usr/bin/env python3
import json
import sys
from http.server import BaseHTTPRequestHandler, ThreadingHTTPServer
from urllib.parse import urlparse

PORT_FILE = sys.argv[1]


class Handler(BaseHTTPRequestHandler):
    def log_message(self, format, *args):
        return

    def _read_json(self):
        length = int(self.headers.get("Content-Length", "0") or 0)
        if length <= 0:
            return {}
        raw = self.rfile.read(length)
        if not raw:
            return {}
        return json.loads(raw.decode("utf-8"))

    def _send_json(self, payload, status=200, headers=None):
        body = json.dumps(payload, ensure_ascii=False).encode("utf-8")
        self.send_response(status)
        self.send_header("Content-Type", "application/json; charset=utf-8")
        self.send_header("Content-Length", str(len(body)))
        if headers:
            for key, value in headers.items():
                self.send_header(key, value)
        self.end_headers()
        self.wfile.write(body)

    def _require_user(self):
        expected = str(self.server.state["user_id"])
        actual = self.headers.get("New-Api-User")
        if actual != expected:
            self._send_json({"success": False, "message": "missing New-Api-User"}, status=401)
            return False
        return True

    def do_GET(self):
        path = urlparse(self.path).path
        if path == "/api/setup":
            self._send_json({"success": True, "data": {"status": self.server.state["setup_done"]}})
            return
        if path == "/api/option/":
            if not self._require_user():
                return
            options = [{"key": key, "value": value} for key, value in self.server.state["options"].items()]
            self._send_json({"success": True, "data": options})
            return
        if path == "/api/channel/":
            if not self._require_user():
                return
            self._send_json({"success": True, "data": {"items": list(self.server.state["channels"])}})
            return
        if path == "/api/models/missing":
            if not self._require_user():
                return
            self._send_json({"success": True, "data": []})
            return
        self._send_json({"success": False, "message": f"unhandled GET {path}"}, status=404)

    def do_POST(self):
        path = urlparse(self.path).path
        body = self._read_json()
        if path == "/api/setup":
            self.server.state["setup_done"] = True
            self._send_json({"success": True, "data": {"status": True}})
            return
        if path == "/api/user/login":
            self._send_json({"success": True, "data": {"id": self.server.state["user_id"]}}, headers={"Set-Cookie": "target=session; Path=/"})
            return
        if path == "/api/channel/":
            if not self._require_user():
                return
            channel = body["channel"]
            channel["id"] = len(self.server.state["channels"]) + 1
            self.server.state["channels"].append(channel)
            self._send_json({"success": True, "data": {"id": channel["id"]}})
            return
        if path == "/api/channel/fix":
            if not self._require_user():
                return
            self._send_json({"success": True, "data": {"channels": len(self.server.state["channels"])}})
            return
        self._send_json({"success": False, "message": f"unhandled POST {path}"}, status=404)

    def do_PUT(self):
        path = urlparse(self.path).path
        body = self._read_json()
        if path == "/api/option/":
            if not self._require_user():
                return
            self.server.state["options"][body["key"]] = body["value"]
            self._send_json({"success": True, "data": {"key": body["key"]}})
            return
        if path == "/api/channel/":
            if not self._require_user():
                return
            channel_id = body["id"]
            updated = False
            for idx, item in enumerate(self.server.state["channels"]):
                if item["id"] == channel_id:
                    self.server.state["channels"][idx] = body
                    updated = True
                    break
            if not updated:
                self._send_json({"success": False, "message": "channel not found"}, status=404)
                return
            self._send_json({"success": True, "data": {"id": channel_id}})
            return
        self._send_json({"success": False, "message": f"unhandled PUT {path}"}, status=404)


server = ThreadingHTTPServer(("127.0.0.1", 3001), Handler)
server.state = {
    "user_id": 7,
    "setup_done": False,
    "options": {
        "UserUsableGroups": json.dumps({"default": "默认分组"}, ensure_ascii=False),
        "GroupRatio": json.dumps({"default": 1}, ensure_ascii=False),
        "ModelRatio": "{}",
        "CompletionRatio": "{}",
        "ModelPrice": "{}",
    },
    "channels": [],
}

with open(PORT_FILE, "w", encoding="utf-8") as fp:
    fp.write("3001")

server.serve_forever()
PY

python3 "$SERVER_SCRIPT" "$TARGET_PORT_FILE" >"$TARGET_LOG" 2>&1 &
TARGET_PID="$!"

for _ in $(seq 1 50); do
  if [ -s "$TARGET_PORT_FILE" ]; then
    break
  fi
  sleep 0.1
done

if [ ! -s "$TARGET_PORT_FILE" ]; then
  echo "FAIL: fake target server did not start" >&2
  exit 1
fi

set +e
DRY_OUTPUT="$(
  bash "$SCRIPT_UNDER_TEST" test \
    --config "$CONFIG_FILE" \
    --profile "$PROFILE_PATH" \
    --dry-run 2>&1
)"
DRY_EXIT_CODE="$?"
set -e

if [ "$DRY_EXIT_CODE" -ne 0 ]; then
  echo "FAIL: dry-run should succeed" >&2
  echo "$DRY_OUTPUT" >&2
  exit 1
fi

if ! printf '%s\n' "$DRY_OUTPUT" | grep -q '"environment": "test"'; then
  echo "FAIL: dry-run should print plan JSON" >&2
  echo "$DRY_OUTPUT" >&2
  exit 1
fi

set +e
OUTPUT="$(
  bash "$SCRIPT_UNDER_TEST" test \
    --config "$CONFIG_FILE" \
    --profile "$PROFILE_PATH" \
    --target-root-username "root" \
    --target-root-password "secret" 2>&1
)"
EXIT_CODE="$?"
set -e

if [ "$EXIT_CODE" -ne 0 ]; then
  echo "FAIL: profile import wrapper should succeed for test" >&2
  echo "$OUTPUT" >&2
  exit 1
fi

if ! printf '%s\n' "$OUTPUT" | grep -q '"target_environment": "test"'; then
  echo "FAIL: summary should include test target environment" >&2
  echo "$OUTPUT" >&2
  exit 1
fi

if ! printf '%s\n' "$OUTPUT" | grep -q '"target_channels_created": 3'; then
  echo "FAIL: wrapper should create three planned channels" >&2
  echo "$OUTPUT" >&2
  exit 1
fi

set +e
PROD_OUTPUT="$(
  bash "$SCRIPT_UNDER_TEST" production \
    --config "$CONFIG_FILE" \
    --profile "$PROFILE_PATH" \
    --target-root-username "root" \
    --target-root-password "secret" 2>&1
)"
PROD_EXIT_CODE="$?"
set -e

if [ "$PROD_EXIT_CODE" -eq 0 ]; then
  echo "FAIL: production target should be rejected" >&2
  echo "$PROD_OUTPUT" >&2
  exit 1
fi

if ! printf '%s\n' "$PROD_OUTPUT" | grep -q 'production'; then
  echo "FAIL: production rejection should mention production" >&2
  echo "$PROD_OUTPUT" >&2
  exit 1
fi

set +e
PROD_CONFIRMED_OUTPUT="$(
  bash "$SCRIPT_UNDER_TEST" production \
    --config "$CONFIG_FILE" \
    --profile "$PROFILE_PATH" \
    --target-base-url "http://127.0.0.1:3001" \
    --target-root-username "root" \
    --target-root-password "secret" \
    --confirm-production 2>&1
)"
PROD_CONFIRMED_EXIT_CODE="$?"
set -e

if [ "$PROD_CONFIRMED_EXIT_CODE" -ne 0 ]; then
  echo "FAIL: production target should be allowed with explicit confirmation and target URL" >&2
  echo "$PROD_CONFIRMED_OUTPUT" >&2
  exit 1
fi

if ! printf '%s\n' "$PROD_CONFIRMED_OUTPUT" | grep -q '"target_environment": "production"'; then
  echo "FAIL: confirmed production import should include production target environment" >&2
  echo "$PROD_CONFIRMED_OUTPUT" >&2
  exit 1
fi

echo "PASS: kkidc host import profile wrapper supports test dry-run/import and guarded production"

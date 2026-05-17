#!/usr/bin/env bash
set -euo pipefail

REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
SCRIPT_UNDER_TEST="${1:-$REPO_ROOT/ops/channel-onboarding/import_profile.py}"
PROFILE_UNDER_TEST="${2:-$REPO_ROOT/ops/channel-onboarding/profiles/yunwu.yaml}"

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

TARGET_PORT_FILE="$ROOT_DIR/target.port"
SERVER_SCRIPT="$ROOT_DIR/fake_target_server.py"
TARGET_LOG="$ROOT_DIR/target.log"

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


server = ThreadingHTTPServer(("127.0.0.1", 0), Handler)
server.state = {
    "user_id": 7,
    "setup_done": False,
    "options": {
        "UserUsableGroups": json.dumps({"default": "默认分组", "vip": "vip分组"}, ensure_ascii=False),
        "GroupRatio": json.dumps({"default": 1, "vip": 1}, ensure_ascii=False),
        "GroupGroupRatio": "{}",
        "ModelRatio": json.dumps({"gpt-4o-mini": 1}, ensure_ascii=False),
        "CompletionRatio": json.dumps({"gpt-4o-mini": 1}, ensure_ascii=False),
        "ModelPrice": "{}",
    },
    "channels": [
        {
            "id": 1,
            "name": "existing-text-channel",
            "type": 1,
            "group": "default",
            "models": "gpt-4o-mini",
            "tag": "existing",
            "settings": "{}",
        }
    ],
}

with open(PORT_FILE, "w", encoding="utf-8") as fp:
    fp.write(str(server.server_address[1]))

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

TARGET_PORT="$(cat "$TARGET_PORT_FILE")"

set +e
OUTPUT="$(
  python3 "$SCRIPT_UNDER_TEST" \
    --profile "$PROFILE_UNDER_TEST" \
    --target-base-url "http://127.0.0.1:$TARGET_PORT" \
    --target-root-username "root" \
    --target-root-password "secret" \
    --target-environment test 2>&1
)"
EXIT_CODE="$?"
set -e

if [ "$EXIT_CODE" -ne 0 ]; then
  echo "FAIL: import should succeed against fake target" >&2
  echo "$OUTPUT" >&2
  exit 1
fi

if ! printf '%s\n' "$OUTPUT" | grep -q '"target_channels_created": 3'; then
  echo "FAIL: import should create three family channels" >&2
  echo "$OUTPUT" >&2
  exit 1
fi

if ! printf '%s\n' "$OUTPUT" | grep -q '"merged_group_ratio_keys": \['; then
  echo "FAIL: summary should report merged group ratio keys" >&2
  echo "$OUTPUT" >&2
  exit 1
fi

if ! printf '%s\n' "$OUTPUT" | grep -q '"preserved_existing_channel_count": 1'; then
  echo "FAIL: existing channels should be preserved" >&2
  echo "$OUTPUT" >&2
  exit 1
fi

if ! printf '%s\n' "$OUTPUT" | grep -q '"preserved_existing_groups": \['; then
  echo "FAIL: existing groups should be preserved" >&2
  echo "$OUTPUT" >&2
  exit 1
fi

set +e
PROD_OUTPUT="$(
  python3 "$SCRIPT_UNDER_TEST" \
    --profile "$PROFILE_UNDER_TEST" \
    --target-base-url "http://127.0.0.1:$TARGET_PORT" \
    --target-root-username "root" \
    --target-root-password "secret" \
    --target-environment production 2>&1
)"
PROD_EXIT_CODE="$?"
set -e

if [ "$PROD_EXIT_CODE" -eq 0 ]; then
  echo "FAIL: production target should be rejected by importer" >&2
  echo "$PROD_OUTPUT" >&2
  exit 1
fi

if ! printf '%s\n' "$PROD_OUTPUT" | grep -q 'production'; then
  echo "FAIL: production rejection should mention production guard" >&2
  echo "$PROD_OUTPUT" >&2
  exit 1
fi

echo "PASS: channel onboarding importer merges safely and rejects production writes"

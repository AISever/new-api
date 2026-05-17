#!/usr/bin/env bash
set -euo pipefail

SCRIPT_UNDER_TEST="${1:-$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)/scripts/kkidc-host-import-profile.sh}"

ROOT_DIR="$(mktemp -d)"
TARGET_PID=""
UPSTREAM_PID=""

cleanup() {
  if [ -n "$TARGET_PID" ]; then
    kill "$TARGET_PID" 2>/dev/null || true
    wait "$TARGET_PID" 2>/dev/null || true
  fi
  if [ -n "$UPSTREAM_PID" ]; then
    kill "$UPSTREAM_PID" 2>/dev/null || true
    wait "$UPSTREAM_PID" 2>/dev/null || true
  fi
  rm -rf "$ROOT_DIR"
}

trap cleanup EXIT

CONFIG_FILE="$ROOT_DIR/.env.lighthouse"
TARGET_PORT_FILE="$ROOT_DIR/target.port"
UPSTREAM_PORT_FILE="$ROOT_DIR/upstream.port"
TARGET_SERVER_SCRIPT="$ROOT_DIR/fake_target_server.py"
UPSTREAM_SERVER_SCRIPT="$ROOT_DIR/fake_upstream_server.py"
TARGET_LOG="$ROOT_DIR/target.log"
UPSTREAM_LOG="$ROOT_DIR/upstream.log"
PROFILE_COPY="$ROOT_DIR/yunwu-local.yaml"

cat > "$CONFIG_FILE" <<'EOF'
IP_1=127.0.0.1
SSH_user=root
SSH_password=test-password
TEST_HOSTNAME=127.0.0.1
ENTERPRISE_HOSTNAME=
EOF

cat > "$TARGET_SERVER_SCRIPT" <<'PY'
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
        self._send_json({"success": False, "message": f"unhandled PUT {path}"}, status=404)


server = ThreadingHTTPServer(("127.0.0.1", 0), Handler)
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
    fp.write(str(server.server_address[1]))

server.serve_forever()
PY

cat > "$UPSTREAM_SERVER_SCRIPT" <<'PY'
#!/usr/bin/env python3
import json
import sys
from http.server import BaseHTTPRequestHandler, ThreadingHTTPServer

PORT_FILE = sys.argv[1]


EXPECTED_REQUESTS = {
    "/kling/v1/videos/text2video": {
        "auth": "Bearer kling-secret",
        "required": {"model_name": "kling-v1"},
    },
    "/ent/v2/text2video": {
        "auth": "Bearer sk-vidu-secret",
        "required": {"model": "viduq2", "duration": 5, "resolution": "720p"},
    },
    "/volc/v1/contents/generations/tasks": {
        "auth": "Bearer doubao-secret",
        "required": {"model": "doubao-seedance-1-0-lite-t2v-250428"},
    },
}


class Handler(BaseHTTPRequestHandler):
    def log_message(self, format, *args):
        return

    def _send_json(self, payload, status=200):
        body = json.dumps(payload, ensure_ascii=False).encode("utf-8")
        self.send_response(status)
        self.send_header("Content-Type", "application/json; charset=utf-8")
        self.send_header("Content-Length", str(len(body)))
        self.end_headers()
        self.wfile.write(body)

    def do_POST(self):
        expected = EXPECTED_REQUESTS.get(self.path)
        if expected is None:
            self._send_json({"error": "unknown path"}, status=404)
            return
        auth = self.headers.get("Authorization", "")
        if auth != expected["auth"]:
            self._send_json({"error": f"bad auth: {auth}"}, status=401)
            return
        length = int(self.headers.get("Content-Length", "0") or 0)
        payload = json.loads(self.rfile.read(length).decode("utf-8")) if length > 0 else {}
        for key, value in expected["required"].items():
            if payload.get(key) != value:
                self._send_json({"error": f"bad payload field {key}: {payload.get(key)!r}"}, status=422)
                return
        self._send_json({"error": "probe accepted"}, status=422)


server = ThreadingHTTPServer(("127.0.0.1", 0), Handler)
with open(PORT_FILE, "w", encoding="utf-8") as fp:
    fp.write(str(server.server_address[1]))
server.serve_forever()
PY

python3 "$TARGET_SERVER_SCRIPT" "$TARGET_PORT_FILE" >"$TARGET_LOG" 2>&1 &
TARGET_PID="$!"
python3 "$UPSTREAM_SERVER_SCRIPT" "$UPSTREAM_PORT_FILE" >"$UPSTREAM_LOG" 2>&1 &
UPSTREAM_PID="$!"

for _ in $(seq 1 50); do
  if [ -s "$TARGET_PORT_FILE" ] && [ -s "$UPSTREAM_PORT_FILE" ]; then
    break
  fi
  sleep 0.1
done

if [ ! -s "$TARGET_PORT_FILE" ] || [ ! -s "$UPSTREAM_PORT_FILE" ]; then
  echo "FAIL: fake target/upstream server did not start" >&2
  exit 1
fi

TARGET_PORT="$(cat "$TARGET_PORT_FILE")"
UPSTREAM_PORT="$(cat "$UPSTREAM_PORT_FILE")"

python3 - "$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)/channel-onboarding/profiles/yunwu.yaml" "$PROFILE_COPY" "$UPSTREAM_PORT" <<'PY'
import pathlib
import sys

src = pathlib.Path(sys.argv[1]).read_text(encoding="utf-8")
src = src.replace("base_url: https://yunwu.ai", f"base_url: http://127.0.0.1:{sys.argv[3]}")
dst = pathlib.Path(sys.argv[2])
dst.write_text(src, encoding="utf-8")
PY

set +e
OUTPUT="$(
  bash "$SCRIPT_UNDER_TEST" test \
    --config "$CONFIG_FILE" \
    --profile "$PROFILE_COPY" \
    --target-base-url "http://127.0.0.1:$TARGET_PORT" \
    --target-root-username "root" \
    --target-root-password "secret" \
    --probe-upstream \
    --require-channel-keys \
    --channel-key kling=kling-secret \
    --channel-key vidu=sk-vidu-secret \
    --channel-key doubao-video=doubao-secret 2>&1
)"
EXIT_CODE="$?"
set -e

if [ "$EXIT_CODE" -ne 0 ]; then
  echo "FAIL: wrapper should support probe and channel-key passthrough" >&2
  echo "$OUTPUT" >&2
  exit 1
fi

if ! printf '%s\n' "$OUTPUT" | grep -q '"passed": 3'; then
  echo "FAIL: wrapper output should include successful probe summary" >&2
  echo "$OUTPUT" >&2
  exit 1
fi

echo "PASS: kkidc host import profile wrapper passes through probe and explicit family keys"

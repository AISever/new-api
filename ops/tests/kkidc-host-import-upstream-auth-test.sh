#!/usr/bin/env bash
set -euo pipefail

SCRIPT_UNDER_TEST="${1:-$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)/scripts/kkidc-host-import-upstream.sh}"

ROOT_DIR="$(mktemp -d)"
UPSTREAM_PID=""
TARGET_PID=""

cleanup() {
  if [ -n "$UPSTREAM_PID" ]; then
    kill "$UPSTREAM_PID" 2>/dev/null || true
    wait "$UPSTREAM_PID" 2>/dev/null || true
  fi
  if [ -n "$TARGET_PID" ]; then
    kill "$TARGET_PID" 2>/dev/null || true
    wait "$TARGET_PID" 2>/dev/null || true
  fi
  rm -rf "$ROOT_DIR"
}

trap cleanup EXIT

CONFIG_FILE="$ROOT_DIR/.env.lighthouse"
UPSTREAM_PORT_FILE="$ROOT_DIR/upstream.port"
TARGET_PORT_FILE="$ROOT_DIR/target.port"
SERVER_SCRIPT="$ROOT_DIR/fake_api_server.py"
UPSTREAM_LOG="$ROOT_DIR/upstream.log"
TARGET_LOG="$ROOT_DIR/target.log"

cat > "$CONFIG_FILE" <<'EOF'
IP_1=127.0.0.1
SSH_user=root
SSH_password=test-password
HOSTNAME=api.aisever.cn
TEST_HOSTNAME=127.0.0.1
ENTERPRISE_HOSTNAME=corp-api.aisever.cn
EOF

cat > "$SERVER_SCRIPT" <<'PY'
#!/usr/bin/env python3
import json
import sys
from http.server import BaseHTTPRequestHandler, ThreadingHTTPServer
from urllib.parse import parse_qs, urlparse


ROLE = sys.argv[1]
PORT_FILE = sys.argv[2]


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

    def _target_auth_failed(self, message):
        self._send_json({"success": False, "message": message}, status=401)

    def _require_target_user(self):
        expected = str(self.server.state["user_id"])
        actual = self.headers.get("New-Api-User")
        if not actual:
            self._target_auth_failed("无权进行此操作，未提供 New-Api-User")
            return False
        if actual != expected:
            self._target_auth_failed("无权进行此操作，New-Api-User 与登录用户不匹配")
            return False
        return True

    def _require_upstream_user(self):
        expected = str(self.server.state["user_id"])
        actual = self.headers.get("New-Api-User")
        if actual != expected:
            self._send_json({"success": False, "message": "missing upstream user header"}, status=401)
            return False
        return True

    def do_GET(self):
        parsed = urlparse(self.path)
        path = parsed.path

        if ROLE == "upstream":
            if path == "/api/user/self/groups":
                if not self._require_upstream_user():
                    return
                self._send_json(
                    {
                        "success": True,
                        "data": {
                            "default": {"desc": "默认分组"},
                        },
                    }
                )
                return
            if path == "/api/user/models":
                if not self._require_upstream_user():
                    return
                self._send_json({"success": True, "data": [{"id": "gpt-4o-mini"}]})
                return
            if path == "/api/pricing":
                self._send_json(
                    {
                        "success": True,
                        "group_ratio": {"default": 1},
                        "auto_groups": ["default"],
                        "data": [
                            {
                                "model_name": "gpt-4o-mini",
                                "enable_groups": ["default"],
                                "quota_type": 0,
                                "model_ratio": 1,
                                "completion_ratio": 1,
                                "supported_endpoint_types": ["openai"],
                            }
                        ],
                    }
                )
                return
            if path == "/api/token/":
                if not self._require_upstream_user():
                    return
                self._send_json({"success": True, "data": {"items": list(self.server.state["tokens"])}})
                return
            if path == "/v1/models":
                auth = self.headers.get("Authorization", "")
                if auth != "Bearer token-default":
                    self._send_json({"error": "invalid token"}, status=401)
                    return
                self._send_json({"data": [{"id": "gpt-4o-mini"}]})
                return

        if ROLE == "target":
            if path == "/api/setup":
                self._send_json({"success": True, "data": {"status": self.server.state["setup_done"]}})
                return
            if path == "/api/channel/":
                if not self._require_target_user():
                    return
                self._send_json({"success": True, "data": {"items": list(self.server.state["channels"])}})
                return
            if path == "/api/models/missing":
                if not self._require_target_user():
                    return
                missing = [] if self.server.state["sync_done"] else ["gpt-4o-mini"]
                self._send_json({"success": True, "data": missing})
                return

        self._send_json({"success": False, "message": f"unhandled GET {path}"}, status=404)

    def do_POST(self):
        parsed = urlparse(self.path)
        path = parsed.path
        body = self._read_json()

        if ROLE == "upstream":
            if path == "/api/user/login":
                self._send_json(
                    {"success": True, "data": {"id": self.server.state["user_id"]}},
                    headers={"Set-Cookie": "upstream=session; Path=/"},
                )
                return
            if path == "/api/token/":
                if not self._require_upstream_user():
                    return
                token = {
                    "id": len(self.server.state["tokens"]) + 1,
                    "name": body["name"],
                    "group": body["group"],
                }
                self.server.state["tokens"].append(token)
                self._send_json({"success": True, "data": token})
                return
            if path.endswith("/key"):
                if not self._require_upstream_user():
                    return
                self._send_json({"success": True, "data": {"key": "token-default"}})
                return

        if ROLE == "target":
            if path == "/api/setup":
                self.server.state["setup_done"] = True
                self._send_json({"success": True, "data": {"status": True}})
                return
            if path == "/api/user/login":
                self._send_json(
                    {"success": True, "data": {"id": self.server.state["user_id"]}},
                    headers={"Set-Cookie": "target=session; Path=/"},
                )
                return
            if path == "/api/channel/":
                if not self._require_target_user():
                    return
                self.server.state["channels"].append(body["channel"])
                self._send_json({"success": True, "data": {"id": len(self.server.state["channels"])}})
                return
            if path == "/api/channel/fix":
                if not self._require_target_user():
                    return
                self._send_json({"success": True, "data": {"channels": len(self.server.state["channels"])}})
                return
            if path == "/api/models/sync_upstream":
                if not self._require_target_user():
                    return
                self.server.state["sync_done"] = True
                self._send_json({"success": True, "data": {"synced": True, "locale": body.get("locale")}})
                return

        self._send_json({"success": False, "message": f"unhandled POST {path}"}, status=404)

    def do_PUT(self):
        parsed = urlparse(self.path)
        path = parsed.path
        body = self._read_json()

        if ROLE == "target" and path == "/api/option/":
            if not self._require_target_user():
                return
            self.server.state["options"][body["key"]] = body["value"]
            self._send_json({"success": True, "data": {"key": body["key"]}})
            return

        self._send_json({"success": False, "message": f"unhandled PUT {path}"}, status=404)


def build_state():
    if ROLE == "upstream":
        return {"user_id": 11, "tokens": []}
    return {
        "user_id": 7,
        "setup_done": False,
        "channels": [],
        "options": {},
        "sync_done": False,
    }


server = ThreadingHTTPServer(("127.0.0.1", 0), Handler)
server.state = build_state()

with open(PORT_FILE, "w", encoding="utf-8") as fp:
    fp.write(str(server.server_address[1]))

server.serve_forever()
PY

python3 "$SERVER_SCRIPT" upstream "$UPSTREAM_PORT_FILE" >"$UPSTREAM_LOG" 2>&1 &
UPSTREAM_PID="$!"
python3 "$SERVER_SCRIPT" target "$TARGET_PORT_FILE" >"$TARGET_LOG" 2>&1 &
TARGET_PID="$!"

for _ in $(seq 1 50); do
  if [ -s "$UPSTREAM_PORT_FILE" ] && [ -s "$TARGET_PORT_FILE" ]; then
    break
  fi
  sleep 0.1
done

if [ ! -s "$UPSTREAM_PORT_FILE" ] || [ ! -s "$TARGET_PORT_FILE" ]; then
  echo "FAIL: fake upstream/target servers did not start" >&2
  exit 1
fi

UPSTREAM_PORT="$(cat "$UPSTREAM_PORT_FILE")"
TARGET_PORT="$(cat "$TARGET_PORT_FILE")"

set +e
OUTPUT="$(
  bash "$SCRIPT_UNDER_TEST" test \
    --config "$CONFIG_FILE" \
    --target-base-url "http://127.0.0.1:$TARGET_PORT" \
    --upstream-base-url "http://127.0.0.1:$UPSTREAM_PORT" \
    --upstream-username "aidev110" \
    --upstream-password "noCqok-wabdej-6juwze" \
    --target-root-username "root" \
    --target-root-password "secret" 2>&1
)"
EXIT_CODE="$?"
set -e

if [ "$EXIT_CODE" -ne 0 ]; then
  echo "FAIL: import should succeed against fake upstream/target" >&2
  echo "$OUTPUT" >&2
  exit 1
fi

if ! printf '%s\n' "$OUTPUT" | grep -q '"target_base_url": "http://127.0.0.1:'; then
  echo "FAIL: summary should include fake target base url" >&2
  exit 1
fi

if ! printf '%s\n' "$OUTPUT" | grep -q '"target_channels_created": 1'; then
  echo "FAIL: import should create one target channel" >&2
  exit 1
fi

if ! printf '%s\n' "$OUTPUT" | grep -q '"target_missing_models_before_official_sync": 1'; then
  echo "FAIL: import should detect one missing model before sync" >&2
  exit 1
fi

if ! printf '%s\n' "$OUTPUT" | grep -q '"target_missing_models_after_official_sync": 0'; then
  echo "FAIL: import should clear missing models after sync" >&2
  exit 1
fi

if ! printf '%s\n' "$OUTPUT" | grep -q '"upstream_tokens_created": 1'; then
  echo "FAIL: import should create one upstream token" >&2
  exit 1
fi

if ! printf '%s\n' "$OUTPUT" | grep -q '"target_docs_manifest_path": "/enterprise-docs/apifox/manifest.json"'; then
  echo "FAIL: import should configure enterprise docs manifest path" >&2
  exit 1
fi

echo "PASS: kkidc host import sync includes target admin New-Api-User header"

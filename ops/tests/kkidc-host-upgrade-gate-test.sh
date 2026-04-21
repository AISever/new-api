#!/usr/bin/env bash
set -euo pipefail

SCRIPT_UNDER_TEST="${1:-$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)/scripts/kkidc-host-upgrade-gate.sh}"

ROOT_DIR="$(mktemp -d)"
trap 'rm -rf "$ROOT_DIR"' EXIT

REPO_DIR="$ROOT_DIR/repo"
REMOTE_REPO_DIR="$ROOT_DIR/remote.git"
BIN_DIR="$ROOT_DIR/bin"
GO_LOG="$ROOT_DIR/go.log"
BUN_LOG="$ROOT_DIR/bun.log"
BASH_LOG="$ROOT_DIR/bash.log"
CURL_LOG="$ROOT_DIR/curl.log"

mkdir -p "$REPO_DIR/web" "$REMOTE_REPO_DIR" "$BIN_DIR"

git init --bare "$REMOTE_REPO_DIR" >/dev/null
git init "$REPO_DIR" >/dev/null
git -C "$REPO_DIR" config user.name test
git -C "$REPO_DIR" config user.email test@example.com
git -C "$REPO_DIR" branch -M codex/release-v0.12.12-p1
git -C "$REPO_DIR" remote add origin "$REMOTE_REPO_DIR"

cat > "$REPO_DIR/README.md" <<'EOF'
seed
EOF
cat > "$REPO_DIR/web/package.json" <<'EOF'
{"name":"test-web"}
EOF
git -C "$REPO_DIR" add README.md web/package.json
git -C "$REPO_DIR" commit -m "seed" >/dev/null
git -C "$REPO_DIR" push -u origin codex/release-v0.12.12-p1 >/dev/null
TEST_SHA="$(git -C "$REPO_DIR" rev-parse --short HEAD)"

cat > "$BIN_DIR/go" <<'EOF'
#!/bin/bash
echo "$*" >> "$GO_LOG"
exit 0
EOF

cat > "$BIN_DIR/bun" <<'EOF'
#!/bin/bash
echo "$*" >> "$BUN_LOG"
exit 0
EOF

cat > "$BIN_DIR/curl" <<'EOF'
#!/bin/bash
echo "$*" >> "$CURL_LOG"
cat <<JSON
{"data":{"version":"codex/release-v0.12.12-p1+${TEST_SHA}+kkidc-test"},"message":"","success":true}
JSON
EOF

cat > "$BIN_DIR/bash" <<'EOF'
#!/bin/bash
echo "$*" >> "$BASH_LOG"
case "$1" in
  *ops/tests/kkidc-host-deploy-pushed-head-test.sh|\
  *ops/tests/kkidc-host-deploy-build-strategy-test.sh|\
  *ops/tests/kkidc-host-deploy-local-build-platform-test.sh|\
  *ops/scripts/kkidc-host-deploy.sh)
    exit 0
    ;;
  *)
    exec /bin/bash "$@"
    ;;
esac
EOF

chmod +x "$BIN_DIR/go" "$BIN_DIR/bun" "$BIN_DIR/curl" "$BIN_DIR/bash"

export PATH="$BIN_DIR:$PATH"
export GO_LOG BUN_LOG BASH_LOG CURL_LOG TEST_SHA REPO_DIR

/bin/bash "$SCRIPT_UNDER_TEST" test >/tmp/upgrade-gate-test.out

if ! grep -q '^test ./controller ./model -count=1$' "$GO_LOG"; then
  echo "FAIL: expected local backend regression suite to run" >&2
  cat "$GO_LOG" >&2 || true
  exit 1
fi

if ! grep -q '^run build$' "$BUN_LOG"; then
  echo "FAIL: expected frontend build to run via bun" >&2
  cat "$BUN_LOG" >&2 || true
  exit 1
fi

for expected in \
  "ops/tests/kkidc-host-deploy-pushed-head-test.sh" \
  "ops/tests/kkidc-host-deploy-build-strategy-test.sh" \
  "ops/tests/kkidc-host-deploy-local-build-platform-test.sh" \
  "ops/scripts/kkidc-host-deploy.sh test" \
  "ops/scripts/kkidc-host-deploy.sh test-stop"
do
  if ! grep -q "$expected" "$BASH_LOG"; then
    echo "FAIL: expected bash invocation containing '$expected'" >&2
    cat "$BASH_LOG" >&2 || true
    exit 1
  fi
done

if ! grep -q '/api/status' "$CURL_LOG"; then
  echo "FAIL: expected test status check via curl" >&2
  cat "$CURL_LOG" >&2 || true
  exit 1
fi

: > "$BASH_LOG"

/bin/bash "$SCRIPT_UNDER_TEST" test --keep-test-running >/tmp/upgrade-gate-test-keep.out

if ! grep -q 'ops/scripts/kkidc-host-deploy.sh test' "$BASH_LOG"; then
  echo "FAIL: expected test deploy call in keep-running mode" >&2
  cat "$BASH_LOG" >&2 || true
  exit 1
fi

if grep -q 'ops/scripts/kkidc-host-deploy.sh test-stop' "$BASH_LOG"; then
  echo "FAIL: keep-running mode should not stop the test environment" >&2
  cat "$BASH_LOG" >&2 || true
  exit 1
fi

echo "PASS: kkidc host upgrade gate delegates local checks and test deployment correctly"

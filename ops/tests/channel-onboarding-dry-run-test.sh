#!/usr/bin/env bash
set -euo pipefail

REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
SCRIPT_UNDER_TEST="${1:-$REPO_ROOT/ops/channel-onboarding/dry_run.py}"
PROFILE_UNDER_TEST="${2:-$REPO_ROOT/ops/channel-onboarding/profiles/yunwu.yaml}"
MANIFEST_TEMPLATE="${3:-$REPO_ROOT/ops/channel-onboarding/manifest-template.yaml}"

TMP_DIR="$(mktemp -d)"
trap 'rm -rf "$TMP_DIR"' EXIT

require_file() {
  local path="$1"
  [ -f "$path" ] || {
    echo "FAIL: missing required file: $path" >&2
    exit 1
  }
}

require_file "$SCRIPT_UNDER_TEST"
require_file "$PROFILE_UNDER_TEST"
require_file "$MANIFEST_TEMPLATE"

run_dry_run() {
  local output_path="$1"
  python3 "$SCRIPT_UNDER_TEST" \
    --profile "$PROFILE_UNDER_TEST" \
    --target-environment test \
    >"$output_path"
}

OUTPUT_ONE="$TMP_DIR/output-one.json"
OUTPUT_TWO="$TMP_DIR/output-two.json"

run_dry_run "$OUTPUT_ONE"
run_dry_run "$OUTPUT_TWO"

if ! cmp -s "$OUTPUT_ONE" "$OUTPUT_TWO"; then
  echo "FAIL: dry-run output is not deterministic" >&2
  diff -u "$OUTPUT_ONE" "$OUTPUT_TWO" >&2 || true
  exit 1
fi

python3 - "$OUTPUT_ONE" <<'PY'
import json
import sys

path = sys.argv[1]
with open(path, "r", encoding="utf-8") as f:
    data = json.load(f)

assert data["success"] is True, "expected success=true"
assert data["target"]["environment"] == "test", "expected target environment=test"
assert data["source"]["slug"] == "yunwu", "expected source slug to be yunwu"

families = data["families"]
assert families == ["doubao-video", "kling", "vidu"], f"unexpected families: {families}"

channels = data["planned_channels"]
assert len(channels) == 3, f"expected 3 planned channels, got {len(channels)}"

family_map = {item["family"]: item for item in channels}

assert family_map["kling"]["type"] == 50, "kling should map to ChannelTypeKling"
assert family_map["vidu"]["type"] == 52, "vidu should map to ChannelTypeVidu"
assert family_map["doubao-video"]["type"] == 54, "doubao-video should map to ChannelTypeDoubaoVideo"

assert family_map["kling"]["test_model"] == "kling-v1", "unexpected kling test model"
assert family_map["vidu"]["test_model"] == "viduq2", "unexpected vidu test model"
assert family_map["doubao-video"]["test_model"] == "doubao-seedance-1-0-lite-t2v-250428", "unexpected doubao test model"

expected_tag = "personal-upstream-yunwu"
for item in channels:
    assert item["tag"] == expected_tag, f"unexpected tag for {item['family']}: {item['tag']}"
    assert item["name"].startswith("newapi::yunwu::"), f"unexpected channel name: {item['name']}"

assert "production_release_guard" in data["verification"], "missing production release guard"
assert data["verification"]["production_release_guard"]["requires_backup"] is True, "expected backup guard"
PY

MISSING_DOUBAO_PROFILE="$TMP_DIR/missing-doubao.yaml"
python3 - "$PROFILE_UNDER_TEST" "$MISSING_DOUBAO_PROFILE" <<'PY'
import pathlib
import sys
import yaml

src = yaml.safe_load(pathlib.Path(sys.argv[1]).read_text(encoding="utf-8"))
for family in src["families"]:
    if family.get("family") == "doubao-video":
        family["model_source"]["models"] = []
dst = pathlib.Path(sys.argv[2])
dst.write_text(yaml.safe_dump(src, allow_unicode=True, sort_keys=False), encoding="utf-8")
PY

set +e
python3 "$SCRIPT_UNDER_TEST" --profile "$MISSING_DOUBAO_PROFILE" --target-environment test >"$TMP_DIR/should-fail.json" 2>"$TMP_DIR/should-fail.err"
STATUS=$?
set -e

if [ "$STATUS" -eq 0 ]; then
  echo "FAIL: expected dry-run to fail when Doubao model list is incomplete" >&2
  cat "$TMP_DIR/should-fail.json" >&2 || true
  exit 1
fi

if ! grep -q "doubao-video" "$TMP_DIR/should-fail.err"; then
  echo "FAIL: expected failure reason to mention doubao-video gap" >&2
  cat "$TMP_DIR/should-fail.err" >&2 || true
  exit 1
fi

echo "PASS: channel onboarding dry-run is deterministic and validates required family models"

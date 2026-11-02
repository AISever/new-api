#!/usr/bin/env bash
set -euo pipefail
task_root="$(git -C "$(dirname "${BASH_SOURCE[0]}")" rev-parse --show-toplevel)"
cd "$task_root"
task_go="${NEW_API_GO:-go}"
exec "$task_go" run ./ops/tools/redemption-reward-backfill "$@"

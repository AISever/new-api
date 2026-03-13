#!/usr/bin/env bash
set -euo pipefail

APP_CONTAINER="${APP_CONTAINER:-new-api-test}"
POSTGRES_CONTAINER="${POSTGRES_CONTAINER:-postgres-test}"
REDIS_CONTAINER="${REDIS_CONTAINER:-redis-test}"
PG_DB="${PG_DB:-new-api-test}"
APP_PORT="${APP_PORT:-3001}"

usage() {
  cat <<'EOF'
Stop the tencent test application container while keeping test data in place.

Usage:
  ./ops/scripts/tencent-test-stop.sh
EOF
}

if [ "${1:-}" = "-h" ] || [ "${1:-}" = "--help" ]; then
  usage
  exit 0
fi

already_stopped="false"
stopped_container="$APP_CONTAINER"

if docker container inspect "$APP_CONTAINER" >/dev/null 2>&1; then
  docker rm -f "$APP_CONTAINER" >/dev/null
else
  already_stopped="true"
  stopped_container=""
fi

cat <<EOF
action=test-stop
target_env=tencent-test
app_container=${APP_CONTAINER}
already_stopped=${already_stopped}
stopped_container=${stopped_container}
retained_postgres_container=${POSTGRES_CONTAINER}
retained_redis_container=${REDIS_CONTAINER}
retained_pg_db=${PG_DB}
retained_app_port=${APP_PORT}
EOF

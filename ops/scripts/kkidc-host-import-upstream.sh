#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"

resolve_shared_repo_root() {
  local common_dir=""
  common_dir="$(git -C "$PROJECT_ROOT" rev-parse --git-common-dir 2>/dev/null || true)"
  if [ -z "$common_dir" ]; then
    printf '%s\n' "$PROJECT_ROOT"
    return
  fi
  cd "$common_dir/.." && pwd
}

resolve_default_repo_file() {
  local relative_path="$1"
  local shared_root="$2"

  if [ -e "$PROJECT_ROOT/$relative_path" ]; then
    printf '%s\n' "$PROJECT_ROOT/$relative_path"
    return
  fi
  printf '%s\n' "$shared_root/$relative_path"
}

SHARED_REPO_ROOT="$(resolve_shared_repo_root)"
CONFIG_FILE="${CONFIG_FILE:-$(resolve_default_repo_file ".kkidc/.env.lighthouse" "$SHARED_REPO_ROOT")}"

TARGET_ENV=""
DRY_RUN="false"

REMOTE_HOST=""
REMOTE_USER=""
REMOTE_PASSWORD=""
TEST_HOSTNAME=""
ENTERPRISE_HOSTNAME=""

TARGET_BASE_URL="${TARGET_BASE_URL:-}"
UPSTREAM_BASE_URL="${UPSTREAM_BASE_URL:-}"
UPSTREAM_USERNAME="${UPSTREAM_USERNAME:-}"
UPSTREAM_PASSWORD="${UPSTREAM_PASSWORD:-}"
TARGET_ROOT_USERNAME="${TARGET_ROOT_USERNAME:-}"
TARGET_ROOT_PASSWORD="${TARGET_ROOT_PASSWORD:-}"
TARGET_SYSTEM_NAME="${TARGET_SYSTEM_NAME:-Aisever 企业版}"
UPSTREAM_TOKEN_NAME_PREFIX="${UPSTREAM_TOKEN_NAME_PREFIX:-enterprise-sync}"
SYNC_LOCALE="${SYNC_LOCALE:-zh-CN}"
DOCS_MANIFEST_PATH="${DOCS_MANIFEST_PATH:-/enterprise-docs/apifox/manifest.json}"

usage() {
  cat <<EOF
Import upstream groups/models/pricing into kkidc test or enterprise via formal APIs.

Usage:
  $(basename "$0") test|enterprise [options]

Options:
  --config PATH                  host config file (default: .kkidc/.env.lighthouse)
  --target-base-url URL          target admin base URL (default from target env)
  --upstream-base-url URL        upstream base URL
  --upstream-username USER       upstream login username
  --upstream-password PASS       upstream login password
  --target-root-username USER    target root username (required for non-dry-run)
  --target-root-password PASS    target root password (required for non-dry-run)
  --target-system-name NAME      target system name (default: Aisever 企业版)
  --upstream-token-prefix NAME   prefix used for upstream relay token names
  --sync-locale LOCALE           official model sync locale (default: zh-CN)
  --dry-run                      print the derived import plan without mutating upstream or target
  --help                         show this help
EOF
}

die() {
  printf '[ERROR] %s\n' "$1" >&2
  exit 1
}

parse_args() {
  if [ $# -lt 1 ]; then
    usage
    exit 1
  fi

  case "$1" in
    test)
      TARGET_ENV="test"
      ;;
    enterprise)
      TARGET_ENV="enterprise"
      ;;
    production|prod)
      die "import does not support production target"
      ;;
    --help|-h)
      usage
      exit 0
      ;;
    *)
      die "unsupported target environment: $1"
      ;;
  esac
  shift

  while [ $# -gt 0 ]; do
    case "$1" in
      --config)
        CONFIG_FILE="$2"
        shift 2
        ;;
      --target-base-url)
        TARGET_BASE_URL="$2"
        shift 2
        ;;
      --upstream-base-url)
        UPSTREAM_BASE_URL="$2"
        shift 2
        ;;
      --upstream-username)
        UPSTREAM_USERNAME="$2"
        shift 2
        ;;
      --upstream-password)
        UPSTREAM_PASSWORD="$2"
        shift 2
        ;;
      --target-root-username)
        TARGET_ROOT_USERNAME="$2"
        shift 2
        ;;
      --target-root-password)
        TARGET_ROOT_PASSWORD="$2"
        shift 2
        ;;
      --target-system-name)
        TARGET_SYSTEM_NAME="$2"
        shift 2
        ;;
      --upstream-token-prefix)
        UPSTREAM_TOKEN_NAME_PREFIX="$2"
        shift 2
        ;;
      --sync-locale)
        SYNC_LOCALE="$2"
        shift 2
        ;;
      --dry-run)
        DRY_RUN="true"
        shift
        ;;
      --help|-h)
        usage
        exit 0
        ;;
      *)
        die "unknown option: $1"
        ;;
    esac
  done
}

load_host_config() {
  [ -f "$CONFIG_FILE" ] || die "missing host config: $CONFIG_FILE"
  set -a
  # shellcheck disable=SC1090
  source "$CONFIG_FILE"
  set +a

  REMOTE_HOST="${IP_1:-${SERVER_IP:-}}"
  REMOTE_USER="${SSH_user:-${SERVER_USER:-}}"
  REMOTE_PASSWORD="${SSH_password:-${SERVER_PASSWORD:-}}"
  TEST_HOSTNAME="${TEST_HOSTNAME:-}"
  ENTERPRISE_HOSTNAME="${ENTERPRISE_HOSTNAME:-}"

  [ -n "$REMOTE_HOST" ] || die "missing IP_1 in $CONFIG_FILE"
  [ -n "$REMOTE_USER" ] || die "missing SSH_user in $CONFIG_FILE"
  [ -n "$REMOTE_PASSWORD" ] || die "missing SSH_password in $CONFIG_FILE"
}

resolve_target_base_url() {
  if [ -n "$TARGET_BASE_URL" ]; then
    return
  fi

  if [ "$TARGET_ENV" = "enterprise" ]; then
    if [ -n "$ENTERPRISE_HOSTNAME" ]; then
      TARGET_BASE_URL="https://${ENTERPRISE_HOSTNAME}"
    else
      TARGET_BASE_URL="http://${REMOTE_HOST}:3002"
    fi
    return
  fi

  if [ -n "$TEST_HOSTNAME" ]; then
    TARGET_BASE_URL="http://${TEST_HOSTNAME}:3001"
    return
  fi
  TARGET_BASE_URL="http://${REMOTE_HOST}:3001"
}

check_dependencies() {
  command -v python3 >/dev/null 2>&1 || die "python3 is required"
  command -v curl >/dev/null 2>&1 || die "curl is required"

  [ -n "$UPSTREAM_BASE_URL" ] || die "--upstream-base-url is required"
  [ -n "$UPSTREAM_USERNAME" ] || die "--upstream-username is required"
  [ -n "$UPSTREAM_PASSWORD" ] || die "--upstream-password is required"

  if [ "$DRY_RUN" != "true" ]; then
    [ -n "$TARGET_ROOT_USERNAME" ] || die "--target-root-username is required for non-dry-run"
    [ -n "$TARGET_ROOT_PASSWORD" ] || die "--target-root-password is required for non-dry-run"
  fi
}

main() {
  parse_args "$@"
  load_host_config
  resolve_target_base_url
  check_dependencies

  export TARGET_ENV DRY_RUN
  export TARGET_BASE_URL UPSTREAM_BASE_URL UPSTREAM_USERNAME UPSTREAM_PASSWORD
  export TARGET_ROOT_USERNAME TARGET_ROOT_PASSWORD TARGET_SYSTEM_NAME
  export UPSTREAM_TOKEN_NAME_PREFIX SYNC_LOCALE DOCS_MANIFEST_PATH

  python3 - <<'PY'
import json
import os
import ssl
import sys
import urllib.error
import urllib.parse
import urllib.request
from http.cookiejar import CookieJar


TARGET_ENV = os.environ["TARGET_ENV"]
DRY_RUN = os.environ["DRY_RUN"] == "true"
TARGET_BASE_URL = os.environ["TARGET_BASE_URL"].rstrip("/")
UPSTREAM_BASE_URL = os.environ["UPSTREAM_BASE_URL"].rstrip("/")
UPSTREAM_USERNAME = os.environ["UPSTREAM_USERNAME"]
UPSTREAM_PASSWORD = os.environ["UPSTREAM_PASSWORD"]
TARGET_ROOT_USERNAME = os.environ.get("TARGET_ROOT_USERNAME", "")
TARGET_ROOT_PASSWORD = os.environ.get("TARGET_ROOT_PASSWORD", "")
TARGET_SYSTEM_NAME = os.environ["TARGET_SYSTEM_NAME"]
UPSTREAM_TOKEN_NAME_PREFIX = os.environ["UPSTREAM_TOKEN_NAME_PREFIX"]
SYNC_LOCALE = os.environ["SYNC_LOCALE"]
DOCS_MANIFEST_PATH = os.environ["DOCS_MANIFEST_PATH"]

OPENAI_FAMILY = "openai"
ANTHROPIC_FAMILY = "anthropic"
GEMINI_FAMILY = "gemini"
JINA_FAMILY = "jina-rerank"
FORBIDDEN_TARGET_HOSTS = {"api.aisever.cn", "newapi.aisever.cn"}

CHANNEL_TYPE_MAP = {
    OPENAI_FAMILY: 1,
    ANTHROPIC_FAMILY: 14,
    GEMINI_FAMILY: 24,
    JINA_FAMILY: 38,
}

FAMILY_ENDPOINT_MAP = {
    "openai": OPENAI_FAMILY,
    "openai-response": OPENAI_FAMILY,
    "openai-video": OPENAI_FAMILY,
    "image-generation": OPENAI_FAMILY,
    "anthropic": ANTHROPIC_FAMILY,
    "gemini": GEMINI_FAMILY,
    "jina-rerank": JINA_FAMILY,
}


def fail(msg: str) -> None:
    print(f"[ERROR] {msg}", file=sys.stderr)
    sys.exit(1)


def make_ssl_context():
    return ssl.create_default_context()


class JsonSession:
    def __init__(self):
        self.cookie_jar = CookieJar()
        self.ssl_context = make_ssl_context()
        self.default_headers = {"Accept": "application/json"}
        self.opener = urllib.request.build_opener(
            urllib.request.HTTPCookieProcessor(self.cookie_jar),
            urllib.request.HTTPSHandler(context=self.ssl_context),
        )

    def set_default_header(self, key: str, value: str):
        self.default_headers[key] = value

    def request(self, method: str, url: str, data=None, headers=None):
        payload = None
        request_headers = dict(self.default_headers)
        if headers:
            request_headers.update(headers)
        if data is not None:
            payload = json.dumps(data, ensure_ascii=False).encode("utf-8")
            request_headers.setdefault("Content-Type", "application/json")
        req = urllib.request.Request(url, data=payload, headers=request_headers, method=method)
        try:
            with self.opener.open(req, timeout=30) as resp:
                body = resp.read().decode("utf-8")
                if not body:
                    return {}
                return json.loads(body)
        except urllib.error.HTTPError as exc:
            body = exc.read().decode("utf-8", errors="replace")
            try:
                parsed = json.loads(body) if body else {}
            except Exception:
                parsed = {"success": False, "message": body or str(exc)}
            raise RuntimeError(f"{method} {url} failed: {parsed}")
        except urllib.error.URLError as exc:
            raise RuntimeError(f"{method} {url} failed: {exc}") from exc


def ensure_success(obj, action: str):
    if isinstance(obj, dict) and obj.get("success") is True:
        return obj
    message = None
    if isinstance(obj, dict):
        message = obj.get("message") or obj.get("error") or obj
    if not message:
        message = obj
    raise RuntimeError(f"{action} failed: {message}")


def sanitize_name(value: str) -> str:
    safe = []
    for ch in value:
        if ch.isalnum() or ch in ("-", "_"):
            safe.append(ch)
        else:
            safe.append("-")
    out = "".join(safe).strip("-")
    return out or "group"


def deterministic_channel_name(group: str, family: str) -> str:
    return f"aigcbest::{group}::{family}"


def deterministic_channel_tag() -> str:
    return "enterprise-upstream-aigcbest"


def deterministic_token_name(group: str) -> str:
    base = f"{UPSTREAM_TOKEN_NAME_PREFIX}-{TARGET_ENV}-{sanitize_name(group)}"
    return base[:50]


def validate_target_base_url() -> None:
    parsed = urllib.parse.urlparse(TARGET_BASE_URL)
    if parsed.scheme not in ("http", "https"):
        fail(f"unsupported target base url scheme: {TARGET_BASE_URL}")

    hostname = parsed.hostname or ""
    if hostname in FORBIDDEN_TARGET_HOSTS:
        fail(f"refusing to target personal production host: {hostname}")

    if TARGET_ENV == "enterprise" and parsed.port == 3001:
        fail(f"target base url looks like test environment, not enterprise: {TARGET_BASE_URL}")
    if TARGET_ENV == "test" and parsed.port == 3002:
        fail(f"target base url looks like enterprise environment, not test: {TARGET_BASE_URL}")


def build_group_descriptions(group_payload: dict) -> dict:
    usable = {}
    for group_name, meta in group_payload.items():
        if not isinstance(meta, dict):
            usable[group_name] = group_name
            continue
        usable[group_name] = meta.get("desc") or group_name
    if "auto" not in usable:
        usable["auto"] = "自动选择可用上游分组"
    return usable


def get_real_groups(group_payload: dict) -> list:
    return [group_name for group_name in group_payload.keys() if group_name != "auto"]


def build_pricing_payloads(pricing_data: dict, usable_groups: list):
    group_ratio = dict(pricing_data.get("group_ratio") or {})
    raw_auto_groups = list(pricing_data.get("auto_groups") or [])
    items = list(pricing_data.get("data") or [])

    usable_group_set = set(usable_groups)
    auto_groups = [group_name for group_name in raw_auto_groups if group_name in usable_group_set]
    filtered_items = []
    missing_endpoint_models = {}
    unsupported_endpoint_models = {}
    for item in items:
        enabled_groups = set(item.get("enable_groups") or [])
        if enabled_groups & usable_group_set:
            filtered_items.append(item)

    model_ratio = {}
    completion_ratio = {}
    model_price = {}
    group_model_names = {group_name: set() for group_name in usable_groups}
    group_family_models = {group_name: {} for group_name in usable_groups}

    for item in filtered_items:
        model_name = item.get("model_name")
        if not model_name:
            continue
        relevant_groups = sorted(group_name for group_name in (item.get("enable_groups") or []) if group_name in usable_group_set)
        for group_name in relevant_groups:
            group_model_names[group_name].add(model_name)

        quota_type = int(item.get("quota_type") or 0)
        if quota_type == 1:
            model_price[model_name] = item.get("model_price", 0)
        else:
            model_ratio[model_name] = item.get("model_ratio", 0)
            completion_ratio[model_name] = item.get("completion_ratio", 0)

        supported_endpoints = list(item.get("supported_endpoint_types") or [])
        if not supported_endpoints:
            missing_endpoint_models[model_name] = relevant_groups
            continue

        unknown_endpoints = sorted({endpoint for endpoint in supported_endpoints if endpoint not in FAMILY_ENDPOINT_MAP})
        if unknown_endpoints:
            unsupported_endpoint_models[model_name] = {
                "groups": relevant_groups,
                "endpoint_types": unknown_endpoints,
            }
            continue

        families = set()
        for endpoint in supported_endpoints:
            families.add(FAMILY_ENDPOINT_MAP[endpoint])

        for group_name in relevant_groups:
            family_map = group_family_models[group_name]
            for family in families:
                family_map.setdefault(family, set()).add(model_name)

    if missing_endpoint_models:
        raise RuntimeError(
            "upstream pricing contains models without supported_endpoint_types: "
            + json.dumps(missing_endpoint_models, ensure_ascii=False)
        )

    if unsupported_endpoint_models:
        raise RuntimeError(
            "upstream pricing contains unsupported endpoint families: "
            + json.dumps(unsupported_endpoint_models, ensure_ascii=False)
        )

    channel_plan = []
    for group_name in usable_groups:
        family_map = group_family_models[group_name]
        for family, models in sorted(family_map.items()):
            sorted_models = sorted(models)
            if not sorted_models:
                continue
            channel_plan.append(
                {
                    "group": group_name,
                    "family": family,
                    "type": CHANNEL_TYPE_MAP[family],
                    "name": deterministic_channel_name(group_name, family),
                    "models": sorted_models,
                    "model_count": len(sorted_models),
                    "test_model": sorted_models[0],
                    "settings": {
                        "upstream_model_update_check_enabled": True,
                        "upstream_model_update_auto_sync_enabled": True,
                        "upstream_model_update_ignored_models": [],
                    },
                }
            )

    return {
        "group_ratio": {k: group_ratio[k] for k in usable_groups if k in group_ratio},
        "auto_groups": auto_groups,
        "group_model_counts": {group_name: len(group_model_names[group_name]) for group_name in usable_groups},
        "model_ratio": model_ratio,
        "completion_ratio": completion_ratio,
        "model_price": model_price,
        "channel_plan": channel_plan,
    }


def login_upstream():
    client = JsonSession()
    res = client.request(
        "POST",
        f"{UPSTREAM_BASE_URL}/api/user/login",
        {"username": UPSTREAM_USERNAME, "password": UPSTREAM_PASSWORD},
    )
    ensure_success(res, "upstream login")
    user_id = ((res.get("data") or {}).get("id"))
    if not user_id:
        raise RuntimeError("upstream login response missing user id")
    return client, int(user_id)


def fetch_upstream_data(client: JsonSession, user_id: int):
    headers = {"New-Api-User": str(user_id)}
    groups = ensure_success(
        client.request("GET", f"{UPSTREAM_BASE_URL}/api/user/self/groups", headers=headers),
        "fetch upstream groups",
    ).get("data") or {}
    models = ensure_success(
        client.request("GET", f"{UPSTREAM_BASE_URL}/api/user/models", headers=headers),
        "fetch upstream models",
    ).get("data") or []
    pricing = ensure_success(
        client.request("GET", f"{UPSTREAM_BASE_URL}/api/pricing"),
        "fetch upstream pricing",
    )
    return groups, models, pricing


def fetch_existing_tokens(client: JsonSession, user_id: int):
    headers = {"New-Api-User": str(user_id)}
    res = ensure_success(
        client.request("GET", f"{UPSTREAM_BASE_URL}/api/token/?p=1&page_size=200", headers=headers),
        "list upstream tokens",
    )
    return list(((res.get("data") or {}).get("items") or []))


def create_or_reuse_group_tokens(client: JsonSession, user_id: int, groups: list):
    headers = {"New-Api-User": str(user_id)}
    existing = fetch_existing_tokens(client, user_id)
    existing_by_name = {item.get("name"): item for item in existing}
    tokens = []
    created = 0
    reused = 0

    for group_name in groups:
        token_name = deterministic_token_name(group_name)
        token = existing_by_name.get(token_name)
        if token is None:
            ensure_success(
                client.request(
                    "POST",
                    f"{UPSTREAM_BASE_URL}/api/token/",
                    {
                        "name": token_name,
                        "remain_quota": 500000,
                        "expired_time": -1,
                        "unlimited_quota": True,
                        "group": group_name,
                    },
                    headers=headers,
                ),
                f"create upstream token for {group_name}",
            )
            created += 1
            existing = fetch_existing_tokens(client, user_id)
            existing_by_name = {item.get("name"): item for item in existing}
            token = existing_by_name.get(token_name)
        else:
            reused += 1

        if token is None:
            raise RuntimeError(f"failed to locate upstream token after create: {token_name}")

        key_resp = ensure_success(
            client.request("POST", f"{UPSTREAM_BASE_URL}/api/token/{token['id']}/key", headers=headers),
            f"fetch upstream token key for {group_name}",
        )
        token_key = ((key_resp.get("data") or {}).get("key"))
        if not token_key:
            raise RuntimeError(f"upstream token key response missing key for {group_name}")

        models_resp = client.request(
            "GET",
            f"{UPSTREAM_BASE_URL}/v1/models",
            headers={"Authorization": f"Bearer {token_key}"},
        )
        if not isinstance(models_resp, dict) or not isinstance(models_resp.get("data"), list):
            raise RuntimeError(f"upstream token verification failed for {group_name}: {models_resp}")
        verified_models = sorted(
            {
                model.get("id")
                for model in (models_resp.get("data") or [])
                if isinstance(model, dict) and model.get("id")
            }
        )

        tokens.append(
            {
                "group": group_name,
                "token_name": token_name,
                "token_id": token["id"],
                "key": token_key,
                "verified_model_count": len(verified_models),
                "verified_models": verified_models,
            }
        )

    return tokens, created, reused


def public_target_setup_status():
    client = JsonSession()
    try:
        res = client.request("GET", f"{TARGET_BASE_URL}/api/setup")
        if isinstance(res, dict):
            return res.get("data") or {}
    except Exception:
        return {}
    return {}


def ensure_target_setup_and_login():
    client = JsonSession()
    setup_resp = client.request("GET", f"{TARGET_BASE_URL}/api/setup")
    setup_data = (setup_resp.get("data") or {}) if isinstance(setup_resp, dict) else {}
    if not setup_data.get("status"):
        ensure_success(
            client.request(
                "POST",
                f"{TARGET_BASE_URL}/api/setup",
                {
                    "username": TARGET_ROOT_USERNAME,
                    "password": TARGET_ROOT_PASSWORD,
                    "confirmPassword": TARGET_ROOT_PASSWORD,
                    "SelfUseModeEnabled": False,
                    "DemoSiteEnabled": False,
                },
            ),
            "target setup",
        )

    login_res = ensure_success(
        client.request(
            "POST",
            f"{TARGET_BASE_URL}/api/user/login",
            {"username": TARGET_ROOT_USERNAME, "password": TARGET_ROOT_PASSWORD},
        ),
        "target root login",
    )
    user_id = ((login_res.get("data") or {}).get("id"))
    if not user_id:
        raise RuntimeError("target root login response missing user id")
    client.set_default_header("New-Api-User", str(int(user_id)))
    return client


def update_option(client: JsonSession, key: str, value):
    ensure_success(
        client.request("PUT", f"{TARGET_BASE_URL}/api/option/", {"key": key, "value": value}),
        f"update option {key}",
    )


def fetch_existing_channels(client: JsonSession):
    res = ensure_success(
        client.request("GET", f"{TARGET_BASE_URL}/api/channel/?p=1&page_size=5000&id_sort=true"),
        "list target channels",
    )
    data = res.get("data") or {}
    items = data.get("items") or []
    return {item.get("name"): item for item in items if item.get("name")}


def upsert_channels(client: JsonSession, channel_plan: list, tokens_by_group: dict):
    existing = fetch_existing_channels(client)
    created = 0
    updated = 0

    for plan in channel_plan:
        token = tokens_by_group[plan["group"]]
        payload_channel = {
            "name": plan["name"],
            "type": plan["type"],
            "key": token["key"],
            "status": 1,
            "base_url": UPSTREAM_BASE_URL,
            "models": ",".join(plan["models"]),
            "group": plan["group"],
            "test_model": plan["test_model"],
            "tag": deterministic_channel_tag(),
            "settings": json.dumps(plan["settings"], ensure_ascii=False),
        }

        existing_channel = existing.get(plan["name"])
        if existing_channel:
            payload_channel["id"] = existing_channel["id"]
            ensure_success(
                client.request("PUT", f"{TARGET_BASE_URL}/api/channel/", payload_channel),
                f"update target channel {plan['name']}",
            )
            updated += 1
            continue

        ensure_success(
            client.request(
                "POST",
                f"{TARGET_BASE_URL}/api/channel/",
                {"mode": "single", "channel": payload_channel},
            ),
            f"create target channel {plan['name']}",
        )
        created += 1

    return created, updated


def build_live_channel_plan(channel_plan: list, tokens_by_group: dict):
    live_plan = []
    dropped_models = {}
    for plan in channel_plan:
        token = tokens_by_group[plan["group"]]
        verified_models = set(token.get("verified_models") or [])
        live_models = [model_name for model_name in plan["models"] if model_name in verified_models]
        missing_models = [model_name for model_name in plan["models"] if model_name not in verified_models]
        if missing_models:
            dropped_models[plan["name"]] = {
                "group": plan["group"],
                "family": plan["family"],
                "dropped_model_count": len(missing_models),
            }
        if not live_models:
            continue

        next_plan = dict(plan)
        next_plan["models"] = live_models
        next_plan["model_count"] = len(live_models)
        next_plan["test_model"] = live_models[0]
        live_plan.append(next_plan)
    return live_plan, dropped_models


def repair_target_abilities(client: JsonSession):
    return ensure_success(
        client.request("POST", f"{TARGET_BASE_URL}/api/channel/fix"),
        "repair target abilities",
    ).get("data") or {}


def run_builtin_model_sync(client: JsonSession):
    missing = ensure_success(
        client.request("GET", f"{TARGET_BASE_URL}/api/models/missing"),
        "fetch missing target models",
    ).get("data") or []

    sync_result = {}
    if missing:
        sync_result = ensure_success(
            client.request(
                "POST",
                f"{TARGET_BASE_URL}/api/models/sync_upstream",
                {"locale": SYNC_LOCALE},
            ),
            "sync official model metadata",
        ).get("data") or {}
    missing_after = ensure_success(
        client.request("GET", f"{TARGET_BASE_URL}/api/models/missing"),
        "fetch missing target models after official sync",
    ).get("data") or []
    return len(missing), len(missing_after), sync_result


validate_target_base_url()
upstream_client, upstream_user_id = login_upstream()
upstream_groups_payload, upstream_models, upstream_pricing = fetch_upstream_data(upstream_client, upstream_user_id)

real_groups = get_real_groups(upstream_groups_payload)
usable_group_descriptions = build_group_descriptions(upstream_groups_payload)
pricing_payloads = build_pricing_payloads(upstream_pricing, real_groups)
target_setup_status = public_target_setup_status()

summary = {
    "action": "import-dry-run" if DRY_RUN else "import",
    "target_env": TARGET_ENV,
    "target_base_url": TARGET_BASE_URL,
    "upstream_base_url": UPSTREAM_BASE_URL,
    "upstream_user_id": upstream_user_id,
    "usable_groups": list(usable_group_descriptions.keys()),
    "real_groups": real_groups,
    "user_model_count": len(upstream_models),
    "auto_groups": pricing_payloads["auto_groups"],
    "group_model_counts": pricing_payloads["group_model_counts"],
    "group_ratio_keys": sorted(pricing_payloads["group_ratio"].keys()),
    "model_ratio_count": len(pricing_payloads["model_ratio"]),
    "completion_ratio_count": len(pricing_payloads["completion_ratio"]),
    "model_price_count": len(pricing_payloads["model_price"]),
    "planned_channel_count": len(pricing_payloads["channel_plan"]),
    "planned_channels": [
        {
            "name": item["name"],
            "group": item["group"],
            "family": item["family"],
            "type": item["type"],
            "model_count": item["model_count"],
            "test_model": item["test_model"],
        }
        for item in pricing_payloads["channel_plan"]
    ],
    "public_target_setup_status": target_setup_status,
    "target_docs_manifest_path": DOCS_MANIFEST_PATH,
}

if DRY_RUN:
    print(json.dumps(summary, ensure_ascii=False, indent=2))
    sys.exit(0)

tokens, created_token_count, reused_token_count = create_or_reuse_group_tokens(upstream_client, upstream_user_id, real_groups)
tokens_by_group = {item["group"]: item for item in tokens}
live_channel_plan, dropped_models = build_live_channel_plan(pricing_payloads["channel_plan"], tokens_by_group)

target_client = ensure_target_setup_and_login()

update_option(target_client, "SystemName", TARGET_SYSTEM_NAME)
update_option(target_client, "ServerAddress", TARGET_BASE_URL)
update_option(target_client, "general_setting.docs_manifest_path", DOCS_MANIFEST_PATH)
update_option(target_client, "UserUsableGroups", json.dumps(usable_group_descriptions, ensure_ascii=False))
update_option(target_client, "AutoGroups", json.dumps(pricing_payloads["auto_groups"], ensure_ascii=False))
update_option(target_client, "GroupRatio", json.dumps(pricing_payloads["group_ratio"], ensure_ascii=False))
update_option(target_client, "GroupGroupRatio", "{}")
update_option(target_client, "ModelRatio", json.dumps(pricing_payloads["model_ratio"], ensure_ascii=False))
update_option(target_client, "CompletionRatio", json.dumps(pricing_payloads["completion_ratio"], ensure_ascii=False))
update_option(target_client, "ModelPrice", json.dumps(pricing_payloads["model_price"], ensure_ascii=False))

created_channels, updated_channels = upsert_channels(target_client, live_channel_plan, tokens_by_group)
ability_fix_result = repair_target_abilities(target_client)
missing_before_sync, missing_after_sync, model_sync_result = run_builtin_model_sync(target_client)

summary.update(
    {
        "upstream_tokens_created": created_token_count,
        "upstream_tokens_reused": reused_token_count,
        "live_channel_count": len(live_channel_plan),
        "target_channels_skipped_due_to_missing_live_models": dropped_models,
        "upstream_tokens": [
            {
                "group": item["group"],
                "token_name": item["token_name"],
                "token_id": item["token_id"],
                "verified_model_count": item["verified_model_count"],
            }
            for item in tokens
        ],
        "target_channels_created": created_channels,
        "target_channels_updated": updated_channels,
        "target_ability_fix": ability_fix_result,
        "target_missing_models_before_official_sync": missing_before_sync,
        "target_missing_models_after_official_sync": missing_after_sync,
        "target_official_model_sync": model_sync_result,
    }
)

print(json.dumps(summary, ensure_ascii=False, indent=2))
PY
}

main "$@"

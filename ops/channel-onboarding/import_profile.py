#!/usr/bin/env python3
import argparse
import json
import ssl
import urllib.error
import urllib.request
from http.cookiejar import CookieJar
from pathlib import Path
from typing import Any, Dict

from dry_run import build_result, load_yaml, normalize_target_environment


ALLOWED_TARGET_ENVIRONMENTS = {"test", "enterprise"}


def fail(message: str) -> None:
    raise SystemExit(message)


class JsonSession:
    def __init__(self):
        self.cookie_jar = CookieJar()
        self.ssl_context = ssl.create_default_context()
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
                return json.loads(body) if body else {}
        except urllib.error.HTTPError as exc:
            body = exc.read().decode("utf-8", errors="replace")
            raise RuntimeError(f"{method} {url} failed: {body}") from exc
        except urllib.error.URLError as exc:
            raise RuntimeError(f"{method} {url} failed: {exc}") from exc


def ensure_success(obj, action: str):
    if isinstance(obj, dict) and obj.get("success") is True:
        return obj
    raise RuntimeError(f"{action} failed: {obj}")


def parse_channel_key_args(items: list[str]) -> Dict[str, str]:
    parsed = {}
    for item in items:
        if "=" not in item:
            fail(f"invalid --channel-key value: {item!r}, expected family=key")
        family, key = item.split("=", 1)
        family = family.strip()
        key = key.strip()
        if not family or not key:
            fail(f"invalid --channel-key value: {item!r}, expected non-empty family and key")
        parsed[family] = key
    return parsed


def merge_dict(base: dict, incoming: dict) -> dict:
    merged = dict(base)
    merged.update(incoming)
    return merged


def join_url(base_url: str, suffix: str) -> str:
    base = str(base_url or "").rstrip("/")
    extra = str(suffix or "").strip()
    if not extra:
        return base
    if extra.startswith("http://") or extra.startswith("https://"):
        return extra.rstrip("/")
    return f"{base}/{extra.lstrip('/')}"


def merge_user_groups(existing_raw: str, new_groups: list[dict]) -> tuple[dict, list[str]]:
    existing = json.loads(existing_raw or "{}")
    preserved = sorted(existing.keys())
    merged = dict(existing)
    for item in new_groups:
        merged[item["target"]] = item["description"]
    return merged, preserved


def merge_group_ratio(existing_raw: str, new_groups: list[dict]) -> dict:
    existing = json.loads(existing_raw or "{}")
    merged = dict(existing)
    for item in new_groups:
        merged.setdefault(item["target"], 1)
    return merged


def merge_model_map(existing_raw: str, planned_channels: list[dict]) -> dict:
    existing = json.loads(existing_raw or "{}")
    merged = dict(existing)
    for channel in planned_channels:
        for model_name in channel["models"]:
            merged.setdefault(model_name, 1)
    return merged


def get_option_map(client: JsonSession, target_base_url: str) -> dict:
    res = ensure_success(client.request("GET", f"{target_base_url}/api/option/"), "get target options")
    items = res.get("data") or []
    return {item["key"]: item["value"] for item in items}


def ensure_target_setup_and_login(target_base_url: str, username: str, password: str) -> JsonSession:
    client = JsonSession()
    setup_resp = client.request("GET", f"{target_base_url}/api/setup")
    setup_data = (setup_resp.get("data") or {}) if isinstance(setup_resp, dict) else {}
    if not setup_data.get("status"):
        ensure_success(
            client.request(
                "POST",
                f"{target_base_url}/api/setup",
                {
                    "username": username,
                    "password": password,
                    "confirmPassword": password,
                    "SelfUseModeEnabled": False,
                    "DemoSiteEnabled": False,
                },
            ),
            "target setup",
        )

    login_resp = ensure_success(
        client.request("POST", f"{target_base_url}/api/user/login", {"username": username, "password": password}),
        "target login",
    )
    user_id = (login_resp.get("data") or {}).get("id")
    if not user_id:
        fail("target login response missing user id")
    client.set_default_header("New-Api-User", str(int(user_id)))
    return client


def update_option(client: JsonSession, target_base_url: str, key: str, value):
    ensure_success(client.request("PUT", f"{target_base_url}/api/option/", {"key": key, "value": value}), f"update option {key}")


def fetch_existing_channels(client: JsonSession, target_base_url: str) -> dict:
    res = ensure_success(client.request("GET", f"{target_base_url}/api/channel/?p=1&page_size=5000&id_sort=true"), "list target channels")
    items = (res.get("data") or {}).get("items") or []
    return {item.get("name"): item for item in items if item.get("name")}


def authorization_header_for_family(family: str, channel_key: str) -> str:
    if family == "vidu" and not channel_key.startswith("sk-"):
        return f"Token {channel_key}"
    return f"Bearer {channel_key}"


def render_probe_value(value: Any, test_model: str) -> Any:
    if isinstance(value, str):
        return value.replace("{test_model}", test_model)
    if isinstance(value, list):
        return [render_probe_value(item, test_model) for item in value]
    if isinstance(value, dict):
        return {key: render_probe_value(item, test_model) for key, item in value.items()}
    return value


def build_probe_request_body(plan: dict, family_config: dict) -> dict:
    verification = family_config.get("verification") or {}
    template = verification.get("probe_request_body")
    if template:
        rendered = render_probe_value(template, plan["test_model"])
        if not isinstance(rendered, dict):
            fail(f"probe_request_body for family {plan['family']} must render to an object")
        return rendered
    return {"model": plan["test_model"], "prompt": "probe"}


def probe_family_channels(planned_channels: list[dict], channel_keys: Dict[str, str]) -> dict:
    passed = 0
    failed = []
    for plan in planned_channels:
        family = plan["family"]
        channel_key = channel_keys.get(family)
        if not channel_key:
            failed.append({"family": family, "reason": "missing channel key"})
            continue
        probe_url = plan.get("probe_url")
        if not probe_url:
            failed.append({"family": family, "reason": "missing probe_url"})
            continue
        request_body = build_probe_request_body(plan, plan.get("family_config") or {})
        req = urllib.request.Request(
            probe_url,
            data=json.dumps(request_body, ensure_ascii=False).encode("utf-8"),
            headers={
                "Content-Type": "application/json",
                "Accept": "application/json",
                "Authorization": authorization_header_for_family(family, channel_key),
            },
            method="POST",
        )
        try:
            with urllib.request.urlopen(req, timeout=15) as resp:
                status = resp.status
                content_type = resp.headers.get("Content-Type", "")
                body = resp.read().decode("utf-8", errors="replace")
        except urllib.error.HTTPError as exc:
            status = exc.code
            content_type = exc.headers.get("Content-Type", "") if exc.headers else ""
            body = exc.read().decode("utf-8", errors="replace")
        except urllib.error.URLError as exc:
            failed.append({"family": family, "reason": str(exc)})
            continue

        if "json" not in content_type.lower():
            failed.append({"family": family, "reason": f"probe returned non-json content-type {content_type or 'unknown'}"})
            continue
        if status == 401:
            failed.append({"family": family, "reason": "probe unauthorized"})
            continue
        if status in (200, 400, 403, 405, 422, 429, 503):
            passed += 1
            continue
        failed.append({"family": family, "reason": f"unexpected probe status {status}", "body": body[:200]})
    return {"passed": passed, "failed": failed}


def upsert_channels(client: JsonSession, target_base_url: str, planned_channels: list[dict], channel_keys: Dict[str, str]) -> tuple[int, int, int]:
    existing = fetch_existing_channels(client, target_base_url)
    preserved_existing = len(existing)
    created = 0
    updated = 0
    for plan in planned_channels:
        channel_key = channel_keys.get(plan["family"], "MANUAL_REQUIRED")
        payload_channel = {
            "name": plan["name"],
            "type": plan["type"],
            "key": channel_key,
            "status": 1,
            "base_url": plan.get("base_url", ""),
            "models": ",".join(plan["models"]),
            "group": plan["group"],
            "test_model": plan["test_model"],
            "tag": plan["tag"],
            "settings": json.dumps(plan["settings"], ensure_ascii=False),
        }

        existing_channel = existing.get(plan["name"])
        if existing_channel:
            payload_channel["id"] = existing_channel["id"]
            ensure_success(client.request("PUT", f"{target_base_url}/api/channel/", payload_channel), f"update channel {plan['name']}")
            updated += 1
        else:
            ensure_success(client.request("POST", f"{target_base_url}/api/channel/", {"mode": "single", "channel": payload_channel}), f"create channel {plan['name']}")
            created += 1
    return created, updated, preserved_existing


def main() -> None:
    parser = argparse.ArgumentParser(description="Import a channel onboarding profile into test or enterprise environments.")
    parser.add_argument("--profile", required=True)
    parser.add_argument("--target-base-url", required=True)
    parser.add_argument("--target-root-username", required=True)
    parser.add_argument("--target-root-password", required=True)
    parser.add_argument("--target-environment", required=True)
    parser.add_argument("--probe-upstream", action="store_true")
    parser.add_argument("--require-channel-keys", action="store_true")
    parser.add_argument("--channel-key", action="append", default=[], help="family=key")
    args = parser.parse_args()

    profile_path = Path(args.profile).resolve()
    profile = load_yaml(profile_path)
    environment = normalize_target_environment(profile, args.target_environment)
    if environment not in ALLOWED_TARGET_ENVIRONMENTS:
        fail(f"production writes are blocked for this importer; allowed environments: {', '.join(sorted(ALLOWED_TARGET_ENVIRONMENTS))}")

    dry_run = build_result(profile, profile_path, environment)
    channel_keys = parse_channel_key_args(args.channel_key)

    if args.require_channel_keys:
        missing = [family for family in dry_run["families"] if family not in channel_keys]
        if missing:
            fail(f"missing required channel keys for families: {', '.join(sorted(missing))}")

    source_base_url = dry_run["source"]["base_url"].rstrip("/")
    family_config_map = {item["family"]: item for item in profile.get("families", [])}
    for plan in dry_run["planned_channels"]:
        family_config = family_config_map.get(plan["family"]) or {}
        plan["family_config"] = family_config
        path_template = family_config.get("path_template") or {}
        verification = family_config.get("verification") or {}
        probe_path = verification.get("probe_endpoint") or path_template.get("model_probe") or path_template.get("create") or ""
        if probe_path:
            plan["probe_url"] = f"{source_base_url}{probe_path}"
        plan["base_url"] = plan.get("base_url") or join_url(source_base_url, family_config.get("channel_base_url_suffix", ""))

    probe_summary = None
    if args.probe_upstream:
        probe_summary = probe_family_channels(dry_run["planned_channels"], channel_keys)
        if probe_summary["failed"]:
            fail(f"upstream probe failed: {json.dumps(probe_summary['failed'], ensure_ascii=False)}")

    client = ensure_target_setup_and_login(args.target_base_url.rstrip("/"), args.target_root_username, args.target_root_password)
    target_base_url = args.target_base_url.rstrip("/")
    option_map = get_option_map(client, target_base_url)

    merged_user_groups, preserved_existing_groups = merge_user_groups(option_map.get("UserUsableGroups", "{}"), dry_run["groups"])
    merged_group_ratio = merge_group_ratio(option_map.get("GroupRatio", "{}"), dry_run["groups"])
    merged_model_ratio = merge_model_map(option_map.get("ModelRatio", "{}"), dry_run["planned_channels"])
    merged_completion_ratio = merge_model_map(option_map.get("CompletionRatio", "{}"), dry_run["planned_channels"])
    merged_model_price = merge_dict(json.loads(option_map.get("ModelPrice", "{}") or "{}"), dry_run["pricing"].get("model_price") or {})

    update_option(client, target_base_url, "UserUsableGroups", json.dumps(merged_user_groups, ensure_ascii=False))
    update_option(client, target_base_url, "GroupRatio", json.dumps(merged_group_ratio, ensure_ascii=False))
    update_option(client, target_base_url, "ModelRatio", json.dumps(merged_model_ratio, ensure_ascii=False))
    update_option(client, target_base_url, "CompletionRatio", json.dumps(merged_completion_ratio, ensure_ascii=False))
    update_option(client, target_base_url, "ModelPrice", json.dumps(merged_model_price, ensure_ascii=False))

    created_channels, updated_channels, preserved_existing_channel_count = upsert_channels(client, target_base_url, dry_run["planned_channels"], channel_keys)
    ability_fix = ensure_success(client.request("POST", f"{target_base_url}/api/channel/fix"), "repair target abilities").get("data") or {}

    summary = {
        "success": True,
        "target_environment": environment,
        "target_base_url": target_base_url,
        "profile": dry_run["manifest_path"],
        "target_channels_created": created_channels,
        "target_channels_updated": updated_channels,
        "preserved_existing_channel_count": preserved_existing_channel_count,
        "preserved_existing_groups": preserved_existing_groups,
        "merged_group_ratio_keys": sorted(merged_group_ratio.keys()),
        "merged_model_ratio_count": len(merged_model_ratio),
        "merged_completion_ratio_count": len(merged_completion_ratio),
        "merged_model_price_count": len(merged_model_price),
        "target_ability_fix": ability_fix,
        "planned_channels": dry_run["planned_channels"],
        "provided_channel_key_families": sorted(channel_keys.keys()),
        "probe_summary": probe_summary or {"passed": 0, "failed": []},
        "risks": dry_run["risks"],
    }
    print(json.dumps(summary, ensure_ascii=False, indent=2))


if __name__ == "__main__":
    main()

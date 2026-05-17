#!/usr/bin/env python3
import argparse
import json
import sys
from pathlib import Path
from typing import List, Optional

import yaml


REQUIRED_KEYS = [
    "version",
    "source",
    "target",
    "families",
    "groups",
    "pricing",
    "channels",
    "verification",
]

REQUIRED_FAMILIES = {"kling", "vidu", "doubao-video"}
REQUIRED_EXPLICIT_MODELS = {"doubao-video"}


def fail(message: str) -> None:
    print(message, file=sys.stderr)
    raise SystemExit(1)


def load_yaml(path: Path) -> dict:
    with path.open("r", encoding="utf-8") as f:
        data = yaml.safe_load(f)
    if not isinstance(data, dict):
        fail(f"invalid manifest/profile: expected object at top level, got {type(data).__name__}")
    return data


def require_keys(data: dict, where: str) -> None:
    for key in REQUIRED_KEYS:
        if key not in data:
            fail(f"missing required key {key!r} in {where}")


def normalize_target_environment(profile: dict, override: Optional[str]) -> str:
    environment = override or profile["target"].get("environment")
    if not environment:
        fail("target.environment is required")
    return str(environment).strip()


def normalize_group_items(profile: dict) -> List[dict]:
    groups = profile["groups"]
    if groups.get("mode") != "explicit":
        fail("only explicit groups mode is supported in dry-run v1")
    items = groups.get("items") or []
    if not items:
        fail("groups.items must not be empty")
    normalized = []
    for item in items:
        source = str(item.get("source", "")).strip()
        target = str(item.get("target", "")).strip()
        description = str(item.get("description", target or source)).strip()
        if not source or not target:
            fail("every group item must include non-empty source and target")
        normalized.append({"source": source, "target": target, "description": description})
    return normalized


def normalize_families(profile: dict) -> List[dict]:
    families = profile.get("families") or []
    if not families:
        fail("families must not be empty")

    normalized = []
    seen = set()
    for item in families:
        family = str(item.get("family", "")).strip()
        if not family:
            fail("every family item must include family")
        if family in seen:
            fail(f"duplicate family definition: {family}")
        seen.add(family)

        model_source = item.get("model_source") or {}
        if model_source.get("mode") != "explicit":
            fail(f"family {family} currently requires explicit model_source.mode")
        models = [str(model).strip() for model in (model_source.get("models") or []) if str(model).strip()]
        ordered_models = list(dict.fromkeys(models))
        if family in REQUIRED_EXPLICIT_MODELS and not models:
            fail(f"family {family} requires explicit non-empty models list")
        if not models:
            fail(f"family {family} has no models defined")

        normalized.append(
            {
                "family": family,
                "channel_type": int(item["channel_type"]),
                "adaptor": item.get("adaptor", ""),
                "channel_base_url_suffix": str(item.get("channel_base_url_suffix", "") or "").strip(),
                "models": ordered_models,
                "verification": item.get("verification") or {},
            }
        )

    missing = sorted(REQUIRED_FAMILIES - seen)
    if missing:
        fail(f"missing required families for video aggregator profile: {', '.join(missing)}")
    return sorted(normalized, key=lambda item: item["family"])


def render_pattern(pattern: str, source_slug: str, family: str, group: str) -> str:
    return (
        pattern.replace("{source_slug}", source_slug)
        .replace("{family}", family)
        .replace("{group}", group)
    )


def join_url(base_url: str, suffix: str) -> str:
    base = str(base_url or "").rstrip("/")
    extra = str(suffix or "").strip()
    if not extra:
        return base
    if extra.startswith("http://") or extra.startswith("https://"):
        return extra.rstrip("/")
    return f"{base}/{extra.lstrip('/')}"


def build_planned_channels(profile: dict, families: List[dict], groups: List[dict], environment: str) -> List[dict]:
    source_slug = profile["source"]["slug"]
    source_base_url = profile["source"]["base_url"]
    naming_pattern = profile["channels"]["naming"]["pattern"]
    tag_pattern = profile["channels"]["tag"]["pattern"]
    settings = profile["channels"].get("settings") or {}

    planned = []
    for family in families:
        for group in groups:
            models = family["models"]
            channel_name = render_pattern(naming_pattern, source_slug, family["family"], group["target"])
            channel_tag = render_pattern(tag_pattern, source_slug, family["family"], group["target"])
            planned.append(
                {
                    "name": channel_name,
                    "family": family["family"],
                    "type": family["channel_type"],
                    "group": group["target"],
                    "models": models,
                    "model_count": len(models),
                    "test_model": models[0],
                    "tag": channel_tag,
                    "environment": environment,
                    "adaptor": family["adaptor"],
                    "base_url": join_url(source_base_url, family["channel_base_url_suffix"]),
                    "settings": settings,
                }
            )
    return planned


def build_result(profile: dict, manifest_path: Path, environment: str) -> dict:
    require_keys(profile, str(manifest_path))
    groups = normalize_group_items(profile)
    families = normalize_families(profile)
    planned_channels = build_planned_channels(profile, families, groups, environment)

    return {
        "success": True,
        "manifest_path": str(manifest_path),
        "source": {
            "slug": profile["source"]["slug"],
            "name": profile["source"]["name"],
            "base_url": profile["source"]["base_url"],
            "docs_url": profile["source"]["docs_url"],
            "auth_mode": profile["source"]["auth_mode"],
            "profile_type": profile["source"]["profile_type"],
        },
        "target": {
            "environment": environment,
            "channel_scope": profile["target"]["channel_scope"],
        },
        "families": [item["family"] for item in families],
        "groups": groups,
        "pricing": profile["pricing"],
        "planned_channels": planned_channels,
        "verification": profile["verification"],
        "risks": [
            "pricing remains manual-review until explicit upstream ratios are confirmed",
            "vidu compatibility is assumed and must be validated in test before production",
        ],
    }


def main() -> None:
    parser = argparse.ArgumentParser(description="Generate a deterministic dry-run plan for channel onboarding manifests.")
    parser.add_argument("--profile", required=True, help="Path to the onboarding profile/manifest YAML")
    parser.add_argument("--target-environment", help="Override target environment")
    args = parser.parse_args()

    manifest_path = Path(args.profile).resolve()
    profile = load_yaml(manifest_path)
    environment = normalize_target_environment(profile, args.target_environment)
    result = build_result(profile, manifest_path, environment)
    print(json.dumps(result, ensure_ascii=False, indent=2, sort_keys=False))


if __name__ == "__main__":
    main()

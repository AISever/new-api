#!/usr/bin/env python3
import argparse
import json
import sys
from pathlib import Path
from typing import Any, List, Optional

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

REQUIRED_EXPLICIT_MODELS = {"doubao-video"}

COMPLEX_PRICING_SHAPES = {
    "per_call_sub_capability",
    "per_duration",
    "per_second",
    "per_duration_matrix",
    "duration_matrix",
    "matrix",
    "upstream_group_matrix",
}

BLOCKING_SUPPORT_STATUSES = {
    "blocked",
    "requires_code",
    "unsupported",
    "unsupported_or_unknown",
    "unknown",
}

EXPRESSION_REQUIRED_STATUSES = {
    "requires_expr",
    "requires_expression",
}

EXPRESSION_BILLING_MODES = {
    "per_call_expr",
    "tiered_expr",
}


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


def upstream_fact_models(profile: dict) -> dict[str, dict[str, Any]]:
    upstream_facts = profile.get("upstream_facts") or {}
    models = upstream_facts.get("models") if isinstance(upstream_facts, dict) else []
    if not isinstance(models, list):
        return {}
    out: dict[str, dict[str, Any]] = {}
    for item in models:
        if not isinstance(item, dict):
            continue
        model = str(item.get("id") or item.get("model") or item.get("model_name") or item.get("name") or "").strip()
        if model:
            out[model] = item
    return out


def normalize_pricing_shape_entry(model: str, entry: Any, facts_by_model: dict[str, dict[str, Any]]) -> dict[str, Any]:
    normalized: dict[str, Any] = {}
    fact = facts_by_model.get(model) or {}
    if isinstance(fact, dict):
        for key in [
            "shape",
            "pricing_shape",
            "support_status",
            "billing_strategy",
            "request_selectors",
            "price_matrix",
            "price_items",
            "split_models",
            "routeable_models",
            "block_reason",
            "evidence_refs",
        ]:
            if key in fact:
                normalized[key] = fact[key]

    if isinstance(entry, str):
        normalized["shape"] = entry
    elif isinstance(entry, dict):
        normalized.update(entry)
    elif entry is not None:
        fail(f"pricing.pricing_shapes.{model} must be a string or object")

    if "shape" not in normalized and "pricing_shape" in normalized:
        normalized["shape"] = normalized["pricing_shape"]
    normalized["shape"] = str(normalized.get("shape") or "").strip()
    normalized["support_status"] = str(normalized.get("support_status") or "").strip()
    normalized["billing_strategy"] = str(normalized.get("billing_strategy") or "").strip()
    return normalized


def normalize_pricing_shapes(profile: dict) -> dict[str, dict[str, Any]]:
    pricing = profile.get("pricing") or {}
    raw_shapes = pricing.get("pricing_shapes") or {}
    if not isinstance(raw_shapes, dict):
        fail("pricing.pricing_shapes must be an object")

    facts_by_model = upstream_fact_models(profile)
    model_names = set(facts_by_model.keys()) | {str(model).strip() for model in raw_shapes.keys()}
    normalized: dict[str, dict[str, Any]] = {}
    for model in sorted(model for model in model_names if model):
        normalized[model] = normalize_pricing_shape_entry(model, raw_shapes.get(model), facts_by_model)
    return normalized


def has_split_route(entry: dict[str, Any]) -> bool:
    split_models = entry.get("split_models") or entry.get("routeable_models")
    if isinstance(split_models, list):
        return bool([item for item in split_models if str(item).strip()])
    if isinstance(split_models, dict):
        return bool(split_models)
    return False


def price_matrix_len(entry: dict[str, Any]) -> int:
    matrix = entry.get("price_matrix")
    if isinstance(matrix, list):
        return len(matrix)
    if isinstance(matrix, dict):
        return len(matrix)
    return 0


def is_group_ratio_fixed_price(shape: str, strategy: str) -> bool:
    return shape == "fixed_per_request" and strategy in {
        "model_price_plus_group_ratio",
        "model_price",
    }


def map_or_empty(pricing: dict, key: str) -> dict:
    value = pricing.get(key) or {}
    if not isinstance(value, dict):
        fail(f"pricing.{key} must be an object")
    return value


def profile_generates_duration_expr(profile: dict) -> bool:
    if not isinstance(profile, dict):
        return False
    shape = str(profile.get("shape") or "").strip()
    unit = str(profile.get("unit") or "").strip()
    duration_selector = str(profile.get("duration_selector") or "").strip()
    try:
        price_per_unit = float(profile.get("price_per_unit") or 0)
    except (TypeError, ValueError):
        price_per_unit = 0
    return shape == "per_duration" and unit == "second" and duration_selector != "" and price_per_unit > 0


def validate_pricing_compatibility(profile: dict, planned_channels: List[dict]) -> List[dict]:
    pricing = profile["pricing"]
    pricing_shapes = normalize_pricing_shapes(profile)
    model_price = map_or_empty(pricing, "model_price")
    model_price_items = map_or_empty(pricing, "model_price_items")
    model_ratio = map_or_empty(pricing, "model_ratio")
    completion_ratio = map_or_empty(pricing, "completion_ratio")
    billing_mode = map_or_empty(pricing, "billing_mode")
    billing_expr = map_or_empty(pricing, "billing_expr")
    model_pricing_profiles = map_or_empty(pricing, "model_pricing_profiles")

    channel_models = sorted({model for channel in planned_channels for model in channel["models"]})
    target_models = set(channel_models) | set(model_price) | set(model_price_items) | set(model_pricing_profiles) | set(model_ratio) | set(completion_ratio) | set(billing_mode) | set(billing_expr)
    models = sorted(target_models | set(pricing_shapes))
    summaries: List[dict] = []

    for model in models:
        shape_entry = pricing_shapes.get(model) or {}
        shape = str(shape_entry.get("shape") or "").strip()
        support_status = str(shape_entry.get("support_status") or "").strip()
        in_target_scope = model in target_models
        mode = str(billing_mode.get(model) or "").strip()
        has_expr = mode in EXPRESSION_BILLING_MODES and bool(str(billing_expr.get(model) or "").strip())
        pricing_profile = model_pricing_profiles.get(model) or {}
        if pricing_profile and not isinstance(pricing_profile, dict):
            fail(f"pricing.model_pricing_profiles.{model} must be an object")
        profile_mode = str(pricing_profile.get("billing_mode") or "").strip()
        profile_expr = str(pricing_profile.get("billing_expr") or "").strip()
        has_profile_expr = profile_mode in EXPRESSION_BILLING_MODES and bool(profile_expr)
        has_generated_profile_expr = profile_generates_duration_expr(pricing_profile)
        has_real_expr = has_expr or has_profile_expr
        has_real_expr = has_real_expr or has_generated_profile_expr
        has_price = model in model_price
        has_items = model in model_price_items
        has_profile_items = isinstance(pricing_profile.get("items"), list) and len(pricing_profile.get("items") or []) > 0
        has_split = has_split_route(shape_entry)
        matrix_count = price_matrix_len(shape_entry)
        strategy = str(shape_entry.get("billing_strategy") or "").strip()

        if in_target_scope and not shape and not bool(shape_entry.get("target_only_existing_metadata")):
            fail(f"planned model {model} requires pricing.pricing_shapes or upstream_facts.models pricing_shape")
        if in_target_scope and support_status in BLOCKING_SUPPORT_STATUSES:
            reason = shape_entry.get("block_reason") or f"support_status={support_status}"
            fail(f"pricing for {model} is blocked: {reason}")
        if in_target_scope and support_status in EXPRESSION_REQUIRED_STATUSES and not (has_real_expr or has_split):
            fail(f"pricing for {model} requires expression billing or split routeable models")
        if in_target_scope and (has_items or has_profile_items) and not (has_price or has_real_expr or has_split):
            fail(f"pricing for {model} has ModelPriceItems only; display rows do not affect real billing")
        if in_target_scope and shape in COMPLEX_PRICING_SHAPES:
            if strategy in {"split_models", "routeable_models"} and not has_split:
                fail(f"pricing for {model} declares {strategy} but has no split_models/routeable_models")
            if strategy in {"per_call_expr", "tiered_expr", "expr", "expression"} and not has_real_expr:
                fail(f"pricing for {model} declares expression billing but missing billing_mode/billing_expr")
            if not (has_real_expr or has_split):
                fail(f"pricing for {model} is {shape}; do not collapse complex real prices into one ModelPrice")
        if in_target_scope and matrix_count > 1 and not (has_real_expr or has_split or is_group_ratio_fixed_price(shape, strategy)):
            fail(f"pricing for {model} has a multi-row price matrix but no real billing strategy")

        summaries.append(
            {
                "model": model,
                "target_scope": in_target_scope,
                "shape": shape or "unspecified",
                "support_status": support_status or "unspecified",
                "billing_mode": mode or "default",
                "has_model_price": has_price,
                "has_display_price_items": has_items or has_profile_items,
                "has_pricing_profile": bool(pricing_profile),
                "has_expression": has_real_expr,
                "has_split_routeable_models": has_split,
                "price_matrix_rows": matrix_count,
            }
        )
    return summaries


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
    pricing_compatibility = validate_pricing_compatibility(profile, planned_channels)

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
        "pricing_compatibility": pricing_compatibility,
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

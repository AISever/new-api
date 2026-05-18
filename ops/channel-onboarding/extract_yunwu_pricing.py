#!/usr/bin/env python3
"""Extract deterministic yunwu provider/group/model/price facts.

This tool is intentionally non-mutating. It converts the upstream
``/api/pricing_new`` response plus documented frontend pricing formulas into
local evidence artifacts and a test-only manifest for models whose real billing
can be represented by ``ModelPrice * GroupRatio`` or a verified request
expression.
"""

import argparse
import csv
import json
from collections import defaultdict
from datetime import datetime
from pathlib import Path
from typing import Any, Optional

import yaml


TARGET_VENDOR_FAMILIES = {
    68: "vidu",
    88: "kling",
}

ACTIVE_UPSTREAM_GROUPS_BY_FAMILY = {
    "kling": {"default", "特价kling"},
    "vidu": {"default", "特价vidu"},
}

OUT_OF_SCOPE_FAMILIES = {
    "doubao-video": "Doubao 本轮不处理",
}

USER_CONFIRMED_DECISIONS = {
    "scope": [
        "Doubao models are out of scope for this run.",
        "Only upstream groups default, 特价kling, and 特价vidu are mapped.",
    ],
    "group_mapping": "Preserve one upstream group as one neutral local aisever group.",
    "fixed_price_write": "Fixed-price-ready models may be written to local/test first.",
    "kling_audio": "kling-audio may use the current per_call_expr billing strategy.",
    "complex_pricing": "Per-second, matrix, video-size, and image models must be handled the same way as yunwu; do not simplify to one fixed ModelPrice.",
    "visibility": "Public names stay neutral; admin-only channel fields may retain upstream provenance.",
    "production": "No production writes are approved by this decision record.",
}

FAMILY_CHANNEL_TYPES = {
    "kling": 50,
    "vidu": 52,
    "doubao-video": 54,
}

FAMILY_ADAPTORS = {
    "kling": "relay/channel/task/kling",
    "vidu": "relay/channel/task/vidu",
    "doubao-video": "relay/channel/task/doubao",
}

GROUP_TARGETS = {
    ("kling", "default"): "video-upstream-kling-default",
    ("kling", "特价kling"): "video-upstream-kling-discount-kling",
    ("vidu", "default"): "video-upstream-vidu-default",
    ("vidu", "特价vidu"): "video-upstream-vidu-discount-vidu",
}

GROUP_DESCRIPTIONS = {
    "video-upstream-kling-default": "Kling standard video pool",
    "video-upstream-kling-discount-kling": "Kling economy video pool",
    "video-upstream-vidu-default": "Vidu standard video pool",
    "video-upstream-vidu-discount-vidu": "Vidu economy video pool",
}

# Main display multipliers extracted from yunwu frontend bundle evidence.
MAIN_MULTIPLIERS = {
    "kling-image": 2.5,
    "kling-omni-image": 20,
    "kling-video": 100,
    "kling-omni-video": 100,
    "kling-avatar-image2video": 100,
    "kling-audio": 5,
    "kling-custom-voices": 5,
    "kling-effects": 100,
    "kling-multi-elements": 100,
    "kling-video-extend": 100,
    "kling-advanced-lip-sync": 50,
    "kling-image-recognize": 10,
    "kling-motion-control": 50,
    "viduq2": 18.75,
    "viduq2-turbo": 18.75,
    "viduq2-pro": 25,
    "viduq3": 156.25,
    "viduq3-pro": 218.75,
    "viduq3-turbo": 125,
    "viduq3-mix": 390.625,
    "viduq1": 62.5,
    "viduq1-classic": 250,
    "vidu2.0": 62.5,
    "audio1.0": 31.25,
    "vidu-tts": 31.25,
}

DIRECT_FIXED_MODELS = {
    "kling-advanced-custom-elements",
    "kling-custom-elements",
    "kling-custom-voices",
    "kling-image-recognize",
}

EXPR_READY_MODELS = {
    "kling-audio",
}

# Models returned by yunwu pricing APIs but not confirmed in the visible UI.
# Keep these out of "confirmed upstream model" decisions until browser evidence
# proves they are searchable/visible in the upstream site.
UI_UNVERIFIED_MODELS = {
    "audio1.0",
}

DURATION_OR_MATRIX_SHAPES = {
    "per_duration",
    "per_duration_matrix",
    "variant_matrix",
    "video_size",
    "image",
}

COMPLEX_SHAPES = {
    "kling-advanced-lip-sync": "per_call_sub_capability",
    "kling-audio": "per_call_sub_capability",
    "kling-avatar-image2video": "per_duration_matrix",
    "kling-effects": "variant_matrix",
    "kling-image": "image",
    "kling-motion-control": "per_duration_matrix",
    "kling-multi-elements": "variant_matrix",
    "kling-omni-image": "image",
    "kling-omni-video": "per_duration_matrix",
    "kling-video": "variant_matrix",
    "kling-video-extend": "variant_matrix",
}

EVIDENCE = [
    {
        "id": "yunwu-pricing-new",
        "url": "https://yunwu.ai/api/pricing_new",
        "kind": "json-api",
        "artifact": "ops/local-test/yunwu-onboarding/evidence/yunwu-pricing-new.raw.json",
    },
    {
        "id": "yunwu-frontend-pricing-formula",
        "url": "https://assets.wlai.vip/assets/js/index-2TJsVMwG.js",
        "kind": "frontend-bundle",
        "artifact": "ops/local-test/yunwu-onboarding/evidence/yunwu-main-index-2TJsVMwG.js",
    },
    {
        "id": "yunwu-frontend-pricing-detail",
        "url": "https://assets.wlai.vip/assets/js/index-NRU3Axrh.js",
        "kind": "frontend-bundle",
        "artifact": "ops/local-test/yunwu-onboarding/evidence/yunwu-pricing-chunk-index-NRU3Axrh.js",
    },
]


def money(value: float) -> float:
    return round(float(value), 10)


def stable_unique(values: list[str]) -> list[str]:
    seen: set[str] = set()
    out: list[str] = []
    for value in values:
        item = str(value).strip()
        if item and item not in seen:
            seen.add(item)
            out.append(item)
    return out


def active_upstream_groups_for(item: dict[str, Any], family: str) -> list[str]:
    allowed_groups = ACTIVE_UPSTREAM_GROUPS_BY_FAMILY.get(family, set())
    return [
        str(group).strip()
        for group in item.get("enable_groups") or []
        if str(group).strip() in allowed_groups
    ]


def target_group_for(family: str, upstream_group: str) -> str:
    if (family, upstream_group) in GROUP_TARGETS:
        return GROUP_TARGETS[(family, upstream_group)]
    slug = (
        upstream_group.lower()
        .replace(" ", "-")
        .replace("/", "-")
        .replace("_", "-")
    )
    return f"video-upstream-{family}-{slug}"


def vendor_by_id(raw: dict[str, Any]) -> dict[int, dict[str, Any]]:
    return {
        int(item["id"]): item
        for item in raw.get("vendors", [])
        if isinstance(item, dict) and item.get("id") is not None
    }


def group_model_ratio(raw: dict[str, Any], group: str, model: str) -> Optional[float]:
    group_ratios = raw.get("group_model_ratio") or {}
    if not isinstance(group_ratios, dict):
        return None
    model_ratios = group_ratios.get(group) or {}
    if not isinstance(model_ratios, dict):
        return None
    value = model_ratios.get(model)
    if value is None:
        return None
    return float(value)


def model_multiplier(model: str) -> float:
    return float(MAIN_MULTIPLIERS.get(model, 1))


def classify_shape(item: dict[str, Any], family: str) -> str:
    model = str(item["model_name"])
    if model in COMPLEX_SHAPES:
        return COMPLEX_SHAPES[model]
    if family == "vidu":
        return "per_duration"
    if family == "doubao-video":
        return "video_size"
    if model in DIRECT_FIXED_MODELS:
        return "fixed_per_request"
    quota_type = int(item.get("quota_type") or 0)
    if quota_type == 4:
        return "per_duration"
    if quota_type == 3:
        return "video_size"
    if quota_type == 2:
        return "image"
    if quota_type == 1:
        return "fixed_per_request"
    return "token_ratio"


def display_unit(shape: str) -> str:
    if shape in {"per_duration", "per_duration_matrix"}:
        return "second"
    if shape == "video_size":
        return "video_size"
    if shape == "image":
        return "image"
    return "request"


def unit_label(unit: str) -> str:
    labels = {
        "request": "按次",
        "second": "按秒",
        "video_size": "按视频规格",
        "image": "按图片",
    }
    return labels.get(unit, unit)


def source_visibility(model: str) -> str:
    if model in UI_UNVERIFIED_MODELS:
        return "api_returned_ui_search_unverified"
    return "api_returned"


def source_visibility_label(status: str) -> str:
    labels = {
        "api_returned": "API返回",
        "api_returned_ui_search_unverified": "API返回但官网搜索未确认",
    }
    return labels.get(status, status)


def support_status(shape: str) -> str:
    if shape == "fixed_per_request":
        return "direct"
    if shape == "per_call_sub_capability":
        return "requires_expr"
    return "blocked"


def block_reason(shape: str) -> str:
    if shape == "fixed_per_request":
        return ""
    if shape == "image":
        return "requires verified image-count billing support or split routeable models"
    if shape == "video_size":
        return "requires verified video-size billing support or split routeable models"
    if shape == "per_duration":
        return "requires verified duration-aware billing support"
    if shape == "per_duration_matrix":
        return "requires verified duration and variant selector billing support"
    if shape == "per_call_sub_capability":
        return "requires verified request selector expression billing"
    if shape == "variant_matrix":
        return "requires verified variant selector billing support"
    return "requires verified real billing support; display-only price is insufficient"


def detail_price_items(model: str, main_price: float) -> list[dict[str, Any]]:
    if model == "kling-audio":
        return [
            {"label": "文生音效", "unit": "request", "price": money(main_price * 5), "multiplier": 5},
            {"label": "视频生音效", "unit": "request", "price": money(main_price * 5), "multiplier": 5},
            {"label": "语音合成", "unit": "request", "price": money(main_price), "multiplier": 1},
        ]
    if model == "kling-advanced-lip-sync":
        return [
            {"label": "人脸识别", "unit": "request", "price": money(main_price * 0.1), "multiplier": 0.1},
            {"label": "对口型", "unit": "5_seconds", "price": money(main_price), "multiplier": 1},
        ]
    if model == "kling-motion-control":
        return [
            {"label": "V2.6 std 720P", "unit": "second", "price": money(main_price), "multiplier": 1},
            {"label": "V2.6 pro 1080P", "unit": "second", "price": money(main_price * 1.6), "multiplier": 1.6},
            {"label": "V3.0 std 720P", "unit": "second", "price": money(main_price * 1.8), "multiplier": 1.8},
            {"label": "V3.0 pro 1080P", "unit": "second", "price": money(main_price * 2.4), "multiplier": 2.4},
        ]
    if model in {"kling-video", "kling-video-extend"}:
        return [
            {"label": "kling-v1 std 4-5s", "unit": "request", "price": money(main_price), "multiplier": 1},
            {"label": "kling-v1 pro 4-5s", "unit": "request", "price": money(main_price * 3.5), "multiplier": 3.5},
            {"label": "kling-v1-5 std 4-5s", "unit": "request", "price": money(main_price * 2), "multiplier": 2},
            {"label": "kling-v1-5 pro 4-5s", "unit": "request", "price": money(main_price * 3.5), "multiplier": 3.5},
            {"label": "kling-v1-6 std 4-5s", "unit": "request", "price": money(main_price * 2), "multiplier": 2},
            {"label": "kling-v1-6 pro 4-5s", "unit": "request", "price": money(main_price * 3.5), "multiplier": 3.5},
        ]
    if model == "kling-avatar-image2video":
        return [
            {"label": "std", "unit": "second", "price": money(main_price), "multiplier": 1},
            {"label": "pro", "unit": "second", "price": money(main_price * 2), "multiplier": 2},
        ]
    if model == "kling-multi-elements":
        return [
            {"label": "kling-v1-6 std 5s", "unit": "request", "price": money(main_price * 3), "multiplier": 3},
            {"label": "kling-v1-6 std 10s", "unit": "request", "price": money(main_price * 6), "multiplier": 6},
            {"label": "kling-v1-6 pro 5s", "unit": "request", "price": money(main_price * 5), "multiplier": 5},
            {"label": "kling-v1-6 pro 10s", "unit": "request", "price": money(main_price * 10), "multiplier": 10},
        ]
    if model == "kling-omni-video":
        return [
            {"label": "std no-ref no-audio", "unit": "second", "price": money(main_price * 0.6), "multiplier": 0.6},
            {"label": "std ref no-audio", "unit": "second", "price": money(main_price * 0.9), "multiplier": 0.9},
            {"label": "std no-ref audio", "unit": "second", "price": money(main_price * 0.8), "multiplier": 0.8},
            {"label": "pro no-ref no-audio", "unit": "second", "price": money(main_price * 0.8), "multiplier": 0.8},
            {"label": "pro no-ref audio", "unit": "second", "price": money(main_price), "multiplier": 1},
            {"label": "pro ref no-audio", "unit": "second", "price": money(main_price * 1.2), "multiplier": 1.2},
        ]
    if model == "kling-effects":
        return [
            {"label": "effect_scene_min", "unit": "request", "price": money(main_price), "multiplier": 1},
            {"label": "effect_scene_max", "unit": "request", "price": money(main_price * 9), "multiplier": 9},
        ]
    return []


def coverage_rows(raw: dict[str, Any]) -> list[dict[str, Any]]:
    vendors = vendor_by_id(raw)
    group_ratio = raw.get("group_ratio") or {}
    usable_group = raw.get("usable_group") or {}
    rows: list[dict[str, Any]] = []

    for item in raw.get("data", []):
        if not isinstance(item, dict):
            continue
        vendor_id = int(item.get("vendor_id") or 0)
        family = TARGET_VENDOR_FAMILIES.get(vendor_id)
        if not family:
            continue
        model = str(item["model_name"])
        shape = classify_shape(item, family)
        multiplier = model_multiplier(model)
        for upstream_group in active_upstream_groups_for(item, family):
            group_ratio_value = float(group_ratio.get(upstream_group, 1))
            group_model_ratio_value = group_model_ratio(raw, upstream_group, model)
            effective_group_ratio = group_ratio_value * (group_model_ratio_value if group_model_ratio_value is not None else 1)
            base_model_price = float(item.get("model_price") or 0)
            main_price = money(base_model_price * multiplier * effective_group_ratio)
            price_items = detail_price_items(model, main_price)
            rows.append(
                {
                    "model": model,
                    "family": family,
                    "provider": vendors.get(vendor_id, {}).get("name", ""),
                    "provider_id": vendor_id,
                    "upstream_group": upstream_group,
                    "group_label": usable_group.get(upstream_group, upstream_group),
                    "target_group_candidate": target_group_for(family, upstream_group),
                    "group_ratio": money(group_ratio_value),
                    "group_model_ratio": group_model_ratio_value,
                    "quota_type": int(item.get("quota_type") or 0),
                    "pricing_shape": shape,
                    "base_model_price": money(base_model_price),
                    "price_multiplier": money(multiplier),
                    "display_unit": display_unit(shape),
                    "effective_price": main_price,
                    "subprices_json": json.dumps(price_items, ensure_ascii=False),
                    "supported_endpoint_types": ",".join(item.get("supported_endpoint_types") or []),
                    "tags": item.get("tags") or "",
                    "description": item.get("description") or "",
                    "source_visibility": source_visibility(model),
                    "support_status": support_status(shape),
                    "block_reason": block_reason(shape),
                    "evidence_refs": ["yunwu-pricing-new", "yunwu-frontend-pricing-formula", "yunwu-frontend-pricing-detail"],
                }
            )
    return sorted(rows, key=lambda row: (row["family"], row["model"], row["upstream_group"]))


def build_facts(rows: list[dict[str, Any]]) -> list[dict[str, Any]]:
    grouped: dict[str, list[dict[str, Any]]] = defaultdict(list)
    for row in rows:
        grouped[row["model"]].append(row)

    facts = []
    for model, model_rows in sorted(grouped.items()):
        first = model_rows[0]
        price_matrix = []
        for row in model_rows:
            subprices = json.loads(row["subprices_json"])
            if subprices:
                for item in subprices:
                    price_matrix.append(
                        {
                            "upstream_group": row["upstream_group"],
                            "target_group": row["target_group_candidate"],
                            **item,
                        }
                    )
            else:
                price_matrix.append(
                    {
                        "upstream_group": row["upstream_group"],
                        "target_group": row["target_group_candidate"],
                        "unit": row["display_unit"],
                        "price": row["effective_price"],
                        "multiplier": row["price_multiplier"],
                    }
                )
        facts.append(
            {
                "id": model,
                "provider": first["provider"],
                "provider_id": first["provider_id"],
                "family": first["family"],
                "pricing_shape": first["pricing_shape"],
                "quota_type": first["quota_type"],
                "base_model_price": first["base_model_price"],
                "price_multiplier": first["price_multiplier"],
                "display_unit": first["display_unit"],
                "enable_groups": stable_unique([row["upstream_group"] for row in model_rows]),
                "target_groups": stable_unique([row["target_group_candidate"] for row in model_rows]),
                "supported_endpoint_types": stable_unique(
                    [
                        endpoint
                        for row in model_rows
                        for endpoint in str(row["supported_endpoint_types"]).split(",")
                        if endpoint
                    ]
                ),
                "tags": first["tags"],
                "description": first["description"],
                "source_visibility": first["source_visibility"],
                "support_status": first["support_status"],
                "block_reason": first["block_reason"],
                "price_matrix": price_matrix,
                "evidence_refs": first["evidence_refs"],
            }
        )
    return facts


def build_summary(rows: list[dict[str, Any]]) -> dict[str, Any]:
    by_family: dict[str, set[str]] = defaultdict(set)
    by_shape: dict[str, set[str]] = defaultdict(set)
    groups: dict[str, dict[str, Any]] = {}
    blocked: set[str] = set()
    direct: set[str] = set()
    expr_ready: set[str] = set()

    for row in rows:
        by_family[row["family"]].add(row["model"])
        by_shape[row["pricing_shape"]].add(row["model"])
        groups[row["upstream_group"]] = {"label": row["group_label"], "ratio": row["group_ratio"]}
        if row["source_visibility"] != "api_returned":
            blocked.add(row["model"])
        elif row["model"] in EXPR_READY_MODELS:
            expr_ready.add(row["model"])
        elif row["support_status"] == "direct":
            direct.add(row["model"])
        else:
            blocked.add(row["model"])

    duration_or_matrix_blocked = {
        row["model"]
        for row in rows
        if row["pricing_shape"] in DURATION_OR_MATRIX_SHAPES
        and row["model"] not in direct
        and row["model"] not in expr_ready
    }

    return {
        "model_count": len({row["model"] for row in rows}),
        "api_returned_model_count": len({row["model"] for row in rows}),
        "ui_confirmed_model_count": len({row["model"] for row in rows}) - len(UI_UNVERIFIED_MODELS & {row["model"] for row in rows}),
        "coverage_rows": len(rows),
        "groups": dict(sorted(groups.items())),
        "by_family": {key: sorted(value) for key, value in sorted(by_family.items())},
        "by_shape": {key: sorted(value) for key, value in sorted(by_shape.items())},
        "direct_model_count": len(direct),
        "direct_models": sorted(direct),
        "expr_ready_model_count": len(expr_ready),
        "expr_ready_models": sorted(expr_ready),
        "blocked_model_count": len(blocked),
        "blocked_models": sorted(blocked),
        "duration_or_matrix_blocked_model_count": len(duration_or_matrix_blocked),
        "duration_or_matrix_blocked_models": sorted(duration_or_matrix_blocked),
        "ui_unverified_model_count": len(UI_UNVERIFIED_MODELS & {row["model"] for row in rows}),
        "ui_unverified_models": sorted(UI_UNVERIFIED_MODELS & {row["model"] for row in rows}),
    }


def build_billing_safe_manifest(rows: list[dict[str, Any]], facts: list[dict[str, Any]], collected_at: str, target_base_url: str) -> dict[str, Any]:
    ready_rows = [
        row
        for row in rows
        if row["support_status"] == "direct" or row["model"] in EXPR_READY_MODELS
        if row["source_visibility"] == "api_returned"
    ]
    group_names = stable_unique([row["target_group_candidate"] for row in ready_rows])
    models_by_group: dict[str, list[str]] = defaultdict(list)
    for row in ready_rows:
        models_by_group[row["target_group_candidate"]].append(row["model"])

    model_prices = {}
    model_price_items = {}
    billing_mode = {}
    billing_expr = {}
    pricing_shapes = {}
    for fact in facts:
        if fact["support_status"] != "direct" and fact["id"] not in EXPR_READY_MODELS:
            continue
        if fact.get("source_visibility") != "api_returned":
            continue
        model = fact["id"]
        if model in EXPR_READY_MODELS:
            model_price_items[model] = [
                {"label": item["label"], "price": item["price"], "unit": item["unit"]}
                for item in fact["price_matrix"]
                if item.get("target_group") == "video-upstream-kling-default"
            ]
            billing_mode[model] = "per_call_expr"
            billing_expr[model] = (
                'str(param("metadata.scenario")) == "speech" || '
                'str(param("metadata.action")) == "speech" || '
                'str(param("metadata.task")) == "speech" || '
                'str(param("metadata.capability")) == "speech" || '
                'str(param("metadata.type")) == "speech" || '
                'str(param("metadata.scenario")) == "tts" || '
                'str(param("metadata.action")) == "tts" || '
                'str(param("metadata.task")) == "tts" || '
                'str(param("metadata.capability")) == "tts" || '
                'str(param("metadata.type")) == "tts" '
                '? tier("语音合成", 0.085) : tier("音效", 0.425)'
            )
        else:
            model_prices[model] = fact["base_model_price"] * fact["price_multiplier"]
        pricing_shapes[model] = {
            "shape": fact["pricing_shape"],
            "support_status": "direct" if fact["support_status"] == "direct" else "requires_expr",
            "billing_strategy": "per_call_expr" if model in EXPR_READY_MODELS else "model_price_plus_group_ratio",
            "quota_type": fact["quota_type"],
            "billing_unit": fact["display_unit"],
            "billing_unit_label": unit_label(fact["display_unit"]),
            "evidence_refs": fact["evidence_refs"],
        }

    groups = []
    for group_name in group_names:
        first = next(row for row in ready_rows if row["target_group_candidate"] == group_name)
        groups.append(
            {
                "name": group_name,
                "description": GROUP_DESCRIPTIONS.get(group_name, group_name),
                "ratio": first["group_ratio"],
            }
        )

    channels = []
    for group_name in group_names:
        models = stable_unique(models_by_group[group_name])
        channels.append(
            {
                "name": f"yunwu-kling-{group_name}-direct",
                "type": FAMILY_CHANNEL_TYPES["kling"],
                "key_ref": "env:YUNWU_KEY",
                "base_url": "https://yunwu.ai",
                "group": group_name,
                "models": models,
                "test_model": models[0],
                "tag": "yunwu-video",
                "status": 1,
                "remark": "Admin provenance: yunwu Kling billing-safe test candidate; upstream group cardinality preserved.",
            }
        )

    ready_facts = [
        fact
        for fact in facts
        if fact["support_status"] == "direct" or fact["id"] in EXPR_READY_MODELS
        if fact.get("source_visibility") == "api_returned"
    ]
    return {
        "target": {
            "base_url": target_base_url,
            "username_ref": "env:AISEVER_USERNAME",
            "password_ref": "env:AISEVER_PASSWORD",
            "environment": "test",
        },
        "source": {
            "slug": "yunwu",
            "base_url": "https://yunwu.ai",
        },
        "upstream_facts": {
            "collected_at": collected_at,
            "evidence": EVIDENCE,
            "providers": [
                {"name": "Kling (可灵)", "family": "kling"},
                {"name": "Vidu", "family": "vidu"},
            ],
            "out_of_scope": [
                {"family": family, "reason": reason}
                for family, reason in sorted(OUT_OF_SCOPE_FAMILIES.items())
            ],
            "confirmed_decisions": USER_CONFIRMED_DECISIONS,
            "models": ready_facts,
        },
        "defaults": {
            "tag": "yunwu-video",
        },
        "visibility": {
            "forbidden_terms": ["yunwu", "yunwu.ai", "云雾"],
        },
        "confirmation": {
            "required": True,
            "artifact": "ops/local-test/yunwu-onboarding/yunwu-upstream-confirmation.md",
            "user_confirmed": True,
            "scope": "local/test only; production requires separate approval",
        },
        "groups": groups,
        "channels": channels,
        "pricing": {
            "pricing_shapes": pricing_shapes,
            "model_price": {key: money(value) for key, value in sorted(model_prices.items())},
            "model_price_items": model_price_items,
            "model_ratio": {},
            "completion_ratio": {},
            "billing_mode": billing_mode,
            "billing_expr": billing_expr,
        },
    }


def build_blocked_candidates(facts: list[dict[str, Any]], collected_at: str) -> dict[str, Any]:
    blocked_facts = [
        {
            **fact,
            "support_status": "blocked",
            "apply_ready": False,
            "requires_selector_verification": fact["pricing_shape"] in DURATION_OR_MATRIX_SHAPES,
            "billing_unit_label": unit_label(fact["display_unit"]),
            "candidate_strategy": candidate_strategy(fact),
        }
        for fact in facts
        if fact["support_status"] != "direct" and fact["id"] not in EXPR_READY_MODELS
    ]
    return {
        "source": {"slug": "yunwu", "base_url": "https://yunwu.ai"},
        "collected_at": collected_at,
        "purpose": "Non-apply candidate report for models whose upstream pricing cannot be safely written as one fixed ModelPrice.",
        "hard_rule": "Do not apply these models until request selectors and real billing expressions or split routeable models are verified.",
        "models": blocked_facts,
    }


def candidate_strategy(fact: dict[str, Any]) -> str:
    shape = fact["pricing_shape"]
    if shape == "per_duration":
        return "verify duration selector and use request expression: price_per_second * duration"
    if shape == "per_duration_matrix":
        return "verify duration plus variant selectors and use matrix expression or split routeable models"
    if shape == "variant_matrix":
        return "verify version/mode/resolution/task selectors and use expression or split routeable models"
    if shape == "video_size":
        return "verify video size selector and use expression or split routeable models"
    if shape == "image":
        return "verify image count/size selector and use expression or split routeable models"
    if shape == "per_call_sub_capability":
        return "verify sub-capability selector and use per_call_expr"
    return "requires manual compatibility analysis"


def md_escape(value: Any) -> str:
    text = str(value if value is not None else "")
    return text.replace("|", "\\|").replace("\n", " ").strip()


def facts_by_model(facts: list[dict[str, Any]]) -> dict[str, dict[str, Any]]:
    return {fact["id"]: fact for fact in facts}


def render_model_matrix_zh(fact: dict[str, Any]) -> list[str]:
    lines = [
        f"### {fact['id']}",
        "",
        f"- 服务商：{fact['provider']}",
        f"- 家族：{fact['family']}",
        f"- 来源可见性：{source_visibility_label(fact.get('source_visibility', 'api_returned'))}",
        f"- 计费形态：{fact['pricing_shape']}",
        f"- 默认计费单位：{unit_label(fact['display_unit'])}",
        f"- 兼容状态：{fact['support_status']}",
    ]
    if fact.get("block_reason"):
        lines.append(f"- 阻断/待决策原因：{fact['block_reason']}")
    lines.extend(
        [
            "",
            "| 上游分组 | 候选本站本地分组 | 变体/能力 | 计费单位 | 价格 |",
            "|---|---|---|---|---:|",
        ]
    )
    for item in fact["price_matrix"]:
        lines.append(
            "| "
            + " | ".join(
                [
                    md_escape(item.get("upstream_group", "")),
                    md_escape(item.get("target_group", "")),
                    md_escape(item.get("label", "base")),
                    md_escape(unit_label(item.get("unit", ""))),
                    md_escape(item.get("price", "")),
                ]
            )
            + " |"
        )
    lines.append("")
    return lines


def render_confirmation_doc(rows: list[dict[str, Any]], facts: list[dict[str, Any]], summary: dict[str, Any], collected_at: str) -> str:
    lines: list[str] = [
        "# Yunwu Upstream Channel Confirmation",
        "",
        "> Status: local/test scope confirmed by user. Production writes are not approved and remain blocked.",
        "",
        "## Scope",
        "",
        f"- Source: yunwu.ai",
        f"- Collected at: {collected_at}",
        "- Confirmed current scope: Doubao is out of scope for this run.",
        "- Confirmed group mapping: only upstream groups `default`, `特价kling`, and `特价vidu` are mapped; each maps one-to-one to a neutral local group.",
        "- Confirmed write scope: fixed-price-ready and approved expression-ready models may proceed to local/test only. Production still requires separate approval.",
        f"- API-returned model count: {summary['api_returned_model_count']}",
        f"- UI-confirmed model count: {summary['ui_confirmed_model_count']}",
        f"- Provider/group/model/price coverage rows: {summary['coverage_rows']}",
        "- Rule: preserve upstream provider/group/model/price cardinality. Do not merge, simplify, rename, or omit rows without explicit user approval.",
        "- Rule: models that cannot be represented by real billing must stay out of write manifests until a handling plan is confirmed.",
        "",
        "## Confirmed User Decisions",
        "",
        "| Decision | Result |",
        "|---|---|",
        "| Provider scope | Doubao models are not processed in this run. |",
        "| Group scope | Map only `default`, `特价kling`, and `特价vidu`. |",
        "| Mapping cardinality | One upstream group maps to one neutral local aisever group. |",
        "| Fixed-price models | May be written to local/test first. |",
        "| `kling-audio` | May use per-call expression billing. |",
        "| Per-second/matrix/video-size/image models | Must be handled the same way as yunwu; no simplified one-price fallback. |",
        "| Visibility | Public names stay neutral; admin-only fields may retain upstream provenance. |",
        "| Production | Not approved in this run. |",
        "",
        "## Out Of Scope",
        "",
        "- Doubao 本轮不处理; it must not appear in coverage rows, write manifests, or active channel plans for this run.",
        "",
        "## Upstream Groups",
        "",
        "| Upstream group | Display label | Group ratio | Target local group candidate |",
        "|---|---|---:|---|",
    ]

    group_targets: dict[str, list[str]] = defaultdict(list)
    for row in rows:
        group_targets[row["upstream_group"]].append(row["target_group_candidate"])
    for group, meta in summary["groups"].items():
        targets = ", ".join(stable_unique(group_targets[group]))
        lines.append(f"| {md_escape(group)} | {md_escape(meta['label'])} | {meta['ratio']} | {md_escape(targets)} |")

    lines.extend(
        [
            "",
            "## Compatibility Summary",
            "",
            "| Bucket | Count | Models |",
            "|---|---:|---|",
            f"| Fixed-price ready | {summary['direct_model_count']} | {md_escape(', '.join(summary['direct_models']))} |",
            f"| Expression ready | {summary['expr_ready_model_count']} | {md_escape(', '.join(summary['expr_ready_models']))} |",
            f"| Blocked pending decision | {summary['blocked_model_count']} | {md_escape(', '.join(summary['blocked_models']))} |",
            f"| Duration/matrix/spec blocked | {summary['duration_or_matrix_blocked_model_count']} | {md_escape(', '.join(summary['duration_or_matrix_blocked_models']))} |",
            f"| API returned but UI-search unverified | {summary['ui_unverified_model_count']} | {md_escape(', '.join(summary['ui_unverified_models']))} |",
            "",
            "## Coverage Table",
            "",
            "| Provider | Family | Upstream group | Target local group candidate | Model | Source visibility | Pricing shape | Unit | Effective price | Status | Reason |",
            "|---|---|---|---|---|---|---|---|---:|---|---|",
        ]
    )
    for row in rows:
        lines.append(
            "| "
            + " | ".join(
                [
                    md_escape(row["provider"]),
                    md_escape(row["family"]),
                    md_escape(row["upstream_group"]),
                    md_escape(row["target_group_candidate"]),
                    md_escape(row["model"]),
                    md_escape(source_visibility_label(row["source_visibility"])),
                    md_escape(row["pricing_shape"]),
                    md_escape(unit_label(row["display_unit"])),
                    md_escape(row["effective_price"]),
                    md_escape(row["support_status"]),
                    md_escape(row["block_reason"]),
                ]
            )
            + " |"
        )

    lines.extend(
        [
            "",
            "## Price Matrix By Model",
            "",
        ]
    )
    for fact in facts:
        lines.extend(render_model_matrix_zh(fact))

    lines.extend(
        [
            "## Price Matrix Details",
            "",
            "| Provider | Family | Model | Upstream group | Target local group candidate | Variant/capability | Unit | Price |",
            "|---|---|---|---|---|---|---|---:|",
        ]
    )
    for fact in facts:
        for item in fact["price_matrix"]:
            lines.append(
                "| "
                + " | ".join(
                    [
                        md_escape(fact["provider"]),
                        md_escape(fact["family"]),
                        md_escape(fact["id"]),
                        md_escape(item.get("upstream_group", "")),
                        md_escape(item.get("target_group", "")),
                        md_escape(item.get("label", "base")),
                        md_escape(unit_label(item.get("unit", ""))),
                        md_escape(item.get("price", "")),
                    ]
                )
                + " |"
            )

    lines.extend(
        [
            "",
            "## Decisions Required Before Any Write",
            "",
            "- Local/test scope is confirmed for fixed-price-ready and approved expression-ready models.",
            "- Production write still requires a separate backup, release checklist, and explicit approval.",
            "- Complex models still require same-as-yunwu expression/UI/billing implementation and verification before write.",
            "",
            "## No-Write Gate",
            "",
            "This document approves only the current local/test planning scope. Production writes remain blocked until separately approved.",
            "",
        ]
    )
    return "\n".join(lines)


def render_confirmation_doc_zh(rows: list[dict[str, Any]], facts: list[dict[str, Any]], summary: dict[str, Any], collected_at: str) -> str:
    lines: list[str] = [
        "# Yunwu 上游渠道信息确认文档",
        "",
        "> 状态：用户已确认本地/测试范围。生产写入未授权，仍然阻断。",
        "",
        "## 范围",
        "",
        "- 上游来源：yunwu.ai",
        f"- 采集时间：{collected_at}",
        "- 已确认本轮范围：Doubao 本轮不处理。",
        "- 已确认分组映射：只映射 `default`、`特价kling`、`特价vidu`；每个上游分组一一映射为一个中性的本站本地分组。",
        "- 已确认写入范围：固定价格可落地模型与已确认表达式模型只允许进入本地/测试验证；生产写入仍需另行确认。",
        f"- API 返回模型数量：{summary['api_returned_model_count']}",
        f"- 官网可见性已确认模型数量：{summary['ui_confirmed_model_count']}",
        f"- provider/group/model/price 覆盖行数：{summary['coverage_rows']}",
        "- 规则：默认保持上游 provider/group/model/price 的一一对应关系。未经明确确认，不允许合并、简化、改名或省略任何行。",
        "- 规则：无法被本站真实扣费系统准确表达的模型，必须先列为待决策项，确认处理方案前不得进入写入清单。",
        "- 数据质量声明：如果某模型仅出现在 API 返回中，但无法在官网页面搜索/筛选中确认，必须标记为“API返回但官网搜索未确认”，不得当作已确认可上线模型。",
        "",
        "## 已确认用户决策",
        "",
        "| 决策项 | 结论 |",
        "|---|---|",
        "| 服务商范围 | Doubao 本轮不处理。 |",
        "| 分组范围 | 只映射 `default`、`特价kling`、`特价vidu`。 |",
        "| 分组映射方式 | 一个上游分组一一映射为一个中性的本站本地分组。 |",
        "| 固定价格模型 | 可以先写入本地/测试环境验证。 |",
        "| `kling-audio` | 同意使用当前建议的 `per_call_expr` 表达式计费方案。 |",
        "| 按秒/矩阵/视频规格/图片模型 | 按 yunwu 侧处理方式在本站实现同等展示与真实扣费，不允许简化成单一固定价格。 |",
        "| 可见性边界 | 普通用户可见名称保持中性；管理员可见渠道字段允许保留上游来源。 |",
        "| 生产写入 | 本轮未授权生产写入。 |",
        "",
        "## 本轮不处理范围",
        "",
        "- Doubao 本轮不处理；不得出现在本轮 coverage 行、写入 manifest 或启用渠道计划中。",
        "",
        "## 上游分组",
        "",
        "| 上游分组 | 上游展示名称 | 分组倍率 | 候选本站本地分组 |",
        "|---|---|---:|---|",
    ]

    group_targets: dict[str, list[str]] = defaultdict(list)
    for row in rows:
        group_targets[row["upstream_group"]].append(row["target_group_candidate"])
    for group, meta in summary["groups"].items():
        targets = ", ".join(stable_unique(group_targets[group]))
        lines.append(f"| {md_escape(group)} | {md_escape(meta['label'])} | {meta['ratio']} | {md_escape(targets)} |")

    lines.extend(
        [
            "",
            "## 兼容性汇总",
            "",
            "| 分类 | 数量 | 模型 |",
            "|---|---:|---|",
            f"| 固定价格可落地 | {summary['direct_model_count']} | {md_escape(', '.join(summary['direct_models']))} |",
            f"| 表达式可落地 | {summary['expr_ready_model_count']} | {md_escape(', '.join(summary['expr_ready_models']))} |",
            f"| 待决策阻断 | {summary['blocked_model_count']} | {md_escape(', '.join(summary['blocked_models']))} |",
            f"| 按秒/矩阵/规格阻断 | {summary['duration_or_matrix_blocked_model_count']} | {md_escape(', '.join(summary['duration_or_matrix_blocked_models']))} |",
            f"| API返回但官网搜索未确认 | {summary['ui_unverified_model_count']} | {md_escape(', '.join(summary['ui_unverified_models']))} |",
            "",
            "## 覆盖表",
            "",
            "| 服务商 | 家族 | 上游分组 | 候选本站本地分组 | 模型 | 来源可见性 | 计费形态 | 计费单位 | 有效价格 | 状态 | 原因 |",
            "|---|---|---|---|---|---|---|---|---:|---|---|",
        ]
    )
    for row in rows:
        lines.append(
            "| "
            + " | ".join(
                [
                    md_escape(row["provider"]),
                    md_escape(row["family"]),
                    md_escape(row["upstream_group"]),
                    md_escape(row["target_group_candidate"]),
                    md_escape(row["model"]),
                    md_escape(source_visibility_label(row["source_visibility"])),
                    md_escape(row["pricing_shape"]),
                    md_escape(unit_label(row["display_unit"])),
                    md_escape(row["effective_price"]),
                    md_escape(row["support_status"]),
                    md_escape(row["block_reason"]),
                ]
            )
            + " |"
        )

    lines.extend(
        [
            "",
            "## 按模型价格矩阵",
            "",
        ]
    )
    for fact in facts:
        lines.extend(render_model_matrix_zh(fact))

    lines.extend(
        [
            "## 价格矩阵明细",
            "",
            "| 服务商 | 家族 | 模型 | 上游分组 | 候选本站本地分组 | 变体/能力 | 计费单位 | 价格 |",
            "|---|---|---|---|---|---|---|---:|",
        ]
    )
    for fact in facts:
        for item in fact["price_matrix"]:
            lines.append(
                "| "
                + " | ".join(
                    [
                        md_escape(fact["provider"]),
                        md_escape(fact["family"]),
                        md_escape(fact["id"]),
                        md_escape(item.get("upstream_group", "")),
                        md_escape(item.get("target_group", "")),
                        md_escape(item.get("label", "base")),
                        md_escape(unit_label(item.get("unit", ""))),
                        md_escape(item.get("price", "")),
                    ]
                )
                + " |"
            )

    lines.extend(
        [
            "",
            "## 写入前需要确认的决策",
            "",
            "- 本地/测试范围已确认：固定价格可落地模型与已确认表达式模型可进入本地/测试验证。",
            "- 生产写入仍需单独备份、发布检查链路和明确批准。",
            "- 复杂模型仍需按 yunwu 的展示与真实扣费方式完成表达式/UI/扣费适配，并验证后才能写入。",
            "",
            "## 禁止写入门禁",
            "",
            "本文档只确认当前本地/测试规划范围。生产写入仍然阻断，直到单独获得明确批准。",
            "",
        ]
    )
    return "\n".join(lines)


def write_outputs(rows: list[dict[str, Any]], output_dir: Path, collected_at: str, target_base_url: str) -> None:
    output_dir.mkdir(parents=True, exist_ok=True)
    facts = build_facts(rows)
    summary = build_summary(rows)

    (output_dir / "yunwu-coverage.effective.json").write_text(
        json.dumps(rows, ensure_ascii=False, indent=2) + "\n",
        encoding="utf-8",
    )
    csv_fields = [
        "model",
        "family",
        "provider",
        "provider_id",
        "upstream_group",
        "group_label",
        "target_group_candidate",
        "group_ratio",
        "group_model_ratio",
        "quota_type",
        "pricing_shape",
        "base_model_price",
        "price_multiplier",
        "display_unit",
        "effective_price",
        "subprices_json",
        "supported_endpoint_types",
        "tags",
        "description",
        "source_visibility",
        "support_status",
        "block_reason",
    ]
    with (output_dir / "yunwu-coverage.effective.csv").open("w", encoding="utf-8", newline="") as f:
        writer = csv.DictWriter(f, fieldnames=csv_fields, extrasaction="ignore")
        writer.writeheader()
        writer.writerows(rows)

    (output_dir / "yunwu-facts-models.json").write_text(
        json.dumps(facts, ensure_ascii=False, indent=2) + "\n",
        encoding="utf-8",
    )
    (output_dir / "yunwu-summary.effective.json").write_text(
        json.dumps(summary, ensure_ascii=False, indent=2) + "\n",
        encoding="utf-8",
    )
    (output_dir / "yunwu-upstream-confirmation.md").write_text(
        render_confirmation_doc(rows, facts, summary, collected_at),
        encoding="utf-8",
    )
    (output_dir / "yunwu-upstream-confirmation.zh-CN.md").write_text(
        render_confirmation_doc_zh(rows, facts, summary, collected_at),
        encoding="utf-8",
    )
    blocked_candidates = build_blocked_candidates(facts, collected_at)
    (output_dir / "yunwu-billing-blocked-candidates.json").write_text(
        json.dumps(blocked_candidates, ensure_ascii=False, indent=2) + "\n",
        encoding="utf-8",
    )
    (output_dir / "yunwu-billing-blocked-candidates.yaml").write_text(
        yaml.safe_dump(blocked_candidates, allow_unicode=True, sort_keys=False),
        encoding="utf-8",
    )

    manifest = build_billing_safe_manifest(rows, facts, collected_at, target_base_url)
    (output_dir / "yunwu-billing-safe.test.manifest.yaml").write_text(
        yaml.safe_dump(manifest, allow_unicode=True, sort_keys=False),
        encoding="utf-8",
    )


def main() -> None:
    parser = argparse.ArgumentParser(description="Extract yunwu pricing facts and generate local onboarding artifacts.")
    parser.add_argument(
        "--input",
        default="ops/local-test/yunwu-onboarding/evidence/yunwu-pricing-new.raw.json",
        help="Path to yunwu /api/pricing_new JSON response.",
    )
    parser.add_argument(
        "--output-dir",
        default="ops/local-test/yunwu-onboarding",
        help="Output directory for generated facts and manifests.",
    )
    parser.add_argument("--target-base-url", default="http://127.0.0.1:3001")
    parser.add_argument("--collected-at", default="")
    args = parser.parse_args()

    input_path = Path(args.input)
    raw = json.loads(input_path.read_text(encoding="utf-8"))
    collected_at = args.collected_at or datetime.now().astimezone().isoformat(timespec="seconds")
    rows = coverage_rows(raw)
    write_outputs(rows, Path(args.output_dir), collected_at, args.target_base_url)

    summary = build_summary(rows)
    print(json.dumps({"success": True, **summary}, ensure_ascii=False, indent=2))


if __name__ == "__main__":
    main()

# Channel Onboarding

This directory contains manifest-driven onboarding assets for new upstream
channels and aggregators.

Artifacts:

- `manifest-template.yaml`: required manifest/profile structure
- `profiles/`: source-specific example profiles such as `yunwu.yaml`
- `dry_run.py`: generates a deterministic onboarding plan without mutating any environment
- `extract_yunwu_pricing.py`: extracts yunwu provider/group/model/price facts from saved upstream pricing data and generates local evidence artifacts
- `import_profile.py`: imports a manifest/profile into `test` or `enterprise` with minimal merges only

## Scope

Use these assets when a new upstream cannot be safely handled by the existing
text-family importer alone, or when a video-family aggregator must be assessed
before touching production.

Current split:

- `ops/scripts/kkidc-host-import-upstream.sh`
  - formal importer for existing text-oriented families such as `openai`,
    `anthropic`, `gemini`, and `jina-rerank`
- `ops/channel-onboarding/*`
  - manifest-driven dry-run path for new or mixed-content upstream onboarding,
    especially video families like `kling`, `vidu`, and `doubao-video`

## Usage

For yunwu-style upstreams, start by regenerating the local fact table from the
saved upstream evidence. This is the required first step before planning writes:

```bash
python3 ops/channel-onboarding/extract_yunwu_pricing.py \
  --input ops/local-test/yunwu-onboarding/evidence/yunwu-pricing-new.raw.json \
  --output-dir ops/local-test/yunwu-onboarding
```

Expected yunwu outputs:

- `yunwu-coverage.effective.csv`: one row per upstream provider/group/model/price mapping
- `yunwu-coverage.effective.json`: JSON equivalent of the coverage table
- `yunwu-facts-models.json`: model facts with price matrices and evidence refs
- `yunwu-summary.effective.json`: grouped fixed/expr-ready/blocked summary
- `yunwu-billing-safe.test.manifest.yaml`: local/test manifest for models whose real billing is representable as `ModelPrice * GroupRatio` or a verified `per_call_expr`
- `yunwu-billing-blocked-candidates.json|yaml`: non-apply candidate report for per-second, matrix, image, and video-size models

Do not apply blocked models. Models classified as `per_duration`,
`per_duration_matrix`, `variant_matrix`, `video_size`, or `image` require
verified request-aware billing, completion-time billing, or split routeable model
names before site writes. `per_call_sub_capability` models may be included only
when the generated manifest contains both display rows and a verified
`billing_mode/billing_expr` pair.

Important: `ModelPrice` is a fixed price. A model that is actually priced per
second, by video size, or by version/mode/resolution matrix must not be written
as a single `ModelPrice`; doing so makes both the UI and real deduction look
like fixed per-call billing.

Example:

```bash
python3 ops/channel-onboarding/dry_run.py \
  --profile ops/channel-onboarding/profiles/yunwu.yaml \
  --target-environment test
```

Expected output:

- deterministic JSON plan
- source and target metadata
- planned channels with channel type, family, group, models, test model, and tag
- verification requirements and rollout risks

This command is intentionally non-mutating. It does not create channels, update
options, or touch production data.

## Minimal Import

For controlled non-production rollout, use:

```bash
python3 ops/channel-onboarding/import_profile.py \
  --profile ops/channel-onboarding/profiles/yunwu.yaml \
  --target-base-url http://127.0.0.1:3001 \
  --target-root-username root \
  --target-root-password '<password>' \
  --target-environment test
```

With family probes and explicit channel keys:

```bash
python3 ops/channel-onboarding/import_profile.py \
  --profile ops/channel-onboarding/profiles/yunwu.yaml \
  --target-base-url http://127.0.0.1:3001 \
  --target-root-username root \
  --target-root-password '<password>' \
  --target-environment test \
  --probe-upstream \
  --require-channel-keys \
  --channel-key kling='<key>' \
  --channel-key vidu='<key>' \
  --channel-key doubao-video='<key>'
```

Current importer guarantees:

- only allows `test` and `enterprise`
- rejects `production`
- merges `UserUsableGroups`, `GroupRatio`, `ModelRatio`, `CompletionRatio`, and `ModelPrice` instead of replacing unrelated entries
- preserves unrelated existing channels
- supports optional family-level upstream probes before import
- supports explicit per-family channel keys instead of `MANUAL_REQUIRED`

Current importer limitations:

- channel keys remain `MANUAL_REQUIRED`
- pricing for profiles like `yunwu` remains manual-review unless the profile provides explicit values
- test execution still must validate actual upstream compatibility before any production recommendation

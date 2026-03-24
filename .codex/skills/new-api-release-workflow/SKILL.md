---
name: new-api-release-workflow
description: Manage official version selection, private patch migration, release branch creation, and test/production deployment sequencing for this `new-api` repository. Use when deciding which official version is safe for production, comparing stable vs alpha releases, creating or updating `codex/release-*`, moving custom commits from one official tag base to another, or planning deployment and verification across local, `kkidc` test, and `kkidc` production environments.
---

# new-api Release Workflow

## Quick Start

1. Read `AGENTS.md`, `ops/README.md`, and `ops/environments/README.md`.
2. Treat the `kkidc` production environment recorded in `ops/environments/README.md` as the only active production target unless the repo docs explicitly record a change.
4. Run `python3 .codex/skills/new-api-release-workflow/scripts/check_upstream_versions.py` when the task asks for the latest official production-safe version.
5. Read `references/branch-strategy.md` when you need the branch model or the exact upgrade sequence.

## Route The Task

- **Version selection**: Run the bundled version-check script, then prefer the latest non-alpha stable release/tag for production.
- **New upgrade cycle**: Create a new `codex/release-vX.Y.Z-pN` branch from the chosen official stable tag.
- **Private patch migration**: Move only the still-needed custom commits from the previous release branch onto the new release branch.
- **Deployment planning**: Open the relevant file in `ops/checklists/` before proposing or executing deployment steps.

## Apply The Repository Rules

- Keep `main` aligned with official `upstream/main`; do not place custom business changes on `main`.
- Keep private modifications on `codex/release-*` branches and keep each change small enough to cherry-pick cleanly.
- Keep `kkidc` test validation before `kkidc` production rollout.

## Follow The Upgrade Workflow

1. Confirm whether the task is asking for the latest official release, the latest stable release, or the next repo upgrade target.
2. Identify the recommended official base version with the bundled version-check script.
3. Fetch official tags and create a fresh `codex/release-*` branch from the selected stable tag.
4. Compare the previous release branch against its old base tag, then cherry-pick or reimplement only the custom commits that still matter.
5. Validate the new branch locally with the smallest relevant checks first.
6. Validate the branch in the test environment before proposing any production rollout.
7. Deploy `kkidc` production only after test validation succeeds.

## Open The Right Files

- Open `ops/README.md` for the operations entry point.
- Open `ops/environments/README.md` for the current truth about `kkidc` production and `kkidc` test.
- Open `ops/checklists/kkidc-test-release.md` before `kkidc` test deployment work.
- Open `ops/checklists/kkidc-production-release.md` before active `kkidc` production deployment work.

## Use The Bundled Resources

- Use `scripts/check_upstream_versions.py` to summarize the latest official release, latest stable release, and recent stable versions.
- Use `references/branch-strategy.md` for the repo-specific branch model and upgrade command skeleton.
- Prefer reading repo docs rather than copying their full contents into responses.

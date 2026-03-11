---
name: new-api-release-workflow
description: Manage official version selection, private patch migration, release branch creation, test/production deployment sequencing, and `tencent` environment planning for this `new-api` repository. Use when deciding which official version is safe for production, comparing stable vs alpha releases, creating or updating `codex/release-*` and `codex/prod-live`, moving custom commits from one official tag base to another, or planning deployment and verification across `kkidc` test, `kkidc` production, `tencent` test, and `tencent` standby environments.
---

# new-api Release Workflow

## Quick Start

1. Read `AGENTS.md`, `ops/README.md`, and `ops/environments/README.md`.
2. Treat `http://202.140.142.149:3000` as the current `kkidc` production environment unless the repo docs explicitly record a completed cutover.
3. Treat `tencent` standby as non-production until `ops/checklists/tencent-cutover.md` has been fully satisfied.
4. Run `python3 .codex/skills/new-api-release-workflow/scripts/check_upstream_versions.py` when the task asks for the latest official production-safe version.
5. Read `references/branch-strategy.md` when you need the branch model or the exact upgrade sequence.

## Route The Task

- **Version selection**: Run the bundled version-check script, then prefer the latest non-alpha stable release/tag for production.
- **New upgrade cycle**: Create a new `codex/release-vX.Y.Z-pN` branch from the chosen official stable tag.
- **Private patch migration**: Move only the still-needed custom commits from the previous release branch onto the new release branch.
- **Deployment planning**: Open the relevant file in `ops/checklists/` before proposing or executing deployment steps.
- **Tencent test work**: Read `docs/installation/TENCENT_TEST.md` after confirming the task targets the `tencent` test environment.
- **Tencent standby work**: Read `docs/installation/DEPLOYMENT.md` only after confirming the task targets the standby `tencent` environment.

## Apply The Repository Rules

- Keep `main` aligned with official `upstream/main`; do not place custom business changes on `main`.
- Keep private modifications on `codex/release-*` branches and keep each change small enough to cherry-pick cleanly.
- Treat `codex/prod-live` as the deployment branch; move it only after the candidate release branch has been validated.
- Keep `kkidc` test validation before `kkidc` production rollout.
- Keep `tencent` test validation and `tencent` standby cutover as separate tracks from the currently active `kkidc` production flow.

## Follow The Upgrade Workflow

1. Confirm whether the task is asking for the latest official release, the latest stable release, or the next repo upgrade target.
2. Identify the recommended official base version with the bundled version-check script.
3. Fetch official tags and create a fresh `codex/release-*` branch from the selected stable tag.
4. Compare the previous release branch against its old base tag, then cherry-pick or reimplement only the custom commits that still matter.
5. Validate the new branch locally with the smallest relevant checks first.
6. Validate the branch in the test environment before proposing any production rollout.
7. Fast-forward or otherwise cleanly update `codex/prod-live` only after validation succeeds.
8. Deploy `kkidc` production first; use `tencent` test for server-side branch validation and keep `tencent` standby isolated unless the task explicitly requires cutover work.

## Open The Right Files

- Open `ops/README.md` for the operations entry point.
- Open `ops/environments/README.md` for the current truth about `kkidc` production, `kkidc` test, `tencent` test, and `tencent` standby.
- Open `ops/checklists/kkidc-test-release.md` before `kkidc` test deployment work.
- Open `ops/checklists/kkidc-production-release.md` before active `kkidc` production deployment work.
- Open `ops/checklists/tencent-test-release.md` before `tencent` test deployment work.
- Open `ops/checklists/tencent-cutover.md` before any `tencent` cutover recommendation.
- Open `docs/installation/TENCENT_TEST.md` for `tencent` test deployment and recovery.
- Open `docs/installation/DEPLOYMENT.md` only for `tencent` standby deployment, restore, HTTPS, or cutover preparation details.

## Use The Bundled Resources

- Use `scripts/check_upstream_versions.py` to summarize the latest official release, latest stable release, and recent stable versions.
- Use `references/branch-strategy.md` for the repo-specific branch model and upgrade command skeleton.
- Prefer reading repo docs rather than copying their full contents into responses.

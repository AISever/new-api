# new-api Branch And Upgrade Strategy

## 1. Repository truth

- `upstream`: official repository `QuantumNous/new-api`
- `origin`: your working repository `AISever/new-api`
- `main`: keep aligned with `upstream/main`
- `codex/release-vX.Y.Z-pN`: custom release branch based on one official stable tag

## 2. Environment truth

- `kkidc` production: see `ops/environments/README.md`
- `kkidc` test: see `ops/environments/README.md`

Read `ops/environments/README.md` before any deployment advice.

## 3. Version selection rule

- Prefer the latest non-alpha stable release/tag for production.
- Treat `alpha`, `beta`, `rc`, `preview`, and similar suffixes as non-production candidates unless the user explicitly asks for them.
- If the latest upstream release is prerelease-only, keep the current stable line until a new stable patch/release appears.

## 4. Upgrade sequence

1. Fetch official tags.
2. Create a new `codex/release-*` branch from the chosen stable tag.
3. Identify custom commits from the previous release branch relative to its base tag.
4. Cherry-pick only the commits that still matter.
5. Resolve conflicts with minimum divergence from upstream.
6. Validate locally, then in test.
7. Deploy active production only after test passes.

## 5. Command skeleton

```bash
git fetch upstream --tags
git switch main
git pull --ff-only upstream main

git switch -c codex/release-vX.Y.Z-pN vX.Y.Z
git cherry-pick <commit>...

```

## 6. Deployment entry points

- `kkidc` test: `ops/checklists/kkidc-test-release.md`
- `kkidc` production: `ops/checklists/kkidc-production-release.md`

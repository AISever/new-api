# Upstream Stable Upgrade Gate Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Add a repeatable upstream stable upgrade workflow that stays compatible with the kkidc environment operations model and blocks risky releases before they reach production.

**Architecture:** Keep the existing `kkidc-host-*` scripts as the only environment mutation entrypoints, then add one upgrade-specific gate script that runs local regression/build checks and, when requested, delegates test-environment deployment and stop actions to the official `kkidc-host-deploy.sh` flow. Document the workflow in `ops` so future stable upgrades use the same gate and release checklists.

**Tech Stack:** Bash, Go tests, Bun/Vite build, existing `ops/tests/*.sh` harnesses, kkidc deploy scripts.

### Task 1: Document the new upgrade workflow

**Files:**
- Create: `ops/checklists/kkidc-upstream-stable-upgrade.md`
- Modify: `ops/README.md`
- Modify: `ops/RUNBOOK.md`
- Modify: `ops/checklists/kkidc-test-release.md`
- Modify: `ops/checklists/kkidc-production-release.md`
- Modify: `ops/checklists/kkidc-enterprise-production-release.md`

**Step 1: Write the upgrade-specific checklist**

Describe the mandatory order:
1. rebase/cherry-pick upgrade work
2. run local upgrade gate
3. run test upgrade gate
4. only then use the normal production checklist

**Step 2: Link the checklist from the ops index and runbook**

Add a short “official stable upgrade” section that points to the new checklist and gate script without replacing the existing `kkidc-host-*` command set.

**Step 3: Patch the existing release checklists**

Add one explicit preflight item that says: if this release is an official stable-version upgrade, the upgrade gate must pass first.

### Task 2: Add the executable upgrade gate

**Files:**
- Create: `ops/scripts/kkidc-host-upgrade-gate.sh`

**Step 1: Write the failing shell test first**

Create a shell test that expects the gate to:
- run the local regression/build suite
- delegate test deployment to `ops/scripts/kkidc-host-deploy.sh test`
- stop the test environment afterward unless explicitly told to keep it running

**Step 2: Run the test and verify it fails**

Run the new shell test before the script exists or before behavior is implemented.

**Step 3: Implement the minimal gate**

The script should support:
- `local`: run local upgrade checks only
- `test`: run local checks, verify pushed `HEAD`, call `kkidc-host-deploy.sh test`, verify `/api/status`, then `test-stop` by default
- `--keep-test-running`: skip `test-stop`

**Step 4: Keep environment mutations delegated**

Do not reimplement remote deploy logic in the gate. Call the official deploy script directly so the gate remains compatible with `kkidc-env-ops`.

### Task 3: Back the gate with automated tests

**Files:**
- Create: `ops/tests/kkidc-host-upgrade-gate-test.sh`

**Step 1: Stub child commands**

Use a temp repo and stub `go`, `bun`, `curl`, and child `bash` calls so the test can verify the control flow deterministically.

**Step 2: Verify default test-mode behavior**

Ensure the gate invokes:
- `go test ./controller ./model -count=1`
- the selected `ops/tests/kkidc-host-deploy-*.sh` local script tests
- `cd web && bun run build`
- `ops/scripts/kkidc-host-deploy.sh test`
- `ops/scripts/kkidc-host-deploy.sh test-stop`

**Step 3: Verify keep-running mode**

Ensure `--keep-test-running` still deploys and verifies the test environment but skips `test-stop`.

### Task 4: Verify the real suite and keep it green

**Files:**
- Test: `controller/user_test.go`
- Test: `ops/tests/kkidc-host-upgrade-gate-test.sh`
- Test: `ops/tests/kkidc-host-deploy-pushed-head-test.sh`
- Test: `ops/tests/kkidc-host-deploy-build-strategy-test.sh`
- Test: `ops/tests/kkidc-host-deploy-local-build-platform-test.sh`

**Step 1: Run the new gate shell test**

Run: `bash ops/tests/kkidc-host-upgrade-gate-test.sh`
Expected: PASS

**Step 2: Run the regression/backend suite used by the gate**

Run: `go test ./controller ./model -count=1`
Expected: PASS

**Step 3: Run the deploy-script smoke tests used by the gate**

Run:
- `bash ops/tests/kkidc-host-deploy-pushed-head-test.sh`
- `bash ops/tests/kkidc-host-deploy-build-strategy-test.sh`
- `bash ops/tests/kkidc-host-deploy-local-build-platform-test.sh`

Expected: PASS

**Step 4: Run the frontend build used by the gate**

Run: `cd web && bun run build`
Expected: PASS

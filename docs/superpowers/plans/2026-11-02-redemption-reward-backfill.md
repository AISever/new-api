# Historical Redemption Reward Backfill Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [x]`) syntax for tracking.

**Goal:** Deliver a dry-run-first, restartable historical reward tool with one inviter-visible log per credited redemption.

**Architecture:** Standalone Go command, no application release or startup migrations. Freeze an explicit cutoff and per-code facts in a hashed plan; create only a dedicated durable ledger on apply. Each code locks/rechecks facts and credits ledger, inviter rewards and system log in one database transaction.

**Tech Stack:** Go, GORM, SQLite/MySQL/PostgreSQL, shopspring decimal, common JSON wrappers.

**Spec:** ../specs/2026-11-02-redemption-reward-backfill-design.md

## Global Constraints

- Current default ratio 5%, personal ratio wins including zero; no reconstructed historical settings.
- No production apply in this task. Preserve wallet quota, subscriptions, fhy 131 reset anchors, options and amount visibility.
- No full redemption key in files, logs, or stdout. No credentials in arguments or reports.
- Only ledger creation on apply; preview performs reads only and refuses to overwrite an output file.
- Support SQLite, MySQL >= 5.7.8, PostgreSQL >= 9.6; reject separate log database for atomic credit/log delivery.

## Review Focus

- A changed invite relationship/config/credited subscription after preview invalidates execution.
- Log insertion failure must leave neither rewards nor permanent success record.
- Concurrent jobs and retries after log cleanup must never credit twice.
- Plan edits, duplicate IDs and an incorrect digest must fail before any database writes.
- Historical subscriptions use granted total even when expired/consumed; no future daily multipliers.

### Task 1: Preview and exact calculation

**Files:** Create `ops/tools/redemption-reward-backfill/backfill.go`, `backfill_test.go`.
**Interfaces:** `Preview(db *gorm.DB, before int64) (Plan,error)`; `Plan` holds version, cutoff, settings, entries (including skip reasons), dataset fingerprint and exact total. `Digest(v any) (string,error)` uses common.Marshal.

- [x] Write tests: quota 500001 at 5% gives 25000; personal 10% wins, explicit 0 skips; subscription 5000000 gives 250000 regardless of quota/used/duration; no inviter/deleted/disabled/unknown basis skip; preview does not create ledger or modify data.
- [x] Run failing tests, then implement projections/decimal calculation/settings validation and plan serialization; run passing tests and commit.

### Task 2: Atomic restartable apply and CLI

**Files:** Extend `backfill.go`, `backfill_test.go`; create `main.go`, `main_test.go`.
**Interfaces:** `Apply(db *gorm.DB, plan Plan, digest string) (ApplyResult,error)`; `run(args []string, stdout io.Writer) error` reads BACKFILL_DSN and driver flags; preview writes new 0600 plan; apply requires exact SHA256 and rejects LOG_SQL_DSN.

- [x] Write failing tests for once-only credit/log, retry after logs removed, 8 concurrent attempts, new plans after completion, existing native reward exclusion, logger failure rollback, overflow rollback, source/settings drift, plan tampering/duplicate rows and wrong digest.
- [x] Implement unique redemption ledger, per-code transaction and row locks, preflight and in-transaction revalidation. Preserve unrelated user fields and all source records. Use ordinary system log ownership and safe content.
- [x] Write failing CLI tests for invalid input/read-only default/output overwrite and secret-safe error paths; implement explicit DSN env handling, driver selection and sanitized errors; run tests and commit.

### Task 3: Database, API and viewer acceptance

**Files:** Create `external_test.go`, `README.md`, `docs/superpowers/results/2026-11-02-redemption-reward-backfill/README.md` and sanitized evidence.
**Interfaces:** Opt-in external tests accept only local test DBs; isolated remote viewer uses existing labeled test database/application, never production.

- [x] Run SQLite suite/race and actual PostgreSQL/MySQL same-code and shared-inviter concurrency tests; inject log failure and verify transactional rollback.
- [x] Build command, preview production read-only if access remains valid, compare candidate totals to independent read-only audit; never apply production.
- [x] Apply only synthetic historical fixtures in isolated test DB; verify log endpoint ownership and actual browser rendering for inviter, denial for unrelated user, no change to original fixtures/settings/subscriptions.
- [x] Run project Go suite, review whole change using a fresh reviewer, fix findings with red/green tests. Record usage, exact verified boundaries and production apply/recovery instructions. Commit the deliverable without main merge or production release.

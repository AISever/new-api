# Invitation Details Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 点击邀请人数查看本人直接受邀用户名，按集中后台权限安全展示充值实付、订阅实付与兑换折算。

**Architecture:** 一个 JSON option 原子保存 off/all/selected 与用户 ID，每次请求读取数据库，默认关闭。聚合接口仅作用于当前用户直接受邀名单；新增独立金额快照保存已核验实付/兑换折算，旧记录明确未确认，订阅镜像按交易号排除。React 弹窗与后台设置复用 Semi UI，金额响应不缓存。

**Tech Stack:** Go/Gin/GORM、shopspring/decimal、React18/Semi UI/Bun、SQLite/MySQL/PostgreSQL。

**Spec:** `docs/superpowers/specs/2026-11-01-invitation-details-design.md`（用户已确认）。

## Global Constraints

- 业务 JSON 使用 common 包封装；数据库代码兼容 SQLite / MySQL >=5.7.8 / PostgreSQL >=9.6。
- 默认只展示用户名；金额授权只由 RootAuth 配置；不接受客户端指定 inviter_id。
- 保留余额、奖励、订阅已用额度/零点锚点、品牌和既有财务字段语义。
- 不查询或修改生产数据，不自动部署；不同币种分项，历史未知金额不假装为 0。
- 兑换快照失败不改变兑换/支付成功结果；报告可重试补录但不能重复记账。

## Review Focus

- 旧已用兑换码软删除后仍计入；没有来源数据的历史覆盖范围明确说明。（Task2）
- 实付为 0、缺币种、折算配置变化分别保持零值、未知与历史快照语义。（Task2）
- 权限撤销、会话切换、迟到的旧请求不能重新显示上个账号的金额。（Task3）
- 搜索包含 SQL 通配符、超大/负分页不能越过所属用户或造成无界分页。（Task1）
- 订阅成功时的充值镜像及重复回调都只计一次，不影响原有返奖。（Task2）

---

### Task 1: Scoped invitation endpoint and atomic visibility option

**Files:** Create `model/invitation.go`, `model/invitation_test.go`, `controller/invitation.go`, `controller/invitation_test.go`; Modify `controller/option.go`, `model/option.go`, `router/api-router.go`.

**Interfaces:** Produces `InvitationVisibility{Mode string, UserIDs []int}`, constant `InvitationVisibilityOption`, `ValidateInvitationVisibility(raw string) (string,error)`, `CanViewInvitationAmounts(userID int) bool`, `GetInvitationPage(inviterID,page,pageSize int,keyword string,withAmounts bool) (*InvitationPage,error)`. DTO contains items (username only by default), total/page/page_size/can_view_amounts; optional `amounts` introduced in Task2. GET `/api/user/self/invitees`; root user-choice GET `/api/option/invitation-users?keyword=&ids=`.

- [ ] Write tests: unauthenticated/other inviter cannot query, zero/default and selected misses omit amounts, selected grant/revoke survives reading DB, malformed policy rejects, deleting option restores default off; literal `%` query, negative/huge pagination and disabled/deleted users.
- [ ] Run `go test ./model ./controller -run Invitation -count=1` and observe feature missing before implementation.
- [ ] Implement DB-backed fail-closed permission, scoped minimal username query, bounded pagination and root-only minimal user choices; validate/single-write JSON option and return actual DB errors.
- [ ] Run same tests, check existing user/option tests; commit `feat: add scoped invitation list and amount visibility policy`.

### Task 2: Exact amount snapshots and duplicate-free aggregation

**Files:** Create `model/invitation_amount.go`, `model/invitation_amount_test.go`, `controller/invitation_payment.go`, `controller/invitation_payment_test.go`; Modify `model/main.go`, `model/redemption.go`, `model/topup.go`, successful verified handlers in `controller/topup{,_stripe,_creem,_waffo,_waffo_pancake}.go` and subscription payment handlers.

**Interfaces:** Consumes Task1 DTO/permission; produces `InvitationAmountSnapshot` (unique source kind/source ID, immutable user ID, decimal string amount, currency, basis, quota/rate snapshot), `RecordInvitationPayment(tradeNo,provider,amount,currency string) error`, `RecordInvitationRedemption(redemption *Redemption) error`, and `AttachInvitationAmounts(items []InvitationUser) error`. DTO `amounts` contains currencies with topup/subscription/redemption/total strings, estimated flags, unknown/unpriced counts and recorded legacy amounts.

- [ ] Tests seed a directly invited user with paid topup 2.10 USD, subscription 3.20 USD plus duplicate topup mirror and redemption 1 USD: total must equal literal 6.30, not 9.50; another currency remains separate.
- [ ] Tests prove signed-success capture only, unpaid/failed exclusion, zero paid amount, unknown historical values not counted as paid, soft-deleted used redemption counted, configuration changes preserve redemption snapshot, unlimited subscription redemption unpriced, idempotent capture/redo no wallet/reward changes.
- [ ] Run failing tests before implementation, then implement unique idempotent snapshots independent of financial fields, streamed bounded-memory per-page aggregation, safe provider normalization and verified-success hooks; migrations added to both ordinary/fast GORM paths.
- [ ] Run `go test ./model ./controller -count=1`, targeted race checks; commit `feat: aggregate invitation payments and redemption values safely`.

### Task 3: Invitation dialog and admin policy editor

**Files:** Create `web/src/components/topup/modals/InvitationUsersModal.jsx`, `web/src/components/topup/utils/invitationDetails.js` and `.test.mjs`, `web/src/pages/Setting/Operation/SettingsInvitationVisibility.jsx`; Modify `InvitationCard.jsx`, `components/settings/OperationSetting.jsx`, seven locale JSON files.

**Interfaces:** Consumes Task1 page/root user-choice API and Task2 amount DTO; saves single `InvitationAmountVisibility` JSON option. Modal receives current viewer ID and open state; stale responses have request-generation fencing, authorization/session changes remove old amounts.

- [ ] Write behavior tests for response normalization, missing vs zero amounts, option serialization, selected ID precision/uniqueness and request fences that reject late old-account/page responses; run failing tests.
- [ ] Implement accessible click target, paginated username search, amount category/currency rows, empty/loading/error/retry and unknown/estimate notices; selected user search/save/reload without serializing full User or treating success=false as saved.
- [ ] Run Node tests, Bun production build, targeted translation coverage; commit `feat: expose invitation details and admin visibility controls`.

### Task 4: Isolated acceptance, branch review and handoff

**Files:** Add `ops/tests/invitation_acceptance.go` or equivalent isolated integration harness, and `docs/superpowers/results/2026-11-01-invitation-details.md`.

**Interfaces:** Consumes all preceding interfaces; fixtures are independent of production, real external payment is never claimed from fixtures.

- [ ] Run full `go test ./...`, frontend test files and build; real SQLite/MySQL/PostgreSQL migration+aggregate fixtures; inspect bounded query behavior and no credentials in DTO/evidence.
- [ ] Start loopback-only fixture app and use actual browser to check invited list/default hiding, authorized sums, admin selected search/save/reload, revoked policy, account switch, paging and narrow viewport.
- [ ] Obtain fresh whole-branch reviewer against spec/plan; critical/important findings get reproducing RED→GREEN fixes and regression suite, minors recorded honestly.
- [ ] Preserve final evidence, source commits and exact validation gaps; no production deployment, push or merge. Commit final report.

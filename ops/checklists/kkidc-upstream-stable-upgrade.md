# kkidc 官方稳定版升级检查清单

适用对象：从官方 `upstream` 稳定版本同步到当前 `codex/release-*` 发布线的升级动作。

目标：把“官方稳定版升级”从一次性的经验操作，变成和 `kkidc-env-ops` 兼容的固定流程。

## 1. 原则

- 升级流程不能绕开现有正式环境脚本。
- 涉及测试环境的部署、验证、关闭，仍然使用 `ops/scripts/kkidc-host-deploy.sh test` / `test-stop`。
- 生产环境发布仍然分别走：
  - `ops/checklists/kkidc-production-release.md`
  - `ops/checklists/kkidc-enterprise-production-release.md`
- 本清单只是升级专用门禁，不替代正式发布脚本。

## 2. 升级前准备

- 确认目标是官方最新稳定标签，而不是 nightly 或 alpha。
- 记录当前基线：
  - 当前发布分支
  - 当前已上线版本
  - 准备同步的官方 tag
- 先阅读：
  - `ops/README.md`
  - `ops/RUNBOOK.md`
  - `ops/environments/README.md`
  - 本清单
- 将官方升级改动与自定义补丁分层处理，避免把“官方变化”和“本地业务补丁”混在一个提交里。

## 3. 本地升级门禁

先执行：

```bash
bash ops/scripts/kkidc-host-upgrade-gate.sh local
```

门禁通过前，不进入任何服务器环境。

本地门禁至少覆盖：

- `go test ./controller ./model -count=1`
- `ops/tests/kkidc-host-deploy-pushed-head-test.sh`
- `ops/tests/kkidc-host-deploy-build-strategy-test.sh`
- `ops/tests/kkidc-host-deploy-local-build-platform-test.sh`
- `cd web && bun run build`

## 4. 测试环境升级门禁

确认当前 `HEAD` 已提交且已 push 后，执行：

```bash
bash ops/scripts/kkidc-host-upgrade-gate.sh test
```

默认行为：

- 先重跑本地门禁
- 再通过正式脚本部署 `kkidc` 测试环境
- 验证 `http://114.66.47.192:3001/api/status`
- 默认执行 `test-stop`

如需要保留测试环境给人工继续验证，显式执行：

```bash
bash ops/scripts/kkidc-host-upgrade-gate.sh test --keep-test-running
```

## 5. 人工重点复核项

每次官方稳定版升级，至少人工复核以下高风险流程：

- 用户管理：
  - 列表页充值
  - 编辑页调整额度
- 兑换码：
  - 金额兑换
  - 订阅套餐兑换
- 公告：
  - 系统设置 -> 仪表盘 -> 添加公告
- 渠道与分组：
  - 渠道失效后的替换与分组接管
  - 上游模型同步
- 企业文档：
  - `/docs`
  - `docs_manifest_path`
  - Logo / favicon / 帮助页入口

## 6. 进入生产环境前

只有在以下条件全部满足时，才允许进入生产发布清单：

- 本地升级门禁通过
- 测试环境升级门禁通过
- 人工重点复核通过
- 已明确本次目标环境：
  - 个人生产
  - 企业生产

随后再按对应正式清单继续，不得直接跳过：

- 个人生产：`ops/checklists/kkidc-production-release.md`
- 企业生产：`ops/checklists/kkidc-enterprise-production-release.md`

## 7. 失败时处理

- 本地门禁失败：先修本地代码或脚本问题，不进入服务器环境。
- 测试环境门禁失败：先停在测试环境，不进入生产环境。
- 若是环境操作失败，按 `kkidc-env-ops` 口径继续：
  - 读取 `ops/RUNBOOK.md`
  - 读取 `ops/environments/README.md`
  - 仅使用正式 `kkidc-host-*` 脚本排查
- 不允许因为赶时间，临时改为手工 SSH 操作替代正式入口。

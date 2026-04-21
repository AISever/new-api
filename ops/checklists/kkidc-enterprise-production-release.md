# kkidc 企业生产环境发布检查清单

适用对象：建议 `https://corp-api.aisever.cn`

## 发布前

- 先阅读 `ops/environments/kkidc-enterprise-production.md`。
- 确认本次改动已经在 `kkidc` 测试环境验证通过。
- 如果本次属于“官方稳定版升级”，确认 `ops/scripts/kkidc-host-upgrade-gate.sh test` 已通过，并记录对应分支与提交号。
- 确认当前操作目标是 `kkidc` 企业生产环境，而不是个人生产环境。
- 确认待部署提交已经 push 到远端校验分支。
- 在共享主机上首次上线企业环境前，先执行 `bash ops/scripts/kkidc-host-backup.sh production`，保留个人生产环境回滚点。
- 若企业环境需要继承现有生产配置或素材，记录计划导入的生产备份目录，并通过 `bash ops/scripts/kkidc-host-restore.sh enterprise --source-backup-dir <production-backup-dir>` 初始化。
- 企业环境已有线上数据后，如涉及数据库、配置、数据迁移，再执行 `bash ops/scripts/kkidc-host-backup.sh enterprise`，并记录回滚方案与备份目录。
- 如需清空企业环境后重新导入，先执行 `bash ops/scripts/kkidc-host-backup.sh enterprise`，再执行 `bash ops/scripts/kkidc-host-reset.sh enterprise`，并记录脚本输出的备份目录。
- 如需接入新的外部上游，提前记录上游地址、账号、目标环境地址，并使用 `bash ops/scripts/kkidc-host-import-upstream.sh enterprise ...`，不要手工拼装 API 调用。
- 若计划直接启用公网域名，先确认 `corp-api.aisever.cn` 已解析到 `114.66.47.192`；若未完成 DNS 切流，则保持 `ENTERPRISE_HOSTNAME` 为空，仅通过 `:3002` 验证。
- 默认走本地构建发布；只有在明确受控窗口内，才允许显式 `BUILD_STRATEGY=remote`。
- 如需显式 `BUILD_STRATEGY=remote`，先记录当前主机资源：
  - `MemAvailable >= 2048MB`
  - `load1 <= 4.00`
  - 企业生产 `/api/status` 当前正常

## 发布中

- 执行：`bash ops/scripts/kkidc-host-deploy.sh enterprise`
- 记录部署分支、提交 SHA、执行时间、操作者。
- 若本次未配置 `ENTERPRISE_HOSTNAME` 或生成的 Caddy 配置未变化，确认部署输出显示未重启 `snowlight-caddy`。
- 如本次显式使用 `BUILD_STRATEGY=remote`，记录门禁检查结果和脚本输出的阶段耗时。
- 如需改动配置文件或数据库 option，逐项记录实际变更。
- 如执行了企业重置或上游导入，记录目标 base URL、脚本参数、导入的分组数量、渠道数量、模型覆盖数。

## 发布后验证

- 若 DNS 已切流：`curl https://corp-api.aisever.cn/api/status`
- 若 DNS 未切流：`curl http://114.66.47.192:3002/api/status`
- 额外执行：`curl https://api.aisever.cn/api/status`，确认个人生产环境未受影响。
- 检查后台核心页面、关键 API、日志写入是否正常。
- 若执行了上游导入，额外核对企业后台中的 `UserUsableGroups`、`AutoGroups`、主要渠道数量，以及至少一条 OpenAI / Anthropic / Gemini 链路的模型可见性。
- 若执行了上游导入，额外执行一次缺失模型检查和官方模型同步结果确认，确保企业环境模型元数据可用。
- 针对企业客户关键链路做最小闭环验证。
- 观察容器状态、错误日志、计费相关日志。

## 回滚

- 回滚到上一个已验证可用版本。
- 若涉及数据变更，按备份方案恢复。
- 回滚完成后重新执行健康检查与关键链路验证。

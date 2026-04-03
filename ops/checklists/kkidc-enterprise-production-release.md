# kkidc 企业生产环境发布检查清单

适用对象：建议 `https://corp-api.aisever.cn`

## 发布前

- 先阅读 `ops/environments/kkidc-enterprise-production.md`。
- 确认本次改动已经在 `kkidc` 测试环境验证通过。
- 确认当前操作目标是 `kkidc` 企业生产环境，而不是个人生产环境。
- 确认待部署提交已经 push 到远端校验分支。
- 如涉及数据库、配置、数据迁移，先执行 `bash ops/scripts/kkidc-host-backup.sh enterprise`，并记录回滚方案与备份目录。
- 默认走本地构建发布；只有在明确受控窗口内，才允许显式 `BUILD_STRATEGY=remote`。
- 如需显式 `BUILD_STRATEGY=remote`，先记录当前主机资源：
  - `MemAvailable >= 2048MB`
  - `load1 <= 4.00`
  - 企业生产 `/api/status` 当前正常

## 发布中

- 执行：`bash ops/scripts/kkidc-host-deploy.sh enterprise`
- 记录部署分支、提交 SHA、执行时间、操作者。
- 如本次显式使用 `BUILD_STRATEGY=remote`，记录门禁检查结果和脚本输出的阶段耗时。
- 如需改动配置文件或数据库 option，逐项记录实际变更。

## 发布后验证

- `curl https://corp-api.aisever.cn/api/status`
- 检查后台核心页面、关键 API、日志写入是否正常。
- 针对企业客户关键链路做最小闭环验证。
- 观察容器状态、错误日志、计费相关日志。

## 回滚

- 回滚到上一个已验证可用版本。
- 若涉及数据变更，按备份方案恢复。
- 回滚完成后重新执行健康检查与关键链路验证。

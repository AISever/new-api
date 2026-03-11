# tencent 测试环境发布检查清单

适用对象：`http://123.206.229.105:3001`

## 发布前

- 先阅读 `ops/environments/tencent-test.md`。
- 明确本次目标分支，优先使用待验证的 `codex/release-*` 分支。
- 确认不会误操作 `tencent` 预备生产环境 `:3000`。
- 如需测试接近生产的数据，先准备好备份目录或恢复方案。

## 发布中

- 按 `docs/installation/TENCENT_TEST.md` 或 `ops/scripts/tencent-test-deploy.sh` 执行部署。
- 记录部署分支、提交 SHA、部署时间。
- 如有配置变更，记录 `.env`、数据库 option、面板配置的实际变更。

## 发布后验证

- `curl http://123.206.229.105:3001/api/status`
- 验证后台登录、关键页面、核心 API、日志写入。
- 验证本次改动对应主流程。
- 观察容器状态、错误日志、计费日志。

## 回滚

- 回滚到上一个已验证可用的候选分支或镜像。
- 若本次测试引入了数据变更，按备份方案恢复。
- 回滚后重新验证 `/api/status`、后台登录、关键链路。

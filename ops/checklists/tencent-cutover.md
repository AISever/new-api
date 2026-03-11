# tencent 切流检查清单

适用对象：`http://123.206.229.105:3000` / `https://api.aisever.art`

## 切流前提

- 先阅读 `ops/environments/tencent-standby.md`。
- 如本次切流包含代码变更，优先完成 `ops/checklists/tencent-test-release.md` 验证。
- 当前 `kkidc` 生产与 `tencent` 数据已完成最新一次同步。
- `tencent` 环境部署成功，`/api/status` 正常。
- `api.aisever.art` 的 DNS、80/443、证书签发、反向代理均已验证通过。
- 已明确回切方案：出现问题时恢复 `kkidc` 生产对外流量。

## 切流前验证

- `curl http://123.206.229.105:3000/api/status`
- `curl -I https://api.aisever.art/api/status`
- 验证后台登录、核心页面、关键 API、日志写入。
- 确认 `X-New-Api-Version` 或 `/api/status` 中版本信息符合预期。

## 切流动作

- 在低峰窗口执行。
- 先确认 DNS / 代理入口变更策略。
- 变更完成后立刻验证域名访问、后台功能、主要请求链路。

## 切流后观察

- 持续观察错误日志、使用日志、计费日志。
- 对比 `kkidc` 生产与 `tencent` 环境的关键指标。
- 保留 `kkidc` 生产环境一段时间，作为快速回切入口。

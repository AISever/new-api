# kkidc 生产环境

## 1. 环境定位

- 状态：**当前真实生产环境**
- 服务器：`43.133.183.213`
- 对外地址：`https://api.aisever.cn`
- 备用访问：`https://newapi.aisever.cn`
- 说明：这是当前真正承接线上流量的 `kkidc` 个人生产环境，当前已切换到最新腾讯云服务器，主要服务个人用户。

## 2. 运行信息

| 项目 | 值 |
| --- | --- |
| 容器名 | `new-api-local` |
| 当前应用源码 | `ba844d202b683219dadc679af3521dc1d8abde9f` |
| 当前运行版本 | `codex/sec-v0.13.2-p1+ba844d202+production` |
| 当前镜像 | `new-api:security-ba844d202`，不可变 ID 见发布报告 |
| 应用端口 | `3000` |
| 数据库 | `new-api` |
| Redis | `new-api-redis`（默认库） |
| 挂载目录 | `/opt/new-api/data`、`/opt/new-api/logs` |
| 构建目录 | `/opt/new-api-build-production` |
| 统一入口脚本 | `ops/scripts/kkidc-host-deploy.sh production` |

## 3. 管理注意事项

- 这是**当前启用中的个人生产**，变更前必须先在 `kkidc` 测试环境验证。
- 2026-10-19 本次安全发布复用已验证镜像，克隆现场 Docker 运行参数，保留原应用容器用于回滚。现场容器没有 Compose 管理标签；不能仅凭统一脚本假定线上实际参数，操作前须重新 inspect 核对。
- 发布、备份恢复验证和回滚记录见 [ba844d202 生产发布报告](../releases/security-20261018/RESULTS.md)。
- 主入口脚本：`ops/scripts/kkidc-host-deploy.sh production`
- 企业客户独立生产环境请查看 `ops/environments/kkidc-enterprise-production.md`。
- 如果需要为该环境补充更细的操作手册，建议新增独立文档并保持 `kkidc` 口径一致。

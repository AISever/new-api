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
| 应用端口 | `3000` |
| 数据库 | `new-api` |
| Redis | `new-api-redis`（默认库） |
| 挂载目录 | `/opt/new-api/data`、`/opt/new-api/logs` |
| 构建目录 | `/opt/new-api-build-production` |
| 统一入口脚本 | `ops/scripts/kkidc-host-deploy.sh production` |

## 3. 管理注意事项

- 这是**当前启用中的个人生产**，变更前必须先在 `kkidc` 测试环境验证。
- 当前生产由新 `kkidc` 主机统一脚本维护。
- 主入口脚本：`ops/scripts/kkidc-host-deploy.sh production`
- 企业客户独立生产环境请查看 `ops/environments/kkidc-enterprise-production.md`。
- 如果需要为该环境补充更细的操作手册，建议新增独立文档并保持 `kkidc` 口径一致。

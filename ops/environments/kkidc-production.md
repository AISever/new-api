# kkidc 生产环境

## 1. 环境定位

- 状态：**当前真实生产环境**
- 服务器：`114.66.47.192`
- 对外地址：`https://api.aisever.cn`
- 备用访问：`https://newapi.aisever.cn`
- 说明：这是当前真正承接线上流量的 `kkidc` 生产环境。旧主机 `202.140.142.149` 仍保留，暂不下线。

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

- 这是**当前唯一真实生产**，变更前必须先在 `kkidc` 测试环境验证。
- 当前生产由新 `kkidc` 主机统一脚本维护，不再以旧 `rsync` 生产流程作为主入口。
- 旧主机 `202.140.142.149` 仍作为回退目标和生产数据补采来源保留。
- 主入口脚本：`ops/scripts/kkidc-host-deploy.sh production`
- 历史脚本：`ops/scripts/kkidc-production-deploy.sh`
- 历史 compose：`ops/compose/kkidc-production.yml`
- 如果需要为该环境补充更细的操作手册，建议新增独立文档，不要和 `tencent` 手册混写。

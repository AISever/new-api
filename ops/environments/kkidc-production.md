# kkidc 生产环境

## 1. 环境定位

- 状态：**当前真实生产环境**
- 服务器：`202.140.142.149`
- 对外地址：`http://202.140.142.149:3000`
- 说明：这是目前真正承接线上流量的环境，对应 `kkidc` 服务器，不是 `tencent` 服务器。

## 2. 运行信息

| 项目 | 值 |
| --- | --- |
| 容器名 | `new-api-local` |
| 应用端口 | `3000` |
| 数据库 | `new-api` |
| Redis | `new-api-redis`（默认库） |
| 挂载目录 | `/opt/new-api/data`、`/opt/new-api/logs` |
| 历史构建目录 | `/tmp/new-api-build` |

## 3. 管理注意事项

- 这是**当前唯一真实生产**，变更前必须先在 `kkidc` 测试环境验证。
- 该环境的信息主要来自 `main-bak` 中保留下来的历史部署说明。
- 该环境使用 `kkidc` 体系的脚本与 compose，不与 `tencent` 预备生产流程混用。
- 对应部署脚本：`ops/scripts/kkidc-production-deploy.sh`
- 对应 compose：`ops/compose/kkidc-production.yml`
- 如果需要为该环境补充更细的操作手册，建议新增独立文档，不要和 `tencent` 手册混写。

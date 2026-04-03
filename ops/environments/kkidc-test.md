# kkidc 测试环境

## 1. 环境定位

- 状态：**按需启动**
- 服务器：`114.66.47.192`
- 对外地址：`http://114.66.47.192:3001`
- 用途：新功能、修复项、发布前验证

## 2. 运行信息

| 项目 | 值 |
| --- | --- |
| 容器名 | `new-api-test` |
| 应用端口 | `3001` |
| 数据库 | `new-api-test` |
| Redis | `new-api-redis` 的 `db 1` |
| 数据目录 | `/opt/new-api-test/data` |
| 日志目录 | `/opt/new-api-test/logs` |

## 3. 管理注意事项

- 所有准备上线 `kkidc` 生产环境的改动，都应先在本环境验证。
- 该环境与 `kkidc` 生产环境同机部署，但数据、端口、Redis 库相互隔离。
- 新 `kkidc` 服务器统一脚本：`ops/scripts/kkidc-host-deploy.sh test`
- 测试完成后必须执行：`ops/scripts/kkidc-host-deploy.sh test-stop`
- `test-stop` 只停止 `new-api-test`，保留测试数据目录、数据库和 Redis `db 1`

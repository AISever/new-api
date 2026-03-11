# tencent 测试环境

## 1. 环境定位

- 状态：**已部署可用**
- 服务器：`123.206.229.105`
- 访问地址：`http://123.206.229.105:3001`
- 说明：该环境用于在 `tencent` 服务器上验证候选发布分支，和 `tencent` 预备生产环境分离运行。

## 2. 运行信息

| 项目 | 值 |
| --- | --- |
| 服务器目录 | `/opt/new-api-test-src` |
| Compose 项目名 | `new-api-test` |
| 容器名 | `new-api-test` |
| PostgreSQL 容器 | `postgres-test` |
| Redis 容器 | `redis-test` |
| 应用端口 | `3001` |
| 数据库 | `new-api-test` |
| 数据目录 | `/opt/new-api-test-src/data` |
| 日志目录 | `/opt/new-api-test-src/logs` |

## 3. 管理注意事项

- 该环境用于验证 `codex/release-*` 等候选分支在 `tencent` 服务器上的真实运行情况。
- 该环境与 `tencent` 预备生产环境使用同一台服务器，但目录、容器名、端口、数据库全部隔离。
- 对应部署脚本：`ops/scripts/tencent-test-deploy.sh`
- 共享部署核心：`ops/scripts/deploy-from-branch.sh`
- 对应恢复脚本：`ops/scripts/tencent-test-restore-from-backup.sh`
- 对应 compose 模板：`ops/compose/tencent-test.yml`
- 详细部署说明：`docs/installation/TENCENT_TEST.md`

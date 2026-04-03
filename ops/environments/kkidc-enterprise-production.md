# kkidc 企业生产环境

## 1. 环境定位

- 状态：**独立部署 / 待公网切流**
- 服务器：`114.66.47.192`
- 建议对外地址：`https://corp-api.aisever.cn`
- 可选别名：`https://enterprise.aisever.cn`
- 说明：该环境作为企业客户独立生产环境，与当前个人生产环境隔离运行；公网域名未切到新主机前，可先以 `3002` 直连方式验证。

## 2. 规划运行信息

| 项目 | 值 |
| --- | --- |
| 容器名 | `new-api-enterprise` |
| 应用端口 | `3002` |
| 数据库 | `new-api-enterprise` |
| Redis | `new-api-redis`（`db 2`） |
| 挂载目录 | `/opt/new-api-enterprise/data`、`/opt/new-api-enterprise/logs` |
| 构建目录 | `/opt/new-api-build-enterprise` |
| 统一入口脚本 | `ops/scripts/kkidc-host-deploy.sh enterprise` |
| 统一备份脚本 | `ops/scripts/kkidc-host-backup.sh enterprise` |
| 统一恢复脚本 | `ops/scripts/kkidc-host-restore.sh enterprise --source-backup-dir <remote-backup-dir>` |

## 3. 管理注意事项

- 企业生产环境应独立使用自己的数据库、Redis DB、数据目录与日志目录，不与个人生产混用。
- 首次在共享主机上发布企业环境前，先备份个人生产环境，再执行企业发布。
- 若需要用现有生产数据初始化企业环境，先执行 `bash ops/scripts/kkidc-host-backup.sh production`，再通过 `bash ops/scripts/kkidc-host-restore.sh enterprise --source-backup-dir <production-backup-dir>` 导入。
- 若 `.kkidc/.env.lighthouse` 未配置 `ENTERPRISE_HOSTNAME`，部署脚本不会写入企业站点的 Caddy 配置，企业环境仅通过 `http://114.66.47.192:3002` 提供验证入口。
- 正式公网启用前，需先确认 `corp-api.aisever.cn` 解析到 `114.66.47.192`，再补齐 `ENTERPRISE_HOSTNAME` / `ENTERPRISE_HOSTNAME_ALIASES` 并重新部署。
- 变更前必须先在 `kkidc` 测试环境验证。
- 服务器环境部署只允许使用已提交且已 push 的 `HEAD`。
- 若实际启用域名与建议值不同，应同步更新 `.kkidc/.env.lighthouse`、本文件与对应发布检查清单。

# kkidc 企业生产环境

## 1. 环境定位

- 状态：**启用中**
- 服务器：`114.66.47.192`
- 对外地址：`https://corp-api.aisever.cn`
- 说明：该环境作为企业客户独立生产环境，与当前个人生产环境隔离运行；当前公网地址已启用 `https://corp-api.aisever.cn`，同时保留 `http://114.66.47.192:3002` 作为直连验证入口。

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
| 统一重置脚本 | `ops/scripts/kkidc-host-reset.sh enterprise` |
| 上游导入脚本 | `ops/scripts/kkidc-host-import-upstream.sh enterprise --upstream-base-url <url> --upstream-username <user> --upstream-password <pass>` |

## 3. 管理注意事项

- 企业生产环境应独立使用自己的数据库、Redis DB、数据目录与日志目录，不与个人生产混用。
- 首次在共享主机上发布企业环境前，先备份个人生产环境，再执行企业发布。
- 若需要用现有生产数据初始化企业环境，先执行 `bash ops/scripts/kkidc-host-backup.sh production`，再通过 `bash ops/scripts/kkidc-host-restore.sh enterprise --source-backup-dir <production-backup-dir>` 导入。
- 若企业环境需要整库重置并重新初始化，必须执行 `bash ops/scripts/kkidc-host-reset.sh enterprise`；该脚本只允许操作企业环境，不允许作用于个人生产环境。
- 若企业环境需要对接新的上游实例，使用 `bash ops/scripts/kkidc-host-import-upstream.sh enterprise ...`；该脚本会写入企业环境分组、倍率与渠道，并为导入后的渠道开启上游模型巡检/自动同步设置。
- 上游导入后，仍建议继续使用系统自带能力完成后续维护：渠道上游模型自动巡检、官方模型同步、缺失模型发现、能力表修复。
- 当前 `.kkidc/.env.lighthouse` 已配置 `ENTERPRISE_HOSTNAME=corp-api.aisever.cn`；后续若调整企业域名，需同步更新私有配置并重新部署。
- 如需排查公网问题，可先以 `http://114.66.47.192:3002` 直连验证应用是否正常，再判断是否为 Caddy / TLS / DNS 问题。
- 变更前必须先在 `kkidc` 测试环境验证。
- 服务器环境部署只允许使用已提交且已 push 的 `HEAD`。
- 若实际启用域名与建议值不同，应同步更新 `.kkidc/.env.lighthouse`、本文件与对应发布检查清单。

# kkidc 企业生产环境

## 1. 环境定位

- 状态：**待部署**
- 服务器：`114.66.47.192`
- 建议对外地址：`https://corp-api.aisever.cn`
- 可选别名：`https://enterprise.aisever.cn`
- 说明：该环境计划作为企业客户独立生产环境，与当前个人生产环境隔离运行。

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

## 3. 管理注意事项

- 企业生产环境应独立使用自己的数据库、Redis DB、数据目录与日志目录，不与个人生产混用。
- 正式部署前，先完成 DNS、证书和访问白名单准备。
- 变更前必须先在 `kkidc` 测试环境验证。
- 服务器环境部署只允许使用已提交且已 push 的 `HEAD`。
- 若实际启用域名与建议值不同，应同步更新 `.kkidc/.env.lighthouse`、本文件与对应发布检查清单。

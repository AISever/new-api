# tencent 预备生产环境

## 1. 环境定位

- 状态：**已部署，未启用对外生产流量**
- 服务器：`123.206.229.105`
- IP 访问：`http://123.206.229.105:3000`
- 规划域名：`https://api.aisever.art`
- 说明：该环境已经完成部署与数据迁移准备，但由于域名/HTTPS 切换问题，当前**不是**真实生产环境。

## 2. 运行信息

| 项目 | 值 |
| --- | --- |
| 部署仓库 | `https://github.com/AISever/new-api.git` |
| 服务器目录 | `/opt/new-api-src` |
| 固定部署分支 | `codex/prod-live` |
| 发布分支示例 | `codex/release-vX.Y.Z-pN` |
| 本机配置入口 | `.tencent/.env.lighthouse` |
| SSH 密钥 | `.tencent/lighthouse-shanghai.pem` |

## 3. 管理注意事项

- **不要把它当作当前线上生产**；它现在属于“预备生产 / 待切换”状态。
- 涉及该环境的部署、备份恢复、证书、Nginx、域名切换，请统一参考 `docs/installation/DEPLOYMENT.md`。
- 如需先验证候选分支，请使用独立的 `tencent` 测试环境：`ops/environments/tencent-test.md`
- 对应脚本：`ops/scripts/tencent-standby-deploy.sh`、`ops/scripts/tencent-backup-from-kkidc.sh`、`ops/scripts/tencent-restore-from-backup.sh`、`ops/scripts/tencent-enable-https-certbot.sh`
- 当前本地配置中可确认的非敏感信息：
  - `IP_1=123.206.229.105`
  - `HOSTNAME=api.aisever.art`
- 只有在域名、HTTPS、流量切换策略全部确认后，才能把它升级为真实生产环境。

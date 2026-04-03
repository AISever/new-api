# 最小运维手册

本页只保留当前常用的发布、验证、关闭命令。

先决条件：

- 已切到目标发布分支。
- 需要发布到 `kkidc` 时，先阅读 `ops/environments/README.md`。
- 测试环境默认不常驻；验证完成后必须执行 stop。
- `kkidc` 发布默认在本地 Docker 构建镜像，再上传到服务器；本地无法构建时应先修复本地环境，而不是常规退回服务器构建。
- 本地 Docker 需要可用；当前默认目标平台是 `linux/amd64`，默认前端堆限制是 `4096`。
- 如需显式远端构建，先确认当前远端主机资源满足门禁：默认 `MemAvailable >= 2048MB` 且 `load1 <= 4.00`。
- 生产环境如需显式远端构建，只能在受控窗口执行，并在发布期间并行探测 `https://api.aisever.cn/api/status`。

## 1. `kkidc` 测试环境

启动 / 更新：

```bash
bash ops/scripts/kkidc-host-deploy.sh test
```

验证：

```bash
curl http://114.66.47.192:3001/api/status
```

关闭：

```bash
bash ops/scripts/kkidc-host-deploy.sh test-stop
```

说明：`test-stop` 只停止 `new-api-test`，保留 `new-api-test` 数据库、Redis `db 1`、`/opt/new-api-test/data`、`/opt/new-api-test/logs`。
近期实测：2026-03-25 本地构建 + 上传测试环境总耗时约 `372s`，其中镜像构建阶段约 `355s`。

从远端备份导入生产数据到测试环境：

```bash
bash ops/scripts/kkidc-host-restore.sh test --source-backup-dir /opt/new-api-backups/<timestamp>
```

说明：该入口会先备份当前测试环境，再恢复指定备份目录中的 `db.dump`、`data.tgz`，默认同时恢复 `logs.tgz`，最后使用当前已推送分支对应的远端镜像重新启动测试环境。

## 2. `kkidc` 生产环境

备份：

```bash
bash ops/scripts/kkidc-host-backup.sh production
```

发布：

```bash
bash ops/scripts/kkidc-host-deploy.sh production
```

验证：

```bash
curl https://api.aisever.cn/api/status
```

## 3. `kkidc` 企业生产环境

首次上线前备份个人生产：

```bash
bash ops/scripts/kkidc-host-backup.sh production
```

企业环境已有线上数据后的常规备份：

```bash
bash ops/scripts/kkidc-host-backup.sh enterprise
```

发布：

```bash
bash ops/scripts/kkidc-host-deploy.sh enterprise
```

验证：

```bash
curl http://114.66.47.192:3002/api/status
```

公网 DNS 切到 `114.66.47.192` 且 `.kkidc/.env.lighthouse` 已配置 `ENTERPRISE_HOSTNAME` 后，再执行：

```bash
curl https://corp-api.aisever.cn/api/status
```

说明：若本次企业发布未配置 `ENTERPRISE_HOSTNAME`，或生成的 Caddy 配置与当前远端一致，部署脚本不会重建 `snowlight-caddy`，个人版 `https://api.aisever.cn` 入口应保持不变。

## 4. 发布顺序

1. 先发 `kkidc` 测试并验证。
2. 验证完成后关闭 `kkidc` 测试。
3. 在共享主机上首次上线企业环境前，先执行 `bash ops/scripts/kkidc-host-backup.sh production`。
4. 再发目标生产环境（个人或企业）。

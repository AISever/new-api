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
- 当前腾讯云个人生产主机 `43.133.183.213` 已验证不适合作为常规远端构建节点；个人生产发布默认坚持本地构建上传。

## 0. 官方稳定版升级门禁

如果本次操作属于“跟随官方稳定版升级”，先执行：

```bash
bash ops/scripts/kkidc-host-upgrade-gate.sh local
bash ops/scripts/kkidc-host-upgrade-gate.sh test
```

说明：

- `local` 只跑本地门禁，不触发服务器环境。
- `test` 会在本地门禁通过后，委托正式脚本 `ops/scripts/kkidc-host-deploy.sh test` 部署测试环境，并在验证通过后默认执行 `test-stop`。
- 如需保留测试环境给人工继续验证，显式加 `--keep-test-running`。
- 本门禁不替代现有生产发布命令；测试通过后，仍需回到对应生产环境检查清单继续执行。

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

按 profile 导入新聚合上游内容：

```bash
bash ops/scripts/kkidc-host-import-profile.sh test --profile ops/channel-onboarding/profiles/yunwu.yaml --dry-run
bash ops/scripts/kkidc-host-import-profile.sh test --profile ops/channel-onboarding/profiles/yunwu.yaml --target-root-username <root> --target-root-password <password>
bash ops/scripts/kkidc-host-import-profile.sh test --profile ops/channel-onboarding/profiles/yunwu.yaml --target-root-username <root> --target-root-password <password> --probe-upstream --require-channel-keys --channel-key kling=<key> --channel-key vidu=<key> --channel-key doubao-video=<key>
```

Yunwu 测试环境联调快捷入口：

```bash
bash ops/scripts/kkidc-test-yunwu-onboarding.sh --target-root-username <root> --target-root-password <password> --channel-key kling=<key> --channel-key vidu=<key> --channel-key doubao-video=<key>
```

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

如需用生产备份初始化企业环境：

```bash
bash ops/scripts/kkidc-host-restore.sh enterprise --source-backup-dir /opt/new-api-backups/<timestamp>
```

发布：

```bash
bash ops/scripts/kkidc-host-deploy.sh enterprise
```

验证：

```bash
curl https://corp-api.aisever.cn/api/status
```

直连排障验证：

```bash
curl http://114.66.47.192:3002/api/status
```

说明：当前企业环境已配置 `ENTERPRISE_HOSTNAME=corp-api.aisever.cn`。若后续企业发布触发 Caddy 重建，发布后应同时验证 `https://corp-api.aisever.cn` 与 `https://api.aisever.cn`。

按 profile 导入新聚合上游内容：

```bash
bash ops/scripts/kkidc-host-import-profile.sh enterprise --profile ops/channel-onboarding/profiles/yunwu.yaml --dry-run
bash ops/scripts/kkidc-host-import-profile.sh enterprise --profile ops/channel-onboarding/profiles/yunwu.yaml --target-root-username <root> --target-root-password <password>
bash ops/scripts/kkidc-host-import-profile.sh enterprise --profile ops/channel-onboarding/profiles/yunwu.yaml --target-root-username <root> --target-root-password <password> --probe-upstream --require-channel-keys --channel-key kling=<key> --channel-key vidu=<key> --channel-key doubao-video=<key>
```

## 4. 发布顺序

1. 先发 `kkidc` 测试并验证。
2. 验证完成后关闭 `kkidc` 测试。
3. 在共享主机上首次上线企业环境前，先执行 `bash ops/scripts/kkidc-host-backup.sh production`。
4. 再发目标生产环境（个人或企业）。

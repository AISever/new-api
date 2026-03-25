# 最小运维手册

本页只保留当前常用的发布、验证、关闭命令。

先决条件：

- 已切到目标发布分支。
- 需要发布到 `kkidc` 时，先阅读 `ops/environments/README.md`。
- 测试环境默认不常驻；验证完成后必须执行 stop。
- `kkidc` 发布默认在本地 Docker 构建镜像，再上传到服务器；本地无法构建时应先修复本地环境，而不是常规退回服务器构建。
- 本地 Docker 需要可用；当前默认目标平台是 `linux/amd64`，默认前端堆限制是 `4096`。

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

## 3. 发布顺序

1. 先发 `kkidc` 测试并验证。
2. 验证完成后关闭 `kkidc` 测试。
3. 如涉及数据库、配置或数据迁移，先执行 `bash ops/scripts/kkidc-host-backup.sh production`。
4. 再发 `kkidc` 生产。

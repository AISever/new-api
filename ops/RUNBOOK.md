# 最小运维手册

本页只保留当前常用的发布、验证、关闭命令。

先决条件：

- 已切到目标发布分支。
- 需要发布到 `kkidc` / `tencent` 时，先阅读 `ops/environments/README.md`。
- 测试环境默认不常驻；验证完成后必须执行 stop。

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

## 2. `kkidc` 生产环境

发布：

```bash
bash ops/scripts/kkidc-host-deploy.sh production
```

验证：

```bash
curl https://api.aisever.cn/api/status
```

## 3. `tencent` 测试环境

部署：

```bash
bash ops/scripts/tencent-test-deploy.sh codex/release-vX.Y.Z-pN
```

验证：

```bash
curl http://123.206.229.105:3001/api/status
```

关闭：

```bash
bash ops/scripts/tencent-test-stop.sh
```

说明：`tencent-test-stop.sh` 只停止 `new-api-test`，保留 `postgres-test`、`redis-test` 和测试数据。

## 4. `tencent` 预备生产环境

部署：

```bash
bash ops/scripts/tencent-standby-deploy.sh codex/prod-live
```

验证：

```bash
curl http://123.206.229.105:3000/api/status
```

## 5. 发布顺序

1. 先发 `kkidc` 测试并验证。
2. 验证完成后关闭 `kkidc` 测试。
3. 再发 `kkidc` 生产。
4. 如需验证 `tencent` 服务器行为，再发 `tencent` 测试并在结束后关闭。
5. `tencent` 预备生产只用于待切流验证，不等于当前生产。

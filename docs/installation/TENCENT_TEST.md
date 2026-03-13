# tencent 测试环境部署手册

本文档用于沉淀 `tencent` 测试环境的部署流程，便于在同一台 `tencent` 服务器上验证候选版本。

> 重要：该环境是 `tencent` 测试环境，当前访问端口为 `3001`，和 `tencent` 预备生产环境 `:3000` 分离。测试环境默认不常驻，验证完成后应执行 stop。

## 1. 适用范围

- 服务器：`123.206.229.105`
- 规划目录：`/opt/new-api-test-src`
- 访问地址：`http://123.206.229.105:3001`
- 推荐用途：部署 `codex/release-*` 候选分支，验证其在 `tencent` 服务器上的真实运行情况

## 2. 一次性初始化

```bash
git clone --depth 1 --single-branch --branch codex/prod-live \
  https://github.com/AISever/new-api.git /opt/new-api-test-src

cd /opt/new-api-test-src
mkdir -p data logs
chmod +x ./ops/scripts/deploy-from-branch.sh
chmod +x ./ops/scripts/tencent-test-deploy.sh
chmod +x ./ops/scripts/tencent-test-restore-from-backup.sh
```

## 3. 部署候选分支

```bash
cd /opt/new-api-test-src
./ops/scripts/tencent-test-deploy.sh codex/release-vX.Y.Z-pN
```

默认结果：

- compose 项目：`new-api-test`
- 容器：`new-api-test` / `postgres-test` / `redis-test`
- 端口：`3001`
- 数据库：`new-api-test`
- 共享部署核心：`ops/scripts/deploy-from-branch.sh`

部署完成后，如测试窗口结束，请执行：

```bash
cd /opt/new-api-test-src
./ops/scripts/tencent-test-stop.sh
```

该命令只停止 `new-api-test` 应用容器，保留 `postgres-test`、`redis-test` 和测试数据。

## 3.1 可选：本地构建并上传镜像（低内存服务器推荐）

当 `tencent` 服务器本机构建发生 OOM 时，可在本地完成构建，然后上传镜像并在服务器 `--no-build` 启动。

```bash
cd /Users/lei/Codes/Kiro/new-api
./ops/scripts/tencent-local-build-upload-deploy.sh \
  --target test \
  --branch codex/release-vX.Y.Z-pN
```

说明：

- 该模式不会替换默认部署方式，只是新增可选路径。
- 本地会构建 `linux/amd64` 镜像并通过 SSH 上传到 `tencent`。
- 该脚本要求本地当前分支、工作区状态、`origin/<branch>` 三者完全一致，避免把本地脏代码伪装成分支版本。

## 4. 用生产备份初始化测试环境

```bash
cd /opt/new-api-test-src
./ops/scripts/tencent-test-restore-from-backup.sh /opt/new-api-src/backups/origin/<timestamp>
```

如需写入测试地址，可显式传入：

```bash
cd /opt/new-api-test-src
SERVER_ADDRESS=http://123.206.229.105:3001 \
  ./ops/scripts/tencent-test-restore-from-backup.sh /opt/new-api-src/backups/origin/<timestamp>
```

## 5. 验证清单

```bash
curl -sS http://127.0.0.1:3001/api/status
curl -sS http://123.206.229.105:3001/api/status
docker ps --format "table {{.Names}}\t{{.Status}}\t{{.Ports}}"
```

预期：

- `new-api-test` 容器为 `healthy`
- `/api/status` 返回 `success=true`
- 版本信息对应候选分支与提交 SHA

## 6. 测试结束后关闭

```bash
cd /opt/new-api-test-src
./ops/scripts/tencent-test-stop.sh
```

预期：

- `new-api-test` 容器已停止
- `postgres-test` / `redis-test` 继续保留
- 下次测试前再重新执行 `./ops/scripts/tencent-test-deploy.sh <branch>`

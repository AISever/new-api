# tencent 预备生产部署手册

本文档用于沉淀 `tencent` 服务器的部署流程，便于后续 AI Agent 和运维人员直接执行，避免重复排障。

> 重要：本手册**只对应 `tencent` 预备生产环境**。当前真实生产流量在新 `kkidc` 生产环境 `https://api.aisever.cn`。环境总览请先看 `ops/environments/README.md`，日常运维入口请先看 `ops/README.md`。

如需部署 `tencent` 测试环境，请改看 `docs/installation/TENCENT_TEST.md`。

## 1. 适用范围

- 目标：预备生产可用、可持续迭代（允许修改代码后部署）
- 当前状态：**已部署，但未启用对外生产流量**
- 服务器：`tencent` `123.206.229.105`
- 域名：`api.aisever.art`
- 代码仓库（部署用）：`https://github.com/AISever/new-api.git`
- 固定部署分支：`codex/prod-live`
- 版本迭代分支示例：`codex/release-vX.Y.Z-pN`

## 2. 分支与版本策略

- 官方基线：从稳定 tag 切发布分支（示例：`vX.Y.Z` → `codex/release-vX.Y.Z-pN`）
- 发布分支验证通过后，合并到固定部署分支 `codex/prod-live`
- 线上部署只跟随 `codex/prod-live`，不直接使用 `main`
- 线上版本通过环境变量注入：`VERSION=<branch>+<short_sha>`
- 验证方式：查看响应头 `X-New-Api-Version` 或 `/api/status` 的 `data.version`

## 3. 一次性初始化（服务器）

```bash
# 1) 克隆部署仓库（指定生产分支）
git clone --depth 1 --single-branch --branch codex/prod-live \
  https://github.com/AISever/new-api.git /opt/new-api-src

cd /opt/new-api-src

# 2) 必须先创建挂载目录，避免容器健康检查异常
mkdir -p data logs
chmod +x ./ops/scripts/deploy-from-branch.sh
chmod +x ./ops/scripts/tencent-standby-deploy.sh
```

## 4. 部署脚本（服务器）

直接使用仓库内脚本：

```bash
cd /opt/new-api-src
./ops/scripts/tencent-standby-deploy.sh codex/prod-live
```

## 5. 日常发布流程

```bash
cd /opt/new-api-src
./ops/scripts/tencent-standby-deploy.sh codex/prod-live
```

说明：

- `ops/scripts/tencent-standby-deploy.sh` 是 `tencent` 预备生产入口。
- 它复用 `ops/scripts/deploy-from-branch.sh` 的公共逻辑：拉分支、重置代码、生成 `Dockerfile.deploy`、注入 `VERSION`、执行 `docker compose build/up`。
- 预备生产继续沿用当前服务器的真实拓扑：官方 `docker-compose.yml` 作为基础文件，脚本只生成 `docker-compose.override.yml`。

### 5.1 可选：本地构建并上传镜像（保留原方式）

当服务器本机构建资源不足时，可使用新增脚本在本地构建并上传镜像，再远端 `--no-build` 启动：

```bash
cd /Users/lei/Codes/Kiro/new-api
./ops/scripts/tencent-local-build-upload-deploy.sh \
  --target standby \
  --branch codex/prod-live
```

该模式与 `./ops/scripts/tencent-standby-deploy.sh` 并存，不会替换原流程。

## 6. 验证清单

```bash
# 容器状态
docker ps --format "table {{.Names}}\t{{.Image}}\t{{.Status}}"

# 本机健康检查
curl -sS http://127.0.0.1:3000/api/status

# 域名检查
curl -I http://api.aisever.art/api/status
```

预期：
- `new-api` 容器为 `healthy`
- `X-New-Api-Version` 为 `codex/prod-live+<sha>`

## 7. 反向代理（HTTP）

Nginx 站点配置文件：

- `/www/server/panel/vhost/nginx/api.aisever.art.conf`

核心逻辑：
- `80` 端口接入
- `/` 反代到 `127.0.0.1:3000`
- 保留 `/.well-known/acme-challenge/` 供证书校验

## 8. HTTPS 现状与处理

当前阻塞点：证书签发时，外部校验流量被 DNSPod `webblock` 页面拦截，导致 `http-01` 失败。

典型报错（`/var/log/letsencrypt/letsencrypt.log`）：
- `unauthorized`
- `Invalid response from https://dnspod.qcloud.com/static/webblock.html?...`

处理顺序：
1. 在 DNSPod 确认 `A` 记录所有线路都指向 `123.206.229.105`
2. 取消域名拦截/未备案拦截策略（或改用可签发证书的域名）
3. 再执行证书签发与 443 配置

推荐在服务器执行脚本（含对 `webblock` 的预检）：

```bash
cd /opt/new-api-src
./ops/scripts/tencent-enable-https-certbot.sh api.aisever.art
```

如需设置邮箱（推荐，用于证书过期提醒）：

```bash
cd /opt/new-api-src
CERTBOT_EMAIL=admin@example.com ./ops/scripts/tencent-enable-https-certbot.sh api.aisever.art
```
## 9. 常见问题

### 9.1 构建阶段被 SIGKILL / OOM

现象：`vite build` 在 `rendering chunks` 附近被系统杀死。

已落地措施：
- 增加 swap（服务器已有 `/www/swap` + `/www/swap2`）
- 构建命令加 `--memory 5g`
- 前端构建限制内存：`NODE_OPTIONS=--max-old-space-size=2048`

说明：
- 目前 `docker compose` 在 BuildKit 下可能提示 `--memory ... will be ignored`
- 如仍频繁 OOM，建议优先升级服务器内存，或改为 CI/本地构建镜像后再推送到服务器拉取

### 9.2 容器 `unhealthy` 且健康检查执行失败

现象：`current working directory is outside of container mount namespace root`

处理：
- 确保宿主机挂载目录存在：`/opt/new-api-src/data`、`/opt/new-api-src/logs`
- 然后执行：`docker compose -p new-api up -d`

## 10. 与官方版本保持同步（Git 策略）

目标：在可改代码的前提下，尽可能减少与官方差异，便于后续持续升级。

### 10.1 官方如何发布版本

- 官方以 Git tag 作为版本：稳定版 `vX.Y.Z`，预览版 `vX.Y.Z-alpha.N`
- GitHub Release 构建会忽略 `*-alpha*` tag（只对稳定版发布 Release）：`.github/workflows/release.yml:5`
- Docker 镜像 `latest` 会随 tag 构建流水线推进（可能跟随 alpha），生产建议固定使用具体 tag 或自建镜像：`.github/workflows/docker-image-arm64.yml:3`

### 10.2 提交代码的原则（保证同步/兼容的关键）

- `main` 只做官方镜像：永远不在 `main` 上提交自定义改动（否则后续升级成本指数上升）
- 自定义改动只放在 `codex/*` 分支，且尽量“少而独立”（一个功能/修复一组提交，避免把大量改动揉成一次提交）
- `codex/prod-live` 只做部署入口：只允许 fast-forward 合并发布分支（保持线上历史线性、可追溯）

### 10.3 同步官方 `main`（镜像）

> 约束：`main` 必须保持与官方 `upstream/main` 一致。

```bash
git fetch upstream --tags
git switch main
git pull --ff-only upstream main
git push origin main
```

### 10.4 以官方稳定 tag 为基线创建发布分支

```bash
# 示例：从 v0.11.2 创建发布分支
git fetch upstream --tags
git switch -c codex/release-vX.Y.Z-pN vX.Y.Z
git push -u origin codex/release-vX.Y.Z-pN
```

### 10.5 升级到新版本（把自定义提交“搬到新 tag 上”）

```bash
# 示例：从旧基线 v0.11.2 升级到新基线 v0.11.3
git fetch upstream --tags
git switch -c codex/release-v0.11.3-p1 v0.11.3

# 把旧发布分支相对旧 tag 的提交整体 cherry-pick 过去
git cherry-pick v0.11.2..codex/release-v0.11.2-patch.2-p1
git push -u origin codex/release-v0.11.3-p1
```

说明：
- 若 cherry-pick 冲突：解决冲突后执行 `git cherry-pick --continue`
- 升级/新功能必须先在测试环境验证通过，再合并到 `codex/prod-live` 并部署到生产

## 11. 源生产 → 轻量 数据备份与恢复

> 场景：`kkidc` 生产环境短期不能下线，需要周期性把“数据/配置/数据库”同步到 `tencent` 新生产环境。
> 注意：恢复会停止新生产环境的容器（有短暂停机），请在低峰操作。

### 11.1 准备本地配置（仅本机保存，不要提交）

1) 复制示例文件：

```bash
cp .tencent/.env.lighthouse.example .tencent/.env.lighthouse
```

2) 编辑 `.tencent/.env.lighthouse`，填写 `IP_1`、`HOSTNAME`（以及可选的 `LIGHTHOUSE_SSH_KEY`）。

### 11.2 执行备份（本地 → 上传到轻量）

在本机仓库目录执行：

```bash
export ORIGIN_SSH_HOST=202.140.142.149
export ORIGIN_SSH_USER=root
export ORIGIN_SSH_PASS='***'   # 或改用 ORIGIN_SSH_KEY

./ops/scripts/tencent-backup-from-kkidc.sh
```

输出为轻量服务器上的备份目录，例如：

- `/opt/new-api-src/backups/origin/20260305-164125`

### 11.3 执行恢复（在轻量服务器上）

SSH 到轻量服务器：

```bash
ssh -i .tencent/lighthouse-shanghai.pem root@123.206.229.105
```

恢复并同步新域名（会写入数据库 options.ServerAddress）：

```bash
cd /opt/new-api-src
SERVER_ADDRESS=https://api.aisever.art \
  ./ops/scripts/tencent-restore-from-backup.sh /opt/new-api-src/backups/origin/<timestamp>
```

验证：

```bash
curl -sS http://127.0.0.1:3000/api/status | head
curl -I http://api.aisever.art/api/status
```

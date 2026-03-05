# 部署手册（腾讯云轻量服务器）

本文档用于沉淀可复用的部署流程，便于后续 AI Agent 和运维人员直接执行，避免重复排障。

## 1. 适用范围

- 目标：生产可用、可持续迭代（允许修改代码后部署）
- 服务器：腾讯云轻量 `123.206.229.105`
- 域名：`newapi.alling.online`
- 代码仓库（部署用）：`https://github.com/AISever/new-api.git`
- 固定部署分支：`codex/prod-live`
- 版本迭代分支示例：`codex/release-v0.11.2-p1`

## 2. 分支与版本策略

- 官方基线：从稳定 tag 切发布分支（示例：`v0.11.2` → `codex/release-v0.11.2-p1`）
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
```

## 4. 部署脚本（服务器）

在服务器 `/opt/new-api-src/deploy-from-branch.sh` 使用以下脚本：

```bash
#!/usr/bin/env bash
set -euo pipefail

BRANCH="${1:-codex/prod-live}"
REPO_DIR="/opt/new-api-src"
PROJECT_NAME="new-api"

cd "$REPO_DIR"

timeout 180 git fetch origin --prune
if git show-ref --verify --quiet "refs/heads/${BRANCH}"; then
  git checkout "$BRANCH"
else
  git checkout -b "$BRANCH" "origin/$BRANCH"
fi
timeout 180 git pull --ff-only origin "$BRANCH"

SHA="$(git rev-parse --short HEAD)"
VERSION_VALUE="${BRANCH}+${SHA}"

cp Dockerfile Dockerfile.deploy
sed -i "s|RUN go mod download|RUN go env -w GOPROXY=https://goproxy.cn,direct \\&\\& go mod download|" Dockerfile.deploy
sed -i "s|RUN DISABLE_ESLINT_PLUGIN='true' VITE_REACT_APP_VERSION=\$(cat VERSION) bun run build|RUN DISABLE_ESLINT_PLUGIN='true' NODE_OPTIONS='--max-old-space-size=2048' VITE_REACT_APP_VERSION=\$(cat VERSION) bun run build|" Dockerfile.deploy

cat > docker-compose.override.yml <<YAML
services:
  new-api:
    image: new-api:codex-prod-live
    build:
      context: .
      dockerfile: Dockerfile.deploy
    environment:
      - VERSION=${VERSION_VALUE}
YAML

docker compose -p "$PROJECT_NAME" build --memory 5g new-api
docker compose -p "$PROJECT_NAME" up -d

echo "deployed_branch=${BRANCH}"
echo "deployed_sha=${SHA}"
echo "deployed_version=${VERSION_VALUE}"
```

赋权：

```bash
chmod +x /opt/new-api-src/deploy-from-branch.sh
```

## 5. 日常发布流程

```bash
cd /opt/new-api-src
./deploy-from-branch.sh codex/prod-live
```

## 6. 验证清单

```bash
# 容器状态
docker ps --format "table {{.Names}}\t{{.Image}}\t{{.Status}}"

# 本机健康检查
curl -sS http://127.0.0.1:3000/api/status

# 域名检查
curl -I http://newapi.alling.online/api/status
```

预期：
- `new-api` 容器为 `healthy`
- `X-New-Api-Version` 为 `codex/prod-live+<sha>`

## 7. 反向代理（HTTP）

Nginx 站点配置文件：

- `/www/server/panel/vhost/nginx/newapi.alling.online.conf`

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

## 9. 常见问题

### 9.1 构建阶段被 SIGKILL / OOM

现象：`vite build` 在 `rendering chunks` 附近被系统杀死。

已落地措施：
- 增加 swap（服务器已有 `/www/swap` + `/www/swap2`）
- 构建命令加 `--memory 5g`
- 前端构建限制内存：`NODE_OPTIONS=--max-old-space-size=2048`

### 9.2 容器 `unhealthy` 且健康检查执行失败

现象：`current working directory is outside of container mount namespace root`

处理：
- 确保宿主机挂载目录存在：`/opt/new-api-src/data`、`/opt/new-api-src/logs`
- 然后执行：`docker compose -p new-api up -d`

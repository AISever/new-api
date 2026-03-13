# 运维入口

本目录用于集中管理当前仓库的**自定义运维资料与脚本**。

原则：

- 环境事实以 `ops/environments/` 为准。
- 环境专属文件按服务器前缀命名：`kkidc-*`、`tencent-*`。
- 通用文档与通用入口不加前缀。
- **不要修改官方原有脚本**；本目录只收纳我们自己增加的运维内容。

## 1. 当前环境口径

- `kkidc` 生产环境：`https://api.aisever.cn`
- `kkidc` 测试环境：`http://114.66.47.192:3001`
- `tencent` 测试环境：`http://123.206.229.105:3001`
- `tencent` 预备生产环境：`http://123.206.229.105:3000` / 规划域名 `https://api.aisever.art`

在执行任何部署、恢复、迁移、切流之前，请先阅读：

1. `ops/environments/README.md`
2. 对应环境文档（`kkidc-production.md` / `kkidc-test.md` / `tencent-test.md` / `tencent-standby.md`）

## 2. 日常操作入口

- `kkidc` 测试发布：`ops/checklists/kkidc-test-release.md`
- `kkidc` 生产发布：`ops/checklists/kkidc-production-release.md`
- `tencent` 测试发布：`ops/checklists/tencent-test-release.md`
- `tencent` 切流准备：`ops/checklists/tencent-cutover.md`
- `tencent` 详细部署/恢复：`docs/installation/DEPLOYMENT.md`
- `tencent` 测试部署：`docs/installation/TENCENT_TEST.md`

## 3. 环境脚本

- 通用 `tencent` 分支部署核心：`ops/scripts/deploy-from-branch.sh`
- 本地构建并上传到 `tencent`（可选）：`ops/scripts/tencent-local-build-upload-deploy.sh`
- `kkidc` 新服务器统一部署：`ops/scripts/kkidc-host-deploy.sh production|test`
- `tencent` 从 `kkidc` 备份导入：`ops/scripts/tencent-backup-from-kkidc.sh`
- `tencent` 预备生产部署：`ops/scripts/tencent-standby-deploy.sh`
- `tencent` 测试部署：`ops/scripts/tencent-test-deploy.sh`
- `tencent` 测试关闭：`ops/scripts/tencent-test-stop.sh`
- `tencent` 测试从备份恢复：`ops/scripts/tencent-test-restore-from-backup.sh`
- `tencent` 从备份恢复：`ops/scripts/tencent-restore-from-backup.sh`
- `tencent` 启用 HTTPS：`ops/scripts/tencent-enable-https-certbot.sh`

## 4. 环境 compose

- `tencent` 测试 compose：`ops/compose/tencent-test.yml`
- `tencent` 预备生产 compose：仓库根 `docker-compose.yml` + 运行期生成的 `docker-compose.override.yml`

## 5. 推荐流程

1. 新功能或修复先在 `kkidc` 测试环境验证。
2. 验证通过后再部署到新 `kkidc` 生产环境（`https://api.aisever.cn`）。
3. 如需验证 `tencent` 服务器行为，再部署到 `tencent` 测试环境。
4. `tencent` 预备生产环境当前只做数据同步、部署验证、域名/HTTPS 切换准备。
5. 未完成切流前，**不要**把 `tencent` 预备生产环境视为真实生产。
6. `kkidc` / `tencent` 测试环境默认不常驻，验证结束后必须执行 stop 关闭测试应用。

## 6. 部署脚本抽象边界

- `kkidc` 与 `tencent` **不是同一套部署拓扑**，不要强行共用一个脚本入口。
- `kkidc` 新服务器使用 `ops/scripts/kkidc-host-deploy.sh` 统一处理 `production` / `test`，并且只打包当前分支已提交的 `HEAD`，不带本地未提交改动。
- `kkidc` 测试环境关闭入口：`ops/scripts/kkidc-host-deploy.sh test-stop`，只停 `new-api-test`，保留测试数据。
- `tencent` 体系基于“服务器上保留 git 仓库、切分支、`docker compose build/up`”运行。
- `tencent` 测试环境关闭入口：`ops/scripts/tencent-test-stop.sh`，只停 `new-api-test`，保留测试数据。
- 因此这里只抽取 `tencent` 内部公共核心：`ops/scripts/deploy-from-branch.sh`。
- `tencent` 测试与 `tencent` 预备生产只在目录、端口、数据库、compose 生成方式上有差异，其余分支切换、构建、版本注入逻辑保持一致。

## 7. tencent 双模式部署

- 模式 A（默认）：服务器拉分支并本机构建，入口仍为 `ops/scripts/tencent-test-deploy.sh` / `ops/scripts/tencent-standby-deploy.sh`。
- 模式 B（新增）：本地构建 `linux/amd64` 镜像并上传，再在服务器 `--no-build` 启动，入口 `ops/scripts/tencent-local-build-upload-deploy.sh`。
- 两种模式并存，互不替代；按服务器资源和发布窗口选择。

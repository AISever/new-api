# 运维入口

本目录用于集中管理当前仓库的**自定义运维资料与脚本**。

原则：

- 环境事实以 `ops/environments/` 为准。
- 环境专属文件按服务器前缀命名：`kkidc-*`。
- 通用文档与通用入口不加前缀。
- **不要修改官方原有脚本**；本目录只收纳我们自己增加的运维内容。
- 本地、测试、生产环境都必须通过仓库内正式脚本入口操作，不使用手工命令流作为常规部署方式。

## 1. 当前环境口径

- `kkidc` 生产环境：`https://api.aisever.cn`
- `kkidc` 测试环境：`http://114.66.47.192:3001`
在执行任何部署、恢复、迁移、切流之前，请先阅读：

1. `ops/environments/README.md`
2. 对应环境文档（`kkidc-production.md` / `kkidc-test.md`）

## 2. 日常操作入口

- 最小运维手册：`ops/RUNBOOK.md`
- `kkidc` 测试发布：`ops/checklists/kkidc-test-release.md`
- `kkidc` 生产发布：`ops/checklists/kkidc-production-release.md`

## 3. 环境脚本

- 本地 Docker 测试环境统一入口：`ops/scripts/local-test-env.sh`
- `kkidc` 新服务器统一部署：`ops/scripts/kkidc-host-deploy.sh production|test`
- `kkidc` 统一备份入口：`ops/scripts/kkidc-host-backup.sh production|test`

要求：

- 本地 Docker 测试环境的部署、更新、重置、备份必须通过 `ops/scripts/local-test-env.sh`。
- `kkidc` 测试/生产环境的部署、启动、停止必须通过 `ops/scripts/kkidc-host-deploy.sh`。
- `kkidc` 测试/生产环境的数据库与运行目录备份必须通过 `ops/scripts/kkidc-host-backup.sh`。
- `ops/scripts/kkidc-host-deploy.sh` 默认使用本地 Docker 构建镜像；本地不可构建时直接失败，不自动退回服务器构建。
- `--build-strategy remote` / `legacy-remote` 仅保留为显式紧急选项，不作为常规发布路径。
- 服务器环境只允许部署或操作已提交的仓库状态；需要验证未提交改动时，先用本地 Docker 测试环境。
- 不要把 SSH 登录后手动执行的临时命令、shell history、`.tmp` 脚本视为正式入口。

## 4. 环境 compose

- 本地 Docker 测试 compose：`ops/compose/local-test.yml`

## 5. 推荐流程

1. 新功能或修复先在 `kkidc` 测试环境验证。
2. 验证通过后再部署到新 `kkidc` 生产环境（`https://api.aisever.cn`）。
3. 发布前如涉及数据库、配置或数据迁移，先执行 `bash ops/scripts/kkidc-host-backup.sh production` 生成备份。
4. `kkidc` 测试环境默认不常驻，验证结束后必须执行 stop 关闭测试应用。

## 6. 部署脚本抽象边界

- `kkidc` 新服务器使用 `ops/scripts/kkidc-host-deploy.sh` 统一处理 `production` / `test`，并且只打包当前分支已提交的 `HEAD`，不带本地未提交改动。
- `kkidc` 测试环境关闭入口：`ops/scripts/kkidc-host-deploy.sh test-stop`，只停 `new-api-test`，保留测试数据。
- `kkidc` 备份入口：`ops/scripts/kkidc-host-backup.sh production|test`，默认在服务器上生成 PostgreSQL dump、globals dump，以及可选的数据/日志归档。
- 服务器环境只允许部署已提交的仓库状态；需要验证未提交改动时，先用本地 Docker 测试环境。

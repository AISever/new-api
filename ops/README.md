# 运维入口

本目录用于集中管理当前仓库的**自定义运维资料与脚本**。

原则：

- 环境事实以 `ops/environments/` 为准。
- 环境专属文件按服务器前缀命名：`kkidc-*`。
- 通用文档与通用入口不加前缀。
- **不要修改官方原有脚本**；本目录只收纳我们自己增加的运维内容。
- 本地、测试、生产环境都必须通过仓库内正式脚本入口操作，不使用手工命令流作为常规部署方式。

## 1. 当前环境口径

- `kkidc` 个人生产环境：`https://api.aisever.cn`
- `kkidc` 企业生产环境：`https://corp-api.aisever.cn`
- `kkidc` 测试环境：`http://114.66.47.192:3001`
在执行任何部署、恢复、迁移、切流之前，请先阅读：

1. `ops/environments/README.md`
2. 对应环境文档（`kkidc-production.md` / `kkidc-enterprise-production.md` / `kkidc-test.md`）

## 2. 日常操作入口

- 最小运维手册：`ops/RUNBOOK.md`
- `kkidc` 测试发布：`ops/checklists/kkidc-test-release.md`
- `kkidc` 个人生产发布：`ops/checklists/kkidc-production-release.md`
- `kkidc` 企业生产发布：`ops/checklists/kkidc-enterprise-production-release.md`

## 3. 环境脚本

- 本地 Docker 测试环境统一入口：`ops/scripts/local-test-env.sh`
- `kkidc` 新服务器统一部署：`ops/scripts/kkidc-host-deploy.sh production|enterprise|test`
- `kkidc` 统一备份入口：`ops/scripts/kkidc-host-backup.sh production|enterprise|test`
- `kkidc` 统一恢复入口：`ops/scripts/kkidc-host-restore.sh test|enterprise --source-backup-dir <remote-backup-dir>`
- `kkidc` 测试/企业环境统一重置入口：`ops/scripts/kkidc-host-reset.sh test|enterprise`
- `kkidc` 测试/企业环境上游导入入口：`ops/scripts/kkidc-host-import-upstream.sh test|enterprise --upstream-base-url <url> --upstream-username <user> --upstream-password <pass>`

要求：

- 本地 Docker 测试环境的部署、更新、重置、备份必须通过 `ops/scripts/local-test-env.sh`。
- `kkidc` 测试/生产环境的部署、启动、停止必须通过 `ops/scripts/kkidc-host-deploy.sh`。
- `kkidc` 测试/生产环境的数据库与运行目录备份必须通过 `ops/scripts/kkidc-host-backup.sh`。
- `kkidc` 测试/企业环境如需从远端备份目录恢复数据库与运行目录，必须通过 `ops/scripts/kkidc-host-restore.sh`。
- `kkidc` 测试/企业环境如需整库清空并重新初始化，必须通过 `ops/scripts/kkidc-host-reset.sh`，禁止手工删库或手工清目录。
- `kkidc` 测试/企业环境如需从外部上游批量导入分组、模型、倍率与渠道，必须通过 `ops/scripts/kkidc-host-import-upstream.sh`。
- 企业版本地文档通过 `bun run sync:enterprise-docs` 生成到 `web/public/enterprise-docs/apifox/`；`ops/scripts/kkidc-host-deploy.sh test|enterprise` 与 `ops/scripts/kkidc-host-import-upstream.sh test|enterprise` 会自动写入 `general_setting.docs_manifest_path=/enterprise-docs/apifox/manifest.json`，不要把该配置同步覆盖到个人生产环境。
- `ops/scripts/kkidc-host-deploy.sh` 默认使用本地 Docker 构建镜像；本地不可构建时直接失败，不自动退回服务器构建。
- `--build-strategy remote` / `legacy-remote` 仅保留为显式紧急选项，不作为常规发布路径。
- 如需显式使用 `BUILD_STRATEGY=remote`，脚本会先检查远端主机当前资源；默认要求 `MemAvailable >= 2048MB` 且 `load1 <= 4.00`，否则直接拒绝远端构建。
- 当前 `kkidc` 本地构建默认使用 `linux/amd64` 目标平台，避免 Apple Silicon 本地镜像直接推到 `amd64` 服务器后出现 `exec format error`。
- 当前 `kkidc` 本地构建默认使用 `FRONTEND_BUILD_NODE_OPTIONS=--max-old-space-size=4096`，并在部署专用 `Dockerfile.deploy` 中把 Alpine 包源切到阿里云镜像，避免本地交叉构建 OOM 或 Alpine 官方源波动导致失败。
- 服务器环境只允许部署或操作已提交且已 push 到远端校验分支的仓库状态；需要验证本地未 push 改动时，先用本地 Docker 测试环境。
- `ops/scripts/kkidc-host-deploy.sh`、`kkidc-host-backup.sh`、`kkidc-host-restore.sh`、`kkidc-host-reset.sh`、`kkidc-host-import-upstream.sh` 默认会优先从 git common root 查找共享的 `.kkidc/.env.lighthouse` 与 `.env.local`，因此在 git worktree 中执行也应使用同一套私有配置。
- `ops/scripts/kkidc-host-deploy.sh` 仅在生成的 Caddy 配置发生变化时才会重建 `snowlight-caddy`；若企业环境未配置 `ENTERPRISE_HOSTNAME`，企业发布只更新 `3002` 应用容器，不触碰个人版公网入口。
- 不要把 SSH 登录后手动执行的临时命令、shell history、`.tmp` 脚本视为正式入口。

## 4. 环境 compose

- 本地 Docker 测试 compose：`ops/compose/local-test.yml`

## 5. 推荐流程

1. 新功能或修复先在 `kkidc` 测试环境验证。
2. 验证通过后，再根据目标客户群部署到对应生产环境：
  - 个人用户：`https://api.aisever.cn`
  - 企业用户：建议 `https://corp-api.aisever.cn`
3. 在共享主机上首次上线企业环境前，先执行 `bash ops/scripts/kkidc-host-backup.sh production` 备份当前个人生产数据；企业环境已有线上数据后，再额外执行 `bash ops/scripts/kkidc-host-backup.sh enterprise`。
4. `kkidc` 测试环境默认不常驻，验证结束后必须执行 stop 关闭测试应用。
5. 如需将企业环境重置并重新接入新的上游，先执行 `bash ops/scripts/kkidc-host-backup.sh enterprise`，再执行 `bash ops/scripts/kkidc-host-reset.sh enterprise`，最后执行 `bash ops/scripts/kkidc-host-import-upstream.sh enterprise ...`；导入前后都额外验证 `https://api.aisever.cn/api/status` 正常。

参考耗时：

- 2026-03-25 使用本地 `linux/amd64` 构建并发布 `kkidc` 测试环境时，`c6407d4d` 的实测结果为：
  - `duration_image_seconds=355`
  - `duration_image_build_seconds` / `duration_image_export_seconds` / `duration_image_upload_seconds` / `duration_image_load_seconds` 已在脚本中单独输出，可用于继续判断是否有必要引入镜像仓库。
  - `duration_remote_start_seconds=2`
  - `duration_verify_seconds=14`
  - `duration_total_seconds=372`
- 2026-03-26 显式使用 `BUILD_STRATEGY=remote` 在 `kkidc` 测试环境验证时，服务器在当前空载条件下可成功完成远端构建，且生产 `https://api.aisever.cn/api/status` 并行探测未出现失败：
  - `duration_image_seconds=243`
  - `duration_image_build_seconds=229`
  - `duration_remote_start_seconds=2`
  - `duration_verify_seconds=13`
  - `duration_total_seconds=258`

## 6. 部署脚本抽象边界

- `kkidc` 新服务器使用 `ops/scripts/kkidc-host-deploy.sh` 统一处理 `production` / `enterprise` / `test`，并且只打包已经 push 到远端校验分支的已提交 `HEAD`，不带本地未提交改动。
- `kkidc` 测试环境关闭入口：`ops/scripts/kkidc-host-deploy.sh test-stop`，只停 `new-api-test`，保留测试数据。
- `kkidc` 备份入口：`ops/scripts/kkidc-host-backup.sh production|enterprise|test`，默认在服务器上生成 PostgreSQL dump、globals dump，以及可选的数据/日志归档。
- `kkidc` 测试/企业环境重置入口：`ops/scripts/kkidc-host-reset.sh test|enterprise`，会先备份目标环境，再清空目标数据库 / Redis DB / 数据目录 / 日志目录，最后通过正式部署脚本重新拉起目标环境。
- `kkidc` 测试/企业环境上游导入入口：`ops/scripts/kkidc-host-import-upstream.sh test|enterprise`，负责拉取外部上游分组、模型、倍率数据，写入目标环境 option，并创建或更新目标渠道。
- `kkidc` 企业环境若未配置 `ENTERPRISE_HOSTNAME`，脚本只保证 `new-api-enterprise` 与 `:3002` 可用；公网域名切流须在 DNS 指向正确后再补齐。
- 服务器环境只允许部署已提交且已 push 的仓库状态；需要验证未提交或未 push 改动时，先用本地 Docker 测试环境。

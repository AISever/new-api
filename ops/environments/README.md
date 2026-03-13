# 部署环境总览

本目录用于统一管理 `new-api` 的部署环境信息，避免把不同服务器环境混在一起。
如需执行具体操作，请回到 `ops/README.md` 选择对应脚本或检查清单。

## 1. 当前口径（2026-03-13）

> 重要：**当前真实对外生产环境是新 `kkidc` 主机 `114.66.47.192` 上的 `https://api.aisever.cn`**。`tencent` 服务器仍未启用对外生产流量。

| 环境 | 当前状态 | 访问地址 | 服务器 | 容器 | 数据库 | 用途 |
| --- | --- | --- | --- | --- | --- | --- |
| `kkidc` 生产环境 | **启用中** | `https://api.aisever.cn` / `https://newapi.aisever.cn` | `114.66.47.192` | `new-api-local` | `new-api` | 当前真实生产流量 |
| `kkidc` 测试环境 | **启用中** | `http://114.66.47.192:3001` | `114.66.47.192` | `new-api-test` | `new-api-test` | 新功能测试验证 |
| `tencent` 测试环境 | **已部署可用** | `http://123.206.229.105:3001` | `123.206.229.105` | `new-api-test` | `new-api-test` | `tencent` 服务器候选分支验证 |
| `tencent` 预备生产环境 | **已部署，未启用** | `http://123.206.229.105:3000` / 规划域名 `https://api.aisever.art` | `123.206.229.105` | `new-api`（compose 项目） | 迁移后数据 | 未来生产切换目标 |

## 2. 信息来源（本次已核对的本地分支）

| 分支 | 提供的信息 |
| --- | --- |
| `main-bak` | `kkidc` 生产 / 测试环境的历史部署信息、端口、容器名、数据库名 |
| `codex/release-*` | `tencent` 部署手册、发布分支与 `codex/prod-live` 策略 |
| `codex/prod-live` | 当前仓库内实际保留的 `tencent` 部署文档、迁移脚本、`.tencent` 配置入口 |

## 3. 目录说明

- `ops/environments/kkidc-production.md`：`kkidc` 生产环境
- `ops/environments/kkidc-test.md`：`kkidc` 测试环境
- `ops/environments/tencent-test.md`：`tencent` 测试环境
- `ops/environments/tencent-standby.md`：`tencent` 预备生产环境
- `.kkidc/.env.lighthouse`：新 `kkidc` 服务器连接信息（本地私有，不入库）
- `ops/checklists/kkidc-test-release.md`：`kkidc` 测试发布检查清单
- `ops/checklists/kkidc-production-release.md`：`kkidc` 生产发布检查清单
- `ops/checklists/tencent-test-release.md`：`tencent` 测试发布检查清单
- `ops/checklists/tencent-cutover.md`：`tencent` 切流检查清单
- `ops/scripts/`：按环境前缀命名的自定义脚本
- `ops/compose/`：按环境前缀命名的 compose 文件
- `docs/installation/TENCENT_TEST.md`：`tencent` 测试环境部署手册
- `docs/installation/DEPLOYMENT.md`：`tencent` 预备生产的详细部署/恢复手册

## 4. 管理规则

1. **功能发布顺序**：先 `kkidc` 测试环境，之后再进新 `kkidc` 生产环境；必要时再走 `tencent` 测试环境。`tencent` 预备生产环境当前只做迁移、验证、切换准备。
2. **不要混淆“测试环境”“已部署”和“已启用”**：`tencent` 测试环境用于候选分支验证，`tencent` 预备生产环境是“已部署但未切流”，不能当作当前生产。
3. **部署前先看本目录**：需要判断“该操作到底影响哪个环境”时，先读本目录，再进入对应环境文档。
4. **`tencent` 相关操作**：涉及 `api.aisever.art`、`/opt/new-api-src`、`codex/prod-live` 时，再继续阅读 `docs/installation/DEPLOYMENT.md`。
5. **新 `kkidc` 服务器**：使用 `ops/scripts/kkidc-host-deploy.sh production|test`，默认读取 `.kkidc/.env.lighthouse` 和 `.env.local`，并只部署当前分支已提交的 `HEAD`。

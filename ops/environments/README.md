# 部署环境总览

本目录用于统一管理 `new-api` 的部署环境信息，避免把不同服务器环境混在一起。
如需执行具体操作，请回到 `ops/README.md` 选择对应脚本或检查清单。

## 1. 当前口径（2026-03-13）

> 重要：**当前真实对外生产环境仍是新 `kkidc` 主机 `114.66.47.192` 上的 `https://api.aisever.cn`；企业生产环境目前仍待部署。**

| 环境 | 当前状态 | 访问地址 | 服务器 | 容器 | 数据库 | 用途 |
| --- | --- | --- | --- | --- | --- | --- |
| `kkidc` 个人生产环境 | **启用中** | `https://api.aisever.cn` / `https://newapi.aisever.cn` | `114.66.47.192` | `new-api-local` | `new-api` | 当前个人用户生产流量 |
| `kkidc` 企业生产环境 | **待部署** | 建议 `https://corp-api.aisever.cn` | `114.66.47.192` | `new-api-enterprise` | `new-api-enterprise` | 企业客户独立生产流量 |
| `kkidc` 测试环境 | **按需启动** | `http://114.66.47.192:3001` | `114.66.47.192` | `new-api-test` | `new-api-test` | 新功能测试验证，结束后关闭 |

## 2. 信息来源（本次已核对的本地分支）

| 分支 | 提供的信息 |
| --- | --- |
| `main-bak` | `kkidc` 生产 / 测试环境的历史部署信息、端口、容器名、数据库名 |

## 3. 目录说明

- `ops/environments/kkidc-production.md`：`kkidc` 生产环境
- `ops/environments/kkidc-enterprise-production.md`：`kkidc` 企业生产环境
- `ops/environments/kkidc-test.md`：`kkidc` 测试环境
- `.kkidc/.env.lighthouse`：新 `kkidc` 服务器连接信息（本地私有，不入库）
- `ops/checklists/kkidc-test-release.md`：`kkidc` 测试发布检查清单
- `ops/checklists/kkidc-production-release.md`：`kkidc` 生产发布检查清单
- `ops/checklists/kkidc-enterprise-production-release.md`：`kkidc` 企业生产发布检查清单
- `ops/scripts/`：按环境前缀命名的自定义脚本
- `ops/compose/`：按环境前缀命名的 compose 文件

## 4. 管理规则

1. **功能发布顺序**：先 `kkidc` 测试环境，之后再进对应的 `kkidc` 生产环境（个人或企业）。
2. **部署前先看本目录**：需要判断“该操作到底影响哪个环境”时，先读本目录，再进入对应环境文档。
3. **新 `kkidc` 服务器**：使用 `ops/scripts/kkidc-host-deploy.sh production|enterprise|test`，默认读取 `.kkidc/.env.lighthouse` 和 `.env.local`，并只部署已经 push 到远端校验分支的已提交 `HEAD`。
4. **测试环境收尾**：`kkidc` 用 `ops/scripts/kkidc-host-deploy.sh test-stop`；只停测试应用，不清空测试数据。

# bc32807fc 安全维护候选（2026-10-18）

> 后续状态：用户另行批准生产发布，2026-10-19 已执行同一不可变镜像的发布；最终状态见 [生产发布报告](../../releases/security-20261018/RESULTS.md)。下文“未发布”描述保留为 10 月 18 日候选验收时点的历史记录。

维护分支 `codex/security-v0.13.2-p1-20261018` 直接从生产源码 `bc32807fcaf6baa118db79cd21996b9b7e4aaca2` 建立。生产基线的 48 个定制提交见 custom-baseline-commits.txt；没有合并 v1 前端、替换品牌、取消定制接口或升级生产环境。

## 修复范围与依据

15 项公开公告逐项映射见 advisories.json。需要新增回补的五项是 quota 整数溢出、Redis 旧用户快照回写额度、匿名 webhook 无限请求体、越权重置 Passkey、用户列表泄露管理令牌。其余十项保留现有防护并复核；具体证据层次逐条标注，源码复核不等同真实提供商验收。

主要参考提交：d0bd8aac7、c9943d37a、bae799ccb、621927f71、d9595831b、dfc0d6324、d2f7f9ee3、0936e2504。另回补 df087b022 的拨号时 DNS/IP 校验与重定向检查；参考 057f71c23 在读取日志时统一隔离历史敏感元数据，保留旧 map 写入接口，避免破坏定制日志/计费调用；45c3fbe8a 的 OAuth 日志脱敏部分也已回补。

适配旧分支的额外修复：冷 Redis 缓存的 HSETNX 初始化、订阅偏好字段级更新、定制 billing/charge 数量乘法上限、邀请奖励原子累加、钱包 bigint 安全换算、NaN/Inf 审计序列化。单请求计费受 int32 安全边界约束；现有 bigint 钱包余额/充值容量保留。

上游的 scoped access token、Argon2id、全新 action-proof/登录绑定 API 及 v1 前端属于跨版本账户体系迁移，本候选不引入这些新增体系；现有 WebAuthn 完成标记、一次性消费、过期校验和身份切换清理已验证。不要把本候选称作 rc.45 全量功能/架构升级。

全量测试发现基线自身的 Claude PDF/文本文件转换和流诊断重置问题，修复并保留原有支持行为。

## 构建依赖与扫描边界

本地工具链 Go 1.27.2、Bun 1.4.3；go.mod 最低 Go 1.26.0。升级 x/net、x/text、x/crypto、x/image、gorilla/websocket、klauspost/compress。前端升级 axios、Vite、React Router、KaTeX 及锁定间接依赖；Docker 构建先复制 patches，再执行 frozen-lockfile 安装。

`braces@3.0.3` 的 GHSA-vfj7-8cjw-p6xm 尚无上游修复版：使用 Bun 原生补丁将 AST/数组递归改为显式栈，保留解析长度和展开数量限制。已比较原版与补丁版 1728 个语法/选项结果，验证最大解析深度与 50000 层调用方 AST。原始 bun audit 仍报告这一条，不能报告成原始扫描零公告。生产镜像只包含 Go 可执行文件和嵌入静态资源，不包含 node_modules 或构建工具。

Go govulncheck 仍可列出 x/crypto 中未引用的 OpenPGP 包公告 GO-2026-5932（无修复版）；须分别记录可达漏洞、导入包漏洞和仅模块级公告。

## 隔离测试与生产边界

`ops/scripts/kkidc-host-security-test.py` 仅管理带专属 label 的 `new-api-security-<run-id>-{app,postgres,redis,fixture}`。新网络、新数据目录、新随机凭据、独立数据库/Redis，应用仅发布 `127.0.0.1:13018`，不改 Caddy/DNS/公网入口。镜像本地编译，不在生产主机构建。容器内存/CPU/PID/日志受限。

每次 deploy/status/stop 对生产 new-api-local、new-api-postgres、new-api-redis、snowlight-caddy 和旧 new-api-test 比较容器 ID、镜像 ID、启动时间、重启计数、运行状态与生产版本。任何目标所有权不符立即拒绝。测试不读取或导入生产数据库；生产 fhy/订阅 131 不参与测试、不被写入。

`ops/tests/security_http_acceptance.py` 只允许操作上述 label 的容器，使用合成管理员、令牌、订阅和上游，验证 HTTP/SSE 转发、显式零值、实际扣费、极端输入拒绝、webhook 上限、并发设置更新、令牌隐藏、北京时间零点订阅锚点以及测试应用重启。真实模型、支付、OAuth 提供商和硬件 WebAuthn 验收不由合成上游替代。

## 生产发布方案（尚未执行）

1. 审阅最终测试回执和不可变镜像 ID，确认候选代码提交；发布前重新只读核验生产版本、挂载、资源、数据库 schema、Caddy 和订阅 131 当前锚点/已用额度。不要用测试数据库覆盖生产。
2. 取得明确生产发布授权后，在维护窗口制作 PostgreSQL 一致性备份（含日志库如分离）、应用数据/配置备份与镜像/运行参数清单；在隔离数据库恢复验证备份可用。记录订阅/余额/邀请比例/分组与任务定价配置基线。
3. 复用已验证镜像，沿用 `/opt/new-api/data`、`/opt/new-api/logs`、现有 production DB/Redis、SESSION_SECRET/CRYPTO_SECRET 与网络。只替换生产应用容器；不更换 DB/Redis/Caddy。检查实际迁移差异后再启动，不执行测试初始化脚本或重置订阅。
4. 启动后先检查内部健康、版本、迁移日志，再检查两个公网域名、管理员/普通用户登录、真实小额聊天/SSE、定制 billing/charge、订阅与钱包扣费、模型/分组定价、邀请奖励。对照余额变化和管理员溢出告警。观察 30–60 分钟，并跨北京时间零点确认订阅调度；保留当前已用额度。
5. 失败时停止新应用并恢复原镜像 `new-api:kkidc-production-bc32807fc` 及完整原参数，保持密钥与挂载不变。发布前须验证旧代码可读候选自动迁移后的 schema；若确需数据库恢复，先停止写入并核对切换后新增交易，不能直接回灌旧备份丢失消费记录。

最终结果与镜像摘要补充在 RESULTS.md 和 evidence/；未经生产发布授权不得执行上述发布步骤。

# 邀请功能远端测试部署

时间：2026-11-01T23:31:18+09:00（Asia/Tokyo）。应用源码提交 `8d4ef4c3546bdc5053d3aca3d9427ff6f2c87009`。

已部署至 `43.133.183.213` 上独立的 `new-api-invitation-20261101-app/postgres/redis`。版本 `codex/invite+8d4ef4c35+test`。新环境使用专用 Docker 网络、独立 PostgreSQL 与 Redis、独立目录 `.local/state/new-api-invitation-preview/20261101`；没有复制生产用户、密钥、支付配置或额度。测试金额均为合成数据。

## 查看

- 这台电脑入口：[钱包及邀请奖励](http://127.0.0.1:13019/console/topup)。通过 SSH 转发至远端回环端口；本机和远端都只监听 127.0.0.1，没有新增公网测试入口，也没有改 Caddy 路由。
- 已在侧栏打开并登录 `invite-owner`，邀请人数 22；打开名单并搜索 `invitee-22` 可看到 USD 6.3、CNY 10 与 1 条未确认历史记录。
- 管理员账号 `inviteadmin`，登录后在“系统设置 → 运营 → 邀请金额可见范围”查看和调整。
- 密码和操作说明只保存在忽略目录 `.env/artifacts/invitation-stage-20261101/access.txt`，权限 0600，不提交 Git。
- 当前合成演示配置为 selected，只授权 inviter ID 2；产品初始默认仍为关闭。未授权账号 `other-inviter` 仅能看自己的名单。
- 留下测试服务运行供用户验收。远端三个测试容器 restart=unless-stopped；本机临时用户服务 `new-api-invitation-preview-20261101.service` 维持 SSH 转发。测试结束后可停止该通道及本次带 `codex.task=new-api-invitation-20261101` 标签的远端容器，保留数据。不要调用旧测试环境的 test-stop 去替代本次专用容器检查。

## 验证

已验证完整 9 项 HTTP 合成数据检查（权限、归属、默认/授权/撤销、币种与镜像去重、余额及订阅锚点保护）。真实远端浏览器：中文界面、邀请人数、20+2 分页、用户名筛选、USD 6.3/CNY 10、历史未确认说明、后台用户名搜索和配置回显、切换管理员和邀请方。

最终复核 18 个原有容器的 ID、镜像、启动时间、运行状态和重启次数均与部署前相同；Caddy 文件摘要相同。生产依然是 `codex/sec-v0.13.2-p1+ba844d202+production`。本次没有查询或写入生产数据库，没有操作 fhy 或订阅 131。新 app/Postgres/Redis 正常运行，重启 0 次、无 OOM。

传输时因完整镜像归档较慢，改为只上传已验证程序的 xz 压缩包；远端校验压缩包摘要、程序摘要，并以已存在的不可变基础镜像 `sha256:d07e5682b016202338524f832d8206f2d99ac4b916a1391db92100cdfb9405e1` 构建专用测试镜像。容器内再次校验 `/new-api` SHA256 为 `740ddc7beff8aed2c7a262d8ef8678219c5d5e22fd873778dcf1ad572200194e`，与本地已验收程序相同。镜像元数据记录此次源码提交，最终远端镜像摘要为 `sha256:c7a888b2899b7758838838458c478d3a47c4538fb09d2a51cd643dfba40fbe47`。未重新编译应用或修改既有镜像标签。

截图：[名单](evidence/preview.png)、[后台](evidence/admin.png)。机器证据：[最终审计](evidence/final-audit.json)、[部署范围](evidence/deployment.json)、[HTTP 验收](evidence/acceptance.log)。真实外部支付尚未执行；本次是测试环境部署与合成数据验收，不代表已发布生产。

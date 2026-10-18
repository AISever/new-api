# ba844d202 生产发布记录与恢复入口

这是针对 `43.133.183.213` 的一次性发布材料。最终结论、校验值和验收边界见 [RESULTS.md](RESULTS.md)，此前的安全修复与测试矩阵见 [安全维护报告](../../security/2026-10-18/RESULTS.md)。所有时间均为北京时间。

## 已批准的范围

用户授权：先备份并验证可恢复，以已验证的 `ba844d202` 镜像发布生产，保留定制功能、配置和用户额度；fhy 订阅 131 继续每日北京时间 00:00 重置，异常立即回滚。

- 不重新构建应用。使用不可变镜像 `sha256:d07e5682b016202338524f832d8206f2d99ac4b916a1391db92100cdfb9405e1`。
- 原应用容器完整保留。PostgreSQL、Redis、Caddy 和原有 `new-api-test` 均保持原容器、启动时间、重启计数。
- 切换窗口临时加载 Caddy 维护响应并添加带专属注释的 DOCKER-USER 规则；流量排空后替换应用，再恢复原始路由并删除该规则。Caddyfile 不改动。
- 应用 Env 仅修改 VERSION；挂载、生产 DB/Redis、密钥、端口、网络、重启策略及资源设置沿用原值。Docker 自动继承新镜像 OCI 来源标签，原有自定义标签保留。
- 生产信息、密码、私钥、数据库、原始 inspect 和配置均放在私有目录，禁止提交到 Git。

## 备份与恢复验证

`backup_remote.py` 导出 PostgreSQL 只读一致性快照及所有表指纹，备份角色、挂载数据/日志、原始 Docker 运行参数、Caddy 配置及旧镜像。`restore_remote.py` 在无公网端口的内部网络完整恢复并对比所有表行内容，实际启动候选验证生产数据兼容性。`redis_backup_remote.py` 导出 RDB，在 network=none 的独立 Redis 校验并加载。所有恢复容器验证后停止。

`release_worker.py` 还在原应用停止、批处理落库后生成新的切换点数据库备份；`verify_cutover_backup_remote.py` 再次隔离恢复并比较账务/配置摘要及订阅 131 全行。禁止用测试数据库覆盖生产数据库。

服务器私有状态：`/home/ubuntu/.local/state/new-api-security-release-20261018/`。
服务器备份：`/opt/new-api-backups/security-20261018-233325/`。
机外备份：`/home/ctyun/Works/new-api/.env/backups/security-production-20261018/`。

## 自动监控与回滚

发布 worker 持有独占锁，观察 30 分钟，检查内部及两个公网域名、镜像/容器身份、重启/OOM、其他服务身份、Caddyfile 摘要及订阅零点锚点。独立 guardian 检查 worker 存活/心跳；独立 ledger audit 检查消费负数及 panic/额度批处理失败。异常触发应用镜像回滚，不回灌旧数据库，不丢弃发布后的交易。

若观察期间需要人工立即回滚：先读取私有 `journal.json`，核实 worker_pid 对应 `/proc/<pid>/cmdline` 确为本次 release_worker.py，再向该进程发送 SIGTERM。worker 的信号处理会进入同一回滚路径；核验最终 phase=rolled_back、旧镜像、三个健康接口和维护规则清理。不要启动第二个并行发布进程。

观察结束后，若出现经确认的版本回归：核实当前应用仍为本次候选、旧容器和原始状态文件仍在，执行服务器状态目录内 `python3 release_worker.py --rollback`，然后核验 journal 与公网健康。该操作继续使用当前生产数据库；如果确需数据恢复，必须先停止写入、保存当前数据并逐笔核对切换后交易，另作恢复决策。

旧容器名：`new-api-local-rollback-security-20261018`，平时停止且 restart=no。回滚函数恢复其原容器名及 restart=always，再启动。初次发布曾真实执行并验证自动回滚，详见结果报告。

## 验收和脚本使用约束

`probe_remote.py` 只操作专用账号 sec261018、限额令牌和极小真实模型调用，既有用户额度不参与验收。结束后禁用账号及令牌，保留账务审计记录。不要重跑它，以免额外发起生产验收调用。

这些脚本依赖服务器私有初始状态，**不是通用部署入口，不应重放本次发布**。仓库中的发布控制另补了已有 journal 拒绝重放、非阻塞锁及切换前错误不回滚的防护；这些运维脚本加固在本次在线进程启动后完成，不改变运行镜像，也未重启健康的生产服务。控制逻辑的离线测试运行方式：

```sh
python3 ops/releases/security-20261018/test_release_controls.py
```

公开证据只包含无凭据摘要和页面截图；原始私有状态留存用于审计与恢复。

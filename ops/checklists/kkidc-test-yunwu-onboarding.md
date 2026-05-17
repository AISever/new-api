# kkidc 测试环境 Yunwu 接入检查清单

适用对象：`http://114.66.47.192:3001`

适用场景：将 `yunwu.ai` 的 `Kling`、`Vidu`、`DoubaoVideo` 聚合内容导入 `kkidc` 测试环境，验证渠道创建、family 探测、后台可见性与测试环境健康状态。

## 导入前

- 先阅读 `ops/environments/kkidc-test.md`。
- 确认本次操作目标是 `kkidc` 测试环境，而不是个人生产或企业生产环境。
- 确认使用的 profile 是预期版本：
  - `ops/channel-onboarding/profiles/yunwu.yaml`
- 准备三类 family 的有效渠道 key：
  - `kling`
  - `vidu`
  - `doubao-video`
- 如需保留当前测试环境数据，先执行：
  - `bash ops/scripts/kkidc-host-backup.sh test`
- 确认不会误用个人生产域名或个人生产 root 凭据。

## Dry Run

先执行：

```bash
bash ops/scripts/kkidc-host-import-profile.sh test --profile ops/channel-onboarding/profiles/yunwu.yaml --dry-run
```

检查输出至少包含：

- `families` 为 `doubao-video`、`kling`、`vidu`
- `planned_channels` 为 3 条
- `tag` 为 `personal-upstream-yunwu`
- `risks` 中仍明确提示倍率为 `manual-review`

## 导入与 Probe

执行：

```bash
bash ops/scripts/kkidc-test-yunwu-onboarding.sh \
  --target-root-username <root> \
  --target-root-password <password> \
  --channel-key kling=<key> \
  --channel-key vidu=<key> \
  --channel-key doubao-video=<key>
```

检查输出至少包含：

- `probe_summary.passed = 3`
- `target_channels_created = 3` 或与预期更新数一致
- `provided_channel_key_families` 覆盖三类 family
- `test_status_check.success = true`

## 导入后验证

- `curl http://114.66.47.192:3001/api/status`
- 登录测试环境后台
- 确认存在以下 3 条渠道或同等更新结果：
  - `newapi::yunwu::kling::yunwu-video`
  - `newapi::yunwu::vidu::yunwu-video`
  - `newapi::yunwu::doubao-video::yunwu-video`
- 确认三条渠道的 `tag` 为 `personal-upstream-yunwu`
- 确认三条渠道的 `group` 为 `yunwu-video`
- 检查后台模型可见性，至少覆盖：
  - `kling-v1`
  - `viduq2`
  - `doubao-seedance-1-0-pro-250528`
- 确认无关既有渠道未被删除
- 交叉验证个人生产健康状态：
  - `curl https://api.aisever.cn/api/status`

## 收尾

- 记录本次使用的 profile、测试时间、family key 来源、探测结果、导入输出摘要。
- 若本次仅为验证，完成后按需要执行：
  - `bash ops/scripts/kkidc-host-deploy.sh test-stop`
- 若测试环境需保留给后续人工验证，明确记录不执行 `test-stop` 的原因。

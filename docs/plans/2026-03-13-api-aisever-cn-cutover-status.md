# api.aisever.cn Cutover Status (2026-03-13)

- Target host: `114.66.47.192`
- Release image: `new-api:kkidc-production-663463bb`
- Version header: `codex/release-v0.11.2-patch.2-p1+663463bb+kkidc-production`
- Primary hostname in new host config: `api.aisever.cn`
- Alias hostname in new host config: `newapi.aisever.cn`

## Completed

- New `kkidc` production host serves both `api.aisever.cn` and `newapi.aisever.cn` over HTTPS locally on the target host.
- `ServerAddress` on the new production database has been switched to `https://api.aisever.cn`.
- Production data was resynced from legacy `kkidc` production before cutover.
- Key production counts match legacy production: `users=184`, `tokens=239`, `channels=26`, `logs=24899`.

## Verified commands

- `curl -kI --resolve api.aisever.cn:443:114.66.47.192 https://api.aisever.cn/help` -> `200`
- `curl -kI --resolve newapi.aisever.cn:443:114.66.47.192 https://newapi.aisever.cn/help` -> `200`
- `curl -kfsS --resolve api.aisever.cn:443:114.66.47.192 https://api.aisever.cn/api/status`

## Latest public verification

- `2026-03-13 16:23`（Asia/Shanghai）从当前工作站再次验证：`https://api.aisever.cn` 已返回 `HTTP/2 200`。
- 响应头版本为 `codex/release-v0.11.2-patch.2-p1+663463bb+kkidc-production`。
- 说明公网入口已经能够命中新 `kkidc` 生产环境。

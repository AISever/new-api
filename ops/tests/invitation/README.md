# 隔离邀请功能验收

只使用合成账号和独立数据库，不接受生产地址或生产数据库配置。脚本创建资源统一使用 `codex.task=new-api-invitation-20261101` 标签；既有同名资源标签不匹配即拒绝继续。需要 Docker、本地 Go 工具链，以及已验证的本地基础镜像 `new-api:security-ba844d202`。

从仓库根目录执行：

```bash
export GO_BIN=/path/to/go
python3 ops/tests/invitation/databases.py
# 先按项目约定完成 web 的 Bun 构建，然后编译包含静态页面的应用。
CGO_ENABLED=0 "$GO_BIN" build -ldflags '-s -w' -o .env/artifacts/invitation-test-20261101/new-api .
python3 ops/tests/invitation/app.py
python3 ops/tests/invitation/proxy.py
# 另一个终端：
python3 ops/tests/invitation/acceptance.py
```

凭据由脚本生成并仅保存在忽略目录 `.env/artifacts/invitation-test-20261101`，权限为 0600。浏览器入口固定为 `http://127.0.0.1:13019`；测试登录账号在该目录 `ui-credentials.json`，均为合成数据。不得上传此目录。

数据库运行在 internal Docker 网络；测试二进制共享对应数据库容器的网络命名空间，以 localhost 访问独立 `invitation_test` 库。代理仅监听本机回环，不能改为公网监听。界面数据在另一个 `invitation_ui` 库中；API 种子写入前核对专用版本标识。

重复运行数据库测试会清除本任务独立 `invitation_test` 库的测试表数据。`app.py` 不替换已存在的应用容器；新构建验收需先核对本任务 app 容器标签，再仅停止和移除该 app 容器，重新运行 app.py；保留数据库容器。代理启动时读取目标地址，若重建导致地址改变则重新启动本任务代理。

结束后仅停止带本任务标签的 app/postgres/mysql 容器和本任务代理。保留私有状态目录以便复核；不要执行全局 Docker 清理，不触碰其他容器或网络。

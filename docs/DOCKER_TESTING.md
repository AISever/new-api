# Docker本地测试环境部署指南

本文档定义了new-api项目本地Docker部署测试的标准流程。所有开发者在提交代码前必须遵循此流程进行测试。

## 快速开始

```bash
# 1. 构建前端
cd web && bun install && bun run build && cd ..

# 2. 构建Docker镜像
docker build -t new-api:local .

# 3. 修改docker-compose.yml
# 将 image: calciumion/new-api:latest 改为 image: new-api:local

# 4. 启动服务
docker-compose up -d

# 5. 验证部署
curl http://localhost:3000/api/status
```

访问 http://localhost:3000，默认账号：root / 123456

## 标准测试流程

### 1. 代码修改后的测试

```bash
# 重新构建镜像
docker build -t new-api:local .

# 重启服务
docker-compose restart new-api

# 查看日志
docker-compose logs -f new-api
```

### 2. 数据库兼容性测试（必须）

根据CLAUDE.md规则2，所有数据库代码必须兼容PostgreSQL、MySQL和SQLite。

#### 测试PostgreSQL（默认）

```bash
docker-compose up -d
# 执行功能测试...
docker-compose logs new-api | grep -i error
```

#### 测试MySQL

```bash
# 1. 停止服务
docker-compose down -v

# 2. 编辑docker-compose.yml
#    - 注释第29行（PostgreSQL SQL_DSN）
#    - 取消注释第30行（MySQL SQL_DSN）
#    - 注释第44行（postgres依赖）
#    - 取消注释第45行（mysql依赖）
#    - 注释第57-68行（postgres服务）
#    - 取消注释第70-80行（mysql服务）
#    - 注释第83行（pg_data卷）
#    - 取消注释第84行（mysql_data卷）

# 3. 启动服务
docker-compose up -d

# 4. 执行相同的功能测试
docker-compose logs new-api | grep -i error
```

#### 测试SQLite

```bash
# 1. 停止服务
docker-compose down -v

# 2. 编辑docker-compose.yml
#    - 注释所有SQL_DSN行（第29-30行）

# 3. 启动服务
docker-compose up -d

# 4. 执行相同的功能测试
docker-compose logs new-api | grep -i error
```

### 3. 前端修改测试

```bash
# 1. 重新构建前端
cd web
bun run build
cd ..

# 2. 重新构建镜像
docker build -t new-api:local .

# 3. 重启服务
docker-compose restart new-api

# 4. 清除浏览器缓存后测试
```

## 常用命令

### 日志查看

```bash
# 查看所有日志
docker-compose logs -f

# 仅查看错误
docker-compose logs new-api | grep -i error

# 查看最近100行
docker-compose logs --tail=100 new-api

# 查看错误日志文件
cat logs/error.log
```

### 数据库操作

```bash
# 连接PostgreSQL
docker-compose exec postgres psql -U root -d new-api

# 连接MySQL
docker-compose exec mysql mysql -uroot -p123456 new-api

# 查看SQLite数据库
docker-compose exec new-api ls -lh /data/new-api.db
```

### 容器调试

```bash
# 进入容器
docker-compose exec new-api sh

# 查看容器资源使用
docker stats new-api

# 重启特定服务
docker-compose restart new-api
```

## 清理环境

```bash
# 停止服务（保留数据）
docker-compose down

# 完全清理（删除所有数据）
docker-compose down -v
rm -rf logs/* data/*

# 删除本地镜像
docker rmi new-api:local
```

## 故障排查

### 问题：服务无法启动

```bash
# 1. 检查端口占用
lsof -i :3000

# 2. 查看详细日志
docker-compose logs new-api

# 3. 检查容器状态
docker-compose ps
```

### 问题：数据库连接失败

```bash
# 1. 确认数据库服务运行
docker-compose ps postgres  # 或 mysql

# 2. 检查数据库日志
docker-compose logs postgres  # 或 mysql

# 3. 验证连接字符串
docker-compose exec new-api env | grep SQL_DSN
```

### 问题：前端404错误

```bash
# 1. 确认前端已构建
ls -la web/dist

# 2. 重新构建
cd web && bun run build && cd ..
docker build -t new-api:local .
docker-compose restart new-api
```

## 提交前检查清单

在提交代码前，必须确认：

- [ ] 代码在PostgreSQL环境下测试通过
- [ ] 代码在MySQL环境下测试通过
- [ ] 代码在SQLite环境下测试通过
- [ ] 无错误日志（检查logs/error.log）
- [ ] 前端功能正常（清除缓存后测试）
- [ ] API健康检查通过（`curl http://localhost:3000/api/status`）
- [ ] 已清理测试环境（`docker-compose down -v`）

## 环境变量说明

测试环境常用配置：

```yaml
# 数据库连接（三选一）
SQL_DSN: postgresql://root:123456@postgres:5432/new-api  # PostgreSQL
SQL_DSN: root:123456@tcp(mysql:3306)/new-api             # MySQL
# 不设置SQL_DSN则使用SQLite

# Redis连接
REDIS_CONN_STRING: redis://redis

# 时区
TZ: Asia/Shanghai

# 日志配置
ERROR_LOG_ENABLED: true
BATCH_UPDATE_ENABLED: true

# 流式超时（秒）
STREAMING_TIMEOUT: 300
```

## 性能测试（可选）

```bash
# 简单压测
ab -n 1000 -c 10 http://localhost:3000/api/status

# 详细压测
wrk -t4 -c100 -d30s http://localhost:3000/api/status

# 监控资源
docker stats
```

## 注意事项

⚠️ **重要**：

1. 测试环境密码（123456）仅用于本地开发
2. 数据库兼容性测试是强制性的，不可跳过
3. 测试完成后必须清理环境（`docker-compose down -v`）
4. 不要将测试配置提交到版本控制
5. 修改数据库相关代码时，必须在三种数据库上都测试

## 使用Claude Code Skill

项目提供了`/docker-test` skill，可以快速查看此文档：

```bash
/docker-test
```

该skill包含完整的部署流程、故障排查和最佳实践。

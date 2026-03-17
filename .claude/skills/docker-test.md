---
skill: docker-test
description: 本地Docker部署测试环境的标准操作流程
tags: [docker, deployment, testing]
---

# Docker测试环境部署标准

本skill定义了new-api项目本地Docker部署测试环境的标准操作流程，确保所有AI Agent开发者遵循统一的部署和测试规范。

## 前置条件检查

在开始部署前，必须确认：

1. Docker和Docker Compose已安装并运行
2. 端口3000、5432（PostgreSQL）、6379（Redis）未被占用
3. 有足够的磁盘空间（至少2GB）

## 标准部署流程

### 1. 构建前端资源

```bash
cd web
bun install
bun run build
cd ..
```

### 2. 构建Docker镜像

```bash
# 构建本地镜像（用于测试未发布的代码）
docker build -t new-api:local .
```

### 3. 修改docker-compose.yml（仅测试环境）

将`image: calciumion/new-api:latest`改为`image: new-api:local`以使用本地构建的镜像。

### 4. 启动服务

```bash
# 启动所有服务
docker-compose up -d

# 查看日志
docker-compose logs -f new-api

# 检查服务状态
docker-compose ps
```

### 5. 验证部署

```bash
# 检查API健康状态
curl http://localhost:3000/api/status

# 预期返回：{"success":true,...}
```

### 6. 访问测试

- 前端界面：http://localhost:3000
- 默认管理员账号：root
- 默认密码：123456（首次登录后必须修改）

## 数据库选择

### 使用PostgreSQL（默认）

无需修改，直接使用docker-compose.yml的默认配置。

### 切换到MySQL

1. 编辑docker-compose.yml：
   - 注释掉postgres服务和第29行的SQL_DSN
   - 取消注释mysql服务和第30行的SQL_DSN
   - 取消注释depends_on中的mysql（第45行）
   - 取消注释volumes中的mysql_data（第84行）

2. 重启服务：
   ```bash
   docker-compose down
   docker-compose up -d
   ```

### 使用SQLite（轻量测试）

1. 编辑docker-compose.yml，注释掉SQL_DSN环境变量
2. SQLite数据库将自动创建在`./data/new-api.db`

## 常见测试场景

### 测试新功能

```bash
# 1. 修改代码后重新构建
docker build -t new-api:local .

# 2. 重启服务
docker-compose restart new-api

# 3. 查看日志确认启动成功
docker-compose logs -f new-api
```

### 测试数据库迁移

```bash
# 1. 停止服务
docker-compose down

# 2. 清理数据卷（⚠️ 会删除所有数据）
docker volume rm new-api_pg_data

# 3. 重新启动
docker-compose up -d
```

### 测试多数据库兼容性

```bash
# 测试PostgreSQL
docker-compose up -d
# 执行测试...

# 切换到MySQL
docker-compose down
# 修改docker-compose.yml切换到MySQL
docker-compose up -d
# 执行相同测试...

# 切换到SQLite
docker-compose down
# 修改docker-compose.yml使用SQLite
docker-compose up -d
# 执行相同测试...
```

## 日志和调试

### 查看日志

```bash
# 查看所有服务日志
docker-compose logs -f

# 仅查看new-api日志
docker-compose logs -f new-api

# 查看最近100行日志
docker-compose logs --tail=100 new-api

# 查看错误日志文件
cat logs/error.log
```

### 进入容器调试

```bash
# 进入new-api容器
docker-compose exec new-api sh

# 进入PostgreSQL容器
docker-compose exec postgres psql -U root -d new-api

# 进入Redis容器
docker-compose exec redis redis-cli
```

### 检查数据库连接

```bash
# PostgreSQL
docker-compose exec postgres psql -U root -d new-api -c "SELECT version();"

# MySQL（如果使用）
docker-compose exec mysql mysql -uroot -p123456 -e "SELECT VERSION();"
```

## 清理环境

### 停止服务（保留数据）

```bash
docker-compose down
```

### 完全清理（删除所有数据）

```bash
# 停止并删除容器、网络、数据卷
docker-compose down -v

# 删除本地镜像
docker rmi new-api:local

# 清理日志和数据目录
rm -rf logs/* data/*
```

## 性能测试

### 基准测试

```bash
# 使用ab进行简单压测
ab -n 1000 -c 10 http://localhost:3000/api/status

# 使用wrk进行更详细的压测
wrk -t4 -c100 -d30s http://localhost:3000/api/status
```

### 监控资源使用

```bash
# 查看容器资源使用情况
docker stats

# 查看特定容器的资源使用
docker stats new-api postgres redis
```

## 故障排查

### 服务无法启动

1. 检查端口占用：`lsof -i :3000`
2. 检查Docker日志：`docker-compose logs new-api`
3. 检查数据库连接：确认SQL_DSN配置正确

### 数据库连接失败

1. 确认数据库服务已启动：`docker-compose ps`
2. 检查数据库日志：`docker-compose logs postgres`
3. 验证连接字符串格式是否正确

### 前端资源404

1. 确认前端已构建：检查`web/dist`目录是否存在
2. 重新构建镜像：`docker build -t new-api:local .`
3. 重启服务：`docker-compose restart new-api`

## 环境变量配置

测试环境常用环境变量：

```yaml
# 必需配置
- SQL_DSN=postgresql://root:123456@postgres:5432/new-api
- REDIS_CONN_STRING=redis://redis
- TZ=Asia/Shanghai

# 调试配置
- ERROR_LOG_ENABLED=true
- BATCH_UPDATE_ENABLED=true
- STREAMING_TIMEOUT=300

# 多机部署测试
- SESSION_SECRET=test_random_string_change_in_production
- SYNC_FREQUENCY=60
```

## 最佳实践

1. **每次修改代码后必须重新构建镜像**
2. **测试前确保数据库状态干净**（使用`docker-compose down -v`清理）
3. **测试多数据库兼容性**（PostgreSQL、MySQL、SQLite都要测试）
4. **检查日志确认无错误**（特别是error.log）
5. **测试完成后清理环境**（避免占用资源）
6. **不要在测试环境使用生产数据**
7. **修改默认密码**（即使是测试环境）

## 持续集成建议

在CI/CD流程中集成Docker测试：

```bash
# CI脚本示例
#!/bin/bash
set -e

# 构建镜像
docker build -t new-api:ci .

# 启动测试环境
docker-compose -f docker-compose.test.yml up -d

# 等待服务就绪
sleep 10

# 运行健康检查
curl -f http://localhost:3000/api/status || exit 1

# 运行集成测试
# ... 你的测试命令 ...

# 清理
docker-compose -f docker-compose.test.yml down -v
```

## 注意事项

⚠️ **重要提醒**：

1. 测试环境的默认密码（123456）仅用于本地开发，生产环境必须修改
2. 测试完成后及时清理Docker资源，避免占用磁盘空间
3. 不要将测试环境的配置文件提交到版本控制系统
4. 测试数据库迁移时，务必备份重要数据
5. 多数据库兼容性测试是强制性的，不可跳过

## 快速参考

```bash
# 快速启动
docker build -t new-api:local . && docker-compose up -d

# 快速重启
docker-compose restart new-api

# 快速查看日志
docker-compose logs -f new-api | grep -i error

# 快速清理
docker-compose down -v && rm -rf logs/* data/*

# 快速健康检查
curl http://localhost:3000/api/status | jq .
```

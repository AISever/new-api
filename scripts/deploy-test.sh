#!/bin/bash

# New-API 测试环境部署脚本
# 用于在服务器上部署独立的测试环境（端口 3001）

set -e

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# 配置
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"
ENV_FILE="$PROJECT_ROOT/.env.local"

# 测试环境配置
TEST_IMAGE_NAME="new-api:test"
TEST_CONTAINER_NAME="new-api-test"
TEST_PORT="3001"
TEST_DATA_DIR="/opt/new-api-test"
TEST_DB_NAME="new-api-test"

# 默认值
DEPLOY_MODE="remote"
SKIP_BUILD=false
SKIP_SYNC=false
CLEAR_CACHE=false
CREATE_DB=false

# 打印带颜色的消息
print_info() { echo -e "${BLUE}[INFO]${NC} $1"; }
print_success() { echo -e "${GREEN}[SUCCESS]${NC} $1"; }
print_warning() { echo -e "${YELLOW}[WARNING]${NC} $1"; }
print_error() { echo -e "${RED}[ERROR]${NC} $1"; }
print_step() { echo -e "${GREEN}[STEP]${NC} $1"; }

# 显示帮助信息
show_help() {
    echo "==========================================
       New-API 测试环境部署脚本
==========================================

用法: $0 [模式] [选项]

模式:
  local           本地部署模式（在当前机器构建和运行）
  remote          远程服务器部署模式（默认）

选项:
  --skip-build    跳过镜像构建，只重启容器
  --skip-sync     跳过代码同步（仅远程模式）
  --clear-cache   部署后清除测试环境的 Redis 缓存
  --create-db     创建测试数据库（首次部署时使用）
  --help          显示此帮助信息

示例:
  $0 remote                    # 远程部署测试环境
  $0 remote --create-db        # 首次部署，创建测试数据库
  $0 remote --skip-build       # 跳过构建，只重启容器
  $0 local                     # 本地部署测试环境
  $0 local --clear-cache       # 本地部署并清除缓存

测试环境信息:
  - 端口: $TEST_PORT
  - 容器名: $TEST_CONTAINER_NAME
  - 镜像名: $TEST_IMAGE_NAME
  - 数据目录: $TEST_DATA_DIR
  - 数据库: $TEST_DB_NAME
"
}

# 加载环境变量
load_env() {
    if [ -f "$ENV_FILE" ]; then
        print_info "加载环境变量: $ENV_FILE"
        set -a
        source "$ENV_FILE"
        set +a
    else
        print_warning "未找到 .env.local 文件，使用默认配置"
    fi
}

# 检查必要的环境变量（远程模式）
check_remote_env() {
    local missing=()
    [ -z "$SERVER_IP" ] && missing+=("SERVER_IP")
    [ -z "$SERVER_USER" ] && missing+=("SERVER_USER")
    [ -z "$SERVER_PASSWORD" ] && missing+=("SERVER_PASSWORD")
    
    if [ ${#missing[@]} -gt 0 ]; then
        print_error "缺少必要的环境变量: ${missing[*]}"
        print_info "请在 .env.local 中配置这些变量"
        exit 1
    fi
}

# 远程执行命令
remote_exec() {
    sshpass -p "$SERVER_PASSWORD" ssh -o StrictHostKeyChecking=no "$SERVER_USER@$SERVER_IP" "$@"
}

# 同步代码到服务器
sync_code() {
    print_step "同步代码到服务器..."
    
    # 使用 rsync 同步代码
    sshpass -p "$SERVER_PASSWORD" rsync -avz \
        --exclude='.git' \
        --exclude='node_modules' \
        --exclude='web/build' \
        --exclude='web/node_modules' \
        --exclude='data' \
        --exclude='logs' \
        --exclude='*.log' \
        "$PROJECT_ROOT/" "$SERVER_USER@$SERVER_IP:/tmp/new-api-build/"
    
    print_success "代码同步完成"
}

# 创建测试数据库
create_test_database() {
    print_step "创建测试数据库: $TEST_DB_NAME"
    
    if [ "$DEPLOY_MODE" = "remote" ]; then
        remote_exec "docker exec new-api-postgres psql -U newapi -d new-api -tc \"SELECT 1 FROM pg_database WHERE datname = '$TEST_DB_NAME'\" | grep -q 1 || docker exec new-api-postgres psql -U newapi -d new-api -c \"CREATE DATABASE \\\"$TEST_DB_NAME\\\";\""
    else
        docker exec new-api-postgres psql -U newapi -d new-api -tc "SELECT 1 FROM pg_database WHERE datname = '$TEST_DB_NAME'" | grep -q 1 || \
        docker exec new-api-postgres psql -U newapi -d new-api -c "CREATE DATABASE \"$TEST_DB_NAME\";"
    fi
    
    print_success "测试数据库准备完成"
}

# 创建测试数据目录
create_test_dirs() {
    print_step "创建测试数据目录..."
    
    if [ "$DEPLOY_MODE" = "remote" ]; then
        remote_exec "mkdir -p $TEST_DATA_DIR/data $TEST_DATA_DIR/logs"
    else
        mkdir -p "$TEST_DATA_DIR/data" "$TEST_DATA_DIR/logs"
    fi
    
    print_success "数据目录创建完成"
}

# 构建测试镜像
build_image() {
    print_step "构建 Docker 镜像: $TEST_IMAGE_NAME"
    
    if [ "$DEPLOY_MODE" = "remote" ]; then
        remote_exec "cd /tmp/new-api-build && docker build --no-cache -t $TEST_IMAGE_NAME --build-arg GOPROXY=https://goproxy.cn,direct ."
    else
        cd "$PROJECT_ROOT"
        docker build --no-cache -t "$TEST_IMAGE_NAME" --build-arg GOPROXY=https://goproxy.cn,direct .
    fi
    
    print_success "镜像构建完成"
}

# 停止并删除旧的测试容器
stop_container() {
    print_step "停止旧的测试容器..."
    
    if [ "$DEPLOY_MODE" = "remote" ]; then
        remote_exec "docker stop $TEST_CONTAINER_NAME 2>/dev/null || true"
        remote_exec "docker rm $TEST_CONTAINER_NAME 2>/dev/null || true"
    else
        docker stop "$TEST_CONTAINER_NAME" 2>/dev/null || true
        docker rm "$TEST_CONTAINER_NAME" 2>/dev/null || true
    fi
}

# 启动测试容器
start_container() {
    print_step "启动测试容器..."
    
    local run_cmd="docker run -d --name $TEST_CONTAINER_NAME --restart always \
        -p $TEST_PORT:3000 \
        -v $TEST_DATA_DIR/data:/data \
        -v $TEST_DATA_DIR/logs:/app/logs \
        -e SQL_DSN='postgresql://newapi:${DB_PASSWORD:-NewApi2024Secure}@new-api-postgres:5432/$TEST_DB_NAME' \
        -e REDIS_CONN_STRING='redis://new-api-redis/1' \
        -e TZ=Asia/Shanghai \
        -e SESSION_SECRET='${SESSION_SECRET:-test-session-secret-2024}' \
        -e CRYPTO_SECRET='${CRYPTO_SECRET:-test-crypto-secret-2024}' \
        -e ERROR_LOG_ENABLED=true \
        -e BATCH_UPDATE_ENABLED=true \
        -e MEMORY_CACHE_ENABLED=true \
        --network new-api_default \
        $TEST_IMAGE_NAME --log-dir /app/logs"
    
    if [ "$DEPLOY_MODE" = "remote" ]; then
        remote_exec "$run_cmd"
    else
        eval "$run_cmd"
    fi
    
    print_success "测试容器启动完成"
}

# 清除测试环境的 Redis 缓存
clear_redis_cache() {
    print_step "清除测试环境 Redis 缓存 (db 1)..."
    
    if [ "$DEPLOY_MODE" = "remote" ]; then
        remote_exec "docker exec new-api-redis redis-cli -n 1 FLUSHDB"
    else
        docker exec new-api-redis redis-cli -n 1 FLUSHDB
    fi
    
    print_success "缓存已清除"
}

# 验证部署
verify_deployment() {
    print_step "验证部署状态..."
    
    if [ "$DEPLOY_MODE" = "remote" ]; then
        remote_exec "docker ps --filter name=$TEST_CONTAINER_NAME --format 'table {{.Names}}\t{{.Status}}\t{{.Ports}}'"
    else
        docker ps --filter "name=$TEST_CONTAINER_NAME" --format 'table {{.Names}}\t{{.Status}}\t{{.Ports}}'
    fi
    
    print_success "容器运行正常"
}

# 主函数
main() {
    echo -e "
${GREEN}==========================================
       New-API 测试环境部署脚本
==========================================${NC}
"
    
    # 解析参数
    while [[ $# -gt 0 ]]; do
        case $1 in
            local)
                DEPLOY_MODE="local"
                shift
                ;;
            remote)
                DEPLOY_MODE="remote"
                shift
                ;;
            --skip-build)
                SKIP_BUILD=true
                shift
                ;;
            --skip-sync)
                SKIP_SYNC=true
                shift
                ;;
            --clear-cache)
                CLEAR_CACHE=true
                shift
                ;;
            --create-db)
                CREATE_DB=true
                shift
                ;;
            --help|-h)
                show_help
                exit 0
                ;;
            *)
                print_error "未知参数: $1"
                show_help
                exit 1
                ;;
        esac
    done
    
    # 加载环境变量
    load_env
    
    print_info "部署模式: $DEPLOY_MODE"
    print_info "测试端口: $TEST_PORT"
    print_info "镜像名称: $TEST_IMAGE_NAME"
    
    # 远程模式检查
    if [ "$DEPLOY_MODE" = "remote" ]; then
        check_remote_env
        print_info "目标服务器: $SERVER_USER@$SERVER_IP"
        
        # 同步代码
        if [ "$SKIP_SYNC" = false ]; then
            sync_code
        fi
    fi
    
    # 创建测试数据库（如果需要）
    if [ "$CREATE_DB" = true ]; then
        create_test_database
    fi
    
    # 创建数据目录
    create_test_dirs
    
    # 构建镜像
    if [ "$SKIP_BUILD" = false ]; then
        build_image
    fi
    
    # 停止旧容器
    stop_container
    
    # 启动新容器
    start_container
    
    # 清除缓存
    if [ "$CLEAR_CACHE" = true ]; then
        clear_redis_cache
    fi
    
    # 验证部署
    verify_deployment
    
    echo -e "
${GREEN}[SUCCESS] ==========================================
[SUCCESS]        测试环境部署完成！
[SUCCESS] ==========================================${NC}
"
    
    if [ "$DEPLOY_MODE" = "remote" ]; then
        print_info "访问地址: http://$SERVER_IP:$TEST_PORT"
    else
        print_info "访问地址: http://localhost:$TEST_PORT"
    fi
    print_info "查看日志: docker logs -f $TEST_CONTAINER_NAME"
    print_info "停止测试: docker stop $TEST_CONTAINER_NAME"
    echo ""
}

# 运行主函数
main "$@"

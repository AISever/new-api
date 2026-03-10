#!/bin/bash
#
# New-API 自动部署脚本
# 支持本地部署和远程服务器部署
#
# 使用方法: ./scripts/deploy.sh [模式] [选项]
#
# 模式:
#   local           本地部署（默认）
#   remote          远程服务器部署（从 .env.local 读取服务器配置）
#
# 选项:
#   --skip-build    跳过构建，只重启容器
#   --skip-sync     跳过代码同步（仅远程模式）
#   --clear-cache   部署后清除 Redis 缓存
#   --image NAME    指定镜像名称（默认: new-api:latest）
#   --help          显示帮助信息
#
# 示例:
#   ./scripts/deploy.sh local                    # 本地部署
#   ./scripts/deploy.sh remote --clear-cache    # 远程部署并清除缓存
#   ./scripts/deploy.sh local --skip-build      # 本地只重启容器
#

set -e

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
NC='\033[0m' # No Color

# 脚本所在目录
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"

# 默认配置
ENV_FILE="$PROJECT_ROOT/.env.local"
REMOTE_BUILD_DIR="/tmp/new-api-build"
IMAGE_NAME="new-api:latest"
CONTAINER_NAME="new-api-local"
NETWORK_NAME="new-api_default"

# 部署模式: local 或 remote
DEPLOY_MODE="local"

# 选项
SKIP_BUILD=false
SKIP_SYNC=false
CLEAR_CACHE=false

# 服务器配置（远程模式）
SERVER_IP=""
SERVER_USER=""
SERVER_PASSWORD=""

# 应用配置
SESSION_SECRET=""
CRYPTO_SECRET=""
DB_PASSWORD=""

# 打印带颜色的消息
log_info() { echo -e "${BLUE}[INFO]${NC} $1"; }
log_success() { echo -e "${GREEN}[SUCCESS]${NC} $1"; }
log_warn() { echo -e "${YELLOW}[WARN]${NC} $1"; }
log_error() { echo -e "${RED}[ERROR]${NC} $1"; }
log_step() { echo -e "${CYAN}[STEP]${NC} $1"; }

# 显示帮助
show_help() {
    cat << EOF
New-API 自动部署脚本

使用方法: $0 [模式] [选项]

模式:
  local           本地部署（默认）- 在当前机器上构建和部署
  remote          远程服务器部署 - 同步代码到服务器并部署

选项:
  --skip-build    跳过构建，只重启容器
  --skip-sync     跳过代码同步（仅远程模式有效）
  --clear-cache   部署后清除 Redis 缓存
  --image NAME    指定镜像名称（默认: new-api:latest）
  --help          显示此帮助信息

配置文件: .env.local
  本地模式需要:
    DB_PASSWORD       数据库密码
    SESSION_SECRET    会话密钥
    CRYPTO_SECRET     加密密钥

  远程模式额外需要:
    SERVER_IP         服务器 IP 地址
    SERVER_USER       SSH 用户名
    SERVER_PASSWORD   SSH 密码

示例:
  $0 local                     # 本地部署
  $0 remote                    # 远程部署
  $0 local --clear-cache       # 本地部署并清除缓存
  $0 remote --skip-build       # 远程只重启容器
  $0 local --image new-api:v2  # 使用指定镜像名

EOF
}

# 解析命令行参数
parse_args() {
    # 第一个参数可能是模式
    if [[ $# -gt 0 && "$1" != --* ]]; then
        case $1 in
            local|remote)
                DEPLOY_MODE=$1
                shift
                ;;
            *)
                log_error "未知模式: $1 (可选: local, remote)"
                exit 1
                ;;
        esac
    fi

    while [[ $# -gt 0 ]]; do
        case $1 in
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
            --image)
                IMAGE_NAME="$2"
                shift 2
                ;;
            --help)
                show_help
                exit 0
                ;;
            *)
                log_error "未知选项: $1"
                show_help
                exit 1
                ;;
        esac
    done
}

# 从 .env.local 读取配置
load_config() {
    if [[ ! -f "$ENV_FILE" ]]; then
        log_error "配置文件不存在: $ENV_FILE"
        log_info "请复制 .env.example 为 .env.local 并填写配置"
        exit 1
    fi

    # 读取应用配置
    SESSION_SECRET=$(grep -E "^SESSION_SECRET=" "$ENV_FILE" | cut -d'=' -f2 | tr -d '"' | tr -d "'" || echo "")
    CRYPTO_SECRET=$(grep -E "^CRYPTO_SECRET=" "$ENV_FILE" | cut -d'=' -f2 | tr -d '"' | tr -d "'" || echo "")
    DB_PASSWORD=$(grep -E "^DB_PASSWORD=" "$ENV_FILE" | cut -d'=' -f2 | tr -d '"' | tr -d "'" || echo "")

    # 设置默认值
    [[ -z "$DB_PASSWORD" ]] && DB_PASSWORD="NewApi2024Secure"
    
    if [[ -z "$SESSION_SECRET" ]]; then
        SESSION_SECRET=$(openssl rand -hex 32 2>/dev/null || cat /dev/urandom | head -c 32 | xxd -p)
        log_warn "SESSION_SECRET 未设置，已生成随机值"
    fi
    
    if [[ -z "$CRYPTO_SECRET" ]]; then
        CRYPTO_SECRET=$(openssl rand -hex 32 2>/dev/null || cat /dev/urandom | head -c 32 | xxd -p)
        log_warn "CRYPTO_SECRET 未设置，已生成随机值"
    fi

    # 远程模式需要额外配置
    if [[ "$DEPLOY_MODE" == "remote" ]]; then
        SERVER_IP=$(grep -E "^SERVER_IP=" "$ENV_FILE" | cut -d'=' -f2 | tr -d '"' | tr -d "'" || echo "")
        SERVER_USER=$(grep -E "^SERVER_USER=" "$ENV_FILE" | cut -d'=' -f2 | tr -d '"' | tr -d "'" || echo "")
        SERVER_PASSWORD=$(grep -E "^SERVER_PASSWORD=" "$ENV_FILE" | cut -d'=' -f2 | tr -d '"' | tr -d "'" || echo "")

        if [[ -z "$SERVER_IP" || -z "$SERVER_USER" || -z "$SERVER_PASSWORD" ]]; then
            log_error "远程模式配置不完整"
            log_error "需要在 .env.local 中设置: SERVER_IP, SERVER_USER, SERVER_PASSWORD"
            exit 1
        fi
        log_info "目标服务器: $SERVER_USER@$SERVER_IP"
    else
        log_info "本地部署模式"
    fi
}

# 检查依赖
check_dependencies() {
    local missing=()
    
    # Docker 是必须的
    if ! command -v docker &> /dev/null; then
        missing+=("docker")
    fi

    # 远程模式需要额外工具
    if [[ "$DEPLOY_MODE" == "remote" ]]; then
        if ! command -v sshpass &> /dev/null; then
            missing+=("sshpass")
        fi
        if ! command -v rsync &> /dev/null; then
            missing+=("rsync")
        fi
    fi

    if [[ ${#missing[@]} -gt 0 ]]; then
        log_error "缺少依赖: ${missing[*]}"
        if [[ "$(uname)" == "Darwin" ]]; then
            log_info "macOS 安装: brew install ${missing[*]}"
        else
            log_info "Linux 安装: apt-get install ${missing[*]}"
        fi
        exit 1
    fi
}

# 检查 Docker 网络
check_docker_network() {
    if ! docker network inspect "$NETWORK_NAME" &>/dev/null; then
        log_info "创建 Docker 网络: $NETWORK_NAME"
        docker network create "$NETWORK_NAME"
    fi
}

# 执行命令（根据模式选择本地或远程）
run_cmd() {
    if [[ "$DEPLOY_MODE" == "remote" ]]; then
        sshpass -p "$SERVER_PASSWORD" ssh -o StrictHostKeyChecking=no "$SERVER_USER@$SERVER_IP" "$@"
    else
        eval "$@"
    fi
}

# 同步代码到服务器（仅远程模式）
sync_code() {
    if [[ "$DEPLOY_MODE" != "remote" ]]; then
        return
    fi

    log_step "同步代码到服务器..."
    
    sshpass -p "$SERVER_PASSWORD" rsync -avz --progress \
        --exclude='.git' \
        --exclude='node_modules' \
        --exclude='web/node_modules' \
        --exclude='web/build' \
        --exclude='*.log' \
        --exclude='logs/' \
        --exclude='data/' \
        --exclude='.env.local' \
        -e "ssh -o StrictHostKeyChecking=no" \
        "$PROJECT_ROOT/" "$SERVER_USER@$SERVER_IP:$REMOTE_BUILD_DIR/"
    
    log_success "代码同步完成"
}

# 构建 Docker 镜像
build_image() {
    log_step "构建 Docker 镜像: $IMAGE_NAME"
    
    local build_dir="$PROJECT_ROOT"
    if [[ "$DEPLOY_MODE" == "remote" ]]; then
        build_dir="$REMOTE_BUILD_DIR"
    fi
    
    run_cmd "cd $build_dir && docker build -t $IMAGE_NAME --build-arg GOPROXY=https://goproxy.cn,direct ."
    
    log_success "镜像构建完成"
}

# 部署容器
deploy_container() {
    log_step "停止旧容器..."
    run_cmd "docker stop $CONTAINER_NAME 2>/dev/null || true"
    run_cmd "docker rm $CONTAINER_NAME 2>/dev/null || true"
    
    # 本地模式检查网络
    if [[ "$DEPLOY_MODE" == "local" ]]; then
        check_docker_network
    fi
    
    log_step "启动新容器..."
    
    # 构建 docker run 命令
    local docker_cmd="docker run -d --name $CONTAINER_NAME --restart always"
    docker_cmd+=" -p 3000:3000"
    
    # 数据卷
    if [[ "$DEPLOY_MODE" == "remote" ]]; then
        docker_cmd+=" -v /opt/new-api/data:/data"
        docker_cmd+=" -v /opt/new-api/logs:/app/logs"
    else
        docker_cmd+=" -v $PROJECT_ROOT/data:/data"
        docker_cmd+=" -v $PROJECT_ROOT/logs:/app/logs"
    fi
    
    # 环境变量
    docker_cmd+=" -e SQL_DSN='postgresql://newapi:${DB_PASSWORD}@new-api-postgres:5432/new-api'"
    docker_cmd+=" -e REDIS_CONN_STRING='redis://new-api-redis'"
    docker_cmd+=" -e TZ=Asia/Shanghai"
    docker_cmd+=" -e SESSION_SECRET='${SESSION_SECRET}'"
    docker_cmd+=" -e CRYPTO_SECRET='${CRYPTO_SECRET}'"
    docker_cmd+=" -e ERROR_LOG_ENABLED=true"
    docker_cmd+=" -e BATCH_UPDATE_ENABLED=true"
    docker_cmd+=" -e MEMORY_CACHE_ENABLED=true"
    docker_cmd+=" --network $NETWORK_NAME"
    docker_cmd+=" $IMAGE_NAME --log-dir /app/logs"
    
    run_cmd "$docker_cmd"
    
    log_success "容器启动完成"
}

# 清除 Redis 缓存
clear_redis_cache() {
    log_step "清除 Redis 缓存..."
    run_cmd "docker exec new-api-redis redis-cli FLUSHALL" || log_warn "Redis 缓存清除失败（可能 Redis 容器未运行）"
    log_success "缓存已清除"
}

# 验证部署
verify_deployment() {
    log_step "验证部署状态..."
    
    # 等待容器启动
    sleep 3
    
    # 检查容器状态
    local status
    status=$(run_cmd "docker inspect -f '{{.State.Status}}' $CONTAINER_NAME 2>/dev/null" || echo "not found")
    
    if [[ "$status" == "running" ]]; then
        log_success "容器运行正常"
        
        # 显示容器信息
        echo ""
        run_cmd "docker ps --filter name=$CONTAINER_NAME --format 'table {{.Names}}\t{{.Status}}\t{{.Ports}}'"
    else
        log_error "容器状态异常: $status"
        log_info "查看日志: docker logs $CONTAINER_NAME"
        exit 1
    fi
}

# 显示部署信息
show_deploy_info() {
    echo ""
    log_success "=========================================="
    log_success "           部署完成！"
    log_success "=========================================="
    echo ""
    
    if [[ "$DEPLOY_MODE" == "remote" ]]; then
        log_info "访问地址: http://$SERVER_IP:3000"
        log_info "查看日志: ssh $SERVER_USER@$SERVER_IP 'docker logs -f $CONTAINER_NAME'"
    else
        log_info "访问地址: http://localhost:3000"
        log_info "查看日志: docker logs -f $CONTAINER_NAME"
    fi
    echo ""
}

# 主函数
main() {
    echo ""
    echo -e "${CYAN}==========================================${NC}"
    echo -e "${CYAN}       New-API 自动部署脚本${NC}"
    echo -e "${CYAN}==========================================${NC}"
    echo ""
    
    parse_args "$@"
    load_config
    check_dependencies
    
    log_info "部署模式: $DEPLOY_MODE"
    log_info "镜像名称: $IMAGE_NAME"
    echo ""
    
    # 远程模式：同步代码
    if [[ "$DEPLOY_MODE" == "remote" && "$SKIP_SYNC" == false ]]; then
        sync_code
    elif [[ "$DEPLOY_MODE" == "remote" && "$SKIP_SYNC" == true ]]; then
        log_warn "跳过代码同步"
    fi
    
    # 构建镜像
    if [[ "$SKIP_BUILD" == false ]]; then
        build_image
    else
        log_warn "跳过镜像构建"
    fi
    
    # 部署容器
    deploy_container
    
    # 清除缓存
    if [[ "$CLEAR_CACHE" == true ]]; then
        clear_redis_cache
    fi
    
    # 验证
    verify_deployment
    
    # 显示信息
    show_deploy_info
}

main "$@"

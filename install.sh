#!/bin/bash

# Komari Monitor 一键安装脚本
# 支持安装、更新、卸载、服务管理等功能

set -e

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[0;33m'
BLUE='\033[0;34m'
PURPLE='\033[0;35m'
CYAN='\033[0;36m'
NC='\033[0m' # No Color

# 日志函数
log_info() {
    echo -e "${CYAN}[INFO]${NC} $1"
}

log_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

log_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

log_step() {
    echo -e "${PURPLE}[STEP]${NC} $1"
}

# 全局变量
INSTALL_DIR="/opt/komari"
DATA_DIR="/opt/komari/data"
SERVICE_NAME="komari"
BINARY_PATH="$INSTALL_DIR/komari"
CONFIG_FILE="$INSTALL_DIR/config.yaml"
DEFAULT_PORT="25774"
LISTEN_PORT=""
WEB_DIR="$INSTALL_DIR/web"
PUBLIC_DIR="$INSTALL_DIR/public"
REPO_URL="https://github.com/komari-monitor/komari.git"
WEB_REPO_URL="https://github.com/komari-monitor/komari-web.git"
GO_VERSION="1.21.0"
NODE_VERSION="20"

# 显示横幅
show_banner() {
    clear
    echo -e "${CYAN}"
    echo "==============================================================="
    echo "            Komari Monitor 一键安装管理脚本"
    echo "       https://github.com/komari-monitor/komari"
    echo "==============================================================="
    echo -e "${NC}"
    echo
}

# 检查root权限
check_root() {
    if [ "$EUID" -ne 0 ]; then
        log_error "请使用 root 权限运行此脚本"
        exit 1
    fi
}

# 检测系统信息
detect_system() {
    if [ -f /etc/os-release ]; then
        . /etc/os-release
        OS=$NAME
        VER=$VERSION_ID
    elif type lsb_release >/dev/null 2>&1; then
        OS=$(lsb_release -si)
        VER=$(lsb_release -sr)
    else
        OS=$(uname -s)
        VER=$(uname -r)
    fi
    
    ARCH=$(uname -m)
    case $ARCH in
        x86_64) ARCH="amd64" ;;
        aarch64) ARCH="arm64" ;;
        i386|i686) ARCH="386" ;;
        riscv64) ARCH="riscv64" ;;
        *) log_error "不支持的架构: $ARCH"; exit 1 ;;
    esac
    
    log_info "检测到系统: $OS $VER ($ARCH)"
}

# 检查systemd
check_systemd() {
    if ! command -v systemctl >/dev/null 2>&1; then
        return 1
    else
        return 0
    fi
}

# 检查是否已安装
is_installed() {
    [ -f "$BINARY_PATH" ]
}

# 检查Go是否安装
check_go() {
    if command -v go >/dev/null 2>&1; then
        local go_version=$(go version | awk '{print $3}' | sed 's/go//')
        log_info "检测到 Go 版本: $go_version"
        return 0
    else
        return 1
    fi
}

# 安装Go
install_go() {
    log_step "安装 Go $GO_VERSION..."
    
    local go_file="go${GO_VERSION}.linux-${ARCH}.tar.gz"
    local go_url="https://golang.org/dl/${go_file}"
    
    # 下载Go
    cd /tmp
    if ! curl -L -o "$go_file" "$go_url"; then
        log_error "下载 Go 失败"
        return 1
    fi
    
    # 安装Go
    rm -rf /usr/local/go
    tar -C /usr/local -xzf "$go_file"
    
    # 设置环境变量
    if ! grep -q "/usr/local/go/bin" /etc/profile; then
        echo 'export PATH=$PATH:/usr/local/go/bin' >> /etc/profile
    fi
    
    export PATH=$PATH:/usr/local/go/bin
    
    log_success "Go 安装完成"
}

# 检查Node.js是否安装
check_nodejs() {
    if command -v node >/dev/null 2>&1; then
        local node_version=$(node --version | sed 's/v//')
        log_info "检测到 Node.js 版本: $node_version"
        return 0
    else
        return 1
    fi
}

# 安装Node.js
install_nodejs() {
    log_step "安装 Node.js $NODE_VERSION..."
    
    # 使用NodeSource仓库安装
    curl -fsSL https://deb.nodesource.com/setup_${NODE_VERSION}.x | bash -
    
    if command -v apt >/dev/null 2>&1; then
        apt-get install -y nodejs
    elif command -v yum >/dev/null 2>&1; then
        yum install -y nodejs npm
    elif command -v dnf >/dev/null 2>&1; then
        dnf install -y nodejs npm
    else
        log_error "无法安装 Node.js，请手动安装"
        return 1
    fi
    
    log_success "Node.js 安装完成"
}

# 安装系统依赖
install_dependencies() {
    log_step "检查并安装系统依赖..."
    
    local packages="curl wget git build-essential"
    
    if command -v apt >/dev/null 2>&1; then
        apt update
        apt install -y $packages
    elif command -v yum >/dev/null 2>&1; then
        yum groupinstall -y "Development Tools"
        yum install -y curl wget git
    elif command -v dnf >/dev/null 2>&1; then
        dnf groupinstall -y "Development Tools"
        dnf install -y curl wget git
    elif command -v apk >/dev/null 2>&1; then
        apk add curl wget git build-base
    else
        log_error "未找到支持的包管理器"
        return 1
    fi
    
    log_success "系统依赖安装完成"
}

# 检查并安装环境依赖
check_and_install_deps() {
    log_step "检查环境依赖..."
    
    # 安装系统依赖
    install_dependencies
    
    # 检查并安装Go
    if ! check_go; then
        log_warning "未检测到 Go，正在安装..."
        install_go
    fi
    
    # 检查并安装Node.js
    if ! check_nodejs; then
        log_warning "未检测到 Node.js，正在安装..."
        install_nodejs
    fi
    
    log_success "环境依赖检查完成"
}

# 获取用户配置
get_user_config() {
    echo
    log_step "配置安装参数"
    
    # 监听端口
    while true; do
        read -p "请输入监听端口 [默认: $DEFAULT_PORT]: " input_port
        if [[ -z "$input_port" ]]; then
            LISTEN_PORT="$DEFAULT_PORT"
            break
        elif [[ "$input_port" =~ ^[0-9]+$ ]] && (( input_port >= 1 && input_port <= 65535 )); then
            LISTEN_PORT="$input_port"
            break
        else
            log_error "端口号无效，请输入 1-65535 之间的数字"
        fi
    done
    
    # 数据库类型选择
    echo
    echo "请选择数据库类型:"
    echo "  1) SQLite (默认，推荐)"
    echo "  2) MySQL"
    read -p "选择 [1-2]: " db_choice
    
    case $db_choice in
        2)
            DB_TYPE="mysql"
            read -p "MySQL 主机 [localhost]: " DB_HOST
            DB_HOST=${DB_HOST:-localhost}
            read -p "MySQL 端口 [3306]: " DB_PORT
            DB_PORT=${DB_PORT:-3306}
            read -p "MySQL 用户名 [root]: " DB_USER
            DB_USER=${DB_USER:-root}
            read -s -p "MySQL 密码: " DB_PASS
            echo
            read -p "数据库名 [komari]: " DB_NAME
            DB_NAME=${DB_NAME:-komari}
            ;;
        *)
            DB_TYPE="sqlite"
            DB_FILE="$DATA_DIR/komari.db"
            ;;
    esac
}

# 从源码构建
build_from_source() {
    log_step "从源码构建 Komari..."
    
    # 创建临时构建目录
    local build_dir="/tmp/komari-build"
    rm -rf "$build_dir"
    mkdir -p "$build_dir"
    cd "$build_dir"
    
    # 克隆前端仓库
    log_info "克隆前端仓库..."
    git clone "$WEB_REPO_URL" komari-web
    cd komari-web
    
    # 构建前端
    log_info "构建前端..."
    npm install
    npm run build
    
    # 克隆后端仓库
    cd "$build_dir"
    log_info "克隆后端仓库..."
    git clone "$REPO_URL" komari
    cd komari
    
    # 复制前端文件
    log_info "复制前端文件..."
    rm -rf public/dist
    cp -r ../komari-web/dist public/
    
    # 构建后端
    log_info "构建后端..."
    export CGO_ENABLED=1
    go mod download
    go build -ldflags "-s -w" -o komari
    
    # 安装二进制文件
    mkdir -p "$INSTALL_DIR"
    cp komari "$BINARY_PATH"
    chmod +x "$BINARY_PATH"
    
    # 清理构建目录
    cd /
    rm -rf "$build_dir"
    
    log_success "源码构建完成"
}

# 下载预编译二进制
download_binary() {
    log_step "下载预编译二进制文件..."
    
    local file_name="komari-linux-${ARCH}"
    local download_url="https://github.com/komari-monitor/komari/releases/latest/download/${file_name}"
    
    log_info "下载 URL: $download_url"
    
    if ! curl -L -o "$BINARY_PATH" "$download_url"; then
        log_error "下载失败，尝试从源码构建"
        return 1
    fi
    
    chmod +x "$BINARY_PATH"
    log_success "二进制文件下载完成"
}

# 创建配置文件
create_config() {
    log_step "创建配置文件..."
    
    cat > "$CONFIG_FILE" << EOF
# Komari Monitor 配置文件
server:
  listen: "0.0.0.0:${LISTEN_PORT}"
  
database:
  type: "${DB_TYPE}"
EOF

    if [ "$DB_TYPE" = "mysql" ]; then
        cat >> "$CONFIG_FILE" << EOF
  host: "${DB_HOST}"
  port: ${DB_PORT}
  user: "${DB_USER}"
  password: "${DB_PASS}"
  name: "${DB_NAME}"
EOF
    else
        cat >> "$CONFIG_FILE" << EOF
  file: "${DB_FILE}"
EOF
    fi
    
    log_success "配置文件创建完成"
}

# 创建systemd服务
create_systemd_service() {
    log_step "创建 systemd 服务..."
    
    local service_file="/etc/systemd/system/${SERVICE_NAME}.service"
    cat > "$service_file" << EOF
[Unit]
Description=Komari Monitor Service
After=network.target

[Service]
Type=simple
ExecStart=${BINARY_PATH} server -l 0.0.0.0:${LISTEN_PORT}
WorkingDirectory=${DATA_DIR}
Restart=always
RestartSec=5
User=root
Environment=GIN_MODE=release
Environment=KOMARI_DB_TYPE=${DB_TYPE}
EOF

    if [ "$DB_TYPE" = "mysql" ]; then
        cat >> "$service_file" << EOF
Environment=KOMARI_DB_HOST=${DB_HOST}
Environment=KOMARI_DB_PORT=${DB_PORT}
Environment=KOMARI_DB_USER=${DB_USER}
Environment=KOMARI_DB_PASS=${DB_PASS}
Environment=KOMARI_DB_NAME=${DB_NAME}
EOF
    else
        cat >> "$service_file" << EOF
Environment=KOMARI_DB_FILE=${DB_FILE}
EOF
    fi
    
    cat >> "$service_file" << EOF

[Install]
WantedBy=multi-user.target
EOF
    
    log_success "systemd 服务文件创建完成"
}

# 安装Komari
install_komari() {
    log_step "开始安装 Komari..."
    
    if is_installed; then
        log_warning "Komari 已安装，如需重新安装请先卸载"
        return 1
    fi
    
    # 检查并安装依赖
    check_and_install_deps
    
    # 获取用户配置
    get_user_config
    
    # 创建目录
    mkdir -p "$INSTALL_DIR" "$DATA_DIR"
    
    # 选择安装方式
    echo
    echo "请选择安装方式:"
    echo "  1) 下载预编译二进制 (推荐)"
    echo "  2) 从源码构建"
    read -p "选择 [1-2]: " install_method
    
    case $install_method in
        2)
            build_from_source
            ;;
        *)
            if ! download_binary; then
                log_warning "预编译二进制下载失败，切换到源码构建"
                build_from_source
            fi
            ;;
    esac
    
    # 创建配置文件
    create_config
    
    # 创建并启动服务
    if check_systemd; then
        create_systemd_service
        systemctl daemon-reload
        systemctl enable "${SERVICE_NAME}.service"
        systemctl start "${SERVICE_NAME}.service"
        
        sleep 3
        if systemctl is-active --quiet "${SERVICE_NAME}.service"; then
            log_success "Komari 服务启动成功"
            show_access_info
        else
            log_error "Komari 服务启动失败"
            log_info "查看日志: journalctl -u ${SERVICE_NAME} -f"
            return 1
        fi
    else
        log_warning "未检测到 systemd，请手动运行:"
        log_info "$BINARY_PATH server -l 0.0.0.0:$LISTEN_PORT"
    fi
    
    log_success "Komari 安装完成！"
}

# 更新Komari
update_komari() {
    log_step "更新 Komari..."
    
    if ! is_installed; then
        log_error "Komari 未安装，请先安装"
        return 1
    fi
    
    # 检查是否有更新
    log_info "检查更新..."
    
    # 停止服务
    if check_systemd && systemctl is-active --quiet "${SERVICE_NAME}.service"; then
        log_step "停止服务..."
        systemctl stop "${SERVICE_NAME}.service"
    fi
    
    # 备份当前版本
    local backup_file="${BINARY_PATH}.backup.$(date +%Y%m%d_%H%M%S)"
    cp "$BINARY_PATH" "$backup_file"
    log_info "已备份当前版本到: $backup_file"
    
    # 选择更新方式
    echo
    echo "请选择更新方式:"
    echo "  1) 下载最新预编译二进制"
    echo "  2) 从最新源码构建"
    read -p "选择 [1-2]: " update_method
    
    local update_success=false
    
    case $update_method in
        2)
            if build_from_source; then
                update_success=true
            fi
            ;;
        *)
            if download_binary; then
                update_success=true
            else
                log_warning "预编译二进制下载失败，尝试源码构建"
                if build_from_source; then
                    update_success=true
                fi
            fi
            ;;
    esac
    
    if [ "$update_success" = true ]; then
        # 重启服务
        if check_systemd; then
            systemctl start "${SERVICE_NAME}.service"
            sleep 3
            if systemctl is-active --quiet "${SERVICE_NAME}.service"; then
                log_success "Komari 更新成功，服务已重启"
                rm -f "$backup_file"  # 删除备份文件
            else
                log_error "服务启动失败，正在恢复备份"
                cp "$backup_file" "$BINARY_PATH"
                systemctl start "${SERVICE_NAME}.service"
            fi
        else
            log_success "Komari 更新成功"
        fi
    else
        log_error "更新失败，正在恢复备份"
        cp "$backup_file" "$BINARY_PATH"
        if check_systemd; then
            systemctl start "${SERVICE_NAME}.service"
        fi
    fi
}

# 卸载Komari
uninstall_komari() {
    log_step "卸载 Komari..."
    
    if ! is_installed; then
        log_info "Komari 未安装"
        return 0
    fi
    
    echo
    log_warning "这将完全删除 Komari 及其数据！"
    read -p "是否保留数据目录？(Y/n): " keep_data
    read -p "确认卸载？(y/N): " confirm
    
    if [[ ! $confirm =~ ^[Yy]$ ]]; then
        log_info "卸载已取消"
        return 0
    fi
    
    # 停止并删除服务
    if check_systemd; then
        systemctl stop "${SERVICE_NAME}.service" 2>/dev/null || true
        systemctl disable "${SERVICE_NAME}.service" 2>/dev/null || true
        rm -f "/etc/systemd/system/${SERVICE_NAME}.service"
        systemctl daemon-reload
        log_success "服务已删除"
    fi
    
    # 删除程序文件
    rm -f "$BINARY_PATH"
    rm -f "$CONFIG_FILE"
    
    # 处理数据目录
    if [[ $keep_data =~ ^[Nn]$ ]]; then
        rm -rf "$DATA_DIR"
        log_info "数据目录已删除"
    else
        log_info "数据目录已保留: $DATA_DIR"
    fi
    
    # 删除安装目录（如果为空）
    rmdir "$INSTALL_DIR" 2>/dev/null || true
    
    log_success "Komari 卸载完成"
}

# 显示访问信息
show_access_info() {
    echo
    log_success "安装完成！"
    echo
    log_info "访问信息:"
    local ip=$(hostname -I | awk '{print $1}' 2>/dev/null || echo "localhost")
    log_info "  URL: http://${ip}:${LISTEN_PORT}"
    echo
    
    # 获取初始密码
    if check_systemd; then
        log_info "正在获取初始登录信息..."
        sleep 2
        local password=$(journalctl -u "${SERVICE_NAME}" --since "2 minutes ago" | grep "admin account created" | tail -n 1 | sed -e 's/.*admin account created.//')
        if [ -n "$password" ]; then
            log_info "初始登录信息: $password"
        else
            log_warning "未能获取初始密码，请查看日志: journalctl -u ${SERVICE_NAME}"
        fi
    fi
    
    echo
    log_info "服务管理命令:"
    log_info "  状态: systemctl status $SERVICE_NAME"
    log_info "  启动: systemctl start $SERVICE_NAME"
    log_info "  停止: systemctl stop $SERVICE_NAME"
    log_info "  重启: systemctl restart $SERVICE_NAME"
    log_info "  日志: journalctl -u $SERVICE_NAME -f"
}

# 显示状态
show_status() {
    if ! is_installed; then
        log_error "Komari 未安装"
        return 1
    fi
    
    echo
    log_info "Komari 状态信息:"
    echo
    
    # 显示版本信息
    if [ -x "$BINARY_PATH" ]; then
        local version=$("$BINARY_PATH" --version 2>/dev/null || echo "未知")
        log_info "版本: $version"
    fi
    
    # 显示服务状态
    if check_systemd; then
        log_info "服务状态:"
        systemctl status "${SERVICE_NAME}.service" --no-pager -l
    else
        log_warning "未检测到 systemd"
    fi
}

# 显示日志
show_logs() {
    if ! is_installed; then
        log_error "Komari 未安装"
        return 1
    fi
    
    if ! check_systemd; then
        log_error "未检测到 systemd"
        return 1
    fi
    
    log_step "显示 Komari 服务日志 (Ctrl+C 退出):"
    journalctl -u "${SERVICE_NAME}" -f --no-pager
}

# 重启服务
restart_service() {
    if ! is_installed; then
        log_error "Komari 未安装"
        return 1
    fi
    
    if ! check_systemd; then
        log_error "未检测到 systemd"
        return 1
    fi
    
    log_step "重启 Komari 服务..."
    systemctl restart "${SERVICE_NAME}.service"
    
    sleep 2
    if systemctl is-active --quiet "${SERVICE_NAME}.service"; then
        log_success "服务重启成功"
    else
        log_error "服务重启失败"
        systemctl status "${SERVICE_NAME}.service" --no-pager -l
    fi
}

# 停止服务
stop_service() {
    if ! is_installed; then
        log_error "Komari 未安装"
        return 1
    fi
    
    if ! check_systemd; then
        log_error "未检测到 systemd"
        return 1
    fi
    
    log_step "停止 Komari 服务..."
    systemctl stop "${SERVICE_NAME}.service"
    log_success "服务已停止"
}

# 启动服务
start_service() {
    if ! is_installed; then
        log_error "Komari 未安装"
        return 1
    fi
    
    if ! check_systemd; then
        log_error "未检测到 systemd"
        return 1
    fi
    
    log_step "启动 Komari 服务..."
    systemctl start "${SERVICE_NAME}.service"
    
    sleep 2
    if systemctl is-active --quiet "${SERVICE_NAME}.service"; then
        log_success "服务启动成功"
    else
        log_error "服务启动失败"
        systemctl status "${SERVICE_NAME}.service" --no-pager -l
    fi
}

# 配置管理
manage_config() {
    if ! is_installed; then
        log_error "Komari 未安装"
        return 1
    fi
    
    echo
    echo "配置管理:"
    echo "  1) 查看当前配置"
    echo "  2) 编辑配置文件"
    echo "  3) 重置配置"
    echo "  4) 返回主菜单"
    echo
    
    read -p "选择操作 [1-4]: " config_choice
    
    case $config_choice in
        1)
            if [ -f "$CONFIG_FILE" ]; then
                log_info "当前配置:"
                cat "$CONFIG_FILE"
            else
                log_warning "配置文件不存在"
            fi
            ;;
        2)
            if command -v nano >/dev/null 2>&1; then
                nano "$CONFIG_FILE"
            elif command -v vi >/dev/null 2>&1; then
                vi "$CONFIG_FILE"
            else
                log_error "未找到文本编辑器"
            fi
            ;;
        3)
            read -p "确认重置配置？(y/N): " confirm
            if [[ $confirm =~ ^[Yy]$ ]]; then
                get_user_config
                create_config
                log_success "配置已重置"
            fi
            ;;
        4)
            return
            ;;
        *)
            log_error "无效选项"
            ;;
    esac
}

# 主菜单
main_menu() {
    while true; do
        show_banner
        
        # 显示当前状态
        if is_installed; then
            if check_systemd && systemctl is-active --quiet "${SERVICE_NAME}.service"; then
                log_success "Komari 已安装并运行中"
            else
                log_warning "Komari 已安装但未运行"
            fi
        else
            log_info "Komari 未安装"
        fi
        
        echo
        echo "请选择操作:"
        echo "  1) 安装 Komari"
        echo "  2) 更新 Komari"
        echo "  3) 卸载 Komari"
        echo "  4) 查看状态"
        echo "  5) 查看日志"
        echo "  6) 启动服务"
        echo "  7) 停止服务"
        echo "  8) 重启服务"
        echo "  9) 配置管理"
        echo "  0) 退出"
        echo
        
        read -p "输入选项 [0-9]: " choice
        
        case $choice in
            1) install_komari; read -p "按回车键继续..." ;;
            2) update_komari; read -p "按回车键继续..." ;;
            3) uninstall_komari; read -p "按回车键继续..." ;;
            4) show_status; read -p "按回车键继续..." ;;
            5) show_logs ;;
            6) start_service; read -p "按回车键继续..." ;;
            7) stop_service; read -p "按回车键继续..." ;;
            8) restart_service; read -p "按回车键继续..." ;;
            9) manage_config; read -p "按回车键继续..." ;;
            0) clear; log_info "感谢使用 Komari 安装脚本！"; exit 0 ;;
            *) log_error "无效选项，请重新选择"; sleep 2 ;;
        esac
    done
}

# 主程序入口
main() {
    # 检查root权限
    check_root
    
    # 检测系统信息
    detect_system
    
    # 如果有命令行参数，直接执行对应功能
    case "${1:-}" in
        "install") install_komari ;;
        "update") update_komari ;;
        "uninstall") uninstall_komari ;;
        "status") show_status ;;
        "start") start_service ;;
        "stop") stop_service ;;
        "restart") restart_service ;;
        "logs") show_logs ;;
        *) main_menu ;;
    esac
    
    # 如果是命令行模式执行完成后也清屏退出
    if [ "$#" -gt 0 ]; then
        clear
        log_info "操作完成，感谢使用 Komari 安装脚本！"
    fi
}

# 执行主程序
main "$@"
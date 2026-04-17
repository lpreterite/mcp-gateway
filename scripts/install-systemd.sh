#!/bin/bash

# mcp-gateway Systemd 安装脚本
# 用于创建和管理 mcp-gateway 的 systemd 服务，解决 PATH 环境变量问题

set -e

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

# 配置
SERVICE_NAME="mcp-gateway"
SERVICE_FILE="/etc/systemd/system/${SERVICE_NAME}.service"
CONFIG_FILE="/etc/mcp-gateway/config.json"
LOG_FILE="/var/log/mcp-gateway.log"
ERR_LOG_FILE="/var/log/mcp-gateway.err.log"

# 检测用户 PATH 组件
detect_paths() {
    local paths=()

    # 添加 Homebrew 路径（Linuxbrew）
    if [ -d "/home/linuxbrew/.linuxbrew/bin" ]; then
        paths+=("/home/linuxbrew/.linuxbrew/bin")
    fi
    if [ -d "/home/linuxbrew/.linuxbrew/sbin" ]; then
        paths+=("/home/linuxbrew/.linuxbrew/sbin")
    fi

    # 检测 Node.js (nvm)
    if [ -n "$NVM_DIR" ] && [ -d "$NVM_DIR/versions/node" ]; then
        local latest_node=$(ls -t "$NVM_DIR/versions/node" 2>/dev/null | head -1)
        if [ -n "$latest_node" ]; then
            paths+=("$NVM_DIR/versions/node/$latest_node/bin")
        fi
    fi

    # 检测 uv (用于 Python 工具)
    if command -v uv &> /dev/null; then
        local uv_path=$(which uv | sed 's|/uv$||')
        paths+=("$uv_path")
    fi

    # 添加标准系统路径
    paths+=("/usr/local/bin" "/usr/bin" "/bin" "/usr/local/sbin" "/usr/sbin" "/sbin")

    # 去重并连接
    echo "$(printf "%s\n" "${paths[@]}" | awk '!seen[$0]++' | tr '\n' ':' | sed 's/:$//')"
}

# 检查是否以 root 运行
check_root() {
    if [ "$EUID" -ne 0 ]; then
        echo -e "${RED}错误: 请使用 sudo 运行此脚本${NC}"
        exit 1
    fi
}

# 创建 systemd service 文件
create_service() {
    echo -e "${GREEN}创建 systemd service 文件...${NC}"

    # 确保目录存在
    mkdir -p "$(dirname "$CONFIG_FILE")"
    mkdir -p "$(dirname "$LOG_FILE")"

    # 检测 PATH
    DETECTED_PATH=$(detect_paths)
    echo -e "${YELLOW}检测到的 PATH: ${DETECTED_PATH}${NC}"

    # 检测二进制文件路径
    GATEWAY_BIN=$(which mcp-gateway 2>/dev/null || echo "/usr/local/bin/mcp-gateway")

    # 创建 service 文件
    cat > "$SERVICE_FILE" <<EOF
[Unit]
Description=MCP Gateway - Unified gateway for connecting multiple MCP servers
After=network.target

[Service]
Type=simple
ExecStart=${GATEWAY_BIN} --config ${CONFIG_FILE}
Restart=always
RestartSec=5
Environment="PATH=${DETECTED_PATH}"
StandardOutput=append:${LOG_FILE}
StandardError=append:${ERR_LOG_FILE}
WorkingDirectory=/etc/mcp-gateway

# 安全加固
NoNewPrivileges=true
ProtectSystem=strict
ProtectHome=true
ReadWritePaths=${LOG_FILE%/*} ${CONFIG_FILE%/*}

[Install]
WantedBy=multi-user.target
EOF

    echo -e "${GREEN}✓ service 文件已创建: ${SERVICE_FILE}${NC}"
}

# 重新加载 systemd
reload_daemon() {
    echo -e "${GREEN}重新加载 systemd...${NC}"
    systemctl daemon-reload
    echo -e "${GREEN}✓ systemd 已重新加载${NC}"
}

# 启动服务
start_service() {
    echo -e "${GREEN}启动服务...${NC}"
    systemctl start "$SERVICE_NAME"
    echo -e "${GREEN}✓ 服务已启动${NC}"
}

# 停止服务
stop_service() {
    echo -e "${YELLOW}停止服务...${NC}"
    systemctl stop "$SERVICE_NAME" 2>/dev/null || true
    echo -e "${GREEN}✓ 服务已停止${NC}"
}

# 重启服务
restart_service() {
    echo -e "${GREEN}重启服务...${NC}"
    systemctl restart "$SERVICE_NAME"
    echo -e "${GREEN}✓ 服务已重启${NC}"
}

# 启用服务开机自启
enable_service() {
    echo -e "${GREEN}启用开机自启...${NC}"
    systemctl enable "$SERVICE_NAME"
    echo -e "${GREEN}✓ 已启用开机自启${NC}"
}

# 检查服务状态
check_status() {
    echo -e "${GREEN}检查服务状态...${NC}"

    if systemctl is-active --quiet "$SERVICE_NAME"; then
        echo -e "${GREEN}✓ 服务正在运行${NC}"
    else
        echo -e "${RED}✗ 服务未运行${NC}"
    fi

    if systemctl is-enabled --quiet "$SERVICE_NAME"; then
        echo -e "${GREEN}✓ 已启用开机自启${NC}"
    else
        echo -e "${YELLOW}⚠ 未启用开机自启${NC}"
    fi

    # 检查健康状态
    if command -v curl &> /dev/null; then
        echo -e "${GREEN}检查健康状态...${NC}"
        HEALTH=$(curl -s http://localhost:4298/health 2>/dev/null || echo "{}")
        echo "$HEALTH" | python3 -m json.tool 2>/dev/null || echo "$HEALTH"
    fi
}

# 查看日志
view_logs() {
    local lines="${1:-50}"
    local follow="${2:-}"

    if [ -n "$follow" ]; then
        echo -e "${GREEN}实时查看日志 (Ctrl+C 退出)...${NC}"
        journalctl -u "$SERVICE_NAME" -f -n "$lines"
    else
        echo -e "${GREEN}=== 最近 ${lines} 行日志 ===${NC}"
        journalctl -u "$SERVICE_NAME" -n "$lines" --no-pager
    fi
}

# 卸载服务
uninstall_service() {
    echo -e "${YELLOW}卸载服务...${NC}"
    stop_service
    systemctl disable "$SERVICE_NAME" 2>/dev/null || true
    rm -f "$SERVICE_FILE"
    echo -e "${GREEN}✓ 服务已卸载${NC}"
    echo -e "${YELLOW}配置文件保留在: ${CONFIG_FILE}${NC}"
}

# 显示帮助
show_help() {
    cat <<EOF
mcp-gateway Systemd 管理工具 (Linux)

用法: sudo $0 <command> [options]

命令:
  install     安装并启动服务 (包含开机自启)
  start       启动服务
  stop        停止服务
  restart     重启服务
  status      检查服务状态
  logs [n]    查看日志 (默认最近 50 行)
  tail        实时查看日志
  uninstall   卸载服务 (保留配置文件)
  help        显示此帮助信息

示例:
  sudo $0 install              # 安装并启动服务
  sudo $0 status              # 检查服务状态
  sudo $0 logs 100            # 查看最近 100 行日志
  sudo $0 tail                # 实时查看日志
  sudo $0 restart             # 重启服务
  sudo $0 uninstall           # 卸载服务

配置文件位置:
  - Systemd service: ${SERVICE_FILE}
  - Gateway 配置: ${CONFIG_FILE}
  - 日志文件: ${LOG_FILE}

注意: 此脚本需要 root 权限运行

EOF
}

# 主函数
main() {
    local command="${1:-}"

    case "$command" in
        install)
            check_root
            create_service
            reload_daemon
            enable_service
            start_service
            sleep 2
            check_status
            ;;
        start)
            check_root
            start_service
            sleep 2
            check_status
            ;;
        stop)
            check_root
            stop_service
            ;;
        restart)
            check_root
            restart_service
            sleep 2
            check_status
            ;;
        status)
            check_status
            ;;
        logs)
            view_logs "${2:-50}" ""
            ;;
        tail)
            view_logs "100" "follow"
            ;;
        uninstall)
            check_root
            uninstall_service
            ;;
        help|--help|-h)
            show_help
            ;;
        *)
            echo -e "${RED}错误: 未知命令 '$command'${NC}"
            echo ""
            show_help
            exit 1
            ;;
    esac
}

main "$@"

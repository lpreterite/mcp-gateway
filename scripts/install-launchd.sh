#!/bin/bash

# mcp-gateway Launchd 安装脚本
# 用于创建和管理 mcp-gateway 的 launchd 服务，解决 PATH 环境变量问题
#
# 注意: 此脚本仅适用于 macOS
# Linux 用户请使用 install-systemd.sh

set -e

# 检测操作系统 (必须在开头调用)
detect_os

# 检测操作系统
detect_os() {
    case "$(uname -s)" in
        Darwin)
            return 0  # macOS
            ;;
        Linux)
            echo -e "${RED}错误: 此脚本仅适用于 macOS${NC}"
            echo -e "${YELLOW}Linux 用户请使用 install-systemd.sh${NC}"
            echo -e "下载: https://github.com/lpreterite/mcp-gateway/releases/download/latest/install-systemd.sh"
            exit 1
            ;;
        *)
            echo -e "${RED}错误: 不支持的操作系统: $(uname -s)${NC}"
            exit 1
            ;;
    esac
}

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# 配置
PLIST_NAME="com.mcp-gateway"
PLIST_PATH="$HOME/Library/LaunchAgents/${PLIST_NAME}.plist"
GATEWAY_BIN="/opt/homebrew/opt/mcp-gateway/bin/mcp-gateway"
CONFIG_FILE="/opt/homebrew/etc/mcp-gateway/config.json"
LOG_FILE="/opt/homebrew/var/log/mcp-gateway.log"
ERR_LOG_FILE="/opt/homebrew/var/log/mcp-gateway.err.log"

# 检测用户 PATH 组件
detect_paths() {
    local paths=()

    # 添加 Homebrew 路径
    if [ -d "/opt/homebrew/bin" ]; then
        paths+=("/opt/homebrew/bin")
    fi
    if [ -d "/opt/homebrew/sbin" ]; then
        paths+=("/opt/homebrew/sbin")
    fi

    # 检测 Node.js (nvm)
    if [ -n "$NVM_DIR" ] && [ -d "$NVM_DIR/versions/node" ]; then
        # 获取最新的 Node.js 版本
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
    paths+=("/usr/local/bin" "/usr/bin" "/bin" "/usr/sbin" "/sbin")

    # 去重并连接
    echo "$(printf "%s\n" "${paths[@]}" | awk '!seen[$0]++' | tr '\n' ':' | sed 's/:$//')"
}

# 创建 launchd plist 文件
create_plist() {
    echo -e "${GREEN}创建 launchd plist 文件...${NC}"

    # 确保 log 目录存在
    sudo mkdir -p "$(dirname "$LOG_FILE")"
    sudo mkdir -p "$(dirname "$CONFIG_FILE")"

    # 检测 PATH
    DETECTED_PATH=$(detect_paths)
    echo -e "${YELLOW}检测到的 PATH: ${DETECTED_PATH}${NC}"

    # 创建 plist 内容
    cat > "$PLIST_PATH" <<EOF
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
  <key>Label</key>
  <string>${PLIST_NAME}</string>
  <key>ProgramArguments</key>
  <array>
    <string>${GATEWAY_BIN}</string>
    <string>--config</string>
    <string>${CONFIG_FILE}</string>
  </array>
  <key>RunAtLoad</key>
  <true/>
  <key>KeepAlive</key>
  <true/>
  <key>EnvironmentVariables</key>
  <dict>
    <key>PATH</key>
    <string>${DETECTED_PATH}</string>
  </dict>
  <key>StandardOutPath</key>
  <string>${LOG_FILE}</string>
  <key>StandardErrorPath</key>
  <string>${ERR_LOG_FILE}</string>
  <key>WorkingDirectory</key>
  <string>/opt/homebrew/etc/mcp-gateway</string>
</dict>
</plist>
EOF

    echo -e "${GREEN}✓ plist 文件已创建: ${PLIST_PATH}${NC}"
}

# 加载服务
load_service() {
    echo -e "${GREEN}加载服务...${NC}"
    launchctl load "$PLIST_PATH"
    echo -e "${GREEN}✓ 服务已加载${NC}"
}

# 启动服务
start_service() {
    echo -e "${GREEN}启动服务...${NC}"
    launchctl start "$PLIST_NAME"
    echo -e "${GREEN}✓ 服务已启动${NC}"
}

# 停止服务
stop_service() {
    echo -e "${YELLOW}停止服务...${NC}"
    launchctl stop "$PLIST_NAME" 2>/dev/null || true
    echo -e "${GREEN}✓ 服务已停止${NC}"
}

# 卸载服务
unload_service() {
    echo -e "${YELLOW}卸载服务...${NC}"
    stop_service
    launchctl unload "$PLIST_PATH" 2>/dev/null || true
    echo -e "${GREEN}✓ 服务已卸载${NC}"
}

# 检查服务状态
check_status() {
    echo -e "${GREEN}检查服务状态...${NC}"

    # 检查服务是否加载
    if launchctl list | grep -q "$PLIST_NAME"; then
        echo -e "${GREEN}✓ 服务已加载${NC}"
    else
        echo -e "${RED}✗ 服务未加载${NC}"
        return 1
    fi

    # 检查进程是否运行
    if pgrep -f "mcp-gateway" > /dev/null; then
        echo -e "${GREEN}✓ 进程正在运行${NC}"
    else
        echo -e "${RED}✗ 进程未运行${NC}"
        return 1
    fi

    # 检查健康状态
    if command -v curl &> /dev/null; then
        echo -e "${GREEN}检查健康状态...${NC}"
        HEALTH=$(curl -s http://localhost:4298/health 2>/dev/null || echo "{}")
        echo "$HEALTH" | python3 -m json.tool 2>/dev/null || echo "$HEALTH"
    fi

    return 0
}

# 查看日志
view_logs() {
    local lines="${1:-50}"

    echo -e "${GREEN}=== 标准输出日志 (最近 ${lines} 行) ===${NC}"
    if [ -f "$LOG_FILE" ]; then
        tail -n "$lines" "$LOG_FILE"
    else
        echo -e "${YELLOW}日志文件不存在: $LOG_FILE${NC}"
    fi

    echo -e "\n${GREEN}=== 错误日志 (最近 ${lines} 行) ===${NC}"
    if [ -f "$ERR_LOG_FILE" ]; then
        tail -n "$lines" "$ERR_LOG_FILE"
    else
        echo -e "${YELLOW}日志文件不存在: $ERR_LOG_FILE${NC}"
    fi
}

# 实时查看日志
tail_logs() {
    echo -e "${GREEN}实时查看错误日志 (Ctrl+C 退出)...${NC}"
    if [ -f "$ERR_LOG_FILE" ]; then
        tail -f "$ERR_LOG_FILE"
    else
        echo -e "${RED}日志文件不存在: $ERR_LOG_FILE${NC}"
        exit 1
    fi
}

# 显示帮助
show_help() {
    cat <<EOF
mcp-gateway Launchd 管理工具

用法: $0 <command> [options]

命令:
  install     安装并启动 launchd 服务
  start       启动服务
  stop        停止服务
  restart     重启服务
  status      检查服务状态
  logs [n]    查看日志 (默认最近 50 行)
  tail        实时查看错误日志
  uninstall   卸载并停止服务
  help        显示此帮助信息

示例:
  $0 install              # 安装并启动服务
  $0 status               # 检查服务状态
  $0 logs 100             # 查看最近 100 行日志
  $0 tail                 # 实时查看日志
  $0 restart              # 重启服务
  $0 uninstall            # 卸载服务

配置文件位置:
  - Launchd plist: ${PLIST_PATH}
  - Gateway 配置: ${CONFIG_FILE}
  - 日志文件: ${LOG_FILE}
  - 错误日志: ${ERR_LOG_FILE}

EOF
}

# 主函数
main() {
    local command="${1:-}"

    case "$command" in
        install)
            create_plist
            load_service
            start_service
            sleep 2
            check_status
            ;;
        start)
            if [ ! -f "$PLIST_PATH" ]; then
                echo -e "${RED}错误: plist 文件不存在，请先运行 'install'${NC}"
                exit 1
            fi
            start_service
            sleep 2
            check_status
            ;;
        stop)
            stop_service
            ;;
        restart)
            stop_service
            sleep 1
            start_service
            sleep 2
            check_status
            ;;
        status)
            check_status
            ;;
        logs)
            view_logs "${2:-50}"
            ;;
        tail)
            tail_logs
            ;;
        uninstall)
            unload_service
            echo -e "${YELLOW}是否删除 plist 文件? (y/N)${NC}"
            read -r response
            if [ "$response" = "y" ] || [ "$response" = "Y" ]; then
                rm -f "$PLIST_PATH"
                echo -e "${GREEN}✓ plist 文件已删除${NC}"
            fi
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

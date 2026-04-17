#!/bin/bash

# mcp-gateway Launchd 安装脚本
# 用于创建和管理 mcp-gateway 的 launchd 服务，解决 PATH 环境变量问题
#
# 注意: 此脚本仅适用于 macOS
# Linux 用户请使用 install-systemd.sh

set -e

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

# 在开头检测操作系统
detect_os

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# 配置
PLIST_NAME="com.mcp-gateway"
PLIST_PATH="$HOME/Library/LaunchAgents/${PLIST_NAME}.plist"

# 使用 brew --prefix 动态获取 Homebrew 前缀，避免硬编码路径
HOMEBREW_PREFIX=$(brew --prefix 2>/dev/null || echo "/opt/homebrew")
GATEWAY_BIN="${HOMEBREW_PREFIX}/opt/mcp-gateway/bin/mcp-gateway"
CONFIG_FILE="${HOMEBREW_PREFIX}/etc/mcp-gateway/config.json"
LOG_DIR="${HOMEBREW_PREFIX}/var/log"
LOG_FILE="${LOG_DIR}/mcp-gateway.log"
ERR_LOG_FILE="${LOG_DIR}/mcp-gateway.err.log"

# 检测用户 PATH 组件
detect_paths() {
    local paths=()

    # 动态添加 Homebrew 路径
    if [ -d "${HOMEBREW_PREFIX}/bin" ]; then
        paths+=("${HOMEBREW_PREFIX}/bin")
    fi
    if [ -d "${HOMEBREW_PREFIX}/sbin" ]; then
        paths+=("${HOMEBREW_PREFIX}/sbin")
    fi

    # 常用 Homebrew 路径（兼容 Intel 和 Apple Silicon）
    for prefix in "/opt/homebrew" "/usr/local" "/home/linuxbrew/.linuxbrew"; do
        if [ -d "${prefix}/bin" ] && [ "${prefix}" != "$HOMEBREW_PREFIX" ]; then
            paths+=("${prefix}/bin")
        fi
        if [ -d "${prefix}/sbin" ] && [ "${prefix}" != "$HOMEBREW_PREFIX" ]; then
            paths+=("${prefix}/sbin")
        fi
    done

    # 检测 Node.js (nvm)
    if [ -n "$NVM_DIR" ] && [ -d "$NVM_DIR/versions/node" ]; then
        local latest_node=$(ls -t "$NVM_DIR/versions/node" 2>/dev/null | head -1)
        if [ -n "$latest_node" ]; then
            paths+=("$NVM_DIR/versions/node/$latest_node/bin")
        fi
    fi

    # 检测 Node.js (fnm)
    if command -v fnm &> /dev/null; then
        local fnm_dir=$(fnm env --dir 2>/dev/null | grep "FNM_DIR" | cut -d'=' -f2)
        if [ -n "$fnm_dir" ]; then
            paths+=("$fnm_dir")
        fi
    fi

    # 检测 uv (用于 Python 工具)
    if command -v uv &> /dev/null; then
        local uv_path=$(which uv | sed 's|/uv$||')
        paths+=("$uv_path")
    fi

    # 检测 pipx
    if command -v pipx &> /dev/null; then
        local pipx_path=$(which pipx | sed 's|/pipx$||')
        paths+=("$pipx_path")
    fi

    # 添加标准系统路径
    paths+=("/usr/local/bin" "/usr/bin" "/bin" "/usr/sbin" "/sbin")

    # 去重并连接
    echo "$(printf "%s\n" "${paths[@]}" | awk '!seen[$0]++' | tr '\n' ':' | sed 's/:$//')"
}

# 创建 launchd plist 文件
create_plist() {
    echo -e "${GREEN}创建 launchd plist 文件...${NC}"

    # 确保配置目录存在（用户可写）
    if [ ! -d "${CONFIG_FILE%/*}" ]; then
        echo -e "${YELLOW}创建配置目录: ${CONFIG_FILE%/*}${NC}"
        mkdir -p "${CONFIG_FILE%/*}" 2>/dev/null || {
            echo -e "${RED}错误: 无法创建配置目录 ${CONFIG_FILE%/*}${NC}"
            echo -e "${YELLOW}请手动运行: mkdir -p ${CONFIG_FILE%/*}${NC}"
            exit 1
        }
    fi

    # 确保日志目录存在（可能需要 sudo）
    if [ ! -d "$LOG_DIR" ]; then
        echo -e "${YELLOW}创建日志目录: $LOG_DIR${NC}"
        sudo mkdir -p "$LOG_DIR" 2>/dev/null || {
            echo -e "${RED}错误: 无法创建日志目录 $LOG_DIR${NC}"
            echo -e "${YELLOW}请手动运行: sudo mkdir -p $LOG_DIR${NC}"
            exit 1
        }
    fi

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
  <string>${CONFIG_FILE%/*}</string>
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

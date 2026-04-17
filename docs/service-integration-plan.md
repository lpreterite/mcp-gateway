# MCP Gateway 服务集成计划 (Service Integration Plan)

## 1. 目标 (Objectives)
将服务管理功能直接内置到 `mcp-gateway` 二进制程序中，取代现有的外部 Shell 脚本 (`install-launchd.sh`, `install-systemd.sh`)，提供统一的跨平台体验。

## 2. 技术选型 (Tech Stack)
- **服务管理库**: `github.com/kardianos/service` (支持 macOS launchd, Linux systemd/Upstart/SysV, Windows Service)
- **CLI 框架**: 继续使用现有的 `github.com/urfave/cli/v2`

## 3. 功能设计 (Feature Design)

### 3.1 新增子命令
用户可以通过以下命令管理服务：
- `mcp-gateway service install [--config PATH]`: 安装为系统服务
- `mcp-gateway service uninstall`: 卸载服务
- `mcp-gateway service start`: 启动服务
- `mcp-gateway service stop`: 停止服务
- `mcp-gateway service restart`: 重启服务
- `mcp-gateway service status`: 检查服务运行状态

### 3.2 环境变量检测 (PATH Detection)
在 Go 代码中移植 `install-launchd.sh` 的逻辑，自动检测并注入以下路径到服务环境：
- Homebrew (`/opt/homebrew/bin`, `/usr/local/bin`, etc.)
- Node.js (nvm, fnm 路径)
- Python (uv, pipx 路径)
- 标准系统路径

### 3.3 日志管理
利用 `kardianos/service` 的日志接口，将输出重定向到：
- **macOS**: `~/Library/Logs/mcp-gateway.log`
- **Linux**: 系统日志 (journald) 或指定文件

## 4. 实施步骤 (Implementation Steps)

1. **依赖添加**:
   ```bash
   go get github.com/kardianos/service
   ```
2. **服务结构体实现**:
   - 定义满足 `service.Interface` 接口的结构体。
   - 实现 `Start()` 和 `Stop()` 方法。
3. **子命令集成**:
   - 在 `cmd/gateway/main.go` 中添加 `service` 子命令。
   - 实现各子命令对应的操作逻辑。
4. **PATH 检测逻辑**:
   - 在 `src/utils` 或新模块中实现 `DetectSystemPaths()` 函数。
5. **验证与测试**:
   - 在 macOS 上测试 `launchd` 集成。
   - 在 Linux 上测试 `systemd` 集成。

## 5. 迁移方案 (Migration)
- 保留现有脚本作为备份，但在 README 中标记为已弃用 (Deprecated)。
- 推荐新用户直接使用 `mcp-gateway service install`。

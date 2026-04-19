# 更新日志 (CHANGELOG)

所有对项目的显著更改都将记录在此文件中。

## [1.2.5] - 2026-04-19

### 修复

- **Pool 连接初始化时序** (`src/pool/pool.go`): 移除 `Connect()` 中硬编码的 `time.Sleep(100ms)`，改为先启动 `readResponses` 协程再标记 connected。npx/uvx 类慢启动服务（playwright、lark、minimax 等）不再因握手超时而注册失败。
- **JSON-RPC notification 与 response ID 冲突** (`src/pool/pool.go`): `handleResponse` 中 `ID` 类型从 `int` 改为 `*int`，notification（无 `id` 字段）反序列化后为 `nil` 直接跳过，避免与 request id=0 冲突导致 searxng 等发 notification 的服务超时。
- **连接池按需创建连接缺少握手** (`src/pool/pool.go`): `acquire()` 创建新连接后增加 `Initialize()` 调用，确保新连接完成 MCP 握手后才返回使用。

### 文档

- 新增 ISSUE-001 问题追踪文档

## [1.2.4] - 2026-04-18

### 修复

- **Homebrew reinstall 问题**: 在 Formula 中添加 `backup: true` 选项，避免 `brew reinstall` 时因配置文件已存在而报错。

## [1.2.3] - 2026-04-18

### 修复

- **服务启动逻辑**: 修复 `service start` 命令在进程已运行时不调用 `bootstrap()` 的问题。当用户直接运行 `mcp-gateway` 后再执行 `service start`，现在能正确将服务注册到 launchd。

## [1.2.2] - 2026-04-21

### 新增

- **多平台 CI 测试**: GitHub Workflow 支持 Ubuntu、macOS、Windows 三平台测试

### 修复

- **CI 问题修复**: 解决了多系统 CI 测试中的各种问题
  - 测试超时问题：修复 `TestGracefulShutdownChannel` 和 `TestServerMultipleStartStops` 超时
  - Windows 编译问题：添加 `facade_windows.go` 及 `newFacadePlatformAdapter` 函数
  - 端口稳定性问题：修复 `Port: 0` 随机端口分配
  - Lint 问题：添加 nolint 注释、参数验证、权限修复
  - CI Workflow 问题：移除 Clean old cache 步骤解决权限问题
  - Windows PowerShell 语法问题：修复 CI Windows Test job 的 bash 语法

### 文档

- **移除 Docker 相关文档**: 服务推荐使用系统服务直接运行
- **更新 M5 Sprint 6 测试验证记录**: 添加 OpenCode MCP 集成测试和 lark MCP 凭证问题调查
- **更新 M6 Sprint 7 发布准备状态**: 确认 CI/CD 配置完整

## [1.2.1] - 2026-04-17

### 新增
- **配置管理**: 增加了 `config info` 和 `config init` 子命令，方便用户查看生效配置路径和快速初始化用户配置。

### 修复
- **服务管理**: 修复了 macOS 上内置服务安装时的权限问题，现在默认安装为用户级服务 (`LaunchAgents`)。
- **日志系统**: 实现了非交互模式下的自动日志重定向到 `~/Library/Logs/mcp-gateway.log`。
- **配置加载**: 完善了 Homebrew 安装环境下的配置查找逻辑，支持自动识别 `/opt/homebrew/etc/...` 路径。
- **代码结构**: 将内部服务包重命名为 `gwservice` 以消除与第三方库的包名冲突。

## [1.2.0] - 2026-04-17

### 新增
- **内置服务管理**: 核心功能，集成 `github.com/kardianos/service`，通过 `mcp-gateway service` 子命令提供跨平台服务管理。
- **自动 PATH 检测**: 在服务启动时自动检测 Node.js (nvm/fnm)、Homebrew 和 Python (uv/pipx) 的环境变量，解决 MCP 服务器无法启动的问题。

### 变更
- **分发方式**: 移除了外部的 `scripts/install-launchd.sh` 和 `scripts/install-systemd.sh` 脚本。
- **Homebrew**: 更新了 Formula 提示信息，推荐使用内置命令进行部署。

## [1.1.1] - 2026-04-17

### 修复
- **配置路径**: 增加了对 Homebrew 默认配置路径的查找支持。

## [1.1.0] - 2026-04-17

### 新增
- **初版服务集成**: 开始尝试将服务管理逻辑内置。

## [1.0.3] - 2026-04-17

### 修复
- **Launchd 脚本**: 修复了脚本中的语法错误并添加了动态 Homebrew 路径检测。

## [1.0.2] - 2026-04-16

### 新增
- **安装脚本**: 添加了最初的 `install-launchd.sh` 和 `install-systemd.sh` 脚本。

## [1.0.1] - 2026-04-16

### 修复
- **发布流程**: 改进了 GitHub Actions 的发布逻辑。

## [1.0.0] - 2026-04-15

### 新增
- **初始版本**: 支持连接池、HTTP/SSE 传输、工具映射映射等核心功能。

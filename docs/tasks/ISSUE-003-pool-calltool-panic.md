# ISSUE-003: Pool.CallTool 类型断言 Panic 导致 MCP 工具调用崩溃

## 问题描述

用户通过 OpenCode 接入 mcp-gateway 后，调用 `gateway_searxng_searxng_web_search` 等 MCP 工具时失败，报错：

```
The socket connection was closed unexpectedly.
```

## 根因分析

### 1. 环境层问题（已解决）

#### 1.1 端口冲突
- **现象**：launchd 日志 (`~/Library/Logs/mcp-gateway.log`) 显示 `listen tcp 0.0.0.0:4298: bind: address already in use`
- **原因**：一个残留的旧 `gateway` 进程 (PID 10863) 占用了 4298 端口，导致新安装的 mcp-gateway 服务无法监听端口，陷入无限重启循环
- **解决**：杀掉残留进程

#### 1.2 Node.js 路径错误
- **现象**：searxng 服务启动失败，日志报错 `env: node: No such file or directory`
- **原因**：存在一个空的 `~/.nvm/versions/node/v22.21.1` 目录，mcp-gateway 的 `DetectSystemPaths()` 自动检测到了这个错误路径。该目录下没有 node 二进制文件，导致子进程启动失败
- **解决**：删除空目录，重新安装服务让 mcp-gateway 检测到正确的 v22.21.0 路径

### 2. 代码层问题（P0 Bug，已修复）

#### 2.1 类型断言 Panic
- **严重等级**：P0（阻塞级）
- **现象**：每次调用 MCP 工具时，Gateway 进程处理线程崩溃，Socket 连接意外关闭
- **证据**：
```
time=2026-04-19T19:33:57.976+08:00 level=INFO msg="http: panic serving [::1]:63459:
interface conversion: interface {} is map[string]interface {}, not *pool.ToolCallResult
goroutine 81 [running]:
github.com/lpreterite/mcp-gateway/src/pool.(*Pool).CallTool(...)
    /home/runner/work/mcp-gateway/mcp-gateway/src/pool/pool.go:637 +0x45c
```

- **根因**：`MCPClientConnection.CallTool` 的返回类型为 `map[string]interface{}`，但在 `Pool.CallTool` 中被错误地强制转换为 `*pool.ToolCallResult`

- **修复方案**：将 `MCPClientConnection.CallTool` 的返回类型从 `map[string]interface{}` 改为 `*ToolCallResult`

## 版本信息

- **修复版本**：v1.2.7
- **修复日期**：2026-04-19
- **修复提交**：待提交

## 修复内容

1. **类型转换修复** (`src/pool/pool.go`): `MCPClientConnection.CallTool` 返回类型更改为 `*ToolCallResult`
2. **Node.js 环境路径** (环境): 清理残留空目录
3. **端口冲突** (环境): 解决 4298 端口占用

## 测试验证

- [x] `make check` 全部通过
- [x] 所有 MCP 服务正常初始化（51 个工具注册成功）
- [x] `searxng_web_search` 调用成功并返回搜索结果

## 相关文件

- `src/pool/pool.go` - 核心修复
- `src/gateway/server.go` - 调用方
- `src/stdio/server.go` - 调用方

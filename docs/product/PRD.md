# PRD: MCP Gateway 统一 MCP 服务网关

**状态**: Draft
**Author**: PO Agent
**Last Updated**: 2026-04-17
**Version**: 1.0
**Stakeholders**: AI Agent 用户、开发者、DevOps

---

## 1. 产品概述

### 1.1 一句话说明

**MCP Gateway** 是一款集中式 MCP 服务器管理服务，通过连接池复用和 HTTP/SSE 传输，让多个 AI Agent 共享同一组 MCP 服务器连接，无需为每个客户端单独启动 MCP 实例。

### 1.2 问题陈述

**用户痛点**：当前架构下，每个 MCP 客户端连接都会创建新的 MCP 服务实例（进程），导致：
- CPU 被大量 node 进程占满而无法使用
- 资源严重浪费，20 个并发客户端 = 20 组 MCP 进程
- 无法复用连接，每次工具调用都有进程创建开销

**根本原因**：
1. `MCPServerManager` 对每个 MCP server 只维护单一连接，无法应对并发
2. 同步串行处理工具调用，高并发下请求堆积
3. 只支持 stdio 进程传输，无法远程调用
4. 通过 `mcporter generate-cli` 每次 spawn 新进程，开销大

### 1.3 解决方案

使用**连接池复用** + **HTTP/SSE 传输**架构：
- 多个客户端通过 HTTP/SSE 连接到此服务
- 服务端管理 MCP server 连接池，复用连接
- 客户端按需调用工具，无需感知后端 MCP server 细节

### 1.4 目标用户

| 用户类型 | 使用场景 |
|---------|---------|
| AI 开发者 | 使用 OpenCode、Claude App 等工具，需要同时访问多个 MCP 服务 |

---

## 2. 用户故事与场景

### 2.1 核心用户故事

**US-1: 连接复用**
> 作为一个 AI 开发者，我希望多个 AI Agent 能共享 MCP 服务器连接，这样我可以节省 80% 的进程资源。

验收标准：
- [ ] 20 个并发客户端连接时，MCP 服务器进程数保持稳定（≤ poolSize × server数）
- [ ] 单个工具调用延迟 < 500ms（复用连接）

**US-2: 远程访问**
> 作为一个使用 OpenCode 的开发者，我希望通过 HTTP/SSE 远程调用 MCP 工具，这样我无需在本地安装 MCP 服务。

验收标准：
- [ ] OpenCode 可通过 `http://localhost:4298/sse` 配置远程 MCP 服务器
- [ ] 工具调用响应格式兼容 MCP 协议

**US-3: 工具映射**
> 作为一个 MCP 用户，我希望工具名有统一前缀（如 `minimax_web_search`），这样我容易区分工具来源。

验收标准：
- [ ] 工具列表 API 返回映射后的工具名（格式：`serverName__originalName`）
- [ ] 支持前缀映射和前缀剥离配置

**US-4: 服务管理**
> 作为一个 DevOps，我希望用标准方式管理 MCP Gateway 服务，这样我可以轻松实现开机自启和进程监控。

验收标准：
- [ ] 支持 `service install/start/stop/restart/status` 命令
- [ ] 状态输出包含分层诊断信息（Config/Install/Registration/Process/Health）

**US-5: Stdio Bridge**
> 作为一个 Claude Desktop 用户，我希望通过 stdio 模式使用 Gateway，这样我可以复用已有的 MCP 配置。

验收标准：
- [ ] Stdio Bridge 可作为独立进程运行
- [ ] Claude Desktop 可通过 Bridge 连接 Gateway

---

## 3. 功能需求

### 3.1 P0 - 核心功能（必须完成）

#### FR-P0-1: Go 版本核心架构
**描述**：将项目从 TypeScript/Node.js 迁移到 Go 语言，实现真正的跨平台零依赖安装。

**详细需求**：
- 单一二进制文件，无需安装 Node.js 运行时
- 编译为 `mcp-gateway` 可执行文件
- 支持 `go install github.com/lpreterite/mcp-gateway@latest` 直接安装

**验收标准**：
- [ ] `go build` 成功生成二进制文件
- [ ] 二进制文件可在 macOS/Linux/Windows 运行
- [ ] 无外部运行时依赖

---

#### FR-P0-2: MCP 连接池
**描述**：为每个 MCP server 管理固定数量的连接，支持获取/归还/复用。

**详细需求**：
- 每个 MCP server 维护 `poolSize` 个连接（可配置，默认：3）
- `acquire()` 获取可用连接，若无可用且未达上限则创建新连接
- `release()` 归还连接到池
- 支持 `minConnections` 预启动
- 支持 `maxConnections` 上限
- 支持 `idleTimeout` 空闲回收

**接口设计**：
```go
type PoolConfig struct {
    MinConnections int  // 每个 server 最少连接数（默认：1）
    MaxConnections int  // 每个 server 最大连接数（默认：5）
    AcquireTimeout int  // 获取连接超时（毫秒，默认：10000）
    IdleTimeout    int  // 空闲清理超时（毫秒，默认：60000）
    MaxRetries     int  // 最大重试次数（默认：3）
}

type MCPConnectionPool interface {
    Acquire(serverName string) (*MCPClient, error)  // 获取可用连接
    Release(serverName string, client *MCPClient)    // 归还连接
    Execute<R>(serverName string, fn func(*MCPClient) R) R  // 执行并自动归还
}
```

**验收标准**：
- [ ] 连接池在启动时初始化 `minConnections` 个连接
- [ ] `acquire()`/`release()` 正确管理连接状态
- [ ] 空闲连接在 `idleTimeout` 后被清理
- [ ] 故障时自动重连

---

#### FR-P0-3: HTTP/SSE 传输层
**描述**：实现 HTTP 服务器和 SSE (Server-Sent Events) 传输，支持客户端远程连接。

**端点定义**：

| 端点 | 方法 | 说明 |
|------|------|------|
| `/sse` | GET | 建立 SSE 连接，接收服务端推送 |
| `/sse` | POST | 发送 JSON-RPC 请求（兼容模式） |
| `/messages` | POST | 发送 JSON-RPC 请求（标准模式，需配合 `/sse`） |
| `/tools` | GET | 查询所有已注册的工具列表 |
| `/tools/call` | POST | 调用指定的工具 |
| `/health` | GET | 健康检查与运行时状态 |

**验收标准**：
- [ ] `GET /sse` 返回 200，响应头 `Content-Type: text/event-stream`
- [ ] 初始消息发送 `sessionId`
- [ ] `POST /messages?sessionId=xxx` 正确处理 JSON-RPC 请求
- [ ] `GET /health` 返回 `{status, ready, sessions, pool}` 结构

---

#### FR-P0-4: 工具注册表
**描述**：集中管理所有 MCP 服务器的工具，支持工具查找和过滤。

**接口设计**：
```go
type ToolInfo struct {
    Name        string  // 映射后的工具名（serverName__originalName）
    Description string  // 工具描述
    ServerName  string  // 所属服务器名
    OriginalName string // 原始工具名
    InputSchema  map[string]interface{}  // 输入参数 schema
}

type Registry interface {
    Register(serverName string, tools []ToolInfo)
    Unregister(serverName string)
    GetTool(name string) (ToolInfo, error)
    ListTools() []ToolInfo
    ListToolsByServer(serverName string) []ToolInfo
}
```

**验收标准**：
- [ ] 工具注册后可通过 `/tools` API 查看
- [ ] 支持按服务器名过滤工具列表
- [ ] 工具名格式为 `serverName__originalName`

---

### 3.2 P1 - 重要功能

#### FR-P1-1: Stdio Bridge
**描述**：实现独立进程模式的 Stdio Bridge，支持 Claude Desktop 等仅支持 stdio 的客户端。

**工作原理**：
```
Claude Desktop (stdio)
       │
       │ stdio (JSON-RPC)
       ▼
┌──────────────────────────────────────┐
│        Stdio Bridge Process          │
│                                      │
│  1. 解析 stdin JSON-RPC 请求         │
│  2. 通过 HTTP/SSE 转发到 Gateway     │
│  3. 将 Gateway 响应通过 stdout 返回   │
└──────────────────────────────────────┘
       │
       │ HTTP/SSE
       ▼
┌──────────────────────────────────────┐
│         MCP Gateway                  │
│         (localhost:4298)             │
└──────────────────────────────────────┘
```

**启动方式**：
```bash
mcp-gateway --stdio
# 或指定 Gateway URL
mcp-gateway --stdio --gateway http://localhost:4298
```

**验收标准**：
- [ ] Bridge 进程可独立启动
- [ ] 支持通过 stdin/stdout 进行 JSON-RPC 通信
- [ ] 与 Gateway 的 SSE 连接正常

---

#### FR-P1-2: 工具映射器
**描述**：处理 Gateway 级工具名与原始服务器工具名之间的名称转换。

**映射规则**：
| 原始工具 | 映射后工具 | 配置 |
|---------|-----------|------|
| `web_search` (minimax) | `minimax__web_search` | 前缀映射，剥离后添加 |
| `understand_image` (minimax) | `minimax__understand_image` | 前缀映射 |
| `analyze_image` (pencil) | `pencil__analyze_image` | 前缀映射 |

**配置示例**：
```json
{
  "mapping": {
    "minimax": { "prefix": "minimax", "stripPrefix": true },
    "pencil": { "prefix": "pencil", "stripPrefix": true }
  },
  "toolFilters": {
    "minimax": { "include": ["understand_image", "web_search"] }
  }
}
```

**验收标准**：
- [ ] 支持前缀映射（添加 server 前缀）
- [ ] 支持前缀剥离（移除原始前缀）
- [ ] 支持工具过滤（include/exclude）

---

#### FR-P1-3: 双轨制服务架构
**描述**：分离"服务管理轨"和"应用运行轨"，实现跨平台服务管理。

**架构**：
```
┌────────────────────────────────────────────────────────────┐
│                    Service Management Track                │
│                                                            │
│  CLI(service) -> ServiceFacade -> PlatformAdapter          │
│                                  ├─ macOS launchd adapter  │
│                                  └─ Linux systemd adapter   │
│                                                            │
│  职责：install / uninstall / load / unload / restart /    │
│       status probe / environment injection                 │
└──────────────────────────────┬─────────────────────────────┘
                               │
                               │ 启动契约：可执行文件、参数、环境
                               ▼
┌────────────────────────────────────────────────────────────┐
│                     Application Runtime Track              │
│                                                            │
│  main -> config.Load -> gateway.NewServer -> Start         │
│                                                            │
│  职责：配置校验、日志初始化、连接池启动、端口监听、健康检查、 │
│       优雅关闭                                             │
└────────────────────────────────────────────────────────────┘
```

**服务命令**：
```bash
mcp-gateway service install
mcp-gateway service start
mcp-gateway service stop
mcp-gateway service restart
mcp-gateway service status
```

**状态输出（分层诊断）**：
```
Config: valid
Install: present
Registration: loaded
Process: running
Health: healthy
Suggested action: none
```

**验收标准**：
- [ ] `service status` 输出分层诊断信息
- [ ] macOS 支持 launchd domain 探测和自愈
- [ ] Linux 支持 systemd 注册态探测
- [ ] 启动失败时给出 `Suggested action`

---

### 3.3 P2 - 优化功能

#### FR-P2-1: 配置管理
**描述**：支持 JSON 配置文件加载和环境变量覆盖。

**配置路径优先级**：
1. `--config` 参数（最高优先级）
2. `MCP_GATEWAY_CONFIG` 环境变量
3. `~/.config/mcp-gateway/config.json`（macOS/Linux）
4. `./config/servers.json`（项目目录，开发模式）

**配置文件格式**：
```json
{
  "gateway": {
    "host": "0.0.0.0",
    "port": 4298,
    "cors": true
  },
  "pool": {
    "minConnections": 1,
    "maxConnections": 5,
    "acquireTimeout": 10000,
    "idleTimeout": 60000
  },
  "servers": [
    {
      "name": "minimax",
      "type": "local",
      "command": ["uvx", "minimax-coding-plan-mcp"],
      "enabled": true,
      "poolSize": 3
    }
  ]
}
```

**验收标准**：
- [ ] 配置文件不存在时给出友好错误
- [ ] 支持配置校验和默认值填充
- [ ] 环境变量可覆盖配置值

---

#### FR-P2-2: 日志与可观测性
**描述**：结构化日志记录，关键操作可追踪。

**日志规范**：
- 使用 `log/slog` 进行结构化日志
- 错误日志包含上下文和堆栈
- 启动日志包含初始化进度

**关键日志**：
```
[gateway] MCP Gateway v1.0.0 starting...
[gateway] Loading config from: /path/to/config.json
[pool] Initialized minimax with 3/3 connections
[gateway] Listening on http://0.0.0.0:4298
[gateway] SSE endpoint: http://0.0.0.0:4298/sse
```

**验收标准**：
- [ ] 启动日志包含版本、配置路径、监听地址
- [ ] 连接池状态变化有日志
- [ ] 工具调用有请求/响应日志

---

#### FR-P2-3: 优雅关闭
**描述**：支持 SIGINT/SIGTERM 信号，实现优雅关闭。

**关闭流程**：
1. 停止接受新连接
2. 等待现有请求处理完成（超时 30s）
3. 关闭所有 MCP 客户端连接
4. 退出进程

**验收标准**：
- [ ] Ctrl+C 触发优雅关闭
- [ ] SIGTERM 触发优雅关闭
- [ ] 关闭超时后强制退出

---

## 4. 非功能需求

### 4.1 性能需求

| 指标 | 目标值 | 说明 |
|------|--------|------|
| 最大并发工具调用数 | `maxConnections` × 服务器数量 | 连接池容量决定 |
| 连接复用率 | 预热后 100% | 连接复用 |
| 典型延迟 | < 100ms | 复用连接的工具调用 |
| 启动时间 | < 10s | Gateway 启动到 ready |
| 内存占用 | ~50MB 基础 + ~10MB/连接 | 每 server |

### 4.2 可靠性需求

| 需求 | 说明 |
|------|------|
| 单服务器故障隔离 | 单个 MCP server 失败不影响 Gateway 和其他服务 |
| 连接失败重试 | 失败连接自动重连（最多 3 次） |
| 启动失败处理 | 部分服务器启动失败不影响整体启动 |
| 超时控制 | 工具调用有超时保护（默认 30s） |

### 4.3 兼容性需求

| 需求 | 说明 |
|------|------|
| API 兼容 | 与现有 API 端点完全兼容 |
| 配置兼容 | 现有 JSON 配置文件无需修改 |
| 客户端兼容 | 支持 OpenCode、其他 MCP 客户端 |
| 跨平台 | macOS (darwin/amd64, darwin/arm64)、Linux (amd64)、Windows |

---

## 5. 技术架构

### 5.1 系统架构图

```
┌─────────────────────────────────────────────────────────────────────┐
│                        MCP Gateway System                            │
│                                                                      │
│  ┌────────────────────────────────────────────────────────────────┐  │
│  │                    MCP Gateway (HTTP Server)                     │  │
│  │                                                                 │  │
│  │  ┌──────────────┐  ┌──────────────────┐  ┌──────────────────┐  │  │
│  │  │ HTTP/SSE     │  │ Connection Pool   │  │ Tool Registry    │  │  │
│  │  │ Transport    │  │ Manager           │  │ (Centralized)    │  │  │
│  │  └──────┬───────┘  └────────┬─────────┘  └──────────────────┘  │  │
│  │         │                   │                                  │  │
│  │         │           ┌───────┴─────────┐                        │  │
│  │         │           │  MCP Server Pool │                        │  │
│  │         │           │  ┌─────────────┐ │                        │  │
│  │         │           │  │ minimax     │ │ (x N connections)      │  │
│  │         │           │  │ pencil      │ │                        │  │
│  │         │           │  │ playwright   │ │                        │  │
│  │         │           │  └─────────────┘ │                        │  │
│  └─────────┼───────────┼───────────────────┼────────────────────────┘  │
│            │           │                   │                           │
│            │ HTTP/SSE  │                   │                           │
│            │           │                   │                           │
│  ┌─────────┴───────────┴───────────────────┴────────────┐              │
│  │                    Stdio Bridge                       │              │
│  │                     (独立进程)                         │              │
│  │  stdin ──> JSON-RPC ──> HTTP/SSE ──> Gateway ──> MCP  │              │
│  │  stdout <── JSON-RPC <── HTTP/SSE <──────────────────┘ │              │
│  │                                                        │              │
│  │  用途: Claude Desktop 等仅支持 stdio 的客户端          │              │
│  └────────────────────┬─────────────────────────────────┘              │
│                       │ stdio                                           │
│  ┌─────────────────────┴────────────────────┐                           │
│  │         Claude Desktop                    │                           │
│  │         (仅支持 stdio MCP 模式)           │                           │
│  └───────────────────────────────────────────┘                           │
└──────────────────────────────────────────────────────────────────────────┘
```

### 5.2 核心组件

#### 5.2.1 HTTP/SSE 传输层 (`src/gateway/server.go`)

处理客户端通过 Server-Sent Events (SSE) 连接的入口点。

**端点**：
- `POST /sse` - MCP 协议端点（Streamable HTTP）
- `GET /sse` - 建立持久 SSE 连接用于工具通知
- `POST /messages?sessionId=xxx` - 发送 JSON-RPC 请求
- `GET /tools` - 列出所有可用工具
- `POST /tools/call` - 通过 REST 直接调用工具
- `GET /health` - 健康检查

#### 5.2.2 连接池管理器 (`src/pool/pool.go`)

这是本架构的**核心创新**。为每个 MCP server 管理连接池。

**设计原则**：
- 每个 MCP server 维护 `poolSize` 个连接（可配置，默认：3）
- 连接在客户端请求之间复用
- 故障时自动连接恢复
- 空闲连接超时和清理

#### 5.2.3 工具注册表 (`src/registry/registry.go`)

集中注册表，将 Gateway 工具名映射到原始 MCP server 工具名。

**功能**：
- 存储所有 MCP server 的所有可用工具
- 按工具名查找服务器
- 基于配置的工具过滤

#### 5.2.4 工具映射器 (`src/registry/mapper.go`)

处理 Gateway 级工具名与原始服务器工具名之间的名称转换。

**映射规则**：
- `minimax__understand_image` → minimax server 上的原始 `understand_image`
- `pencil__create_component` → pencil server 上的原始 `create_component`

#### 5.2.5 服务管理 (`src/gwservice/`)

双轨制服务架构实现：
- `facade.go` - 对外统一命令入口
- `platform_darwin.go` - macOS 平台适配
- `platform_linux.go` - Linux 平台适配
- `status.go` - 分层状态探测与诊断模型
- `contract.go` - 服务轨与应用轨之间的契约

### 5.3 项目结构（Go 版本）

```
mcp-gateway/
├── cmd/
│   └── gateway/
│       └── main.go           # 主程序入口
├── src/
│   ├── gateway/
│   │   ├── server.go         # HTTP/SSE 服务器
│   │   ├── handler.go        # 请求处理器
│   │   └── types.go          # 类型定义
│   ├── pool/
│   │   ├── pool.go           # 连接池实现
│   │   └── client.go         # MCP 客户端
│   ├── registry/
│   │   ├── registry.go       # 工具注册表
│   │   └── mapper.go         # 工具名映射
│   ├── config/
│   │   └── loader.go         # 配置加载
│   └── stdio/
│       ├── bridge.go         # Stdio 桥接器
│       └── types.go          # 类型定义
├── go.mod
├── go.sum
└── Makefile
```

### 5.4 请求流程

```
客户端请求
      │
      ▼
┌─────────────────────────────────┐
│ 1. 将工具名映射到服务器           │
│    (ToolMapper)                  │
└─────────────────────────────────┘
      │
      ▼
┌─────────────────────────────────┐
│ 2. 从连接池获取连接               │
│    (MCPConnectionPool.Acquire)   │
└─────────────────────────────────┘
      │
      ▼
┌─────────────────────────────────┐
│ 3. 在服务器上执行工具调用         │
│    (MCPClient.CallTool)          │
└─────────────────────────────────┘
      │
      ▼
┌─────────────────────────────────┐
│ 4. 将连接归还到池                 │
│    (MCPConnectionPool.Release)   │
└─────────────────────────────────┘
      │
      ▼
    响应
```

---

## 6. 里程碑计划

### 6.1 当前状态

| 分支 | 说明 |
|------|------|
| `fix/mcp-initialize-handshake` | 正在修复 MCP 初始化握手问题 |

**已实现功能**：
- ✅ MCP 连接池（`src/pool/pool.go`）
- ✅ HTTP/SSE 传输层（`src/gateway/server.go`）
- ✅ 双轨制服务架构（`src/gwservice/`）
- ✅ 工具注册与映射（`src/registry/`）

**待解决问题**：
1. ⏳ playwright/lark 的 npx 启动问题（broken pipe）
2. ⏳ OpenCode MCP 工具调用验证
3. ⏳ architecture.md 文档需要更新为 Go 版本
4. ⏳ Stdio Bridge 未实现

### 6.2 里程碑规划

| 里程碑 | 内容 | 优先级 | 目标时间 |
|--------|------|--------|----------|
| **M1: Go 核心功能** | 基础架构、连接池、HTTP/SSE、工具注册表 | P0 | Sprint 1-2 |
| **M2: 工具链完善** | 工具映射、配置管理、日志、优雅关闭 | P1 | Sprint 3 |
| **M3: Stdio Bridge** | 独立进程桥接器，支持 Claude Desktop | P1 | Sprint 4 |
| **M4: 服务管理** | 双轨制服务架构、跨平台安装 | P1 | Sprint 5 |
| **M5: 测试与验证** | 单元测试、集成测试、OpenCode 验证 | P1 | Sprint 6 |
| **M6: 发布准备** | 文档更新、跨平台构建、发布流程 | P2 | Sprint 7 |

### 6.3 Sprint 详细规划

#### Sprint 1: Go 基础设施
**目标**：建立 Go 项目框架，配置管理，基础结构

**任务**：
- [ ] 初始化 Go 模块 (`go mod init`)
- [ ] 配置 `viper` 加载 JSON 配置文件
- [ ] 实现配置结构体与验证
- [ ] 实现日志框架（标准库 `log/slog`）
- [ ] 创建基础项目结构和 Makefile

**交付物**：
- 可运行的 `go build` 基础项目
- 配置加载验证通过
- 开发构建脚本

#### Sprint 2: 核心网关
**目标**：实现 HTTP/SSE 服务器和连接池

**任务**：
- [ ] HTTP 服务器（`/sse`, `/messages`, `/health`, `/tools`, `/tools/call`）
- [ ] 连接池实现（`acquire`/`release`/`execute`）
- [ ] MCP 客户端（`os/exec` 启动子进程，stdio 通信）
- [ ] 优雅关闭实现

**交付物**：
- HTTP 服务器正常运行
- 连接池功能完整
- 与现有 MCP 服务器通信正常

#### Sprint 3: 工具链完善
**目标**：实现工具注册表和名称映射

**任务**：
- [ ] 工具注册表（集中管理，按名称查找）
- [ ] 工具名映射（前缀映射、剥离、过滤）
- [ ] 配置文件格式兼容
- [ ] 结构化日志完善

**交付物**：
- 工具列表 API 正常工作
- 映射规则生效
- 日志可追踪

#### Sprint 4: Stdio Bridge
**目标**：支持 Claude Desktop 的 stdio 模式

**任务**：
- [ ] 实现 stdio 输入输出监听
- [ ] 桥接 stdio 协议与 HTTP/SSE 内部通信
- [ ] 独立进程模式切换（`--stdio` 参数）

**交付物**：
- 可作为独立进程运行
- 支持 Claude Desktop 连接

#### Sprint 5: 服务管理
**目标**：实现双轨制服务架构

**任务**：
- [ ] `ServiceFacade` 统一命令入口
- [ ] macOS `PlatformAdapter`（launchd 适配）
- [ ] Linux `PlatformAdapter`（systemd 适配）
- [ ] 分层状态探测与诊断

**交付物**：
- `service install/start/stop/restart/status` 命令正常
- 分层诊断输出
- 平台自愈能力

#### Sprint 6: 测试与验证
**目标**：功能验证和性能优化

**任务**：
- [ ] 单元测试覆盖（> 80%）
- [ ] 集成测试
- [ ] OpenCode MCP 工具调用验证
- [ ] playwright/lark broken pipe 问题修复

**交付物**：
- 测试覆盖率 > 80%
- 所有已知问题修复
- OpenCode 验证通过

#### Sprint 7: 发布准备
**目标**：准备跨平台发布

**任务**：
- [ ] GitHub Actions CI/CD 配置
- [ ] 多平台构建（darwin/amd64, darwin/arm64, linux/amd64, windows）
- [ ] 发布流程文档
- [ ] 更新 architecture.md 为 Go 版本

**交付物**：
- Release 发布流程
- 预编译二进制文件
- `go install` 支持

---

## 7. 风险与依赖

### 7.1 技术风险

| 风险 | 概率 | 影响 | 对策 |
|------|------|------|------|
| MCP SDK 依赖需重新实现协议解析 | 中 | 高 | 参考现有 SDK 实现，纯 JSON 处理，提前验证协议兼容性 |
| 性能问题（Go vs Node.js） | 低 | 中 | 预留 2 周性能优化时间，进行基准测试对比 |
| 并发模型差异（协程 vs 事件循环） | 中 | 中 | 充分测试连接池场景，使用 sync.Pool 优化资源管理 |
| 现有用户配置迁移 | 低 | 低 | 保持配置格式完全兼容，提供迁移文档 |
| 第三方依赖兼容性 | 低 | 高 | 锁定依赖版本，使用 go.mod replace 备用方案 |
| 跨平台构建复杂性 | 中 | 中 | 使用 GitHub Actions Matrix 构建，测试各平台二进制 |
| 优雅关闭实现遗漏 | 中 | 中 | Sprint 6 专项测试关闭流程，验证资源释放完整 |

### 7.2 待解决问题

| 问题 | 状态 | Owner | 优先级 |
|------|------|-------|--------|
| playwright/lark 的 npx 启动问题（broken pipe） | Open | @dev | P1 |
| OpenCode MCP 工具调用验证 | Open | @dev | P1 |
| architecture.md 文档需要更新为 Go 版本 | Open | @doc | P2 |

### 7.3 依赖关系

```
用户请求
   │
   ▼
┌─────────────────────────────────────────────────────────────┐
│                     External Dependencies                    │
├─────────────────────────────────────────────────────────────┤
│  MCP Servers (minimax, pencil, playwright, lark, etc.)    │
│  - 需要这些服务正常运行                                       │
│  - 工具列表通过连接池获取                                     │
└─────────────────────────────────────────────────────────────┘
   │
   ▼
┌─────────────────────────────────────────────────────────────┐
│                      MCP Gateway                             │
├─────────────────────────────────────────────────────────────┤
│  Go 标准库: net/http, log/slog, os/exec, context            │
│  第三方库: viper (配置), urfave/cli (CLI)                   │
│  运行平台: macOS (launchd), Linux (systemd)                 │
└─────────────────────────────────────────────────────────────┘
   │
   ▼
┌─────────────────────────────────────────────────────────────┐
│                      MCP Clients                             │
├─────────────────────────────────────────────────────────────┤
│  OpenCode, Claude Desktop (via Stdio Bridge), 其他 MCP 客户端 │
└─────────────────────────────────────────────────────────────┘
```

### 7.4 外部依赖

| 依赖 | 说明 | 版本要求 |
|------|------|----------|
| Go | 编程语言 | 1.21+ |
| viper | 配置管理 | latest |
| urfave/cli | CLI 框架 | v2 |
| MCP Servers | MiniMax, Pencil, Playwright, Lark 等 | 各自最新 |

---

## 8. 验收检查清单

### 8.1 功能验收

- [ ] Gateway 启动成功
- [ ] 健康检查返回 200
- [ ] 工具列表 API 返回正确数据
- [ ] 至少一个工具调用成功
- [ ] Stdio Bridge 启动成功（若实现）
- [ ] Stdio Bridge 能获取工具列表（若实现）
- [ ] Stdio Bridge 能调用工具（若实现）
- [ ] 编译无错误: `go build`

### 8.2 服务管理验收

- [ ] `service install` 安装成功
- [ ] `service start` 启动成功
- [ ] `service status` 输出分层诊断
- [ ] `service restart` 正确重启
- [ ] `service stop` 正确停止

### 8.3 性能验收

- [ ] 20 个并发客户端时进程数稳定
- [ ] 连接复用率预热后 100%
- [ ] 典型工具调用延迟 < 100ms

### 8.4 兼容性验收

- [ ] macOS (darwin/arm64) 运行正常
- [ ] macOS (darwin/amd64) 运行正常
- [ ] Linux (linux/amd64) 运行正常
- [ ] Windows (windows/amd64) 运行正常
- [ ] 现有配置文件无需修改

---

## 9. 附录

### 9.1 术语表

| 术语 | 说明 |
|------|------|
| MCP | Model Context Protocol，AI 工具调用协议 |
| SSE | Server-Sent Events，服务器推送事件 |
| 连接池 | 复用一组连接，避免频繁创建/销毁 |
| 双轨制 | 分离服务管理轨和应用运行轨 |
| Stdio Bridge | 将 stdio 协议桥接到 HTTP/SSE |

### 9.2 参考文档

- [架构文档](./architecture.md)
- [Go 迁移计划](./go-migration-plan.md)
- [服务集成方案](./service-integration-plan.md)
- [测试计划](./testing-plan.md)
- [实施计划](./implementation-plan.md)
- [MCP 协议](./mcp-protocol.md)

### 9.3 客户端配置示例

#### OpenCode
```json
{
  "mcpServers": {
    "gateway": {
      "url": "http://localhost:4298/sse",
      "enabled": true,
      "type": "remote"
    }
  }
}
```

#### Claude Desktop (via Stdio Bridge)
```json
{
  "mcpServers": {
    "gateway-bridge": {
      "command": "mcp-gateway",
      "args": ["--stdio", "--config", "/path/to/config.json"]
    }
  }
}
```

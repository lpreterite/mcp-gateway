# 技术架构

> **文档来源**: [PRD.md](../PRD.md)
> **版本**: 1.0
> **更新日期**: 2026-04-17
> **Author**: PO Agent

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

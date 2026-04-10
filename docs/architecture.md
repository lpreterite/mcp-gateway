# MCP Gateway 架构文档

## 愿景

**MCP Gateway** 是一款专为 AI 时代打造的基础设施层工具：

- **统一管理**：一个入口管理所有 MCP 服务，无需为每个 AI Agent 单独配置
- **AI 友好**：AI Agent 可以直接"对话"Gateway，动态发现和调用工具
- **智能桥接**：作为 MCP 服务与 AI Agent 之间的枢纽，支持工具映射、过滤和聚合
- **资源高效**：连接池复用机制，避免资源浪费，让 AI 专注任务而非基础设施

无论是 OpenCode、Claude App 还是其他 AI 工具，只需连接 Gateway，即可访问所有配置的 MCP 服务能力。

---

## 项目背景与目标

### 问题背景

用户反馈当前项目存在**架构缺陷**：每次客户端连接都会创建新的 MCP 服务实例，无法复用，导致 CPU 被大量 node 进程占满而无法使用。

### 根本原因分析

基于代码分析，发现以下问题：

1. **无连接池机制**：`MCPServerManager` 对每个 MCP server 只维护**单一连接**，无法应对并发请求
2. **同步串行处理**：工具调用是串行执行的，高并发场景下请求堆积
3. **无 HTTP 传输层**：当前只支持 stdio 进程传输，客户端必须本地进程调用，无法通过网络远程调用
4. **mcporter 依赖开销**：通过 `mcporter generate-cli` 每次 spawn 新进程，开销大
5. **无连接复用设计**：客户端连接与 MCP server 连接一一绑定，无法共享

### 项目目标

将项目改造为**集中式 MCP 配置管理服务**：
- 多个客户端通过 HTTP/SSE 连接到此服务
- 服务端管理 MCP server 连接池，复用连接
- 客户端按需调用工具，无需感知后端 MCP server 的细节

### 用户确认的决策

- **协议**：SSE (Server-Sent Events) 和 Streamable HTTP
- **向后兼容**：不需要保留 stdio 模式
- **规模**：小规模 (< 20 客户端)

---

## 概述

MCP Gateway 是一个**集中式 MCP 服务器管理服务**，允许多个远程客户端通过 HTTP/SSE 连接并共享 MCP 服务器连接池。这解决了为每个客户端连接生成新 MCP 服务器进程的问题。

## 架构图

```
┌─────────────────────────────────────────────────────────────────┐
│                      MCP Gateway (HTTP Server)                   │
│                                                                  │
│  ┌──────────────┐  ┌──────────────────┐  ┌──────────────────┐  │
│  │ HTTP/SSE     │  │ Connection Pool  │  │ Tool Registry    │  │
│  │ Transport    │  │ Manager          │  │ (Centralized)    │  │
│  │ (客户端连接)  │  │ (复用 MCP 连接)   │  │ (工具映射+过滤)  │  │
│  └──────┬───────┘  └────────┬─────────┘  └──────────────────┘  │
│         │                   │                                   │
│         │           ┌───────┴─────────┐                        │
│         │           │  MCP Server Pool │                        │
│         │           │  ┌─────────────┐ │                        │
│         │           │  │ minimax    │ │ (x N connections)      │
│         │           │  │ zai-mcp    │ │                        │
│         │           │  │ searxng    │ │                        │
│         │           │  └─────────────┘ │                        │
│         │           └───────────────────┘                        │
└─────────┼─────────────────────────────────────────────────────────┘
          │
          │ HTTP/SSE
          │
  ┌───────┴───────┐
  │  多个远程客户端  │
  │  (OpenCode,   │
  │   Claude App, │
  │   其他 MCP    │
  │   客户端)      │
  └───────────────┘
```

## 核心组件

### 1. HTTP/SSE 传输层 (`src/gateway/server.ts`)

处理客户端通过 Server-Sent Events (SSE) 连接的入口点。

**端点：**
- `POST /mcp` - MCP 协议端点（Streamable HTTP）
- `GET /sse` - 建立持久 SSE 连接用于工具通知
- `POST /tools/call` - 通过 REST 直接调用工具
- `GET /tools` - 列出所有可用工具
- `GET /health` - 健康检查

**关键类：**
- `MCPGatewayServer` - 封装 MCP SDK Server 的主服务器类

### 2. 连接池管理器 (`src/gateway/pool.ts`)

这是本架构的**核心创新**。为每个 MCP server 管理连接池。

**设计原则：**
- 每个 MCP server 维护 `poolSize` 个连接（可配置，默认：3）
- 连接在客户端请求之间复用
- 故障时自动连接恢复
- 空闲连接超时和清理

**接口：**
```typescript
interface PoolConfig {
  minConnections: number;  // 每个 server 最少连接数（默认：1）
  maxConnections: number;  // 每个 server 最大连接数（默认：5）
  acquireTimeout: number;   // 获取连接超时（毫秒，默认：10000）
  idleTimeout: number;     // 空闲清理超时（毫秒，默认：60000）
  maxRetries: number;      // 最大重试次数（默认：3）
}

class MCPConnectionPool {
  async acquire(serverName: string): Promise<MCPClient>;  // 获取可用连接
  async release(serverName: string, client: MCPClient): void; // 归还连接到池
  async execute<R>(serverName: string, fn: (client: MCPClient) => Promise<R>): Promise<R>;
}
```

### 3. 工具注册表 (`src/gateway/registry.ts`)

集中注册表，将 gateway 工具名映射到原始 MCP server 工具名。

**功能：**
- 存储所有 MCP server 的所有可用工具
- 按工具名查找服务器
- 基于配置的工具过滤

### 4. 工具映射器 (`src/gateway/mapper.ts`)

处理 gateway 级工具名与原始服务器工具名之间的名称转换。

**映射规则：**
- `minimax_understand_image` → minimax server 上的原始 `understand_image`
- `zhipu_analyze_image` → zai-mcp-server 上的原始 `analyze_image`

## 连接池行为

### 请求流程

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
│    (MCPConnectionPool.acquire)   │
└─────────────────────────────────┘
      │
      ▼
┌─────────────────────────────────┐
│ 3. 在服务器上执行工具调用         │
│    (MCPClient.callTool)          │
└─────────────────────────────────┘
      │
      ▼
┌─────────────────────────────────┐
│ 4. 将连接归还到池                 │
│    (MCPConnectionPool.release)   │
└─────────────────────────────────┘
      │
      ▼
   响应
```

### 连接生命周期

1. **创建**：池在启动时为每个 server 初始化 `minConnections` 个连接
2. **借用**：需要连接时，`acquire()` 返回空闲连接或创建新连接（如果未超过 `maxConnections`）
3. **使用中**：连接被标记为使用中，其他请求不能借用
4. **归还**：使用后，`release()` 将连接返回到空闲池
5. **超时**：空闲连接在 `idleTimeout` 后被清理
6. **故障恢复**：失败的连接会自动替换

## HTTP 传输机制

### 协议

使用 Streamable HTTP 和 SSE（Server-Sent Events），原因如下：
- MCP 协议原生支持流式响应
- 非常适合长时运行的工具调用
- 基于 HTTP（无 WebSocket 复杂性）
- 浏览器和 MCP 客户端原生支持

### 消息类型

1. **tool/call/result** - 工具执行结果
2. **tool/list/result** - 可用工具列表
3. **error** - 错误通知

## 配置参考

### servers.json

```json
{
  "gateway": {
    "host": "0.0.0.0",
    "port": 3000,
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
  ],
  "mapping": {
    "minimax": { "prefix": "minimax", "stripPrefix": true }
  },
  "toolFilters": {
    "minimax": { "include": ["understand_image", "web_search"] }
  }
}
```

### 配置段说明

| 配置段 | 说明 |
|--------|------|
| `gateway` | HTTP 服务器设置（主机、端口、CORS） |
| `pool` | 连接池行为设置 |
| `servers` | MCP 服务器配置 |
| `mapping` | 工具名前缀映射 |
| `toolFilters` | 工具包含/排除过滤器 |

## 文件结构

```
src/
├── gateway/
│   ├── index.ts       # HTTP Server 入口点
│   ├── server.ts      # 带 SSE 传输的 MCP Gateway Server
│   ├── pool.ts        # 连接池管理器
│   ├── registry.ts    # 工具注册表
│   └── mapper.ts      # 工具名映射
├── mcp/
│   ├── client.ts      # MCP 客户端封装
│   └── types.ts       # 类型定义
├── config/
│   ├── loader.ts      # 配置加载器
│   └── validator.ts   # 配置验证
└── utils/
    └── logger.ts      # 日志工具
```

## 性能特性

| 指标 | 值 |
|------|-----|
| 最大并发工具调用数 | `maxConnections` × 服务器数量 |
| 连接复用率 | 预热后 100% |
| 典型延迟 | < 100ms |
| 每个 server 内存占用 | ~50MB 基础 + ~10MB/连接 |

## 与旧架构的对比

| 指标 | 旧架构 | 新架构 |
|------|--------|--------|
| 20 并发客户端 | 每个客户端创建独立进程 | 共享连接池 |
| CPU 使用 | 大量 node 进程占满 | 每 server 仅 N 个进程，稳定 |
| 响应延迟 | 进程创建开销大 | < 100ms（连接复用） |
| 客户端数量 | 1:1 绑定 | 1:N 共享连接池 |

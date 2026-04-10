# MCP Gateway

**MCP 统一网关** - 面向 AI 的 MCP 服务中枢。

## 愿景

MCP Gateway 是一款专为 AI 时代打造的基础设施层工具：

- **统一管理**：一个入口管理所有 MCP 服务，无需为每个 AI Agent 单独配置
- **AI 友好**：AI Agent 可以直接"对话"Gateway，动态发现和调用工具
- **智能桥接**：作为 MCP 服务与 AI Agent 之间的枢纽，支持工具映射、过滤和聚合
- **资源高效**：连接池复用机制，避免资源浪费，让 AI 专注任务而非基础设施

无论是 OpenCode、Claude App 还是其他 AI 工具，只需连接 Gateway，即可访问所有配置的 MCP 服务能力。

## 问题背景

之前的架构每个客户端连接都会创建新的 MCP 服务实例，导致 CPU 被大量 node 进程占满。新的架构通过连接池复用和 HTTP/SSE 传输解决此问题。

## 架构图

```
┌─────────────────────────────────────────────────────────────────────┐
│                        MCP Gateway System                             │
│                                                                      │
│  ┌────────────────────────────────────────────────────────────────┐ │
│  │                    MCP Gateway (HTTP Server)                     │ │
│  │                                                                 │ │
│  │  ┌──────────────┐  ┌──────────────────┐  ┌──────────────────┐ │ │
│  │  │ HTTP/SSE     │  │ Connection Pool  │  │ Tool Registry    │ │ │
│  │  │ Transport    │  │ Manager          │  │ (Centralized)    │ │ │
│  │  └──────┬───────┘  └────────┬─────────┘  └──────────────────┘ │ │
│  │         │                   │                                  │ │
│  │         │           ┌───────┴─────────┐                        │ │
│  │         │           │  MCP Server Pool │                        │ │
│  │         │           │  ┌─────────────┐ │                        │ │
│  │         │           │  │ minimax    │ │ (x N connections)    │ │
│  │         │           │  │ zai-mcp    │ │                        │ │
│  │         │           │  │ searxng    │ │                        │ │
│  │         │           │  └─────────────┘ │                        │ │
│  └─────────┼───────────┼───────────────────┼──────────────────────┘ │
│            │           │                   │                          │
│            │ HTTP/SSE  │                   │                          │
│            │           │                   │                          │
│  ┌─────────┴───────────┴───────────────────┴────────────┐          │
│  │                    Stdio Bridge (独立进程)              │          │
│  │  Claude Desktop ─── stdio ───> Bridge ─── HTTP/SSE    │          │
│  └──────────────────────┬─────────────────────────────────┘          │
│                         │ stdio                                       │
│  ┌──────────────────────┴────────────────────┐                       │
│  │         Claude Desktop (仅支持 stdio)      │                       │
│  └───────────────────────────────────────────┘                       │
└──────────────────────────────────────────────────────────────────────┘
```

## 核心特性

- **连接池化**: 每个 MCP server 维护 N 个连接（可配置，默认 3 个）
- **HTTP/SSE 传输**: 客户端通过 Server-Sent Events 连接，无需本地进程调用
- **工具注册表**: 集中管理所有 MCP 服务器的工具映射
- **REST API**: 提供 HTTP 端点用于简单工具调用

## 快速开始

### 安装方式

**方式一：npm 全局安装（推荐）**
```bash
npm install -g git+https://github.com/packy/mcp-gateway.git

# 首次配置
mkdir -p ~/.config/mcp-gateway
cp /usr/local/lib/node_modules/mcp-gateway/config/servers.example.json ~/.config/mcp-gateway/config.json
# 编辑 ~/.config/mcp-gateway/config.json 添加你的 API keys

# 启动
mcp-gateway
```

**方式二：npx 直接运行**
```bash
# 需要先配置
mkdir -p ~/.config/mcp-gateway
npx git+https://github.com/packy/mcp-gateway.git -- copy-config ~/.config/mcp-gateway/config.json
# 编辑 ~/.config/mcp-gateway/config.json

# 运行
npx git+https://github.com/packy/mcp-gateway.git
```

**方式三：本地安装开发**
```bash
git clone https://github.com/packy/mcp-gateway.git
cd mcp-gateway
npm install && npm run build

# 首次配置
cp config/servers.example.json config/servers.json
# 编辑 config/servers.json

# 启动
npm run dev
```

### 1. 配置

**配置路径优先级：**
1. `MCP_GATEWAY_CONFIG` 环境变量
2. `~/.config/mcp-gateway/config.json` (全局安装)
3. `./config/servers.json` (本地开发)

**默认端口：** `4298`

### 2. 启动网关

**全局安装：**
```bash
mcp-gateway
```

**本地开发：**
```bash
npm run dev

```bash
# 开发模式
npm run dev

# 生产模式
npm run build
npm start
```

## API 端点

### SSE 连接（主要协议）

```
GET /sse
```

建立持久 SSE 连接，接收工具调用结果和服务器通知。SSE 端点同时支持 GET（建立流）和 POST（发送消息）。

### 消息发送

```
POST /messages?sessionId=<session_id>
Content-Type: application/json

{
  "jsonrpc": "2.0",
  "id": 1,
  "method": "tools/call",
  "params": {
    "name": "minimax_web_search",
    "arguments": { "query": "..." }
  }
}
```

### REST API（辅助协议）

**列出工具**
```
GET /tools
```

**调用工具**
```
POST /tools/call
Content-Type: application/json

{
  "name": "minimax_understand_image",
  "arguments": {
    "image_source": "https://example.com/image.png"
  }
}
```

**健康检查**
```
GET /health
```

## 工具名称映射

工具根据其服务器使用前缀暴露：

| 服务器 | 工具前缀 | 示例 |
|--------|----------|------|
| minimax | `minimax_` | `minimax_understand_image` |
| zai-mcp-server | `zhipu_` | `zhipu_analyze_image` |
| searxng | `searxng_` | `searxng_search` |

## 连接池行为

- 每个服务器以 `minConnections` 个连接开始（默认: 1）
- 池根据需要增长到 `maxConnections`（默认: 5）
- 连接在请求之间复用
- 空闲连接在 `idleTimeout` 后清理（默认: 60s）

## 性能

| 指标 | 值 |
|------|-----|
| 最大并发工具调用 | `maxConnections` × 服务器数量 |
| 连接复用率 | 预热后 100% |
| 典型延迟 | < 100ms |

## 文件结构

```
src/
├── gateway/
│   ├── index.ts       # HTTP Server 入口
│   ├── server.ts      # MCP Gateway Server (SSE transport)
│   ├── pool.ts        # 连接池管理器
│   ├── registry.ts    # 工具注册表
│   └── mapper.ts      # 工具名映射
├── stdio-bridge/       # Stdio 桥接器 (支持 Claude Desktop)
│   ├── index.ts       # stdio 入口
│   ├── bridge.ts      # 桥接逻辑
│   └── types.ts       # 类型定义
├── mcp/
│   ├── client.ts      # MCP 客户端封装
│   └── types.ts       # 类型定义
├── config/
│   └── loader.ts      # 配置文件加载
└── test/
    ├── bridge-test.ts  # Bridge 测试
    └── stdio-types-test.ts  # 类型测试
```

## 配置说明

### gateway

HTTP 服务器配置。

| 字段 | 类型 | 默认值 | 说明 |
|------|------|--------|------|
| host | string | "0.0.0.0" | 监听地址 |
| port | number | 3000 | 监听端口 |
| cors | boolean | true | 是否启用 CORS |

### pool

连接池配置。

| 字段 | 类型 | 默认值 | 说明 |
|------|------|--------|------|
| minConnections | number | 1 | 每个 server 最少连接数 |
| maxConnections | number | 5 | 每个 server 最大连接数 |
| acquireTimeout | number | 10000 | 获取连接超时(ms) |
| idleTimeout | number | 60000 | 空闲回收时间(ms) |

### servers

下游 MCP 服务器列表。

| 字段 | 类型 | 说明 |
|------|------|------|
| name | string | 服务器标识名 |
| type | local/remote | 服务器类型 |
| command | string[] | 启动命令（local 类型必填） |
| url | string | 服务器 URL（remote 类型必填） |
| enabled | boolean | 是否启用 |
| env | object | 环境变量 |
| poolSize | number | 此 server 的连接池大小 |

### mapping

工具名称映射规则。

| 字段 | 类型 | 说明 |
|------|------|------|
| prefix | string | 前缀名（如 `minimax`） |
| stripPrefix | boolean | 是否剥离前缀 |
| rename | object | 工具重映射表 |

### toolFilters

工具过滤规则。

| 字段 | 类型 | 说明 |
|------|------|------|
| include | string[] | 只包含的工具名 |
| exclude | string[] | 排除的工具名 |

## License

MIT

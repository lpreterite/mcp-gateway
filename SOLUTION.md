# MCP Gateway - 统一 MCP 服务网关

## 背景问题

之前的架构每个客户端连接都会创建新的 MCP 服务实例，导致 CPU 被大量 node 进程占满。

## 解决方案

使用 **连接池复用** 和 **HTTP/SSE 传输** 架构：

```
┌─────────────────────────────────────────────────────────────────┐
│                      MCP Gateway (HTTP Server)                   │
│                      AI Agent 的 MCP 服务中枢                      │
│                                                                  │
│  ┌──────────────┐  ┌──────────────────┐  ┌──────────────────┐  │
│  │ HTTP/SSE    │  │ Connection Pool  │  │ Tool Registry    │  │
│  │ Transport   │  │ Manager          │  │ (Centralized)    │  │
│  │ (OpenCode   │  │ (复用 MCP 连接)   │  │ (工具映射+过滤)  │  │
│  │  连接)       │  │                  │  │                  │  │
│  └──────────────┘  └──────────────────┘  └──────────────────┘  │
│         │                   │                                   │
│         │           ┌───────┴─────────┐                        │
│         │           │  MCP Server Pool │                        │
│         │           │  ┌─────────────┐ │                        │
│         │           │  │ minimax    │ │ (x N connections)      │
│         │           │  │ zai-mcp    │ │                        │
│         │           │  │ searxng    │ │                        │
│         │           │  └─────────────┘ │                        │
└─────────┼─────────────────────────────────────────────────────────┘
          │
          │ HTTP/SSE / REST
          │
  ┌───────┴───────┐
  │   AI Agents     │
  │  (OpenCode,    │
  │   Claude App,   │
  │   其他 AI 工具)  │
  └───────────────┘
```

## 核心特性

- **连接池化**: 每个 MCP server 维护 N 个连接（可配置，默认 3 个）
- **HTTP/SSE 传输**: 客户端通过 Server-Sent Events 连接，无需本地进程调用
- **工具注册表**: 集中管理所有 MCP 服务器的工具映射
- **REST API**: 提供 HTTP 端点用于简单工具调用

## 项目结构

```
src/
├── gateway/
│   ├── index.ts       # HTTP Server 入口
│   ├── server.ts      # MCP Gateway Server (SSE transport)
│   ├── pool.ts        # 连接池管理器
│   ├── registry.ts    # 工具注册表
│   └── mapper.ts      # 工具名映射
├── mcp/
│   ├── client.ts      # MCP 客户端封装
│   └── types.ts       # 类型定义
├── config/
│   ├── loader.ts      # 配置文件加载
│   └── validator.ts   # 配置校验
```

## 技术实现

### 1. SSE 传输协议

使用 MCP SDK 的 `SSEServerTransport` 实现 SSE 通信：

```
GET /sse     → 建立 SSE 流，接收服务端推送
POST /messages → 发送 JSON-RPC 请求
```

为每个客户端会话创建独立的 MCP Server 实例，确保隔离性。

### 2. 连接池管理

```typescript
// 每个 server 维护固定数量的连接
poolSize: 3  // 默认

// 请求时从池中获取空闲连接
// 自动扩容直到 maxConnections
// 空闲连接在 idleTimeout 后回收
```

### 3. 工具名称映射

| 原始工具 | 映射后工具 | 前缀处理 |
|---------|-----------|---------|
| `web_search` (MiniMax) | `minimax_web_search` | 剥离后添加 |
| `understand_image` (MiniMax) | `minimax_understand_image` | 剥离后添加 |
| `analyze_image` (zai-mcp-server) | `zhipu_analyze_image` | 剥离后添加 |
| `search` (searxng) | `searxng_search` | 剥离后添加 |

### 4. 服务容错

连接失败的服务不会导致 Gateway 启动失败：
- 启动时连接失败的服务器会重试连接
- 调用不可用服务时返回友好的错误信息
- 池化连接支持按需创建新连接

## 配置文件

`config/servers.json`:

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
      "name": "searxng",
      "type": "local",
      "command": ["/path/to/mcp-searxng"],
      "enabled": true,
      "poolSize": 2
    }
  ],
  "mapping": {
    "searxng": {
      "prefix": "searxng",
      "stripPrefix": true
    }
  }
}
```

## API 端点

| 端点 | 方法 | 说明 |
|------|------|------|
| `/sse` | GET | 建立 SSE 连接 |
| `/messages` | POST | 发送 JSON-RPC 请求 |
| `/tools/call` | POST | REST 方式调用工具 |
| `/tools` | GET | 列出所有可用工具 |
| `/health` | GET | 健康检查 |

## 客户端配置

### OpenCode

```json
{
  "mcp": {
    "gateway": {
      "url": "http://localhost:3000/sse",
      "enabled": true,
      "type": "remote"
    }
  }
}
```

### Claude Desktop

不支持 remote MCP 服务器，需使用其他方案。

## 安装和使用

### 前置条件

- Node.js 18+
- npm

### 安装

```bash
cd ~/Documents/Works/mcp-gateway
npm install
npm run build
```

### 启动

```bash
# 开发模式
npm run dev

# 生产模式
npm start
```

### 健康检查

```bash
curl http://localhost:3000/health
```

## 已知问题

### 1. chrome-devtools 服务不存在

`@executeautomation/chrome-devtools-mcp` 包在 npm 上不存在，会连接失败但不影响其他服务。

### 2. 智谱服务响应慢

`zai-mcp-server` 的图像分析工具可能超时，建议设置较长的超时时间。

## 架构优势

1. **连接复用** - 大幅减少进程数量
2. **集中管理** - 单一入口管理所有 MCP 服务
3. **工具映射** - 解决客户端前缀不匹配问题
4. **容错设计** - 单个服务失败不影响整体

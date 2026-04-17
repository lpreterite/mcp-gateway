# MCP Gateway API 使用文档

**状态**: Draft
**文档来源**: PRD.md
**Version**: 1.0
**Last Updated**: 2026-04-17

---

## 1. 概述

MCP Gateway 提供两类接口：

| 类型 | 协议 | 用途 |
|------|------|------|
| HTTP/SSE | REST + Server-Sent Events | 远程客户端连接 |
| JSON-RPC | MCP 协议兼容 | 工具调用和通知 |

---

## 2. HTTP API

### 2.1 健康检查

**端点**: `GET /health`

检查 Gateway 运行状态。

**响应示例**:
```json
{
  "status": "ok",
  "ready": true,
  "sessions": 3,
  "pool": {
    "totalServers": 2,
    "totalConnections": 6,
    "availableConnections": 4
  }
}
```

**状态值**:
- `ok` - 运行正常
- `initializing` - 初始化中
- `degraded` - 部分功能受损

---

### 2.2 工具列表

**端点**: `GET /tools`

查询所有已注册的工具。

**响应示例**:
```json
{
  "tools": [
    {
      "name": "minimax__web_search",
      "description": "Search the web for information",
      "serverName": "minimax"
    },
    {
      "name": "pencil__create_component",
      "description": "Create a UI component",
      "serverName": "pencil"
    }
  ]
}
```

---

### 2.3 工具调用 (REST)

**端点**: `POST /tools/call`

通过 REST API 直接调用工具。

**请求示例**:
```json
{
  "name": "minimax__web_search",
  "arguments": {
    "query": "天气"
  }
}
```

**成功响应**:
```json
{
  "result": {
    "content": [
      {
        "type": "text",
        "text": "今天天气晴，温度 25 度"
      }
    ],
    "isError": false
  }
}
```

**错误响应**:
```json
{
  "error": "tool minimax__web_search not found"
}
```

---

## 3. SSE 端点

### 3.1 建立 SSE 连接

**端点**: `GET /sse`

建立持久 SSE 连接，用于接收服务端推送。

**响应头**:
```
Content-Type: text/event-stream
```

**初始消息**:
```
event: connected
data: {"sessionId":"sse-1744892700123456789"}
```

**后续消息格式**:
```
event: message
data: <JSON-RPC 响应>
```

---

### 3.2 SSE JSON-RPC 请求

**端点**: `POST /sse`

通过 SSE 通道发送 JSON-RPC 请求。

**请求头**:
```
Content-Type: application/json
MCP-Session-ID: <sessionId>  // 可选
```

**请求体**:
```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "method": "tools/call",
  "params": {
    "name": "minimax__web_search",
    "arguments": {"query": "天气"}
  }
}
```

---

### 3.3 标准消息端点

**端点**: `POST /messages?sessionId=<sessionId>`

标准 JSON-RPC 请求端点（需配合 `/sse` 使用）。

**请求体**: 同 SSE JSON-RPC 请求

**响应**: JSON-RPC 响应

---

## 4. JSON-RPC API

Gateway 支持标准 MCP JSON-RPC 方法。

### 4.1 initialize

初始化连接。

**请求**:
```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "method": "initialize",
  "params": {}
}
```

**响应**:
```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "result": {
    "protocolVersion": "2024-11-05",
    "capabilities": {
      "tools": {}
    },
    "serverInfo": {
      "name": "mcp-gateway",
      "version": "1.0.0"
    }
  }
}
```

---

### 4.2 tools/list

列出所有可用工具。

**请求**:
```json
{
  "jsonrpc": "2.0",
  "id": 2,
  "method": "tools/list",
  "params": {}
}
```

**响应**:
```json
{
  "jsonrpc": "2.0",
  "id": 2,
  "result": {
    "tools": [
      {
        "name": "minimax__web_search",
        "description": "Search the web",
        "inputSchema": {
          "type": "object",
          "properties": {
            "query": {"type": "string"}
          }
        }
      }
    ]
  }
}
```

---

### 4.3 tools/call

调用指定工具。

**请求**:
```json
{
  "jsonrpc": "2.0",
  "id": 3,
  "method": "tools/call",
  "params": {
    "name": "minimax__web_search",
    "arguments": {"query": "天气"}
  }
}
```

**响应**:
```json
{
  "jsonrpc": "2.0",
  "id": 3,
  "result": {
    "content": [
      {
        "type": "text",
        "text": "今天天气晴"
      }
    ],
    "isError": false
  }
}
```

---

## 5. 错误码

| 错误码 | 说明 |
|--------|------|
| -32700 | Parse error - 无效的 JSON |
| -32600 | Invalid Request |
| -32601 | Method not found |
| -32602 | Invalid params |
| -32603 | Internal error |
| -32000 | Gateway 错误（如服务初始化中）|

---

## 6. 客户端配置示例

### OpenCode
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

### Claude Desktop (via Stdio Bridge)
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

---

## 7. 请求流程图

```
客户端
   │
   ├─► GET /sse ──────────────────────► 建立 SSE 连接
   │                                      返回 sessionId
   │
   ├─► POST /sse (JSON-RPC) ──────────► 发送请求
   │   或 POST /messages?sessionId=xxx
   │
   └─► GET /tools ───────────────────► 查询工具列表
       POST /tools/call ─────────────► 调用工具
       GET /health ──────────────────► 健康检查
```

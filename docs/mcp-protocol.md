# MCP HTTP/SSE 协议端点

MCP Gateway 使用 HTTP/SSE 协议与 MCP 客户端（如 OpenCode）通信。

## 端点概览

| 端点 | 方法 | 说明 |
|------|------|------|
| `/sse` | GET | 建立 SSE 连接，接收服务器推送的事件 |
| `/sse` | POST | 发送 JSON-RPC 请求（兼容模式） |
| `/messages` | POST | 发送 JSON-RPC 请求（标准模式，需配合 `/sse` 使用） |
| `/tools` | GET | 查询所有已注册的工具列表 |
| `/tools/call` | POST | 调用指定的工具 |
| `/health` | GET | 健康检查与运行时状态 |

## 端点详情

### GET /sse — 建立 SSE 连接

建立与网关的持久 SSE 连接，用于接收服务器推送的事件和 JSON-RPC 响应。

**请求**
```
GET http://localhost:4298/sse
```

**响应**
- 成功：HTTP 200，建立持久连接
- 响应头：
  ```
  Content-Type: text/event-stream
  Cache-Control: no-cache
  Connection: keep-alive
  ```
- 初始消息：发送 `sessionId`，格式为 `{"sessionId":"sse-xxx"}`
  ```
  event: connected
  data: {"sessionId":"sse-1776432890019520000"}
  ```

**后续事件流**
- 服务器通过此连接推送 JSON-RPC 响应
- 格式：`event: message\ndata: {"jsonrpc":"2.0","id":1,"result":{...}}\n\n`

---

### POST /sse — 发送 JSON-RPC 请求（兼容模式）

直接向网关发送 JSON-RPC 请求。与标准 `/messages` 端点的区别是可以自动关联活跃的 SSE 会话。

**请求**
```
POST http://localhost:4298/sse
Content-Type: application/json

{
  "jsonrpc": "2.0",
  "id": 1,
  "method": "initialize",
  "params": {
    "protocolVersion": "2024-11-05",
    "capabilities": {},
    "clientInfo": {"name": "opencode", "version": "1.0.0"}
  }
}
```

**响应**

方式一 — 通过 SSE 通道推送响应（如存在活跃 SSE 连接）：
```
HTTP/1.1 202 Accepted
```
响应通过已建立的 SSE 连接推送。

方式二 — 直接返回响应（如无活跃 SSE 连接）：
```
HTTP/1.1 200 OK
Content-Type: application/json

{"jsonrpc":"2.0","id":1,"result":{"protocolVersion":"2024-11-05","capabilities":{"tools":{}},"serverInfo":{"name":"mcp-gateway","version":"1.0.0"}}}
```

**SessionId 匹配优先级**
1. `MCP-Session-ID` 请求头
2. `sessionId` 查询参数
3. 自动匹配任意活跃的 SSE 会话

---

### POST /messages — 发送 JSON-RPC 请求（标准模式）

通过已建立的 SSE 会话发送 JSON-RPC 请求。

**请求**
```
POST http://localhost:4298/messages?sessionId=sse-1776432890019520000
Content-Type: application/json

{
  "jsonrpc": "2.0",
  "id": 1,
  "method": "tools/list",
  "params": {}
}
```

**响应**
```
HTTP/1.1 200 OK
Content-Type: application/json

{"jsonrpc":"2.0","id":1,"result":{"tools":[...]}}
```

**错误响应**
- `400 Bad Request` — 缺少 sessionId 参数
- `404 Not Found` — sessionId 对应的会话不存在
- `503 Service Unavailable` — 网关尚未 ready（初始化中）

---

### GET /tools — 工具列表

查询网关注册的所有可用工具。

**请求**
```
GET http://localhost:4298/tools
```

**响应**
```json
{
  "tools": [
    {
      "name": "pencil__create_component",
      "description": "Create a new component",
      "serverName": "pencil"
    },
    {
      "name": "playwright__screenshot",
      "description": "Take a screenshot",
      "serverName": "playwright"
    }
  ]
}
```

**工具名称格式**
- `serverName__originalName` — 格式（可配置前缀映射）
- 例如 `pencil__create_component` 表示来自 pencil 服务器的 `create_component` 工具

---

### POST /tools/call — 调用工具

调用指定的工具。

**请求**
```
POST http://localhost:4298/tools/call
Content-Type: application/json

{
  "name": "pencil__create_component",
  "arguments": {
    "path": "./components/Button",
    "type": "react"
  }
}
```

**响应**
```json
{
  "result": {
    "content": [
      {"type": "text", "text": "Component created successfully"}
    ],
    "isError": false
  }
}
```

**错误响应**
```json
{
  "error": "tool pencil__nonexistent not found"
}
```

---

### GET /health — 健康检查

检查网关运行状态和 MCP 服务器连接池状态。

**请求**
```
GET http://localhost:4298/health
```

**响应**
```json
{
  "status": "ok",
  "ready": true,
  "sessions": 2,
  "pool": {
    "pencil": {"active": 1, "idle": 1, "total": 2},
    "playwright": {"active": 0, "idle": 2, "total": 2},
    "lark": {"active": 0, "idle": 2, "total": 2}
  }
}
```

**状态字段说明**
- `status`: `ok`（正常）| `initializing`（初始化中）| `degraded`（有错误）
- `ready`: `true` 表示所有 MCP 服务器已初始化完成，可以处理工具请求；`false` 表示仍在初始化中
- `sessions`: 当前活跃的 SSE 会话数
- `pool`: 各 MCP 服务器的连接池状态

---

## JSON-RPC 方法

### initialize

MCP 握手请求，客户端在开始使用前必须调用。

**请求**
```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "method": "initialize",
  "params": {
    "protocolVersion": "2024-11-05",
    "capabilities": {},
    "clientInfo": {"name": "opencode", "version": "1.0.0"}
  }
}
```

**响应**
```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "result": {
    "protocolVersion": "2024-11-05",
    "capabilities": {"tools": {}},
    "serverInfo": {"name": "mcp-gateway", "version": "1.0.0"}
  }
}
```

### tools/list

查询所有可用工具。

**响应**
```json
{
  "jsonrpc": "2.0",
  "id": 2,
  "result": {
    "tools": [
      {
        "name": "tool_name",
        "description": "Tool description",
        "inputSchema": {"type": "object", "properties": {...}}
      }
    ]
  }
}
```

### tools/call

调用指定工具。

**请求**
```json
{
  "jsonrpc": "2.0",
  "id": 3,
  "method": "tools/call",
  "params": {
    "name": "pencil__create_component",
    "arguments": {"key": "value"}
  }
}
```

**响应**
```json
{
  "jsonrpc": "2.0",
  "id": 3,
  "result": {
    "content": [
      {"type": "text", "text": "Result text"}
    ],
    "isError": false
  }
}
```

---

## MCP 客户端配置示例

### OpenCode 配置

```json
{
  "mcpServers": {
    "gateway": {
      "url": "http://localhost:4298/sse"
    }
  }
}
```

OpenCode 会：
1. GET `/sse` 建立 SSE 连接
2. POST `/sse` 发送 `initialize` 请求
3. 接收响应后，POST `/sse` 发送 `tools/list` 获取可用工具
4. 通过同一 SSE 连接接收响应

### curl 测试

```bash
# 1. 建立 SSE 连接（在另一个终端）
curl http://localhost:4298/sse

# 2. 发送 initialize 请求
curl -X POST http://localhost:4298/sse \
  -H "Content-Type: application/json" \
  -d '{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2024-11-05","capabilities":{},"clientInfo":{"name":"test","version":"1.0.0"}}}'

# 3. 查看健康状态
curl http://localhost:4298/health

# 4. 获取工具列表
curl http://localhost:4298/tools
```

---

## 初始化流程

1. **启动阶段**：网关启动后台 goroutine 初始化各 MCP 服务器连接池
2. **工具收集**：每个 MCP 服务器通过连接池执行 `tools/list` 请求
   - 并行收集（所有服务器同时开始）
   - 超时时间 120 秒/服务器
   - 超时后该服务器标记为失败，但不影响其他服务器
3. **注册**：收集到的工具注册到网关注册表
4. **Ready 状态**：所有服务器收集完成后（或超时后），`ready` 变为 `true`

网关在 `ready=false` 时仍可接收请求，但 `tools/list` 和 `tools/call` 会返回错误 `gateway is still initializing`。

---

## 错误码

| HTTP 状态码 | 说明 |
|------------|------|
| 200 | 成功 |
| 400 | 缺少必要参数（如 sessionId） |
| 404 | 会话不存在 |
| 405 | 方法不允许（端点不支持该 HTTP 方法） |
| 500 | 服务器内部错误 |
| 503 | 服务暂不可用（网关初始化中） |

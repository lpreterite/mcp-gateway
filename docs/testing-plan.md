# MCP Gateway 测试计划

## 测试概述

本文档描述 MCP Gateway 系统的测试策略，包括单元测试、集成测试和手动验证步骤。

## 测试环境

### 前置条件

1. Node.js >= 18.0.0
2. npm 依赖已安装: `npm install`
3. 项目已编译: `npm run build`
4. 配置文件存在: `config/servers.json`

### 测试配置

测试使用的默认 Gateway 配置:
- Host: `0.0.0.0`
- Port: `4298`
- Gateway URL: `http://localhost:4298/sse`

---

## 单元测试

### 1. 配置加载器 (`src/config/loader.ts`)

| 测试用例 | 输入 | 预期结果 |
|---------|------|---------|
| 加载有效配置 | 正确的 `servers.json` | 返回ValidatedConfig 对象 |
| 加载缺失文件 | 不存在的路径 | 抛出 FileNotFound 错误 |
| 验证失败 | 无效的 JSON 或缺少必填字段 | 抛出验证错误 |
| 默认值 | 省略可选字段 | 使用默认值填充 |

**测试命令:**
```bash
npx tsx src/test/config-test.ts
```

### 2. 工具注册表 (`src/gateway/registry.ts`)

| 测试用例 | 输入 | 预期结果 |
|---------|------|---------|
| 注册工具 | ToolInfo 对象 | tools.size === 1 |
| 获取工具 | 已注册的名称 | 返回正确 ToolInfo |
| 获取所有工具 | 无 | 返回所有工具数组 |
| 获取特定服务器工具 | 服务器名 | 仅返回该服务器的工具 |
| 取消注册 | 已存在的工具名 | 工具被移除 |
| 清除服务器工具 | 服务器名 | 该服务器所有工具被移除 |

**测试命令:**
```bash
npx tsx src/test/registry-test.ts
```

### 3. 工具映射器 (`src/gateway/mapper.ts`)

| 测试用例 | 输入 | 预期结果 |
|---------|------|---------|
| 获取服务器工具名 | `understand_image`, `minimax` | `minimax_understand_image` |
| 获取原始工具名 | `minimax_understand_image`, `minimax` | `understand_image` |
| 获取服务器前缀 | 任意 | 返回所有前缀数组 |
| 包含过滤器 | 配置 include 列表 | 仅包含指定的工具 |
| 排除过滤器 | 配置 exclude 列表 | 排除指定的工具 |
| 重命名映射 | 配置 rename 规则 | 返回映射后的名称 |

**测试命令:**
```bash
npx tsx src/test/mapper-test.ts
```

### 4. Stdio Bridge (`src/stdio-bridge/bridge.ts`)

| 测试用例 | 输入 | 预期结果 |
|---------|------|---------|
| 构造函数 | BridgeConfig | 创建 Bridge 实例 |
| 连接状态 | 未连接 | isConnected() === false |
| 断开连接 | 已连接 | 清理 sessionId 和 abortController |

**测试命令:**
```bash
npx tsx src/test/bridge-test.ts
```

---

## 集成测试

### 1. Gateway 启动测试

**目的:** 验证 Gateway 能正确启动并初始化所有组件

**步骤:**
1. 启动 Gateway: `npm run dev`
2. 检查日志输出
3. 验证健康检查端点

**预期结果:**
```
[gateway] Starting MCP Gateway...
[gateway] Loaded configuration with N servers
[gateway] Connecting to <server-name>...
[gateway] Connected to <server-name>
[gateway] Found M tools from <server-name>
[gateway] MCP Gateway listening on http://0.0.0.0:4298
```

**验证命令:**
```bash
curl -s http://localhost:4298/health | jq .
```

**预期响应:**
```json
{
  "status": "ok",
  "sessions": 0,
  "pool": {
    "totalConnections": 3,
    "idleConnections": 3,
    "activeConnections": 0
  }
}
```

### 2. 工具列表 API 测试

**目的:** 验证 REST API 能正确返回工具列表

**步骤:**
```bash
curl -s http://localhost:4298/tools | jq .
```

**预期结果:**
```json
{
  "tools": [
    {
      "name": "minimax_understand_image",
      "description": "...",
      "serverName": "minimax"
    }
  ]
}
```

### 3. 工具调用 API 测试

**目的:** 验证通过 REST API 调用工具的完整流程

**前置条件:** Gateway 必须有至少一个已配置和连接的 MCP 服务器

**步骤:**
```bash
curl -s -X POST http://localhost:4298/tools/call \
  -H "Content-Type: application/json" \
  -d '{
    "name": "<gateway-tool-name>",
    "arguments": {}
  }' | jq .
```

**预期结果:**
- 返回工具执行结果
- 格式符合 ToolCallResult 接口

### 4. SSE 连接测试

**目的:** 验证 SSE 传输层能正确建立连接和处理消息

**步骤:**
1. 启动 Gateway: `npm run dev`
2. 使用 curl 建立 SSE 连接:
   ```bash
   curl -N http://localhost:4298/sse
   ```

**预期结果:**
- 连接建立成功
- 收到包含 sessionId 的初始消息
- 连接保持开放，等待消息

### 5. Stdio Bridge 端到端测试

**目的:** 验证 stdio-bridge 能正确与 gateway 通信

**前置条件:**
- Gateway 运行在 `http://localhost:4298`
- 至少配置了一个 MCP 服务器

**步骤:**

**5.1 启动 Bridge:**
```bash
npm run dev:bridge
```

**预期日志:**
```
[bridge] Starting MCP Gateway Stdio Bridge...
[bridge] Gateway URL: http://localhost:4298/sse
[bridge] Connected to gateway
[bridge] Available tools: tool1, tool2, ...
```

**5.2 发送初始化请求:**
```bash
echo '{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2024-11-05","capabilities":{},"clientInfo":{"name":"test","version":"1.0.0"}}}' \
  | node dist/stdio-bridge/index.js
```

**预期响应 (stdout):**
```json
{"jsonrpc":"2.0","id":1,"result":{"protocolVersion":"2024-11-05","capabilities":{"tools":{}},"serverInfo":{"name":"mcp-gateway-bridge","version":"1.0.0"}}}
```

**5.3 发送工具列表请求:**
```bash
echo '{"jsonrpc":"2.0","id":2,"method":"tools/list","params":{}}' \
  | node dist/stdio-bridge/index.js
```

**预期响应:**
```json
{"jsonrpc":"2.0","id":2,"result":{"tools":[...]}}
```

**5.4 发送工具调用请求:**
```bash
echo '{"jsonrpc":"2.0","id":3,"method":"tools/call","params":{"name":"<tool-name>","arguments":{}}}' \
  | node dist/stdio-bridge/index.js
```

**预期响应:**
```json
{"jsonrpc":"2.0","id":3,"result":{"content":[{"type":"text","text":"..."}]}}
```

### 6. 连接池行为测试

**目的:** 验证连接池的复用和清理机制

**步骤:**
1. 启动 Gateway
2. 并发发送多个工具调用请求
3. 检查连接池状态

**预期结果:**
- 连接被正确复用
- 连接池大小不超过配置的最大值
- 空闲连接被正确清理

**验证命令:**
```bash
curl -s http://localhost:4298/health | jq '.pool'
```

---

## Claude Desktop 集成测试

### 配置步骤

1. 确保 Gateway 运行: `npm run dev`
2. 确保 Bridge 已编译: `npm run build`
3. 编辑 Claude Desktop 配置:

**macOS:**
```bash
~/Library/Application\ Support/Claude/claude_desktop_config.json
```

**Windows:**
```
%APPDATA%\Claude\claude_desktop_config.json
```

4. 添加 mcpServers 配置:
```json
{
  "mcpServers": {
    "gateway-bridge": {
      "command": "/path/to/node",
      "args": ["/absolute/path/to/mcp-gateway/dist/stdio-bridge/index.js", "http://localhost:4298/sse"]
    }
  }
}
```

### 验证步骤

1. 重启 Claude Desktop
2. 检查 MCP 服务器是否连接成功
3. 尝试调用 Gateway 暴露的工具
4. 验证工具响应正确返回

---

## 性能测试

### 连接复用率测试

**目的:** 验证连接池能正确复用连接

**步骤:**
1. 启动 Gateway
2. 通过 Bridge 或 SSE 发送 100 次工具调用
3. 检查进程数量

**预期结果:**
- MCP 服务器进程数量保持稳定 (不超过 pool.maxConnections)
- 响应时间随连接预热而降低

### 并发测试

**目的:** 验证系统能处理并发请求

**步骤:**
```bash
# 并发发送 10 个请求
for i in {1..10}; do
  curl -s -X POST http://localhost:4298/tools/call \
    -H "Content-Type: application/json" \
    -d '{"name":"<tool>","arguments":{}}' &
done
wait
```

**预期结果:**
- 所有请求成功完成
- 无连接泄漏
- 无超时错误

---

## 故障排除

### Bridge 连接失败

**症状:** `[bridge] Failed to connect to gateway`

**排查步骤:**
1. 检查 Gateway 是否运行: `curl http://localhost:4298/health`
2. 检查 Gateway URL 是否正确
3. 检查网络连接
4. 检查 CORS 配置

### 工具调用超时

**症状:** 请求等待 30 秒后返回超时错误

**排查步骤:**
1. 检查 MCP 服务器是否运行正常
2. 检查工具参数是否正确
3. 增加 acquireTimeout 配置
4. 检查 MCP 服务器日志

### SSE 连接断开

**症状:** SSE 连接突然关闭

**排查步骤:**
1. 检查 Gateway 是否重启
2. 检查网络稳定性
3. 实现重连机制

---

## 回归测试清单

每次代码更改后，执行以下测试:

- [ ] Gateway 启动成功
- [ ] 健康检查返回 200
- [ ] 工具列表 API 返回正确数据
- [ ] 至少一个工具调用成功
- [ ] Stdio Bridge 启动成功
- [ ] Stdio Bridge 能获取工具列表
- [ ] Stdio Bridge 能调用工具
- [ ] 编译无错误: `npm run build`

---

## 测试报告模板

```markdown
## 测试报告 - [日期]

### 环境
- Node.js 版本: vXX.X.X
- 操作系统: macOS/Windows/Linux
- Gateway 版本: X.X.X

### 测试结果

| 测试项 | 状态 | 备注 |
|--------|------|------|
| Gateway 启动 | PASS/FAIL | |
| 健康检查 | PASS/FAIL | |
| 工具列表 | PASS/FAIL | |
| 工具调用 | PASS/FAIL | |
| Stdio Bridge | PASS/FAIL | |

### 问题记录

| 问题 | 严重性 | 状态 | 备注 |
|------|--------|------|------|
| #1 | 高/中/低 | open/resolved | |

### 总结

[测试通过/失败总数]
```

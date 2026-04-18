# OpenCode MCP 集成测试

本文档记录如何将 mcp-gateway 作为 MCP 服务器与 OpenCode 等 AI 编程工具集成测试。

---

## 前置条件

在开始测试之前，确保环境已就绪。

### 1. 检查 mcp-gateway 服务

```bash
# 检查进程是否在运行
lsof -i :4298

# 健康检查（返回 JSON 表示服务正常）
curl -s http://localhost:4298/health | jq .
```

预期输出示例：

```json
{
  "status": "ok",
  "ready": true,
  "sessions": 0,
  "version": "1.0.0"
}
```

> **注意**：`/health` 端点可快速响应，但 `/sse` 端点需要流式读取，直接 curl 会卡住等待。这是正常行为，不代表服务异常。

### 2. 检查 OpenCode 配置

确认 OpenCode 的 MCP 配置文件存在且格式正确：

```bash
cat ~/.config/opencode/opencode.json | jq '.mcpServers'
```

配置示例：

```json
{
  "gateway": {
    "url": "http://localhost:4298/sse",
    "enabled": true,
    "type": "remote"
  }
}
```

### 3. 确认 opencode 命令可用

```bash
which opencode
opencode --version
```

---

## opencode mcp 命令详解

`opencode mcp` 是 OpenCode 内置的 MCP 服务器管理命令，是验证集成的核心工具。

### 命令列表

| 命令 | 说明 |
|------|------|
| `opencode mcp list` | 查看所有 MCP 服务器状态 |
| `opencode mcp add` | 交互式添加 MCP 服务器 |
| `opencode mcp debug <name>` | 调试指定服务器的连接问题 |
| `opencode mcp auth [name]` | 管理服务器认证 |
| `opencode mcp logout [name]` | 移除服务器认证 |

### opencode mcp list

这是最重要的验证命令，用于查看所有 MCP 服务器的连接状态。

#### 基本用法

```bash
opencode mcp list
```

#### 输出示例

```
┌  MCP Servers
│
●  ○ searxng          disabled
│      /Users/packy/.nvm/versions/node/v22.21.0/bin/mcp-searxng
│
●  ✗ gateway          failed
│      Failed to get tools
│      http://localhost:4298/sse
│
└  5 server(s)
```

#### 状态图标说明

| 图标 | 含义 | 说明 |
|------|------|------|
| `●` | 正常（running） | 服务器已连接，工具列表获取成功 |
| `○` | 禁用（disabled） | 服务器被禁用，不会建立连接 |
| `✗` | 失败（failed） | 连接或工具获取失败 |

#### 成功的连接

当 gateway 正常连接时，显示如下：

```
●  ● gateway          running
       http://localhost:4298/sse
```

#### 失败的连接

当连接失败时，显示错误详情：

```
●  ✗ gateway          failed
│      Failed to get tools
│      http://localhost:4298/sse
```

错误信息 `Failed to get tools` 表示：
- SSE 连接已建立，但获取工具列表时出错
- 可能是 gateway 后端服务异常或网络问题

### opencode mcp debug

调试特定服务器的详细连接信息：

```bash
opencode mcp debug gateway
```

输出包含：
- 连接建立过程
- 发送的请求
- 收到的响应
- 错误详情

### opencode mcp auth

管理 MCP 服务器认证：

```bash
opencode mcp auth gateway    # 为 gateway 服务器进行认证
opencode mcp logout gateway  # 移除 gateway 的认证
```

---

## 验证流程

按照以下步骤验证 gateway 成功接入 OpenCode。

### Step 1: 启动 mcp-gateway 服务

```bash
# 终端 1：启动服务
cd /Users/packy/Documents/Works/mcp-gateway
go run ./cmd/gateway --config ~/.config/mcp-gateway/config.json
```

### Step 2: 验证服务运行

```bash
# 终端 2：检查进程
lsof -i :4298

# 终端 2：健康检查（快速响应）
curl -s http://localhost:4298/health | jq .
```

### Step 3: 查看 OpenCode MCP 状态

```bash
# 终端 2：核心验证命令
opencode mcp list
```

**成功标志**：gateway 显示为 `● ● gateway running`

**失败标志**：gateway 显示为 `● ✗ gateway failed`

### Step 4: 测试工具调用（可选）

如果连接成功，可以测试实际工具调用：

```bash
# 查看可用的工具列表
curl -s http://localhost:4298/tools | jq '.tools[].name'
```

---

## 故障排查

### 排查矩阵

| 现象 | 可能原因 | 检查方法 |
|------|---------|---------|
| `✗ gateway failed` | 服务未启动 | `lsof -i :4298` |
| `✗ gateway failed` | SSE 连接失败 | `curl -N http://localhost:4298/sse` |
| `✗ gateway failed` | CORS 配置错误 | 检查 gateway 配置 |
| `✗ gateway failed` | 服务未就绪 | 等待 10-30 秒后重试 |

### 常见失败原因

#### 1. mcp-gateway 服务未启动

**检查方法**：

```bash
lsof -i :4298
```

**解决方法**：启动服务

```bash
go run ./cmd/gateway --config ~/.config/mcp-gateway/config.json
```

#### 2. SSE 端点无法连接

**检查方法**：

```bash
curl -N http://localhost:4298/sse
```

> **注意**：此命令会卡住（持续等待 SSE 流），这是正常行为。如果立即返回错误，则表示连接失败。

**解决方法**：
- 检查端口是否正确（默认 4298）
- 检查防火墙设置
- 检查 `~/.config/opencode/opencode.json` 中的 URL 是否正确

#### 3. CORS 问题

**检查方法**：查看 gateway 日志中是否有 CORS 错误

**解决方法**：在 gateway 配置中启用 CORS：

```json
{
  "cors": true,
  "corsOrigins": ["*"]
}
```

#### 4. 服务未就绪

MCP 服务器初始化需要时间（约 10-30 秒），特别是首次连接时。

**解决方法**：等待后重试

```bash
sleep 30
opencode mcp list
```

#### 5. 工具获取失败

即使 SSE 连接成功，如果后端 MCP 服务器异常，`opencode mcp list` 仍可能显示 `Failed to get tools`。

**检查方法**：

```bash
# 查看详细错误
opencode mcp debug gateway

# 检查后端服务器状态
curl -s http://localhost:4298/health | jq '.servers'
```

---

## 测试记录表

每次测试后记录结果，便于追踪问题和回归。

### 测试记录模板

| 测试时间 | 服务状态 | opencode mcp list 结果 | gateway 版本 | 备注 |
|---------|---------|----------------------|-------------|------|
| | | | | |

### 填写说明

- **测试时间**：格式 `YYYY-MM-DD HH:MM`
- **服务状态**：通过 `curl http://localhost:4298/health` 获取
- **opencode mcp list 结果**：复制 gateway 那行的完整输出
- **备注**：任何异常或特殊配置

### 示例记录

| 测试时间 | 服务状态 | opencode mcp list 结果 | gateway 版本 | 备注 |
|---------|---------|----------------------|-------------|------|
| 2026-04-18 14:30 | `{"status":"ok","ready":true}` | `● ● gateway running` | v1.2.1 | 首次测试，正常 |
| 2026-04-18 15:45 | `{"status":"ok","ready":true}` | `● ✗ gateway failed` | v1.2.1 | 端口被占用，重启后恢复 |

---

## 服务端点说明

| 端点 | HTTP 方法 | 说明 | curl 行为 |
|------|----------|------|----------|
| `/health` | GET | 健康检查 | 立即返回 JSON |
| `/tools` | GET | 获取工具列表 | 立即返回 JSON |
| `/sse` | GET | SSE 流式连接 | 持续等待流数据 |

> **重要**：`/sse` 端点使用 Server-Sent Events 协议，需要保持连接。直接 curl 会卡住，这是正常行为。使用 `curl -N` 或 `--no-buffer` 可以禁用缓冲。

---

## 测试命令汇总

```bash
# === 终端 1：启动服务 ===
cd /Users/packy/Documents/Works/mcp-gateway
go run ./cmd/gateway --config ~/.config/mcp-gateway/config.json --log-level debug

# === 终端 2：验证测试 ===

# 检查服务状态
curl http://localhost:4298/health | jq .

# 核心验证命令
opencode mcp list

# 调试特定服务器
opencode mcp debug gateway

# 测试 SSE 连接（会卡住，持续等待）
curl -N http://localhost:4298/sse

# 获取工具列表
curl http://localhost:4298/tools | jq .

# 查看连接池状态
curl http://localhost:4298/health | jq '.pool'
```

---

## 参考链接

- [MCP Gateway 部署说明](./deployment.md)
- [MCP 协议文档](https://modelcontextprotocol.io/)
- [OpenCode 文档](https://opencode.ai)

# MCP Gateway

[![Go Version](https://img.shields.io/badge/Go-1.26%2B-blue.svg)](https://golang.org/)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)
[![Build Status](https://github.com/lpreterite/mcp-gateway/actions/workflows/build.yml/badge.svg)](https://github.com/lpreterite/mcp-gateway/actions)

**MCP 统一网关** - 连接多个 MCP 服务器的统一网关，支持 HTTP/SSE 和 stdio 两种连接方式。

## 特性亮点

- **跨平台零依赖安装** - 单一 Go 编译二进制文件，下载即可运行
- **连接池管理** - 复用 MCP 服务器连接，提升性能
- **工具注册与映射** - 支持前缀映射和自定义重命名
- **优雅关闭** - 支持 SIGINT/SIGTERM，平滑处理现有连接
- **健康检查** - 内置 `/health` 端点便于监控

## 快速开始

### 安装

```bash
# Homebrew (推荐 ⭐)
brew install lpreterite/tap/mcp-gateway
```

### 配置与启动

`mcp-gateway` 遵循 Unix 惯例，优先从用户目录加载配置。

1. **初始化用户配置 (推荐)**：
   ```bash
   mcp-gateway config init
   ```
   这将在 `~/.config/mcp-gateway/config.json` 创建一份默认配置。

2. **查看当前生效的配置信息**：
   ```bash
   mcp-gateway config info
   ```

3. **编辑配置文件添加你的 MCP 服务器**：
   ```bash
   vim ~/.config/mcp-gateway/config.json
   ```


配置示例：

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
    "acquireTimeout": 5000,
    "idleTimeout": 30000
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
  }
}
```

2. **启动服务：**

```bash
# Homebrew 服务方式（推荐 ⭐）
brew services start mcp-gateway

# 或手动运行
mcp-gateway --config $(brew --prefix)/etc/mcp-gateway/config.json
```

服务默认监听 `http://localhost:4298`，可通过以下端点访问：
- `GET /health` - 健康检查
- `GET /tools` - 列出所有工具
- `GET /sse` - SSE 连接
- `POST /messages?sessionId=x` - 发送消息

### 服务管理 (推荐 ⭐)

`mcp-gateway` 内置了跨平台服务管理功能，支持自动检测 PATH 环境变量（包括 Homebrew, nvm, fnm, uv 等），确保服务能正确启动你的 MCP 服务器。

```bash
# 安装为系统服务 (macOS 为当前用户安装，Linux 需要 sudo)
mcp-gateway service install --config $(brew --prefix)/etc/mcp-gateway/config.json

# 启动/停止/重启
mcp-gateway service start
mcp-gateway service stop
mcp-gateway service restart

# 检查状态
mcp-gateway service status

# 查看日志 (macOS 默认位置)
tail -f ~/Library/Logs/mcp-gateway.log

# 卸载服务
mcp-gateway service uninstall
```

> **注意**：在 macOS 上，服务将安装为 `LaunchAgent`（用户级服务），无需 `sudo`。在 Linux 上，服务将安装为 `systemd` 单元，通常需要 `sudo` 权限。

`service status` 现在会输出分层诊断信息，而不仅是 `running/stopped`。典型输出包括：

- `Config`：配置文件是否合法、实际使用的配置路径
- `Install`：服务定义文件是否存在
- `Registration`：是否已加载到 `launchd`/`systemd`
- `Process`：是否存在受服务管理器托管的进程
- `Health`：网关端口是否可访问
- `Suggested action`：下一步建议操作

当网关刚启动且仍在后台初始化 MCP 连接时：

- `GET /health` 返回 `200`，其中 `status` 可能为 `initializing`，`ready` 为 `false`
- `GET /tools` 返回 `503`
- `POST /tools/call` 返回 `503`
- JSON-RPC 的 `tools/list` / `tools/call` 返回 `gateway is still initializing`

常用退出码：

- `10`：配置错误
- `20`：服务未安装
- `30`：服务注册或平台恢复失败
- `40`：运行时错误
- `50`：健康检查失败
- `60`：服务命令执行失败


### CLI 参数

| 参数 | 短选项 | 默认值 | 说明 |
|------|--------|--------|------|
| `--config <path>` | `-c` | 自动查找 | 配置文件路径 |
| `--host <addr>` | - | `0.0.0.0` | 监听地址 |
| `--port <port>` | `-p` | `4298` | 监听端口 |
| `--stdio` | - | `false` | 以 stdio 模式运行 |
| `--log-level` | - | `info` | 日志级别 (debug, info, warn, error) |
| `service <cmd>` | - | - | 服务管理 (install, uninstall, start, stop, restart, status) |
| `--version` | `-v` | - | 显示版本 |
| `--help` | `-h` | - | 显示帮助 |

### 配置路径优先级

1. `--config` 参数
2. `MCP_GATEWAY_CONFIG` 环境变量
3. `~/.config/mcp-gateway/config.json`
4. `./config/servers.json`（本地开发）

## API 端点

| 端点 | 方法 | 说明 |
|------|------|------|
| `/sse` | GET | 建立 SSE 连接 |
| `/messages?sessionId=x` | POST | 发送 JSON-RPC 消息 |
| `/tools` | GET | 列出所有可用工具 |
| `/tools/call` | POST | 调用工具（REST 风格） |
| `/health` | GET | 健康检查 |

### SSE 连接示例

```bash
# 建立 SSE 连接
curl http://localhost:4298/sse

# 发送消息
curl -X POST "http://localhost:4298/messages?sessionId=abc123" \
  -H "Content-Type: application/json" \
  -d '{
    "jsonrpc": "2.0",
    "id": 1,
    "method": "tools/call",
    "params": {
      "name": "minimax_web_search",
      "arguments": {"query": "hello"}
    }
  }'
```

### REST API 示例

```bash
# 列出工具
curl http://localhost:4298/tools

# 调用工具
curl -X POST http://localhost:4298/tools/call \
  -H "Content-Type: application/json" \
  -d '{
    "name": "minimax_web_search",
    "arguments": {"query": "hello"}
  }'

# 健康检查
curl http://localhost:4298/health
```

### 健康检查响应

```json
{
  "status": "ok",
  "version": "1.0.0",
  "uptime": 3600,
  "sessions": 2,
  "pool": {
    "minimax": {"total": 3, "active": 1, "idle": 2}
  }
}
```

## 工具映射

工具根据其服务器使用前缀暴露：

| 服务器 | 工具前缀 | 示例 |
|--------|----------|------|
| minimax | `minimax_` | `minimax_web_search` |
| searxng | `searxng_` | `searxng_search` |

可以在配置中自定义映射规则：

```json
{
  "mapping": {
    "minimax": {
      "prefix": "minimax",
      "stripPrefix": true,
      "rename": {
        "old_name": "new_name"
      }
    }
  }
}
```

## 后台运行

建议优先使用内置的 `service` 命令进行管理（见上文）。

如果您需要使用其他进程管理工具：

### pm2
```bash
pm2 start mcp-gateway -- --config /path/to/config.json
pm2 save
pm2 startup
```


## 项目结构

```
mcp-gateway/
├── cmd/
│   └── gateway/
│       └── main.go           # 主程序入口
├── src/
│   ├── config/
│   │   ├── loader.go         # 配置加载
│   │   └── types.go          # 类型定义
│   ├── gateway/
│   │   ├── server.go         # HTTP/SSE 服务器
│   │   └── types.go          # 类型定义
│   ├── pool/
│   │   ├── pool.go           # 连接池
│   │   └── client.go         # MCP 客户端
│   ├── registry/
│   │   ├── registry.go       # 工具注册表
│   │   └── mapper.go         # 工具映射
│   └── stdio/
│       ├── bridge.go         # Stdio 桥接器
│       └── server.go         # Stdio 服务器
├── config/
│   └── servers.example.json   # 配置示例
├── Makefile
└── go.mod
```

## 配置说明

### gateway

| 字段 | 类型 | 默认值 | 说明 |
|------|------|--------|------|
| host | string | `0.0.0.0` | 监听地址 |
| port | number | `4298` | 监听端口 |
| cors | boolean | `true` | 是否启用 CORS |

### pool

| 字段 | 类型 | 默认值 | 说明 |
|------|------|--------|------|
| minConnections | number | `1` | 每个 server 最少连接数 |
| maxConnections | number | `5` | 每个 server 最大连接数 |
| acquireTimeout | number | `10000` | 获取连接超时(ms) |
| idleTimeout | number | `60000` | 空闲回收时间(ms) |

### servers

| 字段 | 类型 | 说明 |
|------|------|------|
| name | string | 服务器标识名 |
| type | `local`/`remote` | 服务器类型 |
| command | string[] | 启动命令（local 类型必填） |
| url | string | 服务器 URL（remote 类型必填） |
| enabled | boolean | 是否启用 |
| env | object | 环境变量 |
| poolSize | number | 此 server 的连接池大小 |

## 开发

```bash
# 构建
make build

# 测试
make test

# 运行
./mcp-gateway --config config/servers.example.json
```

## License

MIT

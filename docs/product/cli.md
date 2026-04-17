# MCP Gateway CLI 使用文档

**状态**: Draft
**文档来源**: PRD.md
**Version**: 1.0
**Last Updated**: 2026-04-17

---

## 1. 概述

MCP Gateway CLI 提供多种运行模式：

| 模式 | 说明 | 典型场景 |
|------|------|----------|
| HTTP/SSE 服务器 | 默认模式，监听 HTTP/SSE 端口 | 远程客户端连接 |
| Stdio 模式 | 通过标准输入输出通信 | Claude Desktop 集成 |
| 服务管理 | 系统服务安装/卸载/控制 | 开机自启 |

---

## 2. 全局选项

| 选项 | 别名 | 默认值 | 说明 |
|------|------|--------|------|
| `--config` | `-c` | - | 配置文件路径 |
| `--host` | - | `0.0.0.0` | 监听地址 |
| `--port` | `-p` | `4298` | 监听端口 |
| `--log-level` | - | `info` | 日志级别 (debug/info/warn/error) |

---

## 3. 运行命令

### 3.1 启动 HTTP/SSE 服务器

```bash
# 默认配置启动
mcp-gateway

# 指定配置和端口
mcp-gateway -c /path/to/config.json -p 8080

# 开发模式（详细日志）
mcp-gateway -c /path/to/config.json --log-level debug
```

---

### 3.2 Stdio 模式

以 stdio 模式运行，作为 Claude Desktop 等客户端的 MCP 桥接。

```bash
# 默认连接到 localhost:4298
mcp-gateway --stdio

# 指定 Gateway 地址
mcp-gateway --stdio --gateway http://localhost:4298

# 指定配置文件
mcp-gateway --stdio -c /path/to/config.json
```

**stdio 模式工作原理**:
```
Claude Desktop (stdio)
        │
        │ stdio (JSON-RPC)
        ▼
┌──────────────────────────────────────┐
│        Stdio Bridge (mcp-gateway)     │
│                                      │
│  1. 解析 stdin JSON-RPC 请求         │
│  2. 通过 HTTP/SSE 转发到 Gateway     │
│  3. 将 Gateway 响应通过 stdout 返回  │
└──────────────────────────────────────┘
        │
        │ HTTP/SSE
        ▼
┌──────────────────────────────────────┐
│         MCP Gateway                  │
│         (localhost:4298)             │
└──────────────────────────────────────┘
```

---

## 4. 配置命令

### 4.1 查看配置状态

```bash
mcp-gateway config info
```

**输出示例**:
```
MCP Gateway Config Status:
Config paths searched:
  1. --config flag: not specified
  2. MCP_GATEWAY_CONFIG env: not set
  3. ~/.config/mcp-gateway/config.json: found
  4. ./config/servers.json: not found

Active config: ~/.config/mcp-gateway/config.json
```

---

### 4.2 初始化用户配置

创建默认配置文件到 `~/.config/mcp-gateway/config.json`。

```bash
mcp-gateway config init
```

**输出**:
```
✓ Config initialized at ~/.config/mcp-gateway/config.json
```

**默认配置内容**:
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
      "name": "example",
      "type": "local",
      "command": ["echo", "hello"],
      "enabled": true,
      "poolSize": 1
    }
  ],
  "mapping": {}
}
```

---

## 5. 服务管理命令

### 5.1 安装服务

将 MCP Gateway 安装为系统服务。

```bash
# 使用默认配置
mcp-gateway service install

# 指定配置文件
mcp-gateway service install -c /path/to/config.json
```

**输出示例**:
```
Service installed: mcp-gateway
Config path: ~/.config/mcp-gateway/config.json
Install path: ~/Library/LaunchAgents/mcp-gateway.plist
```

---

### 5.2 卸载服务

```bash
mcp-gateway service uninstall
```

---

### 5.3 启动服务

```bash
mcp-gateway service start
```

---

### 5.4 停止服务

```bash
mcp-gateway service stop
```

---

### 5.5 重启服务

```bash
mcp-gateway service restart
```

---

### 5.6 查看服务状态

```bash
mcp-gateway service status
```

**输出示例**:
```
MCP Gateway Service Status
═══════════════════════════════════════════════════════
Config:        valid   (~/.config/mcp-gateway/config.json)
Install:       present (~/Library/LaunchAgents/mcp-gateway.plist)
Registration:  loaded  (com.mcp-gateway.service)
Process:       running (PID: 12345)
Health:        healthy
Suggested action: none
═══════════════════════════════════════════════════════
```

**分层诊断说明**:

| 层级 | 状态 | 说明 |
|------|------|------|
| Config | valid/invalid/missing | 配置文件状态 |
| Install | present/missing | 服务安装文件状态 |
| Registration | loaded/unloaded/not-found | 系统服务注册状态 |
| Process | running/stopped/not-found | 进程运行状态 |
| Health | healthy/unhealthy/unknown | 健康检查结果 |

---

## 6. 命令索引

```
mcp-gateway [全局选项] [命令] [子命令] [参数]

全局选项:
  -c, --config <path>    配置文件路径
      --host <address>   监听地址
  -p, --port <port>      监听端口
      --log-level <level> 日志级别
      --stdio            Stdio 模式

命令:
  config                 配置管理
    info                 查看配置状态
    init                 初始化配置

  service                服务管理
    install              安装服务
    uninstall            卸载服务
    start                启动服务
    stop                 停止服务
    restart              重启服务
    status               查看服务状态

无子命令              启动 HTTP/SSE 服务器
```

---

## 7. 配置路径优先级

1. `--config` 命令行参数（最高）
2. `MCP_GATEWAY_CONFIG` 环境变量
3. `~/.config/mcp-gateway/config.json`（macOS/Linux）
4. `./config/servers.json`（项目目录）

---

## 8. 日志

### 8.1 交互模式

日志输出到 stderr，格式为文本：

```
level=INFO msg="MCP Gateway starting..." version=1.0.0
level=INFO msg="Listening on http://0.0.0.0:4298"
```

### 8.2 服务模式

日志重定向到：
- macOS: `~/Library/Logs/mcp-gateway/mcp-gateway.log`
- Linux: `/var/log/mcp-gateway.log`

---

## 9. 信号处理

| 信号 | 行为 |
|------|------|
| SIGINT (Ctrl+C) | 优雅关闭 |
| SIGTERM | 优雅关闭 |

**优雅关闭流程**:
1. 停止接受新连接
2. 等待现有请求处理完成（超时 30s）
3. 关闭所有 MCP 客户端连接
4. 退出进程

---

## 10. 退出码

| 退出码 | 说明 |
|--------|------|
| 0 | 成功 |
| 1 | 服务命令失败 |
| 2 | 配置错误 |
| 3 | 服务未安装 |


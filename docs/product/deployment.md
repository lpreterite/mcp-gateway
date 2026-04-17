# MCP Gateway 部署说明

本文档面向需要在生产环境或服务器上部署 MCP Gateway 的运维人员，介绍配置管理、部署场景及故障排查。

---

## 配置文件格式

### 配置文件结构

```json
{
  "gateway": { ... },
  "pool": { ... },
  "servers": [ ... ],
  "mapping": { ... },
  "toolFilters": { ... }
}
```

### gateway 配置

网关服务器配置。

| 字段 | 类型 | 默认值 | 说明 |
|------|------|--------|------|
| `host` | string | `0.0.0.0` | 监听地址 |
| `port` | number | `4298` | 监听端口 |
| `cors` | boolean | `true` | 是否启用 CORS |

```json
{
  "gateway": {
    "host": "0.0.0.0",
    "port": 4298,
    "cors": true
  }
}
```

### pool 配置

连接池配置，控制 MCP 服务器连接的复用策略。

| 字段 | 类型 | 默认值 | 说明 |
|------|------|--------|------|
| `minConnections` | number | `1` | 每个 server 预启动的最小连接数 |
| `maxConnections` | number | `5` | 每个 server 允许的最大连接数 |
| `acquireTimeout` | number | `10000` | 获取连接超时时间（毫秒） |
| `idleTimeout` | number | `60000` | 空闲连接回收时间（毫秒） |
| `maxRetries` | number | `3` | 连接失败最大重试次数 |

```json
{
  "pool": {
    "minConnections": 1,
    "maxConnections": 5,
    "acquireTimeout": 10000,
    "idleTimeout": 60000,
    "maxRetries": 3
  }
}
```

### servers 配置

MCP 服务器列表，定义需要代理的 MCP 服务。

| 字段 | 类型 | 必需 | 说明 |
|------|------|------|------|
| `name` | string | 是 | 服务器唯一标识名 |
| `type` | string | 是 | 服务器类型：`local` 或 `remote` |
| `command` | string[] | 是（local） | 启动命令 |
| `url` | string | 是（remote） | 远程服务器 URL |
| `enabled` | boolean | 否 | 是否启用（默认 true） |
| `env` | object | 否 | 环境变量 |
| `poolSize` | number | 否 | 此服务器的连接池大小 |

**local 类型示例**：

```json
{
  "servers": [
    {
      "name": "minimax",
      "type": "local",
      "command": ["uvx", "minimax-coding-plan-mcp"],
      "enabled": true,
      "env": {
        "API_KEY": "your-api-key"
      },
      "poolSize": 3
    }
  ]
}
```

**remote 类型示例**：

```json
{
  "servers": [
    {
      "name": "remote-mcp",
      "type": "remote",
      "url": "https://mcp.example.com/sse",
      "enabled": true,
      "poolSize": 2
    }
  ]
}
```

### mapping 配置

工具名称映射规则，控制对外暴露的工具名格式。

| 字段 | 类型 | 说明 |
|------|------|------|
| `prefix` | string | 工具名前缀 |
| `stripPrefix` | boolean | 是否剥离原始前缀 |
| `rename` | object | 工具重命名映射 |

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

### toolFilters 配置

工具过滤规则，控制哪些工具对客户端可见。

```json
{
  "toolFilters": {
    "minimax": {
      "include": ["understand_image", "web_search"],
      "exclude": ["admin_tool"]
    }
  }
}
```

---

## 环境变量

| 环境变量 | 说明 | 优先级 |
|----------|------|--------|
| `MCP_GATEWAY_CONFIG` | 配置文件路径 | 高于默认路径，低于 `--config` |
| `TZ` | 时区设置 | 影响日志时间戳 |

### 示例

```bash
# 使用指定配置文件
export MCP_GATEWAY_CONFIG=/etc/mcp-gateway/config.json
mcp-gateway

# 或在一行内执行
MCP_GATEWAY_CONFIG=/etc/mcp-gateway/config.json mcp-gateway
```

---

## 配置路径优先级

MCP Gateway 按以下顺序查找配置文件：

1. **`--config` 参数**（最高优先级）
   ```bash
   mcp-gateway --config /path/to/config.json
   ```

2. **`MCP_GATEWAY_CONFIG` 环境变量**
   ```bash
   export MCP_GATEWAY_CONFIG=/path/to/config.json
   ```

3. **`~/.config/mcp-gateway/config.json`**（用户目录）
   - macOS: `$HOME/.config/mcp-gateway/config.json`
   - Linux: `$HOME/.config/mcp-gateway/config.json`

4. **`./config/servers.json`**（本地开发）

5. **Homebrew 系统配置**（仅 macOS）
   - `/opt/homebrew/etc/mcp-gateway/config.json`
   - `/usr/local/etc/mcp-gateway/config.json`

6. **`/etc/mcp-gateway/config.json`**（Linux 系统级）

---

## 部署场景

### 场景一：本地开发

适用于在本地机器上进行 MCP Gateway 开发或测试。

```bash
# 1. 初始化配置
mcp-gateway config init

# 2. 编辑配置
vim ~/.config/mcp-gateway/config.json

# 3. 直接运行（实时查看日志）
mcp-gateway --log-level debug

# 4. 新开终端验证
curl http://localhost:4298/health
```

**配置文件示例**（本地开发）：

```json
{
  "gateway": {
    "host": "127.0.0.1",
    "port": 4298,
    "cors": true
  },
  "pool": {
    "minConnections": 1,
    "maxConnections": 3
  },
  "servers": [
    {
      "name": "minimax",
      "type": "local",
      "command": ["uvx", "minimax-coding-plan-mcp"],
      "enabled": true,
      "poolSize": 2
    }
  ]
}
```

---

### 场景二：单服务器部署

适用于在服务器上长期运行 MCP Gateway 服务。

#### 使用 systemd（Linux）

```bash
# 1. 创建 systemd 服务文件
sudo tee /etc/systemd/system/mcp-gateway.service << 'EOF'
[Unit]
Description=MCP Gateway Service
After=network.target

[Service]
Type=simple
User=your-user
WorkingDirectory=/home/your-user
ExecStart=/usr/local/bin/mcp-gateway --config /etc/mcp-gateway/config.json
Restart=always
RestartSec=5
Environment=NODE_ENV=production

[Install]
WantedBy=multi-user.target
EOF

# 2. 创建配置目录
sudo mkdir -p /etc/mcp-gateway

# 3. 复制并编辑配置
sudo cp /path/to/config.json /etc/mcp-gateway/config.json
sudo vim /etc/mcp-gateway/config.json

# 4. 重载 systemd 并启动服务
sudo systemctl daemon-reload
sudo systemctl enable mcp-gateway
sudo systemctl start mcp-gateway

# 5. 检查状态
sudo systemctl status mcp-gateway
```

#### 使用 launchd（macOS）

```bash
# 1. 创建 LaunchAgent 目录
mkdir -p ~/Library/LaunchAgents

# 2. 创建服务文件
tee ~/Library/LaunchAgents/com.mcp-gateway.plist << 'EOF'
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>Label</key>
    <string>com.mcp-gateway</string>
    <key>ProgramArguments</key>
    <array>
        <string>/usr/local/bin/mcp-gateway</string>
        <string>--config</string>
        <string>/usr/local/etc/mcp-gateway/config.json</string>
    </array>
    <key>RunAtLoad</key>
    <true/>
    <key>KeepAlive</key>
    <true/>
    <key>StandardOutPath</key>
    <string>/tmp/mcp-gateway.log</string>
    <key>StandardErrorPath</key>
    <string>/tmp/mcp-gateway.log</string>
</dict>
</plist>
EOF

# 3. 加载服务
launchctl load ~/Library/LaunchAgents/com.mcp-gateway.plist

# 4. 检查状态
launchctl list | grep mcp-gateway
```

#### 使用内置服务管理命令

```bash
# 安装服务
mcp-gateway service install --config /path/to/config.json

# 启动/停止/重启
mcp-gateway service start
mcp-gateway service stop
mcp-gateway service restart

# 查看状态
mcp-gateway service status
```

---

### 场景三：Docker 部署

适用于需要快速部署或隔离运行的环境。

#### docker-compose 部署（推荐）

```bash
# 1. 创建项目目录
mkdir -p ~/mcp-gateway
cd ~/mcp-gateway

# 2. 创建配置目录
mkdir -p config

# 3. 创建配置文件
cat > config/servers.json << 'EOF'
{
  "gateway": {
    "host": "0.0.0.0",
    "port": 4298,
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
  ]
}
EOF

# 4. 创建 docker-compose.yml
cat > docker-compose.yml << 'EOF'
services:
  mcp-gateway:
    build: .
    container_name: mcp-gateway
    ports:
      - "4298:4298"
    volumes:
      - ./config/servers.json:/app/config.json:ro
    environment:
      - TZ=Asia/Shanghai
    restart: unless-stopped
    healthcheck:
      test: ["CMD", "wget", "-qO-", "http://localhost:4298/health"]
      interval: 30s
      timeout: 10s
      retries: 3
      start_period: 10s
EOF

# 5. 启动服务
docker-compose up -d

# 6. 查看状态
docker-compose ps

# 7. 查看日志
docker-compose logs -f

# 8. 停止服务
docker-compose down
```

#### Docker 手动部署

```bash
# 1. 构建镜像
docker build -t mcp-gateway:latest .

# 2. 运行容器
docker run -d \
  --name mcp-gateway \
  -p 4298:4298 \
  -v /path/to/config.json:/app/config.json:ro \
  -e TZ=Asia/Shanghai \
  --restart unless-stopped \
  mcp-gateway:latest

# 3. 查看日志
docker logs -f mcp-gateway

# 4. 进入容器调试
docker exec -it mcp-gateway sh

# 5. 停止和删除
docker stop mcp-gateway && docker rm mcp-gateway
```

#### Docker 注意事项

> **重要**：Docker 容器内的 MCP 服务器无法直接访问宿主机的程序（如 `uvx`、`node` 等）。如果需要运行本地 MCP 服务器，请使用**主机网络模式**或**在容器内安装相关依赖**。

**使用主机网络**：

```yaml
services:
  mcp-gateway:
    network_mode: host
    # 注意：使用 host 网络时不需要 ports 配置
```

---

## 安全注意事项

### 网络安全

1. **限制监听地址**
   - 开发环境：`127.0.0.1`（仅本地访问）
   - 生产环境：`0.0.0.0`（需要配合防火墙）

2. **配置防火墙**
   ```bash
   # iptables (Linux)
   sudo iptables -A INPUT -p tcp --dport 4298 -s 192.168.1.0/24 -j ACCEPT
   sudo iptables -A INPUT -p tcp --dport 4298 -j DROP

   # ufw (Ubuntu/Debian)
   sudo ufw allow from 192.168.1.0/24 to any port 4298
   ```

3. **启用 HTTPS**
   - 使用 Nginx/Caddy 反向代理
   - 或使用 Docker 时配置 Let's Encrypt

### 配置安全

1. **保护敏感信息**
   - 避免在配置文件中明文存储密钥
   - 使用环境变量代替：
   ```json
   {
     "servers": [
       {
         "name": "minimax",
         "env": {
           "API_KEY": "${MINIMAX_API_KEY}"
         }
       }
     ]
   }
   ```

2. **设置配置文件权限**
   ```bash
   # Linux/macOS
   chmod 600 ~/.config/mcp-gateway/config.json
   ```

3. **以非 root 用户运行**
   - systemd 服务使用 `User=your-user`
   - Docker 镜像默认使用 `appuser`

### 服务安全

1. **定期更新**
   - 关注 GitHub Releases 获取安全更新
   - 使用 `brew upgrade mcp-gateway` 更新 Homebrew 安装

2. **启用日志监控**
   ```bash
   # 查看最近日志
   journalctl -u mcp-gateway -n 100 --no-pager

   # 实时跟踪
   journalctl -u mcp-gateway -f
   ```

---

## 故障排查

### 常见问题

#### 1. 服务启动失败

**症状**：`mcp-gateway service start` 返回错误。

**排查步骤**：

```bash
# 1. 检查配置语法
mcp-gateway config info

# 2. 手动运行查看错误
mcp-gateway --config /path/to/config.json --log-level debug

# 3. 检查端口占用
lsof -i :4298

# 4. 检查服务日志
journalctl -u mcp-gateway -n 50 --no-pager
```

#### 2. MCP 服务器连接失败

**症状**：`/health` 返回 `initializing` 或工具调用超时。

**排查步骤**：

```bash
# 1. 检查 MCP 服务器命令是否可用
which uvx  # 或其他启动命令

# 2. 手动测试启动命令
uvx minimax-coding-plan-mcp --help

# 3. 查看详细日志
mcp-gateway --log-level debug

# 4. 检查环境变量
echo $PATH
```

#### 3. 工具调用返回 503

**症状**：健康检查通过，但工具调用返回 503。

**原因**：连接池未就绪或所有连接都在使用中。

**排查步骤**：

```bash
# 1. 检查健康状态
curl http://localhost:4298/health

# 2. 查看连接池状态
# 在健康检查响应中查看 pool 字段

# 3. 等待连接池初始化
# 通常需要 10-30 秒预热

# 4. 增加连接池大小
```

#### 4. Docker 容器内 MCP 服务器无法启动

**症状**：Docker 部署时 MCP 服务器一直启动失败。

**原因**：容器内缺少运行时（如 `node`、`uvx` 等）。

**解决方案**：

1. **使用主机网络模式**：
   ```yaml
   services:
     mcp-gateway:
       network_mode: host
   ```

2. **在 Dockerfile 中安装依赖**：
   ```dockerfile
   FROM golang:alpine AS builder
   # ... 构建步骤

   FROM alpine:3.19
   RUN apk --no-cache add ca-certificates tzdata nodejs npm
   # ... 其他步骤
   ```

3. **使用远程 MCP 服务器**：
   ```json
   {
     "servers": [
       {
         "name": "remote-mcp",
         "type": "remote",
         "url": "https://mcp.example.com/sse"
       }
     ]
   }
   ```

### 日志分析

#### 日志位置

| 安装方式 | 日志路径 |
|---------|---------|
| systemd | `journalctl -u mcp-gateway` |
| Homebrew (macOS) | `~/Library/Logs/mcp-gateway.log` |
| Docker | `docker logs mcp-gateway` |
| 直接运行 | 输出到 stderr |

#### 日志级别

| 级别 | 使用场景 |
|------|---------|
| `debug` | 排查问题时获取详细信息 |
| `info` | 正常运行（默认） |
| `warn` | 存在潜在问题 |
| `error` | 仅显示错误 |

```bash
# 使用 debug 级别运行
mcp-gateway --log-level debug
```

### 服务状态诊断

```bash
# 查看分层状态
mcp-gateway service status

# 预期输出示例
Config: valid (/home/user/.config/mcp-gateway/config.json)
Install: present
Registration: loaded
Process: running
Health: healthy
Suggested action: none
```

**状态说明**：

| 状态 | 说明 |
|------|------|
| `Config: valid` | 配置文件存在且格式正确 |
| `Install: present` | 服务定义文件存在 |
| `Registration: loaded` | 已注册到服务管理器 |
| `Process: running` | 进程正在运行 |
| `Health: healthy` | 健康检查通过 |
| `Suggested action` | 建议的修复操作 |

### 性能问题

#### 连接池耗尽

**症状**：请求排队或超时。

**解决**：

1. 增加 `maxConnections`：
   ```json
   {
     "pool": {
       "maxConnections": 10
     }
   }
   ```

2. 为特定服务器增加 `poolSize`：
   ```json
   {
     "servers": [
       {
         "name": "high-traffic-server",
         "poolSize": 10
       }
     ]
   }
   ```

#### 内存占用过高

**排查**：

```bash
# 查看进程内存
ps aux | grep mcp-gateway

# Docker 环境
docker stats mcp-gateway
```

**优化建议**：
- 减少 `poolSize`
- 降低 `idleTimeout` 以加快空闲连接回收
- 限制最大并发连接数

---

## 服务管理命令

| 命令 | 说明 |
|------|------|
| `mcp-gateway service install` | 安装系统服务 |
| `mcp-gateway service start` | 启动服务 |
| `mcp-gateway service stop` | 停止服务 |
| `mcp-gateway service restart` | 重启服务 |
| `mcp-gateway service status` | 查看服务状态 |
| `mcp-gateway service uninstall` | 卸载服务 |

### 退出码

| 退出码 | 含义 | 处理建议 |
|--------|------|---------|
| `10` | 配置错误 | 检查配置文件语法和路径 |
| `20` | 服务未安装 | 运行 `service install` |
| `30` | 服务注册失败 | 检查权限或服务定义 |
| `40` | 运行时错误 | 查看日志获取详情 |
| `50` | 健康检查失败 | 等待初始化或检查端口 |
| `60` | 服务命令失败 | 检查命令执行权限 |

---

## 参考链接

- [GitHub 仓库](https://github.com/lpreterite/mcp-gateway)
- [安装指南](./installation.md)
- [配置示例](../config/servers.example.json)

# Go 语言迁移计划

## 背景与目标

当前 `mcp-gateway` 使用 TypeScript/Node.js 实现，存在以下问题：
- 依赖 Node.js 运行时，用户需预先安装
- npm 全局安装路径不一致问题
- 跨平台分发需要处理不同平台的 Node.js 环境

迁移到 Go 语言的目标：
- 编译为单一二进制文件，实现真正的跨平台零依赖安装
- 更好的性能和更低的内存占用
- 简化 CI/CD 交付流程
- 支持 `go install` 直接安装

---

## 技术选型

### 核心依赖

| 功能 | Go 方案 | 说明 |
|------|---------|------|
| HTTP/SSE 服务器 | 标准库 `net/http` | 原生支持，无需第三方 |
| JSON-RPC | 社区实现或手写 | 轻量级协议，简单实现 |
| MCP 协议通信 | 复用现有协议 | 解析 JSON 格式 |
| 配置文件 | 社区库 `viper` | 支持 JSON/YAML/TOML |
| 进程管理 | `os/exec` + stdio | 标准库实现 |
| 连接池 | sync.Pool + 手写管理 | 利用 Go 协程 |
| CLI 框架 | `urfave/cli/v2` | 轻量级、符合 Go 惯用风格 |
| 日志框架 | 标准库 `log/slog` | 结构化日志，运行时开销低 |

### 项目结构

```
mcp-gateway/
├── cmd/
│   └── gateway/
│       └── main.go           # 主程序入口
├── internal/
│   ├── gateway/
│   │   ├── server.go         # HTTP/SSE 服务器
│   │   ├── handler.go        # 请求处理器
│   │   └── types.go          # 类型定义
│   ├── pool/
│   │   ├── pool.go           # 连接池实现
│   │   └── client.go         # MCP 客户端
│   ├── registry/
│   │   ├── registry.go       # 工具注册表
│   │   └── mapper.go         # 工具名映射
│   ├── config/
│   │   └── loader.go         # 配置加载
│   └── stdio/
│       ├── bridge.go         # Stdio 桥接器
│       └── types.go          # 类型定义
├── go.mod
├── go.sum
└── Makefile
```

---

## 迁移阶段

### Phase 1: 基础设施 (第 1 周)

**目标**: 建立 Go 项目框架，配置管理，基础结构

**任务**:
1. 初始化 Go 模块 (`go mod init github.com/packy/mcp-gateway`)
2. 配置 `viper` 加载 JSON 配置文件
3. 实现配置结构体与验证（复用现有验证逻辑）
4. 实现日志框架（标准库 `log/slog` 或第三方 `zerolog`）
5. 创建基础项目结构和 Makefile

**交付物**:
- 可运行的 `go build` 基础项目
- 配置加载验证通过
- 开发构建脚本

### Phase 2: 核心网关 (第 2-3 周)

**目标**: 实现 HTTP/SSE 服务器和连接池

**任务**:
1. **HTTP 服务器**
   - GET `/sse` - 建立 SSE 流
   - POST `/messages` - 处理 JSON-RPC 消息
   - GET `/health` - 健康检查
   - GET `/tools` - 列出工具
   - POST `/tools/call` - REST 风格工具调用

2. **连接池实现**
   - `Pool` 结构体管理每个服务器的连接
   - `acquire`/`release` 获取/释放连接
   - 支持 `minConnections` 预启动
   - 支持 `maxConnections` 上限
   - `idleTimeout` 空闲回收

3. **MCP 客户端**
   - 使用 `os/exec` 启动子进程
   - 通过 stdin/stdout 通信
   - 实现 JSON-RPC 编解码

**交付物**:
- HTTP 服务器正常运行
- 连接池功能完整
- 与现有 MCP 服务器通信正常

### 优雅关闭实现

在 Go 中实现服务优雅关闭（Graceful Shutdown）：

```go
// 创建一个带超时的 context
ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
defer cancel()

// 启动关闭信号监听
sigChan := make(chan os.Signal, 1)
signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

// 在独立协程中执行关闭
go func() {
    sig := <-sigChan
    slog.Info("Received signal, initiating graceful shutdown", "signal", sig)

    // 停止接受新连接
    if err := srv.Shutdown(ctx); err != nil {
        slog.Error("Server shutdown error", "error", err)
    }

    // 关闭所有 MCP 客户端连接
    pool.CloseAll()

    slog.Info("Graceful shutdown completed")
    os.Exit(0)
}()
```

**关键点**:
- `srv.Shutdown(ctx)` 停止接受新连接，但保持现有连接处理完成
- 设置超时避免无限等待
- 先关闭 HTTP 服务器，再关闭连接池
- 捕获 SIGINT (Ctrl+C) 和 SIGTERM (systemd) 两种信号

### Phase 3: 工具注册与映射 (第 4 周)

**目标**: 实现工具注册表和名称映射

**任务**:
1. 工具注册表
   - 集中管理所有服务器的工具
   - 支持按名称查找工具
   - 返回工具列表（带映射后名称）

2. 工具名映射
   - 前缀映射（`minimax_`, `zhipu_` 等）
   - 剥离前缀选项
   - 自定义重命名

**交付物**:
- 工具列表 API 正常工作
- 映射规则生效

### Phase 4: Stdio 桥接 (第 5 周)

**目标**: 支持 Claude Desktop 的 stdio 模式

**任务**:
1. 实现 stdio 输入输出监听
2. 桥接 stdio 协议与 HTTP/SSE 内部通信
3. 独立进程模式切换（`--stdio` 参数）

**交付物**:
- 可作为独立进程运行
- 支持 Claude Desktop 连接

### Phase 5: 测试与优化 (第 6 周)

**目标**: 功能验证和性能优化

**任务**:
1. 单元测试覆盖
2. 集成测试
3. 性能基准测试
4. 内存和连接泄漏检查
5. 错误处理完善

**交付物**:
- 测试覆盖率 > 80%
- 性能达标
- 稳定运行

### Phase 6: 发布准备 (第 7 周)

**目标**: 准备跨平台发布

**任务**:
1. GitHub Actions CI/CD 配置
2. 多平台构建（darwin/amd64, darwin/arm64, linux/amd64, windows）
3. 发布流程文档
4. 安装脚本（install.sh）

**交付物**:
- Release 发布流程
- 预编译二进制文件
- `go install` 支持

---

## API 兼容性

保持与现有 API 完全兼容：

| 端点 | 方法 | 说明 |
|------|------|------|
| `/sse` | GET | 建立 SSE 连接 |
| `/messages?sessionId=x` | POST | 发送 JSON-RPC |
| `/tools` | GET | 列出工具 |
| `/tools/call` | POST | 调用工具 |
| `/health` | GET | 健康检查 |

---

## 配置兼容性

保持 JSON 配置文件格式完全兼容：

```json
{
  "gateway": { "host": "0.0.0.0", "port": 4298 },
  "pool": { "minConnections": 1, "maxConnections": 5 },
  "servers": [...],
  "mapping": {...}
}
```

---

## 错误处理规范

Go 错误处理采用显式错误返回机制，建议遵循以下规范：

### 错误类型定义

```go
// 定义错误类型，便于分类处理
var (
    ErrServerNotFound = errors.New("server not found")
    ErrConnectionPoolExhausted = errors.New("connection pool exhausted")
    ErrInvalidRequest = errors.New("invalid request")
    ErrTimeout = errors.New("operation timeout")
)

// 错误结构体，包含上下文信息
type PoolError struct {
    ServerName string
    Err        error
}

func (e *PoolError) Error() string {
    return fmt.Sprintf("pool error for %s: %v", e.ServerName, e.Err)
}
```

### 错误处理策略

| 场景 | 处理策略 |
|------|----------|
| 配置缺失/无效 | 返回错误并退出程序，避免带病运行 |
| MCP 服务器连接失败 | 重试 + 记录日志，继续服务其他服务器 |
| 单个请求超时 | 返回 JSON-RPC 错误，不影响其他请求 |
| 连接池耗尽 | 排队等待 + 超时控制，避免无限阻塞 |
| 未知错误 | 记录完整堆栈，返回通用错误信息 |

### 日志规范

使用 `log/slog` 进行结构化日志记录：

```go
// 错误日志：包含错误类型、上下文、堆栈
slog.Error("MCP request failed",
    "server", serverName,
    "method", method,
    "error", err,
)

// 警告日志：记录可恢复的问题
slog.Warn("Connection pool near capacity",
    "server", serverName,
    "active", active,
    "max", max,
)

// 信息日志：记录关键操作
slog.Info("Connection established",
    "server", serverName,
    "clientId", clientID,
)
```

---

## 构建与发布

### 开发构建

```bash
go build -o mcp-gateway ./cmd/gateway
```

### 跨平台构建

```bash
# macOS ARM64
GOOS=darwin GOARCH=arm64 go build -o dist/mcp-gateway-darwin-arm64 ./cmd/gateway

# macOS AMD64
GOOS=darwin GOARCH=amd64 go build -o dist/mcp-gateway-darwin-amd64 ./cmd/gateway

# Linux AMD64
GOOS=linux GOARCH=amd64 go build -o dist/mcp-gateway-linux-amd64 ./cmd/gateway

# Windows
GOOS=windows GOARCH=amd64 go build -o dist/mcp-gateway-windows-amd64.exe ./cmd/gateway
```

### Makefile 常用目标

```makefile
build: go build -o mcp-gateway ./cmd/gateway
test: go test -v ./...
lint: golangci-lint run
release: make build-all && make release-draft
```

---

## 安装方式对比

### 当前（Node.js）

```bash
npm install -g git+https://github.com/packy/mcp-gateway.git
```

### 迁移后（Go）

```bash
# 方式一: go install (推荐)
go install github.com/packy/mcp-gateway@latest

# 方式二: 下载预编译二进制
curl -L https://github.com/packy/mcp-gateway/releases/latest/download/mcp-gateway-darwin-arm64 -o /usr/local/bin/mcp-gateway
chmod +x /usr/local/bin/mcp-gateway

# 方式三: Homebrew
brew install packy/tap/mcp-gateway
```

---

## CLI 使用方法

### 命令行参数

```bash
mcp-gateway [选项]
```

| 参数 | 短选项 | 默认值 | 说明 |
|------|--------|--------|------|
| `--config <path>` | `-c` | 自动查找 | 指定配置文件路径 |
| `--host <addr>` | `-h` | `0.0.0.0` | 监听地址 |
| `--port <port>` | `-p` | `4298` | 监听端口 |
| `--stdio` | - | false | 以 stdio 模式运行（供 Claude Desktop 使用） |
| `--version` | `-v` | - | 显示版本号 |
| `--help` | - | - | 显示帮助信息 |

### 启动服务（HTTP/SSE 模式）

```bash
# 默认启动（自动查找配置）
mcp-gateway

# 指定配置路径
mcp-gateway --config /path/to/config.json

# 指定端口启动
mcp-gateway --port 8080

# 开发模式（查看详细日志）
mcp-gateway --log-level debug
```

**输出示例：**
```
2024/04/10 12:00:00 [gateway] MCP Gateway v1.0.0 starting...
2024/04/10 12:00:00 [gateway] Loading config from: /Users/packy/.config/mcp-gateway/config.json
2024/04/10 12:00:00 [pool] Initialized minimax with 3/3 connections
2024/04/10 12:00:00 [pool] Initialized zhipu with 2/3 connections
2024/04/10 12:00:00 [gateway] Listening on http://0.0.0.0:4298
2024/04/10 12:00:00 [gateway] SSE endpoint: http://0.0.0.0:4298/sse
2024/04/10 12:00:00 [gateway] REST endpoint: http://0.0.0.0:4298/tools/call
```

### Stdio 模式（Claude Desktop）

以 stdio 模式运行时，程序从标准输入读取 JSON-RPC 请求并向标准输出写入响应，适合 Claude Desktop 等仅支持 stdio 的客户端：

```bash
# 启动 stdio 模式
mcp-gateway --stdio

# 或指定配置
mcp-gateway --stdio --config /path/to/config.json
```

**Claude Desktop 配置示例（~/.claude.json）：**
```json
{
  "mcpServers": {
    "gateway": {
      "command": "mcp-gateway",
      "args": ["--stdio"]
    }
  }
}
```

### 配置路径优先级

程序按以下顺序查找配置文件：

1. **`--config` 参数**（最高优先级）
   ```bash
   mcp-gateway --config /custom/path/config.json
   ```

2. **`MCP_GATEWAY_CONFIG` 环境变量**
   ```bash
   export MCP_GATEWAY_CONFIG=/custom/path/config.json
   mcp-gateway
   ```

3. **`~/.config/mcp-gateway/config.json`**（macOS/Linux）
   ```bash
   mkdir -p ~/.config/mcp-gateway
   cp /path/to/config.json ~/.config/mcp-gateway/config.json
   ```

4. **`./config/servers.json`**（项目目录，开发模式）

### 健康检查

启动后可通过以下方式验证服务状态：

```bash
# HTTP 请求
curl http://localhost:4298/health

# 响应示例
{
  "status": "ok",
  "version": "1.0.0",
  "uptime": 3600,
  "sessions": 2,
  "pool": {
    "minimax": { "total": 3, "active": 1, "idle": 2 },
    "zhipu": { "total": 3, "active": 0, "idle": 3 }
  }
}
```

### 列出可用工具

```bash
curl http://localhost:4298/tools

# 响应示例
{
  "tools": [
    { "name": "minimax_web_search", "description": "...", "serverName": "minimax" },
    { "name": "minimax_understand_image", "description": "...", "serverName": "minimax" },
    { "name": "zhipu_analyze_image", "description": "...", "serverName": "zhipu" }
  ]
}
```

### 停止服务

```bash
# Ctrl+C 优雅关闭
# 或发送 SIGTERM
kill $(pgrep mcp-gateway)
```

---

## 后台运行

### 方式一：nohup（通用）

```bash
# 后台启动，输出日志到文件
nohup mcp-gateway > /var/log/mcp-gateway.log 2>&1 &

# 指定配置和端口
nohup mcp-gateway --config /path/to/config.json --port 4298 >> /var/log/mcp-gateway.log 2>&1 &

# 获取进程 PID
echo $!

# 查看进程是否运行
pgrep -f mcp-gateway

# 查看日志
tail -f /var/log/mcp-gateway.log
```

### 方式二：systemd（Linux）

**创建服务文件：**
```bash
sudo tee /etc/systemd/system/mcp-gateway.service << 'EOF'
[Unit]
Description=MCP Gateway
After=network.target

[Service]
Type=simple
User=<your-user>
WorkingDirectory=/path/to/config/directory
ExecStart=/usr/local/bin/mcp-gateway --config /path/to/config.json
Restart=always
RestartSec=5
Nice=5
LimitNOFILE=65536
StandardOutput=journal
StandardError=journal

[Install]
WantedBy=multi-user.target
EOF
```

**参数说明：**
- `Nice=5`：降低进程优先级，避免影响系统其他服务
- `LimitNOFILE=65536`：允许打开的文件描述符上限，确保高并发下不会耗尽

**管理服务：**
```bash
# 重新加载 systemd
sudo systemctl daemon-reload

# 启动服务
sudo systemctl start mcp-gateway

# 设置开机自启
sudo systemctl enable mcp-gateway

# 查看状态
sudo systemctl status mcp-gateway

# 查看日志
sudo journalctl -u mcp-gateway -f

# 重启服务
sudo systemctl restart mcp-gateway

# 停止服务
sudo systemctl stop mcp-gateway
```

### 方式三：launchd（macOS）

**创建 plist 文件：**
```bash
mkdir -p ~/Library/LaunchAgents
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
        <string>/Users/packy/.config/mcp-gateway/config.json</string>
    </array>
    <key>RunAtLoad</key>
    <true/>
    <key>KeepAlive</key>
    <true/>
    <key>StandardOutPath</key>
    <string>/var/log/mcp-gateway.log</string>
    <key>StandardErrorPath</key>
    <string>/var/log/mcp-gateway.error.log</string>
    <key>WorkingDirectory</key>
    <string>/Users/packy</string>
</dict>
</plist>
EOF
```

**管理服务：**
```bash
# 加载服务
launchctl load ~/Library/LaunchAgents/com.mcp-gateway.plist

# 卸载服务
launchctl unload ~/Library/LaunchAgents/com.mcp-gateway.plist

# 启动服务
launchctl start com.mcp-gateway

# 停止服务
launchctl stop com.mcp-gateway

# 查看状态
launchctl list | grep mcp-gateway
```

### 方式四：pm2（跨平台）

```bash
# 安装 pm2
npm install -g pm2

# 启动服务
pm2 start mcp-gateway -- --config /path/to/config.json

# 保存进程列表（开机自启）
pm2 save

# 设置开机自启
pm2 startup

# 查看状态
pm2 status

# 查看日志
pm2 logs mcp-gateway

# 重启服务
pm2 restart mcp-gateway

# 停止服务
pm2 stop mcp-gateway

# 删除进程
pm2 delete mcp-gateway
```

### 方式五：Docker（推荐）

**Dockerfile：**
```dockerfile
FROM golang:1.22-alpine AS builder

WORKDIR /build
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o mcp-gateway ./cmd/gateway

FROM alpine:3.19
RUN apk --no-cache add ca-certificates tzdata && \
    adduser -D -u 1000 appuser

WORKDIR /app
COPY --from=builder /build/mcp-gateway .
COPY config/servers.example.json /app/config.json

RUN chown -R appuser:appuser /app

USER appuser
EXPOSE 4298

HEALTHCHECK --interval=30s --timeout=10s --start-period=5s --retries=3 \
    CMD wget -qO- http://localhost:4298/health || exit 1

ENTRYPOINT ["./mcp-gateway"]
CMD ["--config", "/app/config.json"]
```

**安全加固说明：**
- 使用非 root 用户 `appuser` 运行，降低容器被攻陷后的风险
- `HEALTHCHECK` 让 Docker 定期检查服务健康状态
- `chown` 确保应用用户拥有配置目录的读取权限

**构建镜像：**
```bash
docker build -t mcp-gateway:latest .
```

**运行容器：**
```bash
# 基本运行
docker run -d \
  --name mcp-gateway \
  -p 4298:4298 \
  -v /path/to/config.json:/app/config.json \
  mcp-gateway:latest

# 带日志查看
docker run -d \
  --name mcp-gateway \
  -p 4298:4298 \
  -v /path/to/config.json:/app/config.json \
  -v $(pwd)/logs:/app/logs \
  mcp-gateway:latest

# 查看日志
docker logs -f mcp-gateway

# 进入容器调试
docker exec -it mcp-gateway sh

# 停止删除
docker stop mcp-gateway && docker rm mcp-gateway
```

### 方式六：docker-compose（推荐）

**docker-compose.yml：**
```yaml
services:
  mcp-gateway:
    image: mcp-gateway:latest
    container_name: mcp-gateway
    ports:
      - "4298:4298"
    volumes:
      - ./config.json:/app/config.json:ro
      - ./logs:/app/logs
    environment:
      - TZ=Asia/Shanghai
    restart: unless-stopped
    healthcheck:
      test: ["CMD", "wget", "-qO-", "http://localhost:4298/health"]
      interval: 30s
      timeout: 10s
      retries: 3
      start_period: 10s

  # 可选：前端控制面板
  # mcp-dashboard:
  #   image: mcp-dashboard:latest
  #   ports:
  #     - "4299:80"
  #   environment:
  #     - API_BASE=http://mcp-gateway:4298
  #   depends_on:
  #     - mcp-gateway

networks:
  default:
    name: mcp-network
```

**目录结构：**
```
mcp-gateway-deploy/
├── docker-compose.yml
├── config.json
└── logs/
```

**管理命令：**
```bash
# 启动服务（后台运行）
docker-compose up -d

# 查看状态
docker-compose ps

# 查看日志
docker-compose logs -f

# 查看特定服务日志
docker-compose logs -f mcp-gateway

# 重启服务
docker-compose restart

# 停止服务
docker-compose down

# 重新构建镜像
docker-compose build --no-cache

# 拉取最新镜像并重启
docker-compose pull && docker-compose up -d
```

### 进程管理常用命令

```bash
# 查看进程
ps aux | grep mcp-gateway

# 强制终止
kill -9 $(pgrep mcp-gateway)

# 查看端口占用
lsof -i :4298

# 测试服务是否响应
curl -s http://localhost:4298/health | jq .status
```

---

## 风险评估与对策

| 风险 | 概率 | 影响 | 对策 |
|------|------|------|------|
| MCP SDK 依赖需重新实现协议解析 | 中 | 高 | 参考现有 SDK 实现，纯 JSON 处理，提前验证协议兼容性 |
| 性能问题（Go vs Node.js） | 低 | 中 | 预留 2 周性能优化时间，进行基准测试对比 |
| 并发模型差异（协程 vs 事件循环） | 中 | 中 | 充分测试连接池场景，使用 sync.Pool 优化资源管理 |
| 现有用户配置迁移 | 低 | 低 | 保持配置格式完全兼容，提供迁移文档 |
| 第三方依赖兼容性 | 低 | 高 | 锁定依赖版本，使用 go.modreplace 备用方案 |
| 跨平台构建复杂性 | 中 | 中 | 使用 GitHub Actions Matrix 构建，测试各平台二进制 |
| 优雅关闭实现遗漏 | 中 | 中 | Phase 5 专项测试关闭流程，验证资源释放完整 |

---

## 验收标准

1. **功能对齐**: 所有现有 API 端点正常工作
2. **配置兼容**: 现有配置文件无需修改即可使用
3. **性能达标**: 相同负载下内存降低 50%+
4. **零依赖安装**: 单一二进制文件即可运行
5. **测试覆盖**: 核心模块覆盖率 > 80%

---

## 附录

### 参考资料

- [Go net/http 文档](https://pkg.go.dev/net/http)
- [Viper 配置库](https://github.com/spf13/viper)
- [SSE 规范](https://html.spec.whatwg.org/multipage/server-sent-events.html)
- [JSON-RPC 2.0 规范](https://www.jsonrpc.org/specification)

### 现有 TypeScript 实现参考

- `src/gateway/server.ts` - HTTP/SSE 服务器
- `src/gateway/pool.ts` - 连接池
- `src/mcp/client.ts` - MCP 客户端
- `src/config/loader.ts` - 配置加载
# Pool 初始化握手失败问题

> MCP 连接池初始化阶段握手（Initialize）失败/超时，导致 npx/uvx 类慢启动服务全部无法连接

**所属目录**：`docs/tasks/`
**文档状态**：评审中
**当前版本**：v0.1
**发布日期**：2026-04-18
**最后更新**：2026-04-18

---

## 1. 问题概述

| 属性 | 值 |
|------|-----|
| **问题编号** | ISSUE-001 |
| **严重程度** | P0（核心功能不可用） |
| **状态** | 🔄 修复中（代码已修改，待验证） |
| **关联里程碑** | M5: 测试与验证 |
| **关联分支** | `fix/mcp-initialize-handshake` |
| **关联文件** | `src/pool/pool.go` |
| **发现日期** | 2026-04-18 |

### 现象

Pool 初始化阶段，各 MCP 服务子进程启动后 MCP 握手（Initialize）失败或超时，导致连接实际未建立。后续 `collectTools` 时全部返回 "not connected"。

**影响范围**：

| 服务 | 状态 | 工具数 | 启动方式 |
|------|------|--------|----------|
| pencil | ✅ 正常 | 13 | 原生二进制 |
| playwright | ❌ 失败 | 0（应 21） | npx |
| lark | ❌ 失败 | 0（应 5） | npx |
| searxng | ❌ 失败 | 0 | uvx |
| minimax | ❌ 失败 | 0 | npx |
| zai-mcp-server | ❌ 失败 | 0 | npx |

**历史佐证**：日志显示曾成功加载 39 个工具（pencil + playwright + lark），说明服务本身没有问题，问题出在连接初始化流程。

---

## 2. 根因分析

### 2.1 根因 1：Connect() 中硬编码 100ms 等待

**位置**：`src/pool/pool.go` 第 79 行

```go
// 原始代码
c.process = process
time.Sleep(100 * time.Millisecond)  // ← 硬编码 100ms
go c.readResponses()                 // ← 读取协程在 sleep 之后才启动
```

**问题**：进程启动后只等 100ms 就标记为 connected。对于 npx/uvx 类服务（playwright、lark、searxng、minimax、zai-mcp-server），启动需要先下载/安装包，100ms 远不够。而且 `readResponses` 协程在 sleep 之后才启动，导致子进程的早期输出可能丢失。

**影响**：连接被标记为 "已连接" 但实际子进程尚未就绪，后续 Initialize 请求无法得到正确响应。

### 2.2 根因 2：Initialize() 握手超时导致连接被丢弃

**位置**：`Pool.Initialize()` 方法

**流程**：
1. `Pool.Initialize()` 先调 `Connect()` 再调 `Initialize()`
2. Initialize 有 30s 超时
3. 如果子进程还没启动完成，发送的 initialize 请求无法得到响应
4. 30s 后超时，连接直接被跳过（`continue`），不加入 pool

**影响**：超时的连接不会重试，直接被丢弃。即使子进程后续启动完成，也无法再被连接。

### 2.3 根因 3：acquire() 按需创建连接时缺少 Initialize() 握手

**位置**：`Pool.acquire()` 方法

```go
// 原始代码（简化）
if err := client.Connect(ctx); err != nil {
    // 错误处理...
} else {
    pool = append(pool, client)  // ← 直接加入池，没有 Initialize()
}
```

**问题**：`acquire()` 在连接不足时创建新连接，但只调了 `Connect()` 没调 `Initialize()`。新连接无法正常使用 MCP 协议。

**影响**：即使在运行时动态扩展连接池，新连接也无法正常工作。

### 2.4 根因关系图

```
Connect() 硬编码 100ms 等待
       │
       ▼
子进程未就绪就被标记为 "connected"
       │
       ▼
Initialize() 发送握手请求
       │
       ▼
子进程无法响应 → 30s 超时
       │
       ▼
连接被跳过（continue），不加入 pool
       │
       ▼
collectTools() 时所有服务返回 "not connected"
```

---

## 3. 已完成的修复

代码已修改（在工作区，未提交），修改内容如下：

### 3.1 修复根因 1：移除硬编码 sleep，提前启动 readResponses

```diff
  c.process = process
- time.Sleep(100 * time.Millisecond)
+ 
+ go c.readResponses()

  c.connected = true
```

**变更说明**：
- 移除 `time.Sleep(100ms)` —— 不再用固定等待来"赌"子进程就绪
- 将 `readResponses` 协程启动提前到标记 connected 之前 —— 确保子进程输出从第一行开始就被读取

### 3.2 修复根因 3：acquire() 中增加 Initialize() 调用

```diff
  if err := client.Connect(ctx); err != nil {
      // 错误处理...
+ } else if err := client.Initialize(); err != nil {
+     slog.Warn("Failed to initialize new connection",
+         "server", serverName,
+         "error", err,
+     )
+     client.Disconnect()
      p.mu.Unlock()
  } else {
      pool = append(pool, client)
```

**变更说明**：
- 在 `acquire()` 中增加了 `Initialize()` 调用
- 如果握手失败，断开连接并释放锁，避免无效连接进入池

### 3.3 修复状态

| 修改 | 文件 | 行号 | 状态 |
|------|------|------|------|
| 移除 time.Sleep | `src/pool/pool.go` | L79 | ✅ 已修改 |
| 提前启动 readResponses | `src/pool/pool.go` | L79 | ✅ 已修改 |
| acquire() 增加 Initialize | `src/pool/pool.go` | L535-543 | ✅ 已修改 |
| 提交到版本控制 | — | — | ⏳ 未提交 |
| 验证通过 | — | — | ⏳ 未验证 |

---

## 4. 待完成的验证步骤

### 4.1 单元测试验证

- [ ] `go test ./src/pool/...` 通过
- [ ] `go test ./...` 全量通过
- [ ] `golangci-lint run` 无新增警告

### 4.2 集成测试验证

- [ ] 启动 gateway，验证所有服务连接成功：

```bash
go run ./cmd/gateway
# 观察日志中所有服务的 Initialize 结果
```

- [ ] 调用 `/tools` 接口，验证工具数符合预期：

```bash
curl http://localhost:4298/tools | jq '. | length'
# 预期：远大于 13（pencil 独占时）
```

- [ ] 逐个验证 npx/uvx 类服务的工具加载：

| 服务 | 预期工具数 | 验证方式 |
|------|-----------|----------|
| pencil | 13 | `/tools` 响应中检查 |
| playwright | 21 | `/tools` 响应中检查 |
| lark | 5 | `/tools` 响应中检查 |
| searxng | ≥1 | `/tools` 响应中检查 |
| minimax | ≥1 | `/tools` 响应中检查 |
| zai-mcp-server | ≥1 | `/tools` 响应中检查 |

### 4.3 OpenCode 集成验证

- [ ] `opencode mcp list` 显示 gateway 为 `●` 状态
- [ ] `opencode mcp call gateway/list-services` 返回所有服务
- [ ] 工具调用（如 `playwright_browser_navigate`）正常工作

### 4.4 CI 验证

- [ ] 推送到远端后 GitHub Actions CI 通过
- [ ] 多平台测试（Ubuntu、macOS、Windows）通过

---

## 5. 建议的后续改进

### 5.1 短期（本次修复范围内）

| 优先级 | 改进项 | 说明 |
|--------|--------|------|
| P0 | 提交修复代码到 `fix/mcp-initialize-handshake` 分支 | 当前修改在工作区未提交 |
| P0 | 完成上述验证步骤 | 确保修复有效 |
| P1 | 合并到 main 并发布 | 验证通过后 |

### 5.2 中期（架构改进）

| 优先级 | 改进项 | 说明 |
|--------|--------|------|
| P2 | 健康检查机制 | Connect() 后增加进程就绪探测，替代固定等待 |
| P2 | 连接重试策略 | Initialize 失败后支持有限次重试，而非直接丢弃 |
| P2 | 启动超时可配置 | 不同服务的启动超时应该可配置（如 npx 服务给 60s，原生二进制给 5s） |
| P3 | 连接池预热 | 支持异步预热连接池，不阻塞启动流程 |

### 5.3 长期（健壮性提升）

| 优先级 | 改进项 | 说明 |
|--------|--------|------|
| P3 | 进程生命周期管理 | 监控子进程状态，自动重连 |
| P3 | 连接池指标 | 暴露连接池状态指标（活跃连接数、等待队列长度等） |
| P3 | 优雅降级 | 部分服务不可用时，其他服务仍可正常工作 |

---

## 6. 相关文档

| 文档 | 路径 |
|------|------|
| 项目状态卡 | [docs/status.md](../status.md) |
| M5 Sprint 6 任务 | [docs/tasks/M5-Sprint-6-测试与验证.md](./M5-Sprint-6-测试与验证.md) |
| 架构文档 | [docs/product/architecture.md](../product/architecture.md) |
| 里程碑计划 | [docs/product/milestone.md](../product/milestone.md) |

---

## 修订记录

| 版本 | 日期 | 修订内容 | 修订人 |
|------|------|----------|--------|
| v0.1 | 2026-04-18 | 初始版本：记录根因分析、已完成修复、待验证步骤 | PM Agent |

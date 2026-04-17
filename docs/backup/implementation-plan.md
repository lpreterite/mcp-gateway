# MCP Gateway 实施计划

## 任务清单

### 已完成

| 任务 | 状态 | 文件 |
|------|------|------|
| 创建技术文档 | ✅ 完成 | `docs/architecture.md` |
| 重构项目结构 | ✅ 完成 | `src/gateway/`, `src/mcp/`, `src/config/` |
| 实现连接池管理器 | ✅ 完成 | `src/gateway/pool.ts` |
| 实现工具注册表 | ✅ 完成 | `src/gateway/registry.ts` |
| 实现工具映射器 | ✅ 完成 | `src/gateway/mapper.ts` |
| 实现 HTTP/SSE 服务器 | ✅ 完成 | `src/gateway/server.ts` |
| 实现配置加载器 | ✅ 完成 | `src/config/loader.ts` |
| 更新配置文件 | ✅ 完成 | `config/servers.json` |
| 更新依赖 | ✅ 完成 | `package.json` |
| 更新 README | ✅ 完成 | `README.md` |

---

## 里程碑

### Milestone 1: 基础架构 ✅
- [x] 项目结构重构
- [x] 连接池管理器核心实现
- [x] HTTP/SSE 传输层

### Milestone 2: 工具链 ✅
- [x] 工具注册表
- [x] 工具映射器
- [x] 配置加载与验证

### Milestone 3: 集成测试 ✅
- [x] TypeScript 编译通过
- [x] 依赖安装完成
- [ ] 运行时测试（待用户验证）

---

## 文件清单

### 新增文件

| 文件 | 说明 |
|------|------|
| `src/gateway/index.ts` | HTTP Server 入口，启动 Express + SSE |
| `src/gateway/server.ts` | MCP Gateway Server，封装 MCP SDK Server |
| `src/gateway/pool.ts` | **核心** - MCP 连接池管理器，实现 acquire/release/execute |
| `src/gateway/registry.ts` | 工具注册表 |
| `src/gateway/mapper.ts` | 工具名映射 |
| `src/mcp/client.ts` | MCP 客户端封装，封装 StdioClientTransport |
| `src/mcp/types.ts` | 类型定义 |
| `src/config/loader.ts` | 配置文件加载器 |
| `src/test/direct-connection-test.ts` | MCP Server 直连测试脚本 |
| `src/test/pool-test.ts` | Gateway 连接池测试脚本 |
| `docs/architecture.md` | 架构设计文档（中文） |
| `docs/implementation-plan.md` | 实施计划文档（本文档） |

### 修改文件

| 文件 | 操作 | 说明 |
|------|------|------|
| `config/servers.json` | 修改 | 添加 gateway、pool 配置段 |
| `package.json` | 修改 | 添加 `express`, `cors` 依赖 |
| `README.md` | 修改 | 更新架构说明和快速开始指南 |

### 保留文件

| 文件 | 说明 |
|------|------|
| `src/index.ts` | 旧版入口（保留但未使用） |
| `src/test-connection.ts` | 调试工具，可直连 MCP server |

### 废弃文件

| 文件 | 说明 |
|------|------|
| `dist/` | 旧编译产物（已重新编译） |

---

## 快速开始

### 启动 Gateway

```bash
# 开发模式
npm run gateway

# 生产模式
npm run build
npm start
```

### 测试端点

```bash
# 健康检查
curl http://localhost:4298/health

# 列出工具
curl http://localhost:4298/tools

# 调用工具
curl -X POST http://localhost:4298/tools/call \
  -H "Content-Type: application/json" \
  -d '{"name":"minimax_understand_image","arguments":{"image_source":"..."}}'
```

---

## 验证步骤

### 功能验证

1. **启动服务**
   ```bash
   npm run gateway
   ```
   预期：看到 `MCP Gateway listening on http://0.0.0.0:4298`

2. **检查健康状态**
   ```bash
   curl http://localhost:4298/health
   ```
   预期：返回 `{ "status": "ok", "sessions": 0, "pool": {...} }`

3. **列出工具**
   ```bash
   curl http://localhost:4298/tools
   ```
   预期：返回已注册的工具列表

### 性能验证

1. **监控进程数**
   ```bash
   ps aux | grep -E "minimax|zai|mcp" | grep -v grep | wc -l
   ```
   预期：稳定在配置的数量（不随请求增加）

2. **多并发测试**
   ```bash
   # 并发 10 个请求
   for i in {1..10}; do
     curl -X POST http://localhost:4298/tools/call \
       -H "Content-Type: application/json" \
       -d '{"name":"minimax_web_search","arguments":{"query":"test"}}' &
   done
   wait
   ```
   预期：所有请求完成，进程数不增加

---

## 测试脚本

提供了两个测试脚本用于验证系统功能：

### 1. 直连测试 - 验证 MCP Server 本身

```bash
npx tsx src/test/direct-connection-test.ts
```

此脚本直接启动各个 MCP server 进程，验证其是否正常工作：
- 启动 MCP server 进程
- 发送 MCP 协议初始化请求
- 获取工具列表
- 执行工具调用测试

**前置条件：**
- minimax 需要设置 `MINIMAX_API_KEY` 环境变量
- searxng 需要设置 `SEARXNG_URL` 环境变量

### 2. 连接池测试 - 验证 Gateway 功能

```bash
# 先启动 Gateway
npm run gateway

# 在另一个终端运行测试
npx tsx src/test/pool-test.ts
```

此脚本测试 Gateway 的完整功能：
- **测试 1**: MCP Server 连接测试 - 验证 Gateway 能连接到各个 MCP server
- **测试 2**: 工具调用测试 - 验证工具调用是否正常工作
- **测试 3**: 连接池进程数量限制测试 - 验证不会创建过多进程
- **测试 4**: 并发请求测试 - 验证并发请求时连接被正确复用
- **测试 5**: 压力测试 - 短时间内大量请求，验证稳定性

**关键验证指标：**
- 进程数不会随并发请求增加而爆炸
- 连接池统计显示正确的活动/空闲连接数
- 所有请求都能得到响应

---

## 已知问题与限制

1. **连接池大小**：每个 server 默认最多 5 个连接，超出时请求会排队等待
2. **空闲清理**：目前未实现空闲连接自动清理（可在 future 版本中添加）
3. **故障恢复**：连接失败时需要手动重启 gateway（可在 future 版本中添加自动重连）

---

## 未来优化方向

1. **连接池动态调整**：根据负载自动调整 `poolSize`
2. **健康检查与自动重连**：检测到连接失败时自动重建
3. **指标监控**：添加 Prometheus 指标导出
4. **认证授权**：添加 API Key 或 OAuth 认证
5. **TLS 支持**：启用 HTTPS 传输加密

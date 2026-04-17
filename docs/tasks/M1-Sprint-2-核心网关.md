# Sprint 2: 核心网关

> **所属里程碑**: M1: Go 核心功能
> **文档来源**: [milestone.md](../../product/milestone.md)
> **版本**: 1.0
> **更新日期**: 2026-04-17

---

## Sprint 信息

| 属性 | 值 |
|------|-----|
| **里程碑** | M1: Go 核心功能 |
| **Sprint 编号** | 2 |
| **状态** | 🔄 进行中 |
| **开始日期** | 2026-04-__ |
| **结束日期** | 2026-04-__ |

---

## 目标描述

实现 HTTP/SSE 服务器和连接池。构建核心网关功能：

- HTTP 服务器端点（/sse, /messages, /health, /tools, /tools/call）
- MCP 连接池管理（acquire/release/execute）
- MCP 客户端（通过 os/exec 启动子进程，stdio 通信）
- 优雅关闭机制

---

## 任务清单

- [ ] HTTP 服务器（`/sse`, `/messages`, `/health`, `/tools`, `/tools/call`）
- [ ] 连接池实现（`acquire`/`release`/`execute`）
- [ ] MCP 客户端（`os/exec` 启动子进程，stdio 通信）
- [ ] 优雅关闭实现

---

## 交付物列表

| 交付物 | 描述 | 状态 |
|--------|------|------|
| HTTP 服务器正常运行 | 所有端点可访问 | ⏳ |
| 连接池功能完整 | acquire/release/execute 正常工作 | ⏳ |
| 与现有 MCP 服务器通信正常 | 可以启动并通信 | ⏳ |

---

## 验收标准

- [ ] HTTP 服务器在端口上正常监听
- [ ] /health 端点返回 200 OK
- [ ] /sse 端点支持 SSE 连接
- [ ] /tools 端点返回工具列表
- [ ] /tools/call 端点可以调用工具
- [ ] 连接池可以正确管理连接
- [ ] MCP 客户端可以启动子进程并通过 stdio 通信
- [ ] 服务可以优雅关闭（处理 SIGTERM）

---

## 前置依赖

- Sprint 1: Go 基础设施（必须完成）

---

## 技术考量

- HTTP/SSE 实现参考现有 `src/gateway/server.go`
- 连接池实现参考现有 `src/pool/pool.go`
- MCP 客户端需要处理进程生命周期管理

---

## 备注

- 这是核心功能 Sprint，与现有已实现功能重叠
- 需要确保与现有代码兼容

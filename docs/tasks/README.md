# 任务总览

> **文档来源**: [milestone.md](../product/milestone.md)
> **版本**: 1.0
> **更新日期**: 2026-04-17
> **Author**: PO Agent

---

## 里程碑总览

| 里程碑 | 内容 | 优先级 | Sprint | 状态 |
|--------|------|--------|--------|------|
| **M1: Go 核心功能** | 基础架构、连接池、HTTP/SSE、工具注册表 | P0 | Sprint 1-2 | 🔄 进行中 |
| **M2: 工具链完善** | 工具映射、配置管理、日志、优雅关闭 | P1 | Sprint 3 | ⏳ 待开始 |
| **M3: Stdio Bridge** | 独立进程桥接器，支持 Claude Desktop | P1 | Sprint 4 | ⏳ 待开始 |
| **M4: 服务管理** | 双轨制服务架构、跨平台安装 | P1 | Sprint 5 | ⏳ 待开始 |
| **M5: 测试与验证** | 单元测试、集成测试、OpenCode 验证 | P1 | Sprint 6 | ⏳ 待开始 |
| **M6: 发布准备** | 文档更新、跨平台构建、发布流程 | P2 | Sprint 7 | ⏳ 待开始 |

---

## 当前 Sprint

**M1: Go 核心功能 - Sprint 2: 核心网关**

当前正在进行的 Sprint，请查看 [M1-Sprint-2-核心网关](./M1-Sprint-2-核心网关.md)

---

## 任务清单快速链接

### M1: Go 核心功能

| Sprint | 文档 | 状态 |
|--------|------|------|
| Sprint 1: Go 基础设施 | [M1-Sprint-1-Go-基础设施.md](./M1-Sprint-1-Go-基础设施.md) | 🔄 进行中 |
| Sprint 2: 核心网关 | [M1-Sprint-2-核心网关.md](./M1-Sprint-2-核心网关.md) | 🔄 进行中 |

### M2: 工具链完善

| Sprint | 文档 | 状态 |
|--------|------|------|
| Sprint 3: 工具链完善 | [M2-Sprint-3-工具链完善.md](./M2-Sprint-3-工具链完善.md) | ⏳ 待开始 |

### M3: Stdio Bridge

| Sprint | 文档 | 状态 |
|--------|------|------|
| Sprint 4: Stdio Bridge | [M3-Sprint-4-Stdio-Bridge.md](./M3-Sprint-4-Stdio-Bridge.md) | ⏳ 待开始 |

### M4: 服务管理

| Sprint | 文档 | 状态 |
|--------|------|------|
| Sprint 5: 服务管理 | [M4-Sprint-5-服务管理.md](./M4-Sprint-5-服务管理.md) | ⏳ 待开始 |

### M5: 测试与验证

| Sprint | 文档 | 状态 |
|--------|------|------|
| Sprint 6: 测试与验证 | [M5-Sprint-6-测试与验证.md](./M5-Sprint-6-测试与验证.md) | ⏳ 待开始 |

### M6: 发布准备

| Sprint | 文档 | 状态 |
|--------|------|------|
| Sprint 7: 发布准备 | [M6-Sprint-7-发布准备.md](./M6-Sprint-7-发布准备.md) | ⏳ 待开始 |

---

## 已完成功能

- ✅ MCP 连接池（`src/pool/pool.go`）
- ✅ HTTP/SSE 传输层（`src/gateway/server.go`）
- ✅ 双轨制服务架构（`src/gwservice/`）
- ✅ 工具注册与映射（`src/registry/`）

## 待解决问题

- ⏳ playwright/lark 的 npx 启动问题（broken pipe）→ **根因已定位为 Pool 初始化握手失败，详见 [ISSUE-001](./ISSUE-001-pool-initialize-handshake.md)**
- ⏳ OpenCode MCP 工具调用验证
- ⏳ architecture.md 文档需要更新为 Go 版本
- ⏳ Stdio Bridge 未实现

---

## 问题追踪

| 编号 | 问题 | 优先级 | 状态 | 文档 |
|------|------|--------|------|------|
| ISSUE-001 | Pool 初始化握手失败 | P0 | 🔄 修复中 | [ISSUE-001-pool-initialize-handshake.md](./ISSUE-001-pool-initialize-handshake.md) |

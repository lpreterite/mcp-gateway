# Sprint 4: Stdio Bridge

> **所属里程碑**: M3: Stdio Bridge
> **文档来源**: [milestone.md](../../product/milestone.md)
> **版本**: 1.0
> **更新日期**: 2026-04-17

---

## Sprint 信息

| 属性 | 值 |
|------|-----|
| **里程碑** | M3: Stdio Bridge |
| **Sprint 编号** | 4 |
| **状态** | ⏳ 待开始 |
| **开始日期** | 2026-04-__ |
| **结束日期** | 2026-04-__ |

---

## 目标描述

支持 Claude Desktop 的 stdio 模式。实现独立进程桥接器：

- stdio 输入输出监听
- 桥接 stdio 协议与 HTTP/SSE 内部通信
- 独立进程模式切换（`--stdio` 参数）

---

## 任务清单

- [ ] 实现 stdio 输入输出监听
- [ ] 桥接 stdio 协议与 HTTP/SSE 内部通信
- [ ] 独立进程模式切换（`--stdio` 参数）

---

## 交付物列表

| 交付物 | 描述 | 状态 |
|--------|------|------|
| 可作为独立进程运行 | 支持 --stdio 参数启动 | ⏳ |
| 支持 Claude Desktop 连接 | Claude Desktop 可以通过 stdio 连接 | ⏳ |

---

## 验收标准

- [ ] `--stdio` 参数可以切换到 stdio 模式
- [ ] stdio 模式不启动 HTTP 服务器
- [ ] JSON-RPC 消息通过 stdio 正确收发
- - [ ] HTTP/SSE 内部通信与 stdio 之间正确桥接
- [ ] 可以作为 Claude Desktop MCP 服务器配置使用

---

## 前置依赖

- Sprint 2: 核心网关（必须完成）

---

## 技术考量

- stdio 模式需要禁用 HTTP 服务器
- 需要处理 Claude Desktop 的 MCP 协议
- 桥接逻辑需要处理协议转换

---

## 备注

- Stdio Bridge 是 Claude Desktop 支持的关键
- 这是里程碑 3 的核心功能
- 需要与 Claude Desktop 进行集成测试

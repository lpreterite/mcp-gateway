# MCP Gateway 项目状态卡

> **更新时间**: 2026-04-17
> **维护者**: PO Agent

---

## 1. 项目概览

| 项目 | 信息 |
|------|------|
| **项目名称** | MCP Gateway |
| **项目描述** | MCP 统一网关 - 连接多个 MCP 服务器的统一网关，支持 HTTP/SSE 和 stdio 两种连接方式 |
| **当前版本** | `v1.2.1` |
| **开发状态** | 🚧 Development |
| **Git 分支** | `fix/mcp-initialize-handshake` |
| **许可证** | MIT |

### 核心技术栈

| 组件 | 技术 |
|------|------|
| 语言 | Go 1.26+ |
| HTTP 框架 | 标准库 `net/http` |
| 配置管理 | Viper |
| 日志 | `log/slog` |
| 服务管理 | launchd (macOS) / systemd (Linux) |

---

## 2. 里程碑进度

| 里程碑 | 内容 | Sprint | 状态 | 进度 |
|--------|------|--------|------|------|
| **M1** | Go 核心功能 | Sprint 1-2 | 🔄 进行中 | ████████░░ 80% |
| **M2** | 工具链完善 | Sprint 3 | ⏳ 待开始 | ░░░░░░░░░░ 0% |
| **M3** | Stdio Bridge | Sprint 4 | ⏳ 待开始 | ░░░░░░░░░░ 0% |
| **M4** | 服务管理 | Sprint 5 | ⏳ 待开始 | ░░░░░░░░░░ 0% |
| **M5** | 测试与验证 | Sprint 6 | ⏳ 待开始 | ░░░░░░░░░░ 0% |
| **M6** | 发布准备 | Sprint 7 | ⏳ 待开始 | ░░░░░░░░░░ 0% |

### 里程碑详情

#### M1: Go 核心功能 (🔄 进行中 - 80%)
- [x] Go 项目框架初始化
- [x] 配置管理 (Viper)
- [x] MCP 连接池
- [x] HTTP/SSE 传输层
- [x] 工具注册与映射
- [ ] 优雅关闭完善

#### M2: 工具链完善 (⏳ 待开始)
- [ ] 工具映射规则完善
- [ ] 配置格式兼容
- [ ] 结构化日志完善
- [ ] 优雅关闭

#### M3: Stdio Bridge (⏳ 待开始)
- [ ] stdio 输入输出监听
- [ ] 桥接 stdio 协议与 HTTP/SSE
- [ ] `--stdio` 参数模式

#### M4: 服务管理 (⏳ 待开始)
- [ ] ServiceFacade 统一命令入口
- [ ] macOS launchd 适配
- [ ] Linux systemd 适配
- [ ] 分层状态探测与诊断

#### M5: 测试与验证 (⏳ 待开始)
- [ ] 单元测试覆盖 > 80%
- [ ] 集成测试
- [ ] OpenCode MCP 验证

#### M6: 发布准备 (⏳ 待开始)
- [ ] GitHub Actions CI/CD
- [ ] 多平台构建
- [ ] 发布流程文档

---

## 3. Sprint 进度

### 当前 Sprint

| Sprint | 名称 | 状态 | 进度 |
|--------|------|------|------|
| **M1-Sprint-2** | 核心网关 | 🔄 进行中 | ████████░░ 80% |

**Sprint 目标**: 实现 HTTP/SSE 服务器和连接池

**进行中任务**:
- [x] HTTP 服务器 (`/sse`, `/messages`, `/health`, `/tools`, `/tools/call`)
- [x] 连接池实现 (`acquire`/`release`/`execute`)
- [x] MCP 客户端 (`os/exec` 启动子进程，stdio 通信)
- [ ] 优雅关闭实现

---

## 4. 已完成功能

### 核心功能

| 功能 | 模块 | 状态 | 文件 |
|------|------|------|------|
| MCP 连接池 | pool | ✅ | `src/pool/pool.go` |
| HTTP/SSE 服务器 | gateway | ✅ | `src/gateway/server.go` |
| 工具注册与映射 | registry | ✅ | `src/registry/` |
| 双轨制服务架构 | gwservice | ✅ | `src/gwservice/` |
| 配置管理 | config | ✅ | `src/config/` |

### API 端点

| 端点 | 方法 | 状态 |
|------|------|------|
| `/health` | GET | ✅ |
| `/tools` | GET | ✅ |
| `/sse` | GET | ✅ |
| `/messages` | POST | ✅ |
| `/tools/call` | POST | ✅ |

---

## 5. 待解决问题

| # | 问题 | 优先级 | 状态 | 关联 |
|---|------|--------|------|------|
| 1 | playwright/lark 的 npx 启动问题 (broken pipe) | P1 | ⏳ | M5 |
| 2 | OpenCode MCP 工具调用验证 | P1 | ⏳ | M5 |
| 3 | architecture.md 文档需更新为 Go 版本 | P2 | ⏳ | M6 |
| 4 | Stdio Bridge 未实现 | P1 | ⏳ | M3 |
| 5 | MCP 初始化握手问题修复 | P0 | 🔄 | 当前分支 |

---

## 6. 快速链接

### 文档

| 文档 | 路径 |
|------|------|
| README | [README.md](../README.md) |
| 产品需求 | [PRD.md](./product/PRD.md) |
| 里程碑计划 | [milestone.md](./product/milestone.md) |
| 架构文档 | [architecture.md](./product/architecture.md) |
| API 文档 | [api.md](./product/api.md) |
| CLI 文档 | [cli.md](./product/cli.md) |
| 部署文档 | [deployment.md](./product/deployment.md) |

### 任务

| Sprint | 文档 |
|--------|------|
| M1-Sprint-1: Go 基础设施 | [M1-Sprint-1-Go-基础设施.md](./tasks/M1-Sprint-1-Go-基础设施.md) |
| M1-Sprint-2: 核心网关 | [M1-Sprint-2-核心网关.md](./tasks/M1-Sprint-2-核心网关.md) |
| M2-Sprint-3: 工具链完善 | [M2-Sprint-3-工具链完善.md](./tasks/M2-Sprint-3-工具链完善.md) |
| M3-Sprint-4: Stdio Bridge | [M3-Sprint-4-Stdio-Bridge.md](./tasks/M3-Sprint-4-Stdio-Bridge.md) |
| M4-Sprint-5: 服务管理 | [M4-Sprint-5-服务管理.md](./tasks/M4-Sprint-5-服务管理.md) |
| M5-Sprint-6: 测试与验证 | [M5-Sprint-6-测试与验证.md](./tasks/M5-Sprint-6-测试与验证.md) |
| M6-Sprint-7: 发布准备 | [M6-Sprint-7-发布准备.md](./tasks/M6-Sprint-7-发布准备.md) |

### 外部链接

| 链接 | URL |
|------|-----|
| GitHub 仓库 | https://github.com/lpreterite/mcp-gateway |
| CI/CD 构建 | https://github.com/lpreterite/mcp-gateway/actions |
| Homebrew 安装 | `brew install lpreterite/tap/mcp-gateway` |

---

## 更新日志

| 日期 | 版本 | 更新内容 |
|------|------|----------|
| 2026-04-17 | v1.2.1 | 初始化状态卡文档 |

---

> **提示**: 此文档会随项目进展定期更新。请查看 [milestone.md](./product/milestone.md) 了解详细规划。

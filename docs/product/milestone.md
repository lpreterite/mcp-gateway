# 里程碑计划

> **文档来源**: [PRD.md](../PRD.md)
> **版本**: 1.0
> **更新日期**: 2026-04-17
> **Author**: PO Agent

---

## 6. 里程碑计划

### 6.1 当前状态

| 分支 | 说明 |
|------|------|
| `fix/mcp-initialize-handshake` | 正在修复 MCP 初始化握手问题 |

**已实现功能**：
- ✅ MCP 连接池（`src/pool/pool.go`）
- ✅ HTTP/SSE 传输层（`src/gateway/server.go`）
- ✅ 双轨制服务架构（`src/gwservice/`）
- ✅ 工具注册与映射（`src/registry/`）

**待解决问题**：
1. ⏳ playwright/lark 的 npx 启动问题（broken pipe）
2. ⏳ OpenCode MCP 工具调用验证
3. ⏳ architecture.md 文档需要更新为 Go 版本
4. ⏳ Stdio Bridge 未实现

### 6.2 里程碑规划

| 里程碑 | 内容 | 优先级 | 目标时间 |
|--------|------|--------|----------|
| **M1: Go 核心功能** | 基础架构、连接池、HTTP/SSE、工具注册表 | P0 | Sprint 1-2 |
| **M2: 工具链完善** | 工具映射、配置管理、日志、优雅关闭 | P1 | Sprint 3 |
| **M3: Stdio Bridge** | 独立进程桥接器，支持 Claude Desktop | P1 | Sprint 4 |
| **M4: 服务管理** | 双轨制服务架构、跨平台安装 | P1 | Sprint 5 |
| **M5: 测试与验证** | 单元测试、集成测试、OpenCode 验证 | P1 | Sprint 6 |
| **M6: 发布准备** | 文档更新、跨平台构建、发布流程 | P2 | Sprint 7 |

### 6.3 Sprint 详细规划

#### Sprint 1: Go 基础设施
**目标**：建立 Go 项目框架，配置管理，基础结构

**任务**：
- [ ] 初始化 Go 模块 (`go mod init`)
- [ ] 配置 `viper` 加载 JSON 配置文件
- [ ] 实现配置结构体与验证
- [ ] 实现日志框架（标准库 `log/slog`）
- [ ] 创建基础项目结构和 Makefile

**交付物**：
- 可运行的 `go build` 基础项目
- 配置加载验证通过
- 开发构建脚本

#### Sprint 2: 核心网关
**目标**：实现 HTTP/SSE 服务器和连接池

**任务**：
- [ ] HTTP 服务器（`/sse`, `/messages`, `/health`, `/tools`, `/tools/call`）
- [ ] 连接池实现（`acquire`/`release`/`execute`）
- [ ] MCP 客户端（`os/exec` 启动子进程，stdio 通信）
- [ ] 优雅关闭实现

**交付物**：
- HTTP 服务器正常运行
- 连接池功能完整
- 与现有 MCP 服务器通信正常

#### Sprint 3: 工具链完善
**目标**：实现工具注册表和名称映射

**任务**：
- [ ] 工具注册表（集中管理，按名称查找）
- [ ] 工具名映射（前缀映射、剥离、过滤）
- [ ] 配置文件格式兼容
- [ ] 结构化日志完善

**交付物**：
- 工具列表 API 正常工作
- 映射规则生效
- 日志可追踪

#### Sprint 4: Stdio Bridge
**目标**：支持 Claude Desktop 的 stdio 模式

**任务**：
- [ ] 实现 stdio 输入输出监听
- [ ] 桥接 stdio 协议与 HTTP/SSE 内部通信
- [ ] 独立进程模式切换（`--stdio` 参数）

**交付物**：
- 可作为独立进程运行
- 支持 Claude Desktop 连接

#### Sprint 5: 服务管理
**目标**：实现双轨制服务架构

**任务**：
- [ ] `ServiceFacade` 统一命令入口
- [ ] macOS `PlatformAdapter`（launchd 适配）
- [ ] Linux `PlatformAdapter`（systemd 适配）
- [ ] 分层状态探测与诊断

**交付物**：
- `service install/start/stop/restart/status` 命令正常
- 分层诊断输出
- 平台自愈能力

#### Sprint 6: 测试与验证
**目标**：功能验证和性能优化

**任务**：
- [ ] 单元测试覆盖（> 80%）
- [ ] 集成测试
- [ ] OpenCode MCP 工具调用验证
- [ ] playwright/lark broken pipe 问题修复

**交付物**：
- 测试覆盖率 > 80%
- 所有已知问题修复
- OpenCode 验证通过

#### Sprint 7: 发布准备
**目标**：准备跨平台发布

**任务**：
- [ ] GitHub Actions CI/CD 配置
- [ ] 多平台构建（darwin/amd64, darwin/arm64, linux/amd64, windows）
- [ ] 发布流程文档
- [ ] 更新 architecture.md 为 Go 版本

**交付物**：
- Release 发布流程
- 预编译二进制文件
- `go install` 支持

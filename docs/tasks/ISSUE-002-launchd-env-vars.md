# launchd 环境下 MCP 子进程环境变量缺失问题

> mcp-gateway 作为 launchd 服务运行时，子进程继承贫瘠环境导致多个 MCP 服务无法注册工具

**所属目录**：`docs/tasks/`
**文档状态**：已完成
**当前版本**：v0.1
**发布日期**：2026-04-19
**最后更新**：2026-04-19

---

## 1. 问题概述

| 属性 | 值 |
|------|-----|
| **问题编号** | ISSUE-002 |
| **严重程度** | P0（核心功能不可用） |
| **状态** | ✅ 已修复并验证通过 |
| **关联里程碑** | M5: 测试与验证 |
| **关联分支** | `main`（已合并） |
| **关联文件** | `src/pool/starter.go`, `src/pool/pool.go`, `config.json` |
| **前序问题** | ISSUE-001（Pool 初始化握手） |
| **发现日期** | 2026-04-19 |

### 现象

mcp-gateway 通过 Homebrew 安装后作为 launchd 服务运行时，多个 MCP 服务（minimax、searxng、zai-mcp-server、playwright、lark）无法正常注册工具，只有 pencil（原生二进制）成功。直接在终端运行 brew 二进制则一切正常。

**launchd 环境下的表现**：

| 服务 | 状态 | 工具数 | 启动方式 |
|------|------|--------|----------|
| pencil | ✅ 正常 | 13 | 原生二进制 |
| minimax | ❌ 失败 | 0 | npx |
| searxng | ❌ 失败 | 0 | uvx |
| zai-mcp-server | ❌ 失败 | 0 | npx |
| playwright | ❌ 失败 | 0 | npx |
| lark | ❌ 失败 | 0 | npx |

**终端直接运行的对照结果**：6/6 服务全部成功，51 个工具注册。

**关键特征**：
- 唯一成功的 pencil 是原生 Go 二进制，对 shell 环境无依赖
- 所有 npx/uvx 类服务全部失败，这些服务依赖 Node.js/Python 运行时，需要 PATH、HOME 等环境变量
- 同一个二进制在不同环境下表现完全不同，指向环境问题而非代码逻辑问题

---

## 2. 根因分析

### 2.1 根因 1：launchd 提供最小环境

macOS 的 launchd 服务管理器只为服务进程提供极简环境，仅包含 plist 中显式设置的 PATH 变量，缺少以下关键变量：

| 环境变量 | 作用 | launchd 环境下 |
|----------|------|---------------|
| `HOME` | 用户主目录，npx/uvx/nvm 依赖 | 缺失 |
| `USER` | 当前用户名 | 缺失 |
| `TMPDIR` | 临时目录，Node.js 文件操作依赖 | 缺失 |
| `LANG` | 语言编码 | 缺失 |
| `NVM_DIR` | nvm Node.js 版本管理路径 | 缺失 |
| `XDG_CONFIG_HOME` | 配置目录 | 缺失 |
| `XDG_DATA_HOME` | 数据目录 | 缺失 |
| `XDG_CACHE_HOME` | 缓存目录，npx/uvx 缓存依赖 | 缺失 |

### 2.2 根因 2：子进程继承父进程的贫瘠环境

**位置**：`src/pool/starter.go` 的 `DefaultProcessStarter.Start()` 方法

```go
// 原始代码
cmd := exec.CommandContext(ctx, command, args...)
// ← 没有显式设置 cmd.Env
```

`exec.CommandContext()` 创建子进程时，若未显式设置 `cmd.Env`，子进程自动继承父进程（即 mcp-gateway 进程）的全部环境变量。当 mcp-gateway 由 launchd 启动时，父进程环境本身就很贫瘠，子进程继承后同样缺少必要变量。

**影响**：所有子进程（无论启动方式）都只能拿到 launchd 的最小环境。

### 2.3 根因 3：config.json 的 env 字段未传递给子进程

`ServerConfig` 结构体定义了 `Env map[string]string` 字段，config.json 中也可以为每个服务配置环境变量。但 `DefaultProcessStarter.Start()` 从未将这些变量传递给子进程——环境变量定义了，但在启动流程中被丢弃了。

### 2.4 根因 4：部分服务的环境变量依赖 shell 配置

如 `MINIMAX_API_HOST` 在 `~/.zshrc` 中全局设置，launchd 环境下不加载 shell 配置文件，导致该变量不可用。这类变量本应在 config.json 的 `env` 字段中显式配置。

### 2.5 根因关系图

```
launchd 启动 mcp-gateway
        │
        ▼
父进程只有最小环境（仅 PATH）
        │
        ▼
exec.CommandContext() 未设置 cmd.Env
        │
        ▼
子进程继承父进程的贫瘠环境
        │
        ├── config.json 的 env 字段未传递 ──→ 定义了但没用上
        │
        ├── shell 配置未加载 ──→ .zshrc 中的变量不可用
        │
        ▼
npx/uvx 子进程缺少 HOME、PATH、NVM_DIR 等
        │
        ▼
Node.js/Python 运行时无法正常初始化
        │
        ▼
5/6 MCP 服务无法注册工具
```

---

## 3. 已完成的修复

### 修复方案：运行时通过 login shell 动态获取真实系统环境变量

核心思路：子进程启动时，先通过 login shell 获取用户的完整系统环境（就如同用户在终端中一样），与父进程环境合并后传给子进程。

### 3.1 ProcessStarter.Start() 接口增加 env 参数

**文件**：`src/pool/starter.go`

```diff
  // ProcessStarter 接口
  type ProcessStarter interface {
-     Start(ctx context.Context, command string, args ...string) (*exec.Cmd, error)
+     Start(ctx context.Context, env map[string]string, command string, args ...string) (*exec.Cmd, error)
  }
```

**变更说明**：接口增加 `env map[string]string` 参数，用于传递 config.json 中配置的服务级环境变量。

### 3.2 新增 buildChildEnv() 构建子进程环境

**文件**：`src/pool/starter.go`

```go
func (s *DefaultProcessStarter) buildChildEnv(extra map[string]string) []string {
    // 1. 从 os.Environ() 继承父进程环境
    // 2. 调用 ensureEssentialEnv() 补充缺失变量
    // 3. 合并 extra（config.json 的 env，优先级最高）
    // 4. 返回完整的 []string 格式环境变量
}
```

**环境变量优先级（从低到高）**：

1. `os.Environ()` — 父进程环境，launchd 下很贫瘠
2. `ensureEssentialEnv()` 补充 — login shell 真实环境或硬编码 fallback
3. config.json `env` 字段 — 优先级最高，不被覆盖

### 3.3 新增 fetchLoginShellEnv() 获取真实系统环境

**文件**：`src/pool/starter.go`

```go
func fetchLoginShellEnv() map[string]string {
    // macOS: /bin/zsh -l -c env
    // Linux: /bin/bash -l -c env
    // 5 秒超时限制
    // 解析 env 输出为 map[string]string
    // 失败返回 nil
}
```

**变更说明**：通过 login shell 执行 `env` 命令获取用户的完整环境变量（如同用户手动打开终端后的环境）。设置 5 秒超时防止阻塞启动流程。

### 3.4 重写 ensureEssentialEnv() 补充缺失变量

**文件**：`src/pool/starter.go`

```go
func ensureEssentialEnv(environ []string) []string {
    // 首次调用通过 sync.Once 触发 fetchLoginShellEnv()
    // login shell 成功时：用缓存的真实系统环境补充缺失项
    // login shell 失败时：fallback 到硬编码默认值
}
```

**fallback 默认值**：HOME、USER、PATH、TMPDIR、LANG、TERM、XDG_CONFIG_HOME、XDG_DATA_HOME、XDG_CACHE_HOME

**关键设计**：
- 使用 `sync.Once` 确保 login shell 只执行一次，结果缓存
- login shell 结果仅用于"补充缺失"，不覆盖已有变量
- fallback 机制保证即使 login shell 不可用也能提供基本环境

### 3.5 新增 readStderr() 协程输出子进程日志

**文件**：`src/pool/pool.go`

```go
// 将子进程 stderr 输出到日志，便于诊断子进程启动失败原因
go readStderr(process)
```

**变更说明**：子进程的 stderr 输出原本被丢弃，现在输出到 mcp-gateway 的日志中，方便排查子进程启动失败的原因。

### 3.6 修复 MCP 协议 notifications/initialized

**文件**：`src/pool/pool.go`

```diff
- Method: "initialized",
+ Method: "notifications/initialized",
```

**变更说明**：MCP 协议规范要求 initialized 通知的 method 为 `notifications/initialized`，原代码使用 `initialized` 不符合规范。

### 3.7 config.json 修复 minimax 服务环境变量

**文件**：`config.json`

```diff
  "minimax": {
+     "env": {
+         "MINIMAX_API_HOST": "https://api.minimaxi.com"
+     },
      ...
  }
```

**变更说明**：`MINIMAX_API_HOST` 原本依赖 `.zshrc` 全局设置，在 config.json 中显式配置，确保 launchd 环境下也可用。

### 3.8 修复状态

| 修改 | 文件 | 状态 |
|------|------|------|
| Start() 接口增加 env 参数 | `src/pool/starter.go` | ✅ 已修改 |
| 新增 buildChildEnv() | `src/pool/starter.go` | ✅ 已修改 |
| 新增 fetchLoginShellEnv() | `src/pool/starter.go` | ✅ 已修改 |
| 重写 ensureEssentialEnv() | `src/pool/starter.go` | ✅ 已修改 |
| 新增 readStderr() | `src/pool/pool.go` | ✅ 已修改 |
| 修复 notifications/initialized | `src/pool/pool.go` | ✅ 已修改 |
| config.json 添加 minimax env | `config.json` | ✅ 已修改 |

---

## 4. 测试覆盖

### 4.1 新增测试文件

**文件**：`src/pool/starter_test.go`，共 22 个测试用例。

### 4.2 测试分类

| 分类 | 用例数 | 测试内容 |
|------|--------|----------|
| `TestBuildChildEnv_*` | 6 | 环境继承、extra 合并、优先级验证、输出格式 |
| `TestEnsureEssentialEnv_*` | 13 | login shell 补充缺失变量、已有变量保留、fallback 默认值 |
| `TestFetchLoginShellEnv` | 1 | login shell 实际执行与结果解析 |
| `TestLoginShellEnvCached` | 1 | sync.Once 缓存验证 |
| `TestEnsureEssentialEnv_LoginShellDoesNotOverwrite` | 1 | 确保不覆盖已有变量 |

### 4.3 关键测试场景

| 场景 | 验证点 |
|------|--------|
| 父进程环境继承 | `os.Environ()` 的变量出现在子进程环境中 |
| config.json env 合并 | `extra` 参数的变量出现在子进程环境中 |
| 优先级：extra > 父进程 | 同名变量以 extra 的值为准 |
| launchd 最小环境 | 空环境下 ensureEssentialEnv 补充必要变量 |
| login shell 缓存 | 多次调用 ensureEssentialEnv 只触发一次 login shell |
| login shell 不覆盖 | 已有变量不会被 login shell 结果替换 |
| fallback 机制 | login shell 失败时使用硬编码默认值 |

---

## 5. 验证结果

### 5.1 编译验证

- [x] `go build ./...` — 编译通过

### 5.2 单元测试验证

- [x] `go test ./...` — **35/35 测试全部通过**（含 22 个新增）

### 5.3 模拟 launchd 最小环境验证

使用 `env -i` 模拟 launchd 的最小环境，验证修复后 gateway 能否正常工作：

```bash
env -i PATH=/usr/bin:/bin HOME=$HOME go run ./cmd/gateway
```

| 服务 | 工具数 | 状态 |
|------|--------|------|
| pencil | 13 | ✅ |
| playwright | 21 | ✅ |
| lark | 5 | ✅ |
| searxng | 2 | ✅ |
| minimax | 2 | ✅ |
| zai-mcp-server | 8 | ✅ |
| **合计** | **51** | **✅ 6/6 服务全部成功** |

### 5.4 正常终端环境验证

在正常终端环境下运行，确保修复不影响正常使用场景：

```bash
go run ./cmd/gateway
```

| 服务 | 工具数 | 状态 |
|------|--------|------|
| pencil | 13 | ✅ |
| playwright | 21 | ✅ |
| lark | 5 | ✅ |
| searxng | 2 | ✅ |
| minimax | 2 | ✅ |
| zai-mcp-server | 8 | ✅ |
| **合计** | **51** | **✅ 6/6 服务全部成功** |

### 5.5 环境变量优先级验证

| 优先级层级 | 来源 | 验证方式 |
|-----------|------|----------|
| 最低 | `os.Environ()`（父进程环境） | 确认继承正常 |
| 中等 | `ensureEssentialEnv()` 补充 | 确认缺失变量被补充 |
| 最高 | config.json `env` 字段 | 确认不被其他层级覆盖 |

---

## 6. 建议的后续改进

### 6.1 短期

| 优先级 | 改进项 | 说明 |
|--------|--------|------|
| P1 | 审查所有服务的 env 配置 | 检查是否有其他环境变量应从 shell 配置迁移到 config.json |
| P2 | launchd plist 中设置必要环境变量 | 作为额外保障，在 plist 中也显式设置 HOME、USER 等 |

### 6.2 中期

| 优先级 | 改进项 | 说明 |
|--------|--------|------|
| P2 | login shell 结果缓存持久化 | 避免每次重启 gateway 都要执行一次 login shell |
| P2 | 环境变量诊断接口 | 暴露端点用于查看子进程实际使用的环境变量，方便排查 |
| P3 | 支持环境变量模板 | config.json 中支持 `$HOME`、`$USER` 等变量引用 |

### 6.3 长期

| 优先级 | 改进项 | 说明 |
|--------|--------|------|
| P3 | 跨平台环境变量策略 | Linux systemd 环境有类似问题，需要统一方案 |
| P3 | 环境变量变更热更新 | config.json env 变更后无需重启即可生效 |

---

## 7. 相关文档

| 文档 | 路径 |
|------|------|
| ISSUE-001（前序问题） | [docs/tasks/ISSUE-001-pool-initialize-handshake.md](./ISSUE-001-pool-initialize-handshake.md) |
| 项目状态卡 | [docs/status.md](../status.md) |
| 架构文档 | [docs/product/architecture.md](../product/architecture.md) |
| 里程碑计划 | [docs/product/milestone.md](../product/milestone.md) |

---

## 8. 验收测试报告

### 基本信息

| 项目 | 值 |
|------|-----|
| **项目名称** | MCP Gateway |
| **测试范围** | launchd 环境变量修复（ISSUE-002） |
| **测试日期** | 2026-04-19 |
| **测试版本** | v1.2.5 |
| **测试环境** | macOS (darwin/arm64) |

### 测试结果总览

| 指标 | 数值 |
|------|------|
| 测试用例总数 | 35（含 ISSUE-001 的 13 个） |
| 通过 | 35 |
| 失败 | 0 |
| 通过率 | 100% |
| 新增用例 | 22 |

### 单元测试结果

| 包 | 用例数 | 通过 | 失败 | 状态 |
|----|--------|------|------|------|
| pool | 35 | 35 | 0 | ✅ PASS |

### 集成测试结果

**模拟 launchd 最小环境（env -i）**：

| 服务 | 工具数 | 状态 |
|------|--------|------|
| pencil | 13 | ✅ |
| playwright | 21 | ✅ |
| lark | 5 | ✅ |
| searxng | 2 | ✅ |
| minimax | 2 | ✅ |
| zai-mcp-server | 8 | ✅ |
| **合计** | **51** | **✅ 6/6 服务全部成功** |

**正常终端环境**：

| 服务 | 工具数 | 状态 |
|------|--------|------|
| pencil | 13 | ✅ |
| playwright | 21 | ✅ |
| lark | 5 | ✅ |
| searxng | 2 | ✅ |
| minimax | 2 | ✅ |
| zai-mcp-server | 8 | ✅ |
| **合计** | **51** | **✅ 6/6 服务全部成功** |

### 遗留问题

无新增遗留问题。

### 验收结论

**状态**：✅ **PASS**

**说明**：ISSUE-002 launchd 环境变量修复验证通过。通过 login shell 动态获取真实系统环境变量的方案，在模拟 launchd 最小环境和正常终端环境下均表现正确。6/6 服务成功注册 51 个工具。新增 22 个单元测试全部通过，环境变量优先级机制工作正常：config.json env > login shell 补充 > 父进程继承。

**报告时间**：2026-04-19

---

## 修订记录

| 版本 | 日期 | 修订内容 | 修订人 |
|------|------|----------|--------|
| v0.1 | 2026-04-19 | 初始版本：记录根因分析、已完成修复、测试覆盖、验证结果 | PM Agent |

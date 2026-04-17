# Sprint 3: 工具链完善

> **所属里程碑**: M2: 工具链完善
> **文档来源**: [milestone.md](../../product/milestone.md)
> **版本**: 1.1
> **更新日期**: 2026-04-17

---

## Sprint 信息

| 属性 | 值 |
|------|-----|
| **里程碑** | M2: 工具链完善 |
| **Sprint 编号** | 3 |
| **状态** | ✅ 完成 |
| **开始日期** | 2026-04-17 |
| **结束日期** | 2026-04-17 |

---

## 目标描述

实现工具注册表和名称映射，完善工具链功能：

- 工具注册表（集中管理，按名称查找）
- 工具名映射（前缀映射、剥离、过滤）
- 配置文件格式兼容
- 结构化日志完善

---

## 任务清单

- [x] 工具注册表（集中管理，按名称查找）
- [x] 工具名映射（前缀映射、剥离、过滤）
- [x] 配置文件格式兼容
- [x] 结构化日志完善

---

## 交付物列表

| 交付物 | 描述 | 状态 |
|--------|------|------|
| 工具列表 API 正常工作 | /tools 端点返回正确格式的工具列表 | ✅ |
| 映射规则生效 | 工具名按配置正确映射 | ✅ |
| 日志可追踪 | 结构化日志包含请求 ID 等追踪信息 | ✅ |

---

## 验收标准

- [x] 工具注册表可以注册和查找工具
- [x] 工具名映射规则按配置生效
- [x] 工具调用时名称正确转换
- [x] 日志包含结构化字段（时间、级别、请求 ID、工具名等）
- [x] 配置文件支持工具映射规则

---

## 前置依赖

- Sprint 2: 核心网关（必须完成）

---

## 技术考量

- 工具注册表实现参考现有 `src/registry/`
- 需要支持动态注册和静态配置两种方式
- 映射规则需要可配置、可扩展

---

## 备注

- 工具名映射是 Claude Desktop 兼容性的关键
- 需要考虑映射的性能开销

---

## 实现说明

### 工具注册表 (`src/registry/registry.go`)

- `Registry` 结构体：线程安全的工具注册表
- `RegisterTool`：注册工具到注册表
- `GetTool`：按名称查找工具
- `GetAllTools`：获取所有已注册工具
- `Count`：返回已注册工具数量
- `GetToolsByServer`：按服务器名称筛选工具

### 工具映射器 (`src/registry/mapper.go`)

- `Mapper` 结构体：工具名映射器
- `GetGatewayToolName`：将原始工具名转换为网关工具名（添加前缀）
- `GetOriginalToolName`：将网关工具名反向映射为原始工具名（去除前缀+重命名）
- `ShouldIncludeTool`：根据过滤规则判断工具是否应该包含
- 支持前缀映射、剥离前缀、重命名映射、include/exclude 过滤

### 配置文件 (`src/config/types.go`)

- `MappingConfig`：工具名映射配置（Prefix、StripPrefix、Rename）
- `ToolFilterConfig`：工具过滤配置（Include、Exclude）
- `Config`：完整配置结构，包含 Mapping 和 ToolFilters

### 结构化日志

- 使用 `log/slog` 实现结构化日志
- 网关模式 (`src/gateway/server.go`)：
  - 请求入口日志包含 session ID、method
  - 工具调用日志包含 tool、originalName、server、isError
- stdio 模式 (`src/stdio/server.go`)：
  - 工具调用日志包含 tool、originalName、server、isError

### 测试覆盖

| 模块 | 覆盖率 |
|------|--------|
| src/registry | 61.7% |
| src/config | 50.5% |
| src/pool | 2.1% |
| 整体 | - |


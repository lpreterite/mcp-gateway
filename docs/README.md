# MCP Gateway 文档中心

## 文档结构

```
docs/
├── README.md              # 文档中心入口
└── product/
    ├── PRD.md             # 产品需求文档
    ├── api.md             # API 使用文档
    ├── architecture.md     # 技术架构
    ├── cli.md             # CLI 使用文档
    ├── deployment.md       # 部署说明
    ├── installation.md     # 用户安装说明
    └── milestone.md        # 里程碑计划
```

---

## 文档索引

### PRD.md — 产品需求文档
**作用**：产品功能需求、功能点优先级、验收标准

**内容概览**：
- 产品概述与问题陈述
- 目标用户（AI 开发者）
- 用户故事与核心场景
- P0/P1/P2 功能需求详情
- 非功能需求（性能、可靠性、兼容性）
- 技术架构
- 里程碑规划
- 风险与依赖

**读者**：产品经理、开发团队、DevOps

---

### api.md — API 使用文档
**作用**：HTTP/SSE 和 JSON-RPC 接口规范

**内容概览**：
- HTTP API（/health、/tools、/tools/call）
- SSE 端点（/sse 建立连接、POST 发送请求）
- JSON-RPC API（initialize、tools/list、tools/call）
- 错误码说明
- 客户端配置示例（OpenCode、Claude Desktop）

**读者**：前端开发者、API 集成方

---

### architecture.md — 技术架构
**作用**：系统设计和技术实现细节

**内容概览**：
- 系统架构图
- 核心组件说明（HTTP/SSE、连接池、工具注册表、工具映射器、服务管理）
- 项目结构
- 请求流程
- 关键技术选型

**读者**：开发工程师、技术评审

---

### cli.md — CLI 使用文档
**作用**：命令行工具使用手册

**内容概览**：
- 全局选项
- 运行命令（HTTP/SSE 服务器、Stdio 模式）
- 配置命令（config info/init）
- 服务管理命令（service install/start/stop/restart/status）
- 配置路径优先级
- 日志和信号处理

**读者**：运维工程师、开发人员

---

### deployment.md — 部署说明
**作用**：生产环境部署指南

**内容概览**：
- 配置文件格式详解
- 环境变量说明
- 部署场景（本地开发、单服务器、Docker）
- 安全注意事项
- 故障排查指南

**读者**：运维工程师、DevOps

---

### installation.md — 用户安装说明
**作用**：快速安装指引

**内容概览**：
- 前置要求
- 安装方式（Homebrew、go install、二进制、源码编译）
- 快速开始四步指南
- 卸载说明
- 常见问题

**读者**：所有用户

---

### milestone.md — 里程碑计划
**作用**：版本规划和迭代计划

**内容概览**：
- 当前状态和已实现功能
- 里程碑规划（M1-M6）
- Sprint 详细规划

**读者**：产品经理、开发团队、项目管理者

---

## 快速导航

| 需求 | 文档 |
|------|------|
| 了解产品功能 | [PRD.md](./product/PRD.md) |
| 集成 MCP Gateway | [api.md](./product/api.md) |
| 了解系统设计 | [architecture.md](./product/architecture.md) |
| 使用命令行 | [cli.md](./product/cli.md) |
| 生产部署 | [deployment.md](./product/deployment.md) |
| 安装服务 | [installation.md](./product/installation.md) |
| 查看版本计划 | [milestone.md](./product/milestone.md) |

---

## 贡献指南

文档使用 Markdown 格式编写，遵循以下约定：
- 标题层级：H1（页面标题）→ H2（章节）→ H3（子章节）
- 代码块标注语言：bash、json、go 等
- 文档头部包含元信息：状态、文档来源、版本、更新日期

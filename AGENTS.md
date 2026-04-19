# 项目：mcp-gateway

## AI 研发规范

遵循 AI 软件研发工程体系。

当需要时，读取以下规范文件作为强制指令：
- docs/ai-engineering/principles.md — 核心原则
- docs/ai-engineering/process.md — 研发流程
- docs/ai-engineering/collaboration.md — 协作协议
- docs/ai-engineering/checklists.md — 检查清单
- docs/ai-engineering/deliverables.md — 产出物要求
- docs/ai-engineering/document-management.md — 文档管理

## 版本发布流程

打 tag 和推送前**必须**先运行验证：

```bash
make check
```

验证通过后才能执行 commit、tag、push。验证不通过则修复后再验证，不允许跳过。

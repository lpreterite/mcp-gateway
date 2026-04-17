# Sprint 7: 发布准备

> **所属里程碑**: M6: 发布准备
> **文档来源**: [milestone.md](../../product/milestone.md)
> **版本**: 1.0
> **更新日期**: 2026-04-17

---

## Sprint 信息

| 属性 | 值 |
|------|-----|
| **里程碑** | M6: 发布准备 |
| **Sprint 编号** | 7 |
| **状态** | ⏳ 待开始 |
| **开始日期** | 2026-04-__ |
| **结束日期** | 2026-04-__ |

---

## 目标描述

准备跨平台发布，建立发布流程：

- GitHub Actions CI/CD 配置
- 多平台构建（darwin/amd64, darwin/arm64, linux/amd64, windows）
- 发布流程文档
- 更新 architecture.md 为 Go 版本

---

## 任务清单

- [ ] GitHub Actions CI/CD 配置
- [ ] 多平台构建（darwin/amd64, darwin/arm64, linux/amd64, windows）
- [ ] 发布流程文档
- [ ] 更新 architecture.md 为 Go 版本

---

## 交付物列表

| 交付物 | 描述 | 状态 |
|--------|------|------|
| Release 发布流程 | GitHub Release 流程可用 | ⏳ |
| 预编译二进制文件 | 多平台二进制文件可用 | ⏳ |
| `go install` 支持 | 可以通过 go install 安装 | ⏳ |

---

## 验收标准

- [ ] GitHub Actions 工作流可以正常运行
- [ ] CI 在 PR 时运行测试
- [ ] CD 在 tag 时构建并发布
- [ ] 构建产出 darwin/amd64 二进制
- [ ] 构建产出 darwin/arm64 二进制
- [ ] 构建产出 linux/amd64 二进制
- [ ] 构建产出 windows 二进制
- [ ] 发布流程文档完整
- [ ] architecture.md 已更新为 Go 版本

---

## 前置依赖

- Sprint 6: 测试与验证（必须完成）

---

## 技术考量

- GitHub Actions 需要配置多个构建 job
- 可能需要 CGO 交叉编译支持
- goreleaser 或类似工具可以简化发布
- architecture.md 需要反映当前的 Go 实现

---

## 发布流程

1. 创建 git tag（格式：vX.Y.Z）
2. GitHub Actions 自动触发
3. 在所有目标平台构建
4. 创建 GitHub Release
5. 上传二进制文件到 Release

---

## 备注

- 发布准备是最后一个 Sprint
- 确保所有功能稳定后再发布
- 文档更新同样重要

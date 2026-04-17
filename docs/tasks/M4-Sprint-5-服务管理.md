# Sprint 5: 服务管理

> **所属里程碑**: M4: 服务管理
> **文档来源**: [milestone.md](../../product/milestone.md)
> **版本**: 1.0
> **更新日期**: 2026-04-17

---

## Sprint 信息

| 属性 | 值 |
|------|-----|
| **里程碑** | M4: 服务管理 |
| **Sprint 编号** | 5 |
| **状态** | ⏳ 待开始 |
| **开始日期** | 2026-04-__ |
| **结束日期** | 2026-04-__ |

---

## 目标描述

实现双轨制服务架构，支持跨平台服务管理：

- `ServiceFacade` 统一命令入口
- macOS `PlatformAdapter`（launchd 适配）
- Linux `PlatformAdapter`（systemd 适配）
- 分层状态探测与诊断

---

## 任务清单

- [ ] `ServiceFacade` 统一命令入口
- [ ] macOS `PlatformAdapter`（launchd 适配）
- [ ] Linux `PlatformAdapter`（systemd 适配）
- [ ] 分层状态探测与诊断

---

## 交付物列表

| 交付物 | 描述 | 状态 |
|--------|------|------|
| `service install/start/stop/restart/status` 命令正常 | 服务管理命令完整可用 | ⏳ |
| 分层诊断输出 | 可以输出详细的状态诊断信息 | ⏳ |
| 平台自愈能力 | 某些错误条件下自动恢复 | ⏳ |

---

## 验收标准

- [ ] `service install` 可以在对应平台安装服务
- [ ] `service start` 可以启动服务
- [ ] `service stop` 可以停止服务
- [ ] `service restart` 可以重启服务
- [ ] `service status` 可以查看服务状态
- [ ] macOS 上使用 launchd
- [ ] Linux 上使用 systemd
- [ ] 诊断输出包含分层状态信息

---

## 前置依赖

- Sprint 4: Stdio Bridge（建议完成）

---

## 技术考量

- 需要为每个平台实现 PlatformAdapter 接口
- launchd 使用 launchctl 命令
- systemd 使用 systemctl 命令
- 需要处理服务脚本生成

---

## 备注

- 服务管理是生产环境部署的关键
- 需要考虑权限问题（服务安装通常需要 root）
- Windows 平台暂不在本次范围

# MCP Gateway 服务集成与双轨制架构方案

## 1. 文档目标

本文档用于统一 `mcp-gateway` 的服务管理设计，明确以下问题：

- 为什么当前服务管理容易出现“已安装但未运行”“配置可读但状态显示 stopped”这类问题
- 为什么单纯依赖跨平台服务库不足以覆盖 macOS `launchctl` / Linux `systemd` 的异常状态
- 如何采用“双轨制”架构，将“服务管理”和“应用运行”解耦，降低生命周期管理复杂度
- 如何在尽量小的改动范围内，为后续实现更稳的 `install/start/stop/restart/status` 打下基础

本文档替代旧版“仅将服务命令接入 `kardianos/service`”的实施计划，作为后续服务治理的设计基线。

## 2. 背景与问题定义

当前 `mcp-gateway` 已经具备以下能力：

- 通过 `mcp-gateway service install` 安装为系统服务
- 通过 `mcp-gateway service start|stop|restart|status` 管理服务
- 使用 `github.com/kardianos/service` 统一对接 macOS `launchd` 和 Linux `systemd`

但在实际使用中，已经暴露出几个关键问题：

1. 服务状态来源不止一个
- 配置文件是否有效
- 服务定义文件是否存在
- 系统服务管理器是否已加载该服务
- 服务进程是否存活
- 应用监听端口和健康检查是否正常

2. 当前命令语义过于依赖底层抽象
- `Restart()`、`Start()`、`Status()` 主要依赖三方库返回值
- 无法准确表达“已安装但未加载”“已加载但未健康”“进程存在但应用未就绪”等状态

3. 平台状态机与跨平台抽象之间存在鸿沟
- macOS 的 `launchctl` 关注 `plist`、domain、bootstrap、bootout、kickstart
- Linux 的 `systemd` 关注 unit、enabled、active、failed
- 三方库适合覆盖通用路径，但不擅长处理平台异常恢复

4. 失败恢复逻辑缺失
- 例如 `plist` 已存在，但服务未加载到 `gui/<uid>` 域
- 例如配置合法，但 `status` 只能返回 `stopped`
- 用户需要自己理解底层平台命令才能恢复

## 3. 设计目标

新的服务集成设计应满足以下目标：

1. 明确分离“服务管理”和“应用运行”两个关注点
2. 让 `start/restart` 的语义从“调用一次命令”升级为“尽量收敛到目标状态”
3. 让 `status` 提供诊断价值，而不是仅输出布尔结果
4. 保留 `kardianos/service` 的跨平台价值，但不把平台异常恢复完全交给它
5. 为后续扩展 macOS/Linux 平台定制逻辑保留接口边界

## 4. 核心方案：双轨制架构

### 4.1 定义

双轨制的核心思想是将系统拆成两条职责明确的轨道：

1. 服务管理轨
- 负责系统级安装、注册、加载、启动、停止、状态探测
- 面向 `launchd` / `systemd` 等平台能力
- 处理服务声明文件、环境变量、开机启动、用户域/系统域等问题

2. 应用运行轨
- 负责 `mcp-gateway` 业务进程本身的启动、配置加载、日志、端口监听、健康检查、优雅退出
- 面向网关业务逻辑，不感知服务平台实现细节

### 4.2 设计原则

1. 服务管理轨不直接承载业务判断
- 它的目标是“把进程正确交给平台管理器”

2. 应用运行轨不承担平台恢复逻辑
- 它的目标是“在给定配置和环境下稳定运行”

3. 两条轨道通过清晰契约连接
- 服务层传入配置路径、环境变量、启动参数
- 应用层返回退出状态、健康状态、错误日志

4. 出现问题时优先判断是哪条轨道出错
- 配置不合法、端口冲突、健康失败属于应用运行轨
- `plist` 未加载、unit 未注册、用户域异常属于服务管理轨

## 5. 目标架构

```text
┌────────────────────────────────────────────────────────────┐
│                    Service Management Track                │
│                                                            │
│  CLI(service) -> ServiceFacade -> PlatformAdapter          │
│                                  ├─ macOS launchd adapter  │
│                                  └─ Linux systemd adapter  │
│                                                            │
│  职责：install / uninstall / load / unload / restart /    │
│       status probe / environment injection                 │
└──────────────────────────────┬─────────────────────────────┘
                               │
                               │ 启动契约：可执行文件、参数、环境
                               ▼
┌────────────────────────────────────────────────────────────┐
│                     Application Runtime Track              │
│                                                            │
│  main -> config.Load -> gateway.NewServer -> Start         │
│                                                            │
│  职责：配置校验、日志初始化、连接池启动、端口监听、健康检查、 │
│       优雅关闭                                             │
└────────────────────────────────────────────────────────────┘
```

## 6. 职责边界

### 6.1 服务管理轨职责

- 生成或安装服务定义
- 维护服务 label / unit name
- 处理用户级或系统级服务域
- 注入运行所需的 PATH 和其他环境变量
- 执行平台级 start/stop/restart/status
- 在必要时做平台级自愈

### 6.2 应用运行轨职责

- 读取配置文件并校验语义
- 初始化日志输出
- 构建连接池和工具注册表
- 启动 HTTP/SSE 服务
- 暴露 `/health` 等运行时探针
- 响应 SIGINT/SIGTERM 并完成优雅退出

### 6.3 边界约束

- 服务管理轨不解析业务状态，例如不判断某个 MCP server 是否注册成功
- 应用运行轨不直接调用 `launchctl` / `systemctl`
- 服务命令不应直接嵌入业务启动细节，只消费应用层暴露的成功/失败结果

## 7. 生命周期模型

为了避免状态混淆，服务状态至少应按以下层次分层探测：

### 7.1 配置层

- 配置文件是否存在
- JSON 是否合法
- 配置是否通过语义校验

### 7.2 安装层

- macOS: `~/Library/LaunchAgents/mcp-gateway.plist` 是否存在
- Linux: systemd unit 是否存在
- 服务定义是否与当前二进制、当前参数一致

### 7.3 注册层

- macOS: 是否加载到正确的 `launchd` domain
- Linux: unit 是否已被 systemd 识别

### 7.4 运行层

- 进程是否存在
- 端口是否监听
- 健康检查是否通过

### 7.5 推荐状态枚举

服务管理层建议抽象出以下状态，而不是只有 `running/stopped`：

- `ConfigInvalid`
- `NotInstalled`
- `InstalledNotLoaded`
- `LoadedStopped`
- `RunningUnhealthy`
- `RunningHealthy`
- `Unknown`

## 8. 命令语义重定义

### 8.1 install

目标：将服务安装到平台服务管理器可识别的位置。

要求：

- 保留 `kardianos/service` 作为安装层默认实现
- 安装后立即验证服务定义文件是否存在
- 输出实际安装位置和生效配置路径

### 8.2 start

目标：将服务收敛到 `RunningHealthy` 或至少 `Running`。

要求：

- 若未安装，返回明确错误
- 若已安装但未注册，执行平台加载操作
- 若已注册但未运行，执行平台启动操作
- 启动后增加一次运行态验证

### 8.3 stop

目标：将服务收敛到 `LoadedStopped` 或 `NotLoaded`。

要求：

- 如果服务不存在，返回可解释结果，不做误导性失败
- 停止后验证进程已退出

### 8.4 restart

目标：无论当前处于何种中间状态，都尽量回到 `RunningHealthy`。

要求：

- 不假设服务当前已经被平台加载
- 先探测当前状态，再执行最小必要动作
- 必要时允许 `reload` / `bootout + bootstrap` / `kickstart` 这类平台恢复路径

### 8.5 status

目标：提供分层诊断信息，而不是仅输出 `running/stopped`。

建议输出结构：

```text
Config: valid
Install: present
Registration: loaded
Process: running
Health: healthy
Suggested action: none
```

对于异常情况，建议输出：

```text
Config: valid
Install: present
Registration: missing
Process: not running
Health: unknown
Suggested action: bootstrap service into launchd domain
```

## 9. 平台适配策略

### 9.1 保留三方库的部分

`github.com/kardianos/service` 继续承担以下职责：

- 跨平台服务定义生成
- 基础的安装/卸载能力
- 基本的服务对象创建

原因：

- 它已经覆盖大部分正常路径
- 可以减少平台样板代码
- 对 Linux/macOS 的基础支持足够成熟

### 9.2 自定义平台补充层

在 `gwservice` 之上增加一层轻量封装，例如：

- `ServiceFacade`
- `PlatformAdapter`
- `MacOSLaunchdAdapter`
- `LinuxSystemdAdapter`

该层负责：

- 状态探测
- 平台级异常恢复
- 统一的诊断输出
- 将“命令调用”转换成“目标状态收敛”

### 9.3 为什么不完全交给三方库

因为三方库主要解决“通用操作”，而不是“平台异常恢复”。

典型缺口包括：

- macOS 中 `plist` 已存在但服务未加载到当前用户 domain
- `launchctl` 的 `bootstrap`、`bootout`、`kickstart` 等恢复路径
- Linux 中 unit 已安装但处于 failed/inactive 状态时的恢复策略

## 10. 推荐模块拆分

建议在 `src/gwservice` 下逐步演进为以下结构：

```text
src/gwservice/
  manager.go          # 现有服务入口，保留兼容
  facade.go           # 对外统一命令入口
  status.go           # 分层状态探测与诊断模型
  contract.go         # 服务轨与应用轨之间的契约
  platform_darwin.go  # macOS 平台适配
  platform_linux.go   # Linux 平台适配
```

建议契约模型：

- `ServiceStatusReport`
- `InstallResult`
- `RunTargetState`
- `SuggestedAction`

## 11. 失败处理与自愈策略

### 11.1 设计原则

- 优先探测，再执行动作
- 优先最小动作，再升级恢复手段
- 所有恢复动作都要有验证步骤
- 所有失败都要给出“建议下一步”

### 11.2 典型异常与策略

1. 配置非法
- 直接失败
- 不进入平台级服务控制

2. 服务已安装但未加载
- 执行平台加载操作
- 更新 `status` 输出为“注册层异常已恢复”

3. 服务已加载但应用未健康
- 判定为应用运行轨问题
- 输出应用日志路径和健康检查失败原因

4. 服务定义存在但路径失效
- 判定为安装层漂移
- 建议重新安装服务

## 12. 实施路线

### Phase 1：文档与边界收敛

- 更新本设计文档
- 明确双轨制术语和职责边界
- 停止继续把平台异常逻辑堆进 CLI 命令里

### Phase 2：状态探测模型

- 为 `status` 增加分层探测
- 定义统一状态报告结构
- 统一错误文案和建议动作

### Phase 3：macOS 平台自愈

- 在 macOS 上增加 launchd 注册态探测
- 为 `start/restart` 增加“未加载时自愈加载”的逻辑
- 验证 `status` 与真实 `launchctl` 状态一致

### Phase 4：Linux 平台对齐

- 补齐 systemd 状态探测
- 统一 Linux/macOS 的状态报告结构
- 确保 CLI 输出风格一致

### Phase 5：运行轨契约增强

- 增加更明确的健康检查定义
- 让服务层能判断“进程已运行”与“应用已就绪”的差异
- 视需要引入结构化退出码

## 12.1 当前落地状态

截至目前，文档规划已推进到以下阶段：

### Phase 1：已完成

- 已更新本文档，统一双轨制术语与职责边界
- 已停止继续把 macOS 异常恢复逻辑直接堆在 CLI 层

### Phase 2：已完成第一版

- `service status` 已从布尔状态升级为分层诊断输出
- 当前已能输出：
  - `Config`
  - `Install`
  - `Registration`
  - `Process`
  - `Health`
  - `Suggested action`
- 已实现配置无副作用检查与路径解析复用

### Phase 3：macOS 已完成第一版，Linux 待补齐

- macOS 已具备：
  - launchd 注册态探测
  - `start/restart` 的 `bootstrap` / `kickstart` / `bootout + bootstrap` 自愈逻辑
  - `stop` 的平台级 `bootout` 收敛
- Linux 已补第一版 `systemd` 平台适配：
  - `start/stop/restart` 已走 `systemctl` 控制路径
  - 支持 `daemon-reload` 后重试
  - `status` 已补 `systemctl show` 的注册态探测
- 但 Linux 仍未做到与 macOS 完全对称的运行态/健康态恢复细化

### Phase 4：已开始结构铺垫，尚未完成

- 已引入 `Facade + PlatformAdapter` 结构
- 已拆出：
  - `facade.go`
  - `contract.go`
  - `platform_darwin.go`
  - `platform_linux.go`
- 另已补：
  - `platform_generic.go`
  - `platform_shared.go`
  - `contract.go`
- 当前 CLI 已统一走 `Facade`，不再直接混用 `manager` 与零散控制函数
- Linux 平台适配已可用，但仍未具备与 macOS 同等级的自愈能力

### Phase 5：已部分完成

- HTTP 启动顺序已改为“先监听，后初始化”
- `/health` 已补充 `ready` 与 `status`（如 `initializing`）
- 初始化阶段的 `/tools`、`/tools/call`、JSON-RPC 请求已返回明确的“still initializing”语义
- 已增加初始化日志、每个 server 的工具收集结果日志以及 ready 切换汇总日志
- 已补第一版结构化退出码
- 已补第一版契约类型：
  - `ServiceState`
  - `SuggestedActionCode`
  - `SuggestedAction`
  - `InstallResult`
  - `RunTargetState`
- 尚未完成更细粒度的运行轨契约与全部状态枚举的彻底收口

## 12.2 当前状态总结

当前系统已经具备“双轨制”的核心骨架：

- 服务管理轨：
  - 通过 `gwservice.Facade` 统一接管 `install/start/stop/restart/status/uninstall`
  - 通过 `PlatformAdapter` 分发平台相关控制逻辑
  - macOS 已具备第一版异常恢复能力

- 应用运行轨：
  - 已从“阻塞式启动”改为“先监听、后后台初始化”
  - 已具备初始化期与就绪期的明确行为边界
  - 已增强健康检查与启动日志可观测性

当前尚未完成的主要缺口有两类：

1. Linux / systemd 对齐
- 仍缺少与 macOS 对称的“已安装未加载 / 已加载未健康”完整恢复细化
- 仍缺少更细粒度的 process/health 探测与失败分类

2. 契约模型与退出码完善
- 第一版契约类型和退出码已经落地
- 但状态字符串尚未百分之百收口到统一枚举体系
- `SuggestedAction` 仍主要用于 CLI 文本展示，尚未扩展为更强的结构化动作模型

## 12.3 下一步

建议下一阶段按以下顺序推进：

1. 完成 Linux 平台适配层
- 为 systemd 增加探测、启动、停止、重载、恢复逻辑
- 统一 Linux/macOS 的状态报告语义

2. 收敛契约模型
- 将状态字符串逐步替换为统一的类型化状态
- 让 `SuggestedAction` 不再只是自由文本，而是可组合的结构

3. 完成 install/stop 的验证闭环
- install 后立即验证服务定义是否生效
- stop 后显式验证注册态/进程态是否已退出

4. 引入结构化退出码
- 已完成第一版，后续可继续扩展到更多子命令和更细粒度错误

5. 同步 README
- 已完成第一版同步，后续可继续补充更多示例与故障排查说明

## 13. 测试策略

### 13.1 单元测试

- 状态枚举与状态合成逻辑
- 平台命令输出解析
- 建议动作生成逻辑

### 13.2 集成测试

- install 后 status 检查
- 已安装未加载场景下的 start/restart
- 运行中 restart 的幂等性验证
- 配置非法时的 fail-fast 行为

### 13.3 手工验证

macOS：

- 验证 `plist` 存在但未加载的恢复路径
- 验证用户域 `gui/<uid>` 下的状态探测

Linux：

- 验证 unit 安装、enable、active、failed 的探测和恢复

## 14. 迁移策略

- 保留现有 `service` 子命令名称，避免用户接口变化
- 内部逐步将命令从“直接调用三方库方法”迁移为“通过服务门面收敛目标状态”
- 优先升级 `status` 和 `restart`，因为这两个命令最能体现诊断和恢复能力
- 在 README 中逐步补充更精确的服务状态说明

## 15. 当前结论

对于 `mcp-gateway` 这类同时涉及系统服务管理和业务网关运行的程序，最佳实践不是完全放弃三方库，也不是继续把所有逻辑堆进一个 `manager` 里，而是：

- 继续使用 `kardianos/service` 处理基础安装与跨平台封装
- 在其上增加平台适配和状态探测层
- 采用双轨制架构，明确分离“服务管理轨”和“应用运行轨”

这样可以在保持当前实现连续性的前提下，显著提升以下能力：

- 服务命令的可解释性
- 异常状态的可恢复性
- 生命周期管理的可测试性
- 后续平台扩展的可维护性

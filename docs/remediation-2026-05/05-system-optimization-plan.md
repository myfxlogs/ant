# AntTrader 系统级优化方案

## 1. 文档定位

本文档是 AntTrader 后续整改实施的**系统级方向依据**。

`01-master-remediation-plan.md` 负责说明整改路线，`04-target-architecture.md` 负责说明目标架构，`10-migration-checklist.md` 负责跟踪任务；本文档负责回答：

```text
为什么要这样整改
哪些方向必须坚持
哪些问题优先解决
哪些重构不能偏离交易系统本质
```

后续实施如果出现方向争议，应优先回到本文档确认系统级原则。

## 2. 当前系统判断

AntTrader 当前已经不是简单 Demo，而是一个功能较完整的交易系统雏形。

已具备能力包括：

- 用户与权限。
- MT4/MT5 账户连接。
- 手动交易。
- 持仓、订单、历史查询。
- 行情、K 线、账户流。
- 策略模板。
- Python 策略校验、执行、回测。
- 策略调度与 Online 运行。
- 自动交易配置与部分风控。
- AI Provider、Agent、Debate V2、代码辅助。
- 管理后台。
- 日志、审计、分析、运维脚本。

整体判断：

```text
项目已经进入“功能密集堆叠期”。
下一阶段重点不应继续无序加功能，而应转向系统边界、交易稳定性、通信收敛、风控治理、代码规模控制和文档重建。
```

## 3. 主要系统问题

### 3.1 文档与代码长期漂移

当前旧文档存在以下问题：

- 部分文档描述的是历史计划，不是当前实现。
- 部分文档把占位接口描述得过于完整。
- 部分文档没有区分“已实现”和“目标设计”。
- 文档之间存在重复、过期、互相覆盖。
- 新开发如果继续参考旧文档，容易走错方向。

优化方向：

```text
建立新文档体系
  → 标注旧文档状态
  → 迁移有效内容
  → 归档历史文档
  → 删除误导性文档
```

### 3.2 前后端职责边界被侵蚀

交易系统中，前端如果承担业务计算，会带来严重风险：

- 多端结果不一致。
- 用户可篡改计算过程。
- 风控和交易判断不可审计。
- 后续移动端或 API 客户端会重复实现逻辑。
- 出现盈亏、风险、策略状态争议时无法确定权威来源。

必须坚持：

```text
前端只展示
后端做计算
策略服务做策略运行与回测
```

前端可以格式化数据，但不能生成业务事实。

### 3.3 ConnectRPC 体系被 REST 业务接口稀释

项目原计划采用 ConnectRPC，但当前混入了多个 `/api/...` 业务 REST 端点。

风险：

- 类型契约分裂。
- 前端调用方式不统一。
- 鉴权、错误处理、日志追踪不一致。
- proto 不再是完整业务地图。
- 后续重构和客户端生成困难。

必须回归：

```text
浏览器到后端业务通信统一 ConnectRPC
```

REST 只允许健康检查等非业务能力。

### 3.4 后端组装层过重

当前后端主服务存在功能集中组装问题。

典型表现：

- 服务初始化集中。
- Connect handler 与 REST handler 混杂。
- API 生命周期与 worker 生命周期混杂。
- repository、service、external client 依赖关系隐式。
- 新增功能容易继续堆到同一个入口。

风险：

- 启动复杂。
- 测试困难。
- 拆分困难。
- 局部修改影响全局。
- 运行时故障难定位。

优化方向：

```text
server.New 巨型组装
  → app.Container
  → transport/connect
  → transport/rest health only
  → worker lifecycle
  → domain service
  → infra adapter
```

### 3.5 交易核心未被作为最高优先级治理

项目中 AI、策略、回测、管理后台都很重要，但交易系统的生命线是：

```text
账户连接
  → 状态同步
  → 风控
  → 执行
  → 记录
  → 审计
  → 恢复
```

任何策略、AI 或自动化能力都必须经过这条链路，不能旁路。

当前需要加强：

- 风控决策统一。
- 手动交易与自动交易共用风险语义。
- 自动交易风控从轻量检查升级为配置驱动。
- 执行失败、拒绝、超时、幂等命中必须完整审计。
- MT 连接状态需要更明确的状态机。

### 3.6 策略生命周期不够强约束

当前策略已有模板、草稿、回测、调度等能力，但需要形成强生命周期。

目标是避免：

- 未校验代码直接发布。
- 未回测策略直接 Online。
- AI 生成代码绕过人工确认。
- 策略运行状态和配置状态混在一起。
- 调度异常缺少结构化追踪。

必须形成：

```text
Draft
  → Validate
  → Backtest
  → RiskAssessment
  → Publish
  → Schedule
  → Online
  → Execute
  → Audit
```

### 3.7 文件规模失控风险

项目已有多个超 800 行文件，且包括生成代码。

风险：

- 单文件职责过多。
- 代码审查困难。
- 修改冲突频繁。
- 生成代码不可控。
- proto 领域边界过粗。

必须坚持：

```text
除 Markdown 文档外，所有文件 <= 800 行
```

生成文件超限时，不能忽略，必须从 proto 拆分源头解决。

## 4. 系统级目标

### 4.1 一个权威业务入口

业务接口全部通过 ConnectRPC 暴露。

目标：

```text
proto = 业务接口地图
connect handler = 唯一浏览器业务入口
frontend client = 唯一业务调用方式
```

### 4.2 一个权威计算位置

所有业务事实由后端产生。

包括：

- 账户状态。
- 交易权限。
- 风险等级。
- 盈亏统计。
- 回撤。
- 策略状态。
- 回测评分。
- AI 流程状态。

### 4.3 一个稳定交易内核

交易内核必须独立于 AI 和前端。

目标分层：

```text
Trading Core
  - Account State
  - Market Snapshot
  - Permission Gate
  - Risk Engine
  - Execution Gateway
  - Trade Record
  - Audit

Strategy Extension
  - Template
  - Validation
  - Backtest
  - Schedule
  - Signal

AI Assistance
  - Provider
  - Agent
  - Debate
  - Code Assist
  - Report

Operation Layer
  - Admin
  - Logs
  - Metrics
  - Health
```

### 4.4 一个可演进的模块化单体

短期不拆大量微服务。

目标是：

```text
先模块化
再 worker 化
最后按需要服务化
```

### 4.5 一个可审计的策略自动化闭环

自动交易必须可解释、可追踪、可停用。

每次自动化执行都应能回答：

- 哪个策略触发？
- 为什么触发？
- 输入行情是什么？
- 生成了什么信号？
- 风控是否通过？
- 是否下单？
- 下单结果是什么？
- 若失败，失败在哪一步？

## 5. 推荐目标架构

### 5.1 总体形态

```text
Browser
  ↓ Connect-Web
Backend API
  ↓ Domain Services
Backend Workers
  ↓ Internal Clients / Repositories
Strategy Runtime Service
  ↓
PostgreSQL / Redis / MT4 / MT5 / AI Providers
```

### 5.2 Backend API

Backend API 只负责：

- ConnectRPC handler。
- 鉴权与权限校验入口。
- 请求转换。
- 调用 domain service。
- 返回后端已计算结果。

Backend API 不应负责：

- 长时间阻塞任务。
- 策略循环调度。
- 大型回测执行。
- 连接恢复循环。
- 前端专用计算。

### 5.3 Backend Worker

Worker 负责：

- MT 连接恢复。
- 行情与账户同步。
- 策略调度。
- 异步回测任务。
- 定时清理。
- 指标采集。

短期可以与 API 同进程，但必须在代码和 lifecycle 上拆开。

### 5.4 Strategy Runtime Service

策略服务负责：

- Python 策略静态校验。
- 沙箱执行。
- 回测。
- 技术指标。
- 策略记忆。

策略服务不负责：

- 用户鉴权。
- 账户权限。
- 实盘下单。
- 最终风控。
- 审计。

### 5.5 Frontend

前端负责：

- 页面渲染。
- 表单。
- Connect client。
- 图表。
- 国际化。
- 操作反馈。

前端不负责：

- 风控。
- 交易决策。
- 账户统计。
- 回测指标。
- AI 流程裁决。

## 6. 核心链路重构方案

### 6.1 账户连接链路

目标链路：

```text
AccountService ConnectRPC
  → AccountDomainService
  → ConnectionOrchestrator
  → MTAdapter
  → AccountStateStore
  → EventPublisher
  → AuditLogger
```

关键优化：

- 建立连接状态机。
- 连接状态与账户基础信息分离。
- 连接失败使用标准错误码。
- 连接恢复由 worker 管理。
- 状态变化全部审计。

建议状态：

```text
DISCONNECTED
CONNECTING
CONNECTED
RECONNECTING
ERROR
DISABLED
```

### 6.2 手动交易链路

目标链路：

```text
TradingService ConnectRPC
  → AuthInterceptor
  → PermissionGate
  → AccountStateChecker
  → RiskEngine.ValidateManualOrder
  → ExecutionGateway
  → MTAdapter
  → TradeRecordWriter
  → StreamPublisher
  → AuditLogger
```

关键优化：

- 统一风险返回。
- 统一错误码。
- 统一审计结构。
- 所有订单动作经过 ExecutionGateway。

### 6.3 自动交易链路

目标链路：

```text
ScheduleRunner
  → StrategyRuntime.Execute
  → SignalNormalizer
  → RiskEngine.ValidateStrategySignal
  → ExecutionGateway
  → MTAdapter
  → TradeRecordWriter
  → ScheduleRunLogger
  → AuditLogger
```

关键优化：

- 策略信号标准化。
- 自动交易风控配置化。
- 重复信号拦截。
- 冷却时间。
- 最大持仓。
- 最大日内亏损。
- 最大回撤自动停用。
- 每次执行结构化记录。

### 6.4 风控链路

目标是统一 RiskEngine。

```text
RiskEngine
  - ValidateManualOrder
  - ValidateOrderModify
  - ValidateOrderClose
  - ValidateStrategySignal
  - ValidateScheduleActivation
```

统一结果：

```text
RiskDecision
  allowed
  level
  code
  message
  details
  evaluated_rules
```

风险等级建议：

```text
allow
warn
block
fatal
```

### 6.5 策略生命周期链路

目标：

```text
Create Draft
  → Edit Code
  → Validate
  → Extract Params
  → Backtest
  → Risk Assessment
  → Publish
  → Schedule
  → Enable Online
  → Runtime Evaluation
  → Execution
  → Logs and Audit
```

策略状态建议：

```text
draft
validated
backtested
published
scheduled
online
disabled
archived
```

AI 生成代码只允许进入 Draft，不允许直接进入 Online。

### 6.6 回测链路

目标将回测视为独立子系统。

```text
BacktestRequest
  → DatasetResolver
  → CostModelResolver
  → StrategyRuntime.Backtest
  → MetricsCalculator
  → RiskAssessment
  → ResultStore
  → Frontend Render
```

建议独立存储：

- 回测运行。
- 成交明细。
- 权益曲线。
- 风险事件。
- 成本模型。

### 6.7 AI 链路

AI 定位为辅助，不是交易决策核心。

目标链路：

```text
AI Debate / Code Assist
  → Generate Draft
  → Validate
  → Backtest
  → Risk Assessment
  → User Confirm
  → Publish
```

AI 不允许：

- 直接下单。
- 绕过回测。
- 绕过风控。
- 绕过用户确认。
- 修改实盘配置后自动启用。

## 7. 数据与协议优化

### 7.1 proto 拆分原则

proto 必须按领域拆分，不能继续膨胀。

建议拆分方向：

```text
common.proto
pagination.proto
risk.proto
auth.proto
account.proto
trading.proto
market.proto
stream.proto
strategy_template.proto
strategy_schedule.proto
backtest.proto
ai_provider.proto
ai_debate.proto
ai_code_assist.proto
system_ai.proto
admin_user.proto
admin_system.proto
log.proto
```

目标：

- proto 文件小于 800 行。
- 生成代码小于 800 行。
- service 职责单一。
- message 结构清晰。

### 7.2 数据库文档化

每个核心表必须说明：

- 表用途。
- 写入方。
- 读取方。
- 生命周期。
- 是否可清理。
- 是否涉及审计。
- 是否与交易结果相关。

### 7.3 运行态与配置态分离

重点对象：

- 策略调度。
- 账户连接。
- 自动交易风控。
- AI 工作流。

原则：

```text
配置表保存用户意图
运行态表保存系统事实
日志表保存过程证据
```

## 8. 可观测性优化

### 8.1 健康检查

目标保留 REST 健康探针：

```text
/health/live
/health/ready
/health/deps
```

`/health/deps` 应覆盖：

- PostgreSQL。
- Redis。
- Strategy Runtime。
- MT4 Gateway。
- MT5 Gateway。
- Scheduler。
- Backtest Worker。

### 8.2 指标体系

必须补充核心指标：

交易：

- 下单次数。
- 下单成功次数。
- 下单失败次数。
- 风控拒绝次数。
- 执行延迟。

策略：

- 调度评估次数。
- 信号次数。
- 自动下单次数。
- 策略错误次数。
- Online 策略数量。

回测：

- pending 数。
- running 数。
- 成功数。
- 失败数。
- 耗时。

连接：

- 已连接账户数。
- 重连次数。
- 连接失败次数。

AI：

- 请求次数。
- 错误次数。
- 延迟。
- provider 错误分类。

### 8.3 日志分层

日志应分为：

```text
application log
operation audit log
trade execution log
strategy schedule log
risk decision log
system health log
```

不同日志服务不同目的，不应混成一个大而全的日志桶。

## 9. 安全与权限优化

### 9.1 密钥治理

要求：

- 生产密钥不得进入仓库。
- AI Provider Key 后端加密存储。
- API Key 只显示一次。
- Docker Compose 示例不得包含真实密钥。

### 9.2 权限模型

建议统一权限 scope：

```text
auth:read
account:read
account:write
trade:read
trade:write
strategy:read
strategy:write
strategy:run
backtest:run
ai:use
admin:read
admin:write
log:read
```

前端路由权限、后端用户角色、API Key scope 应逐步统一。

## 10. 分阶段实施优先级

### 阶段一：先锁方向

交付：

- 新整改文档体系。
- 工程硬规则。
- 行数基线。
- REST 迁移清单。

目标：防止继续走偏。

### 阶段二：收敛接口

交付：

- 浏览器业务 REST 迁移 ConnectRPC。
- 前端 REST fetch 清理。
- 后端业务 REST handler 删除。

目标：让 proto 重新成为业务契约地图。

### 阶段三：收敛计算

交付：

- 前端业务计算清点。
- 计算逻辑迁回后端。
- proto response 补充展示字段。

目标：后端成为业务事实唯一来源。

### 阶段四：拆分大文件

交付：

- 行数检查脚本。
- `make verify` 接入。
- 手写超限文件拆分。
- proto 超限拆分。
- 生成代码超限消除。

目标：把可维护性纳入硬约束。

### 阶段五：重构交易核心

交付：

- RiskEngine。
- RiskDecision。
- ExecutionGateway 统一入口。
- 自动交易风控增强。
- 审计链路完善。

目标：让实盘链路稳定、可控、可审计。

### 阶段六：重构策略生命周期

交付：

- 策略状态机。
- 发布前强制校验和回测。
- Online 前风险检查。
- 调度运行态与配置态分离。

目标：让策略从生成到实盘有完整管控。

### 阶段七：Worker 化与运维增强

交付：

- API lifecycle 与 worker lifecycle 分离。
- 健康检查增强。
- 指标体系增强。
- 容器重启流程标准化。

目标：提高运维稳定性和故障定位能力。

## 11. 禁止偏离项

整改过程中禁止：

- 新增业务 REST 接口。
- 前端新增业务计算。
- AI 结果直接进入实盘下单。
- 未回测策略直接 Online。
- 超 800 行文件继续增长。
- 以“自动生成”为理由接受超限代码。
- 在未完成替代文档前删除旧文档。
- 把健康检查接口扩展为业务接口。
- 把策略服务直接暴露给浏览器。
- 将风控逻辑分散在前端、策略服务和多个 handler 中。

## 12. 成功标准

系统级优化成功后，应达到：

- 文档体系清晰，旧文档不再误导。
- 前端只负责展示和交互。
- 业务通信统一 ConnectRPC。
- proto 是完整业务接口地图。
- 后端模块边界清楚。
- Worker 与 API 生命周期可分离。
- 交易核心链路稳定、可审计。
- 自动交易风控配置化、可解释。
- 策略生命周期完整。
- AI 辅助能力受控。
- 非 MD 文件全部不超过 800 行。
- 每轮整改都有质量检查和容器验证。

## 13. 最终方向总结

AntTrader 后续整改的总方向是：

```text
不推倒重写
不继续堆功能
先统一文档
再收敛协议
再下沉计算
再拆分大文件
再稳定交易核心
再完善策略生命周期
最后增强运维与可观测性
```

这是一个交易系统，不是普通内容管理系统。

因此所有架构取舍必须优先满足：

```text
安全
稳定
可审计
可恢复
可维护
可演进
```

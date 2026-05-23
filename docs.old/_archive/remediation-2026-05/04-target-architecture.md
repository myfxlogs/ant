# AntTrader 目标架构规划

## 1. 本文档目标

本文档定义 AntTrader 整改后的目标架构。

目标不是立即微服务化，而是先完成：

```text
功能堆叠型单体
  → 模块化单体
  → API / Worker 职责解耦
  → 核心交易链路稳定化
  → 必要时再拆分独立服务
```

## 2. 架构总原则

### 2.1 前端不计算

前端只做：

- 展示。
- 渲染。
- 交互。
- 表单输入。
- ConnectRPC client 封装。
- UI 本地状态。

所有金额、风险、策略、交易、回测、AI 流程判断都以后端返回为准。

### 2.2 ConnectRPC 是唯一业务入口

浏览器到后端的业务通信只有一种：

```text
ConnectRPC over HTTP/2 h2c / Connect-Web
```

REST 只允许非业务健康探针。

### 2.3 后端模块化单体优先

当前阶段不建议立刻拆大量微服务。

目标是先让代码结构具备清晰边界：

```text
backend/internal/app
backend/internal/transport
backend/internal/domain
backend/internal/worker
backend/internal/infra
```

### 2.4 交易核心最高优先级

系统所有扩展都围绕交易核心，不得绕过交易核心。

```text
账户连接
  → 行情/账户状态同步
  → 风控
  → 执行
  → 记录
  → 审计
  → 恢复
```

## 3. 目标系统分层

```text
Frontend
  ↓ Connect-Web
Backend API
  ↓ domain service
Backend Domain Core
  ↓ infra adapter
PostgreSQL / Redis / MT Gateway / Strategy Runtime / AI Provider
```

进一步拆分：

```text
Frontend
  - routes
  - pages
  - feature components
  - connect clients
  - rendering only

Backend API
  - auth interceptor
  - connect handler
  - request validation
  - response mapping

Domain Core
  - account
  - trading
  - risk
  - strategy
  - backtest
  - ai
  - admin
  - logs

Worker Runtime
  - connection recovery
  - stream sync
  - schedule runner
  - backtest queue
  - cleanup jobs

Infrastructure
  - postgres repository
  - redis cache/idempotency/leader
  - mt4/mt5 adapter
  - strategy runtime client
  - ai provider client
```

## 4. 建议后端目录结构

目标结构：

```text
backend/internal/
  app/
    bootstrap.go
    container.go
    lifecycle.go
    config.go

  transport/
    connect/
      register.go
      auth.go
      account.go
      trading.go
      strategy.go
      backtest.go
      ai.go
      admin.go
      logs.go
    rest/
      health.go

  domain/
    account/
      service.go
      connection_state.go
      repository.go
    trading/
      command_service.go
      execution_gateway.go
      record_writer.go
    risk/
      engine.go
      decision.go
      rule_account.go
      rule_order.go
      rule_strategy.go
    strategy/
      template_service.go
      schedule_service.go
      lifecycle.go
    backtest/
      run_service.go
      worker.go
      result_store.go
    ai/
      provider_service.go
      debate_service.go
      code_assist_service.go
    logs/
      audit_service.go
      operation_service.go

  worker/
    scheduler.go
    backtest.go
    connection_recovery.go
    stream_sync.go

  infra/
    postgres/
    redis/
    mt/
    strategyruntime/
    ai/
```

迁移时不要求一次性改到位，但新代码应朝该结构靠拢。

## 5. 目标前端目录结构

目标结构：

```text
frontend/src/
  app/
    router.tsx
    providers.tsx

  client/
    connect.ts
    transport.ts
    errors.ts

  features/
    auth/
    accounts/
    trading/
    analytics/
    strategy/
    backtest/
    ai/
    admin/
    logs/

  shared/
    components/
    hooks/
    i18n/
    formatting/
```

每个 feature 内部建议：

```text
api.ts
pages/
components/
hooks/
types.ts
```

前端允许做的转换：

- 时间格式化。
- 数字格式化。
- 表格排序筛选。
- 图表数据映射。
- UI 状态切换。

前端禁止做的转换：

- 重新计算盈亏。
- 重新计算回撤。
- 重新裁决风险。
- 重新判断策略状态。
- 重新聚合账户级指标。

## 6. 交易核心目标链路

### 6.1 手动交易

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

### 6.2 自动交易

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

### 6.3 风控决策统一结构

目标统一结构：

```text
RiskDecision
  allowed
  level
  code
  message
  details
  evaluated_rules
```

所有手动交易、自动交易、策略调度都应使用统一风险语义。

## 7. 策略生命周期目标链路

```text
Draft
  → Validate
  → ExtractParams
  → Backtest
  → RiskAssessment
  → Publish
  → CreateSchedule
  → EnableOnline
  → Evaluate
  → Execute
  → Audit
```

AI 生成策略必须进入同一流程，不允许 AI 结果直接下单。

## 8. Worker 化目标

短期可以仍然一个 binary，但生命周期必须拆清楚：

```text
API lifecycle
  - HTTP server
  - Connect handlers

Worker lifecycle
  - account recovery
  - stream workers
  - strategy scheduler
  - backtest queue
  - cleanup tasks
```

中期可以拆成：

```text
backend-api
backend-worker
```

拆分条件：

- ConnectRPC 迁移基本完成。
- Worker 与 API 依赖边界清楚。
- 启停、健康检查、日志、指标已具备。

## 9. Strategy Runtime 目标

当前 `strategy-service` 可继续作为策略运行时，但目标边界应明确：

```text
Go Backend
  → StrategyRuntimeClient
  → Strategy Runtime Service
```

策略运行时负责：

- 代码校验。
- 沙箱执行。
- 回测。
- 技术指标。
- 策略记忆。

策略运行时不负责：

- 用户权限。
- 账户权限。
- 实盘下单。
- 最终风控。
- 审计记录。

## 10. 数据层目标

按领域组织 repository：

```text
account repository
trading repository
risk repository
strategy repository
backtest repository
ai repository
log repository
admin repository
```

数据库文档必须补充：

- 表用途。
- 写入方。
- 读取方。
- 生命周期。
- 清理策略。
- 是否审计敏感。

## 11. 健康检查目标

保留 REST 健康探针：

```text
/health/live
/health/ready
/health/deps
```

`/health/deps` 应能展示：

- PostgreSQL。
- Redis。
- Strategy Runtime。
- MT4 Gateway。
- MT5 Gateway。
- Scheduler。
- Backtest Worker。

## 12. 架构完成标准

目标架构阶段完成时，应满足：

- 前端业务通信全部 ConnectRPC。
- 前端不再承担业务计算。
- 后端业务 REST 已删除或只剩健康探针。
- `server.go` 不再是巨型组装中心。
- Worker 生命周期独立。
- 风控决策统一。
- 策略生命周期清晰。
- 非 MD 文件全部不超过 800 行。

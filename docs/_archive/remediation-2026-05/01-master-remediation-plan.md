# AntTrader 项目整体整改规划

## 1. 整改背景

当前 AntTrader 已形成较完整的产品雏形，包含：

- 账户绑定与 MT4/MT5 连接。
- 手动交易。
- 自动交易。
- 策略模板。
- 策略调度。
- Python 策略校验、执行、回测。
- AI Provider、Agent、Debate V2。
- 管理后台、日志、分析、运维辅助脚本。

但随着功能累积，项目出现以下问题：

- 历史文档与代码现状差距较大。
- 前端混入部分计算和业务接口细节。
- 原定 ConnectRPC 通信体系混入了 REST 业务接口。
- 后端 `server.New()` 和 handler 注册承担过多职责。
- 策略、AI、回测、自动交易的边界需要重新收敛。
- 部分文件存在继续膨胀风险。

本轮整改目标不是推倒重写，而是在保留现有能力的基础上完成架构收敛。

## 2. 总体目标

目标架构：

```text
Frontend
  - 展示
  - 渲染
  - 交互
  - ConnectRPC client

Backend API
  - 认证授权
  - ConnectRPC 业务接口
  - 账户/交易/策略/AI/管理/日志查询与命令入口
  - 不承担长时间阻塞任务

Backend Worker
  - 账户连接恢复
  - 行情与账户同步
  - 策略调度
  - 异步回测队列
  - 日志与指标采集

Strategy Runtime Service
  - 策略校验
  - 沙箱执行
  - 回测引擎
  - 指标计算

Infrastructure
  - PostgreSQL
  - Redis
  - MT4/MT5 Gateway
  - AI Providers
```

短期仍可维持一个后端 Go binary，但代码结构必须按 API、Worker、Domain、Infra 拆清楚。

## 3. 核心整改原则

### 3.1 交易核心优先

项目最高优先级链路是：

```text
账户连接
  → 行情/持仓同步
  → 风控
  → 执行
  → 订单记录
  → 审计
  → 状态恢复
```

AI、回测、策略生成都不能绕过交易核心链路。

### 3.2 ConnectRPC 优先

所有浏览器到后端的业务接口统一使用 ConnectRPC。

REST 业务接口必须迁移。

### 3.3 后端计算优先

前端不得做最终业务计算。前端显示的统计数据、风险状态、策略状态、账户状态必须来自后端。

### 3.4 模块化单体优先

不立即拆微服务，先把 Go 后端改造成模块化单体。

推荐结构：

```text
backend/internal/app
backend/internal/transport/connect
backend/internal/transport/rest
backend/internal/domain/account
backend/internal/domain/trading
backend/internal/domain/strategy
backend/internal/domain/backtest
backend/internal/domain/ai
backend/internal/domain/admin
backend/internal/worker
backend/internal/infra/postgres
backend/internal/infra/redis
backend/internal/infra/mt
backend/internal/infra/strategyruntime
```

## 4. 阶段路线图

### 阶段 0：文档与边界确立

目标：建立新文档体系，停止继续依赖旧文档。

交付物：

- 新建 `docs/remediation-2026-05/`。
- 明确前后端职责。
- 明确 ConnectRPC 回归目标。
- 明确 800 行硬约束。
- 建立旧文档清理清单。

验收标准：

- 新整改文档存在。
- 旧文档不再作为新增开发依据。
- 工程规则已更新。

### 阶段 1：接口清点与 ConnectRPC 迁移

目标：清点并迁移浏览器可见业务 REST。

重点迁移对象：

- `/api/economic-calendar`：已完成
- `/api/economic-indicators`：已完成
- `/api/strategy/indicator-catalog`：已完成
- `/api/debate/v2/*`：已完成
- `/api/ai/code/revise`
- `/api/ai/code/explain`
- `/api/ai/primary`
- `/api/strategy/validate-extended`
- `/api/backtest-runs/*`

迁移方式：

```text
REST handler
  → proto service/rpc
  → backend internal/connect implementation
  → frontend connect client
  → 删除前端 fetch
  → 删除 REST handler
```

验收标准：

- 前端业务请求不再直接 `fetch('/api/...')`。
- 后端不再注册业务 `/api/...` handler。
- proto 和生成代码全部小于 800 行。

### 阶段 2：前端计算下沉

目标：把前端计算全部迁回后端。

清点对象：

- 账户权益与收益率计算。
- 交易统计聚合。
- 风险指标。
- 回测结果二次计算。
- 策略运行状态派生。
- AI 流程状态推导。

迁移方式：

```text
前端计算函数
  → 后端 service 计算
  → proto response 增加展示字段
  → 前端只渲染后端返回结果
```

验收标准：

- 前端只保留格式化、排序、筛选、渲染。
- 涉及金额、风险、交易决策的数据都以后端为准。

### 阶段 3：后端模块化重构

目标：降低 `server.New()`、Connect 注册、REST handler、service 初始化耦合。

建议拆分：

```text
app.Container
app.Bootstrap
app.Lifecycle
transport/connect.Register
transport/rest.RegisterHealthOnly
domain/*
worker/*
```

验收标准：

- `server.go` 不再承担全部组装逻辑。
- API handler 注册与 service 构造分离。
- Worker 生命周期可独立管理。
- 单文件不超过 800 行。

### 阶段 4：交易核心重构

目标：把风控、执行、审计、状态恢复统一成明确交易内核。

目标链路：

```text
TradingCommand
  → PermissionGate
  → RiskEngine
  → ExecutionGateway
  → MTAdapter
  → TradeRecordWriter
  → EventPublisher
  → AuditLogger
```

重点：

- 统一 `RiskDecision`。
- 手动交易与自动交易共用底层风控能力。
- 自动交易风控从轻量放行升级为严格配置驱动。
- 执行结果、风控拒绝、异常全部可审计。

### 阶段 5：策略生命周期重构

目标：策略从“模板 + 调度”升级为完整生命周期。

目标状态：

```text
draft
  → validated
  → backtested
  → published
  → scheduled
  → online
  → disabled / archived
```

AI 生成策略不得直接 Online，必须经过：

```text
代码生成
  → 校验
  → 参数抽取
  → 回测
  → 风险评分
  → 用户确认发布
  → 调度启用
```

### 阶段 6：旧文档与旧接口删除

目标：完成清理，不再保留误导性文档和接口。

删除条件：

- 新文档已有替代。
- 代码已完成迁移。
- 前端不再引用旧接口。
- 测试和构建通过。
- 容器已重建并验证。

## 5. 优先级建议

第一优先级：

1. 建立新文档体系。
2. 清点并迁移业务 REST。
3. 加入 800 行检查。
4. 清理前端业务计算。

第二优先级：

1. 拆 `server.New()`。
2. 重构自动交易风控。
3. 梳理策略生命周期。
4. 强化回测结果存储。

第三优先级：

1. Worker 化。
2. 观测体系升级。
3. 文档归档与删除。

## 6. 暂不建议立即做的事

暂不建议：

- 立即拆成大量微服务。
- 一次性删除全部旧文档。
- 一次性重写前端。
- 一次性重写策略服务。
- 在未完成 ConnectRPC 迁移前继续新增业务功能。

## 7. 每阶段质量要求

每个阶段必须完成：

- 静态检查。
- 编译检查。
- 测试检查。
- 单文件行数检查。
- 容器重建与重启。
- 前端人工验收。

Markdown 文档变更可只做文本级检查；代码变更必须完成完整质量流程。

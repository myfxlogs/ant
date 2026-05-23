# 市场状态与 K 线缓存专项设计

本文档定义 AntTrader 借鉴 QuantDinger Market Regime 与 Kline Cache 能力时的市场状态识别、回测数据缓存和外部数据源边界。

## 1. 目标与非目标

目标：

- 为策略生成、回测评分、策略实验和调度健康分析提供市场状态上下文。
- 建立 K 线缓存作为回测、报告和实验的性能优化。
- 在结果 metadata 中记录缓存命中、数据源和时间范围。
- 为未来只读外部行情源预留工厂边界。

非目标：

- K 线缓存不替代真实交易实时行情权威。
- 不用缓存行情直接驱动真实交易下单。
- 不以数据源工厂为名新增非 MT 市场实盘接入。
- 不在前端计算市场状态或策略评分。

## 2. 现有依赖模块

- MarketService：行情读取与对外契约。
- KlineService：K 线读取、聚合或缓存入口。
- BacktestRunService：回测数据范围、结果 metadata。
- PythonStrategyService：回测、指标和评分计算。
- StrategyExperimentService：参数实验和候选评分。
- StrategyScheduleService：调度健康展示，不直接采信缓存下单。

## 3. QuantDinger 可借鉴点

- Market Regime 用于策略优化和评分上下文。
- K 线缓存按 symbol、周期和时间范围缓存。
- 数据源工厂隔离不同行情来源。
- 缓存 TTL 按周期区分。

AntTrader 当前以 MT4/MT5 Gateway 行情与订单流为核心，缓存只作为回测与实验优化。

## 4. Market Regime 输出草案

建议输出：

| 字段 | 说明 |
|------|------|
| `regime` | 市场状态，如趋势、震荡、高波动、过渡 |
| `confidence` | 置信度 |
| `features` | 特征指标 |
| `segments` | 分段市场状态 |
| `strategy_families` | 推荐策略族 |
| `data_range` | 数据时间范围 |
| `model_version` | 规则或模型版本 |

初始特征：

- 价格涨跌幅。
- EMA 快慢线差距。
- ATR 百分比。
- 已实现波动率。
- 方向效率。
- 成交量比例，如数据可得。

第一阶段可使用可解释规则，不要求引入复杂模型。

## 5. 使用场景

- AI 生成策略时作为 prompt 上下文。
- 策略实验评分时计算 regime fit。
- 调度启用前提供提示，不直接替代 RiskEngine。
- 调度健康分析中解释策略表现变化。
- 回测报告中说明样本市场环境。

## 6. K 线缓存模型草案

缓存 key 建议：

```text
account_id + symbol + timeframe + range + source
```

缓存 metadata 建议字段：

| 字段 | 说明 |
|------|------|
| `cache_key` | 缓存键 |
| `source` | 数据源 |
| `symbol` | 品种 |
| `timeframe` | 周期 |
| `start_time` | 开始时间 |
| `end_time` | 结束时间 |
| `hit` | 是否命中 |
| `ttl_seconds` | TTL |
| `fetched_at` | 拉取时间 |
| `expired_at` | 过期时间 |

TTL 原则：

- 短周期 TTL 更短。
- 长周期 TTL 可更长。
- 历史闭合区间可使用更长 TTL。
- 当前未闭合 K 线不得长期缓存。

## 7. 数据源工厂边界

如果未来接入非 MT 只读数据源，优先建立只读工厂：

```text
MarketDataSource
  -> MTGatewaySource
  -> ExternalKlineSource
  -> CachedSource
```

约束：

- 工厂只服务行情读取、回测、报告和实验。
- 交易执行仍以现有 MT4/MT5 核心为准。
- 新增数据源不得扩展为多交易所实盘适配。
- 数据源差异不得泄漏到前端业务计算。

## 8. ConnectRPC 接口草案

```text
MarketRegimeService
  DetectMarketRegime
  GetMarketRegime

KlineCacheService
  GetKlineCacheStatus
  InvalidateKlineCache
```

`InvalidateKlineCache` 属管理或维护能力，Agent 不应默认开放。

回测和策略实验接口应在结果 metadata 中携带数据源与缓存信息，而不是要求前端二次拼接。

## 9. 后端、策略服务与前端职责

后端负责：

- 数据源选择、缓存命中判断、metadata 入库。
- Market Regime 结果的持久化或引用。
- ConnectRPC 契约与权限控制。

策略服务负责：

- 特征计算、市场状态识别、回测指标与评分。
- 返回可解释 feature 与评分组件。

前端负责：

- 展示市场状态、置信度、特征摘要和缓存 metadata。
- 不计算 regime、评分或交易建议。

## 10. 分阶段实施

1. 在回测 metadata 中记录数据源与时间范围。
2. 建立 K 线缓存 key 与 TTL 规则。
3. 实现轻量 Market Regime 规则计算。
4. 将 regime fit 接入策略评分。
5. 为策略实验和 AI prompt 提供 regime 上下文。
6. 评估只读外部数据源工厂。

## 11. 验收标准与禁止事项

验收标准：

- 回测结果可追溯行情数据源、时间范围和缓存命中状态。
- Market Regime 输出有版本和特征解释。
- 前端只展示后端或策略服务返回结果。
- 缓存不会驱动真实交易下单。

禁止事项：

- 禁止用 K 线缓存替代真实交易实时行情权威。
- 禁止缓存数据直接触发真实交易。
- 禁止前端计算市场状态、评分或调度建议。
- 禁止以数据源工厂为名新增非 MT 实盘接入。

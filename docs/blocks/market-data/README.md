# market-data — 市场数据

> K 线/Tick 存储、实时报价流、去重/质量/归一化、回填、在线指标。

## 代码位置

```
backend/internal/mdgateway/      ← 60+ 文件：adapter(mt4/mt5)、tick 去重、质量检测、bar 聚合、回填
backend/internal/source/         ← 实时数据源接口
backend/internal/symbol/         ← 品种规范化/解析
backend/internal/repository/     ← market_data_pg.go、ch_market_data_store.go
```

## 关键设计

- 双存储：PG（业务数据）+ ClickHouse（时序 K 线/Tick）
- NATS JetStream 推送实时 bar
- 在线指标（SMA/EMA/Bollinger/RSI/MACD…）——流式计算，不停机
- 回填系统（backfiller/）：source → target → trigger → clock
- 熔断器（CircuitBreaker）按 broker endpoint 隔离

## 依赖

```
mt-gateway(报价流) → market-data(去重/归一化/存储)
```

## 被依赖

```
market-data → strategy-runtime(实时bar源)
market-data → backtest-engine(历史K线)
market-data → frontend(行情展示)
```

## 关联文档

- [spec/11-mdgateway.md](spec/11-mdgateway.md)
- [spec/13-clickhouse-schema.md](spec/13-clickhouse-schema.md)
- [spec/19-md-doctor.md](spec/19-md-doctor.md)
- [plans/resilience-gaps.md](plans/resilience-gaps.md) — bar_aggregator 重启恢复 + tick 去重验证

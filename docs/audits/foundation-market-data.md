# 地基审计 · market-data

## 1. 核心设计替代方案

**当前**：MT adapter 推送 tick → tick_dedup → quality → bar_aggregator → PG/CH/NATS。在线指标流式计算。回填系统独立。

**替代方案**：不用 ClickHouse，全部存 PG。

不选。CH 对时序 K 线/Tick 的压缩率和查询性能是 PG 的 10-50 倍。PG 存业务数据、CH 存行情数据——混合存储是正确的。

**替代方案**：NATS 替代 SSE 推送实时 bar。

当前就是用 NATS JetStream 推实时 bar。SSE 是前端消费 NATS 事件的管道。设计正确。

**结论**：✅ 架构最优。多存储（PG+CH+NATS）分层正确，不需要改。

## 2. 上下游契约

**上游**：mt-gateway 推送原始 tick → mdtick 层标准化 → market-data 管线。

**下游**：strategy-runtime 通过 `BarSource` 接口消费 bar → backtest-engine 通过 PG/CH 查询历史 K 线。

**隐患**：tick 去重依赖 symbol+timestamp 唯一性。mt-gateway 断开重连后会重推已收到的 tick。去重逻辑是否正确处理了重连边界？（需要验证 `tick_dedup.go` 的去重窗口是否覆盖重连间隙）

**另一个隐患**：bar_aggregator 在内存中聚合。进程重启后内存中的未完成 bar 丢失，导致重启时刻的 bar 数据有缺口。当前有持久化吗？（需要验证）

## 3. 已知架构债务

| 债务 | 严重度 | 方案 |
|------|--------|------|
| bar_aggregator 重启丢未完成 bar | 🟡 | 启动时从 PG 读最近 N 根 bar 恢复聚合状态，或改为从 tick 重建 |
| tick 去重窗口与重连间隙的关系未验证 | 🟡 | 确认去重窗口 > 最坏重连时间（当前指数退避 max 5min） |

## 4. 总评

架构最优。两个黄标不是设计缺陷——是实现细节的验证问题。验证后大概率不需要改。

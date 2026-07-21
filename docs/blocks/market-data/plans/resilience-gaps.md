# market-data 韧性缺口 · 施工清单

> 来源：`docs/audits/foundation-market-data.md` 黄标

## 模块 R1 · bar_aggregator 重启恢复

**Why**：进程重启后内存中未完成的 bar 丢失，导致重启时刻前后 K 线数据有缺口。

- [ ] **R1a** 启动时从 PG 读最近 N 根已持久化的 bar，恢复内存聚合状态
- [ ] **R1b** 如 PG 中的 bar 不完整（重启时正在进行中的那根 bar），从 tick 重建该 bar
- [ ] **验收**：进程重启 → 检查重启时刻前后 bar 数据连续无缺口

## 模块 R2 · tick 去重窗口验证

**Why**：mt-gateway 重连后会重推已收到的 tick。去重窗口必须覆盖最坏重连时间。

- [ ] **R2a** 确认 `tick_dedup.go` 的去重窗口大小
- [ ] **R2b** 确认 mt-gateway 指数退避的最坏重连时间（当前 max 5min）
- [ ] **R2c** 如果去重窗口 < 最坏重连时间 → 扩大窗口到 10min
- [ ] **验收**：断开 mt-gateway → 等待 5min → 重连 → 无重复 tick 入库

# 施工交接：按需订阅（P0，资源 + bar 送达，替代 LIVE-PRICE-4 的"订全 462"）

> **审计方定论 2026-08-14，可行性已验证。** 触发：LIVE-PRICE-4 把订阅从 37 改成 FetchAllSymbols 全 462 → 每账户 462 symbol × 实时 tick × 8 周期聚合 × 多账户 → 把 192MiB 小服务器吃到 60%（"全红了"+ build 失败）+ 可能导致 bar 不送达 run（负载下 broker 阻塞）。
>
> **解法**：按需订阅——gateway 连接时订空（只启动 recvLoop），策略启动时 `mtHub.SubscribeSymbols` 按需加订其 symbol。负载从 462 降到每策略 ~1 个。
>
> **铁律**：对抗证明 + commit + 部署 + 回填。**服务器资源紧，build 选择低负载时段**。

## 可行性（审计方已验证，接线已存在）

- `mthub/service.go:418 SubscribeSymbols(ctx, accountID, symbols)` → `exec.AddSymbols(ctx, symbols)`（OrderExecutor 接口，mt4/mt5 Gateway 实现，order_types.go:121）。
- mt4/mt5 `Subscribe(ctx, syms, handler)`（quotes.go:15）：`len(syms)==0` 时跳过 SubscribeMany（:28）、仍启动 recvLoop（:40）。AddSymbols 后 recvLoop 自动交付新 symbol。
- gateway 的 `g.subscribedSymbols`（quotes.go:26）持久化，`reSubscribeSymbols`（:182）重连时重订这些 → 按需加的 symbol 重连后自动恢复。

## 改动（2 处）

### ① gateway connect 订空（runner_gateway.go:125-143）
现状（LIVE-PRICE-4 引入）：
```go
syms := cfg.Symbols
if len(syms) == 0 {
    available, _ := fetcher.FetchAllSymbols(ctx)  // 订全 462 ← 太重
    syms = available
}
gw.Subscribe(ctx, syms, mgr.HandleTick)
```
改成：
```go
// Demand-driven: gateway 启动 recvLoop 但不预订 symbol；策略启动时按需 AddSymbols。
gw.Subscribe(ctx, nil, mgr.HandleTick)
```
（FetchAllSymbols 调用 + 逐 symbol SubscribeMany 逻辑移除——策略会通过 SubscribeSymbols 加订。）

### ② RunLiveStrategy 按需加订（live_runner.go，source.Subscribe 之后）
```go
barCh, barCancel := source.Subscribe(barAccountID)
defer barCancel()
// Demand-driven: 确保 gateway 订阅本策略的 symbol（不然 OnQuote 不交付）。
if s.mtHub != nil && cfg.Symbol != "" {
    if err := s.mtHub.SubscribeSymbols(barAccountID, []string{cfg.Symbol}); err != nil {
        s.log.Warn("LiveStrategyRunner: SubscribeSymbols failed", zap.String("symbol", cfg.Symbol), zap.Error(err))
        // 非 fatal：策略仍订阅了 broker，gateway 订阅失败只意味着收不到该 symbol bar。
    }
}
```
（`barAccountID` = cfg.AccountID 或 cfg.DataSourceAccountID，与 bar 订阅一致。）

## 影响

- **资源**：每账户订阅数从 462 → 0（无策略时）或 ~1-5（有策略时）。backend 内存压力骤降，192MiB 松绑，build 可行。
- **bar 送达**：broker 只收所需 symbol 的 bar，channel 不再被 462 symbol 压满 → 大概率修复"bar 不送达 run"。部署后重测 eager 1m。
- **重连**：gateway 重连后 reSubscribeSymbols 重订按需 symbol（g.subscribedSymbols 已持久化）。

## 对抗证明

- **①**：mock Subscribe 调用 → 旧代码传 462 symbol（RED，订全）；新代码传 nil/空（GREEN，订空）。
- **②**：mock RunLiveStrategy + 注入 mock mtHub → 断言 SubscribeSymbols 被调用且 symbol=cfg.Symbol（GREEN）；删调用 → 未触发（RED）。
- **端到端**：部署后（低负载时段）跑 eager 1m run → bar_source 账户只订了 BTCUSDm → 出信号（铁证：按需订 + bar 送达 + 执行链通）。

## 红队自审
- [ ] ② 的 barAccountID 与 source.Subscribe 用同一个（cfg.AccountID 或 DataSourceAccountID），别串。
- [ ] 订空后，无策略的账户 gateway 仍 healthy（recvLoop 在跑，只是没数据）——不影响健康检查。
- [ ] reSubscribeSymbols 重连时重订的是 g.subscribedSymbols（按需加的），不是空的——确认 AddSymbols 有 append 到 subscribedSymbols（quotes.go:69 已有）。
- [ ] SubscribeSymbols 失败非 fatal（log warn，不阻断策略启动）。
- [ ] MT4 + MT5 的 Subscribe/AddSymbols 同构，都覆盖。

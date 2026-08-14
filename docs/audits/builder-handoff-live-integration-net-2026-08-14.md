# 施工交接：Live 路径端到端集成测试（"网"——系统性暴露所有残留断点）

> **审计方定论 2026-08-14。** 背景：live 路径逐层修了 10+ 个 bug（LIVE-PRICE-1/2/3/4、BAR-ALIGN、ACCT-LOOKUP、demand-subscribe、MDGATEWAY-1/2/3/4），**每一层都是手测发现**。审计方证实 bar 已送达 run（DIAG log pass:true）、shouldRunOnBar 通过、handleBar 被调、VM 响应 success=true——但 **VM 响应里 0 信号**（OrderSend 没产生 StrategySignal）。
>
> **根因模式**：整条 live 路径只做了单测、从没做过端到端集成测试 → 每个组件"单测过了"但组件间集成全是 bug。手动剥洋葱太慢。
>
> **本任务**：写一个 live 路径端到端集成测试，从"喂 bar → VM → 信号 → dispatch → 下单"整条链一次性跑，**把所有残留断点一次暴露**。

---

## 测试设计

### 测试目标
用一个最简单的 eager MQL 策略（每根 bar OrderSend BUY），mock bar 喂进 RunLiveStrategy，断言整条链的每一步，**精确定位哪一步断**。

### 断言链（逐环节，哪步断哪步红）

```
mock bar (BTCUSDm 1m closed)
  → ① bar 到达 runLiveEventLoop barCh           [已证实 OK，DIAG log]
  → ② shouldRunOnBar pass                        [已证实 OK，pass:true]
  → ③ handleBar 被调                             [已证实 OK]
  → ④ initVMSession / CompileMQL 成功            [待验——可能 compile 静默退化]
  → ⑤ session.Start / SendEvent 成功（无 error）  [已证实 OK，无 error log]
  → ⑥ VM 执行 OnBar                              [待验——OnBar 是否被调？]
  → ⑦ OrderSend 产生 StrategySignal              [待验——当前 0 信号，疑似这步断]
  → ⑧ ExecuteLiveResponse.Signals 非空            [待验——当前为空]
  → ⑨ dispatchFromBytes 提取到信号               [待验——上游空所以没提取到]
  → ⑩ dispatchPaperSignal / paperEngine 下单     [待验]
```

**当前已知断点：⑥→⑧ 之间**（VM 执行了 success=true 但响应 0 信号）。测试要覆盖 ⑥-⑩，**精确到哪一步断**。

### 实现方案

**Go 集成测试**（`backend/internal/connect/strategy/live_integration_test.go`），用 mock 组件：

```go
func TestLivePath_E2E_BarToSignalToOrder(t *testing.T) {
    // 1. 最简 eager MQL 策略
    code := `void OnBar() { OrderSend(Symbol(), OP_BUY, 0.01, Ask, 3, 0, 0); }`

    // 2. mock barSource（可控推 bar）
    barSource := newMockBarSource()
    
    // 3. mock/paper engine（捕获信号/下单）
    paperEngine := newCapturePaperEngine()
    
    // 4. 构造 StrategyExecutionServer（注入 mock 组件）
    srv := newTestServer(barSource, paperEngine, ...)
    
    // 5. 启动 RunLiveStrategy（goroutine）
    go srv.RunLiveStrategy(ctx, cfg)
    
    // 6. 推一根 closed bar
    barSource.pushBar(&mthub.BarUpdate{
        Symbol: "BTCUSDm", Period: "1m", Closed: true,
        Open: dec("100"), High: dec("101"), Low: dec("99"), Close: dec("100.5"),
    })
    
    // 7. 断言链——每一步独立断言，哪步断哪步报
    require.Eventually(t, func() bool {
        return paperEngine.orderCount() > 0  // 最终：paper 下了单
    }, 5*time.Second, 100*time.Millisecond)
    
    // 如果上面超时——逐步诊断：
    // ⑧ 检查 VM 响应有无信号（需要 instrument 或 log）
    // ⑥ 检查 OnBar 有没有被调
    // ④ 检查 compile 有没有成功
}
```

### 关键设计决策

1. **不用真 mtapi/broker**：mock barSource（可控推 bar）+ mock/paper engine（捕获下单）。这隔离了 VM→signal→dispatch→order 链。

2. **先测最简策略**：eager（OrderSend 每 bar）。如果这个都不产信号 → VM bug。如果产信号 → MACD 的问题是策略逻辑（正常）。

3. **逐步断言**：如果最终断言（paper 下单）失败，中间加诊断断言（compile 成功？OnBar 被调？响应有信号？），**一次跑测定位精确断点**。

4. **覆盖 paper + live**：paper 模式（capturePaperEngine）+ live 模式（mockOrderExecutor）各一个。

### 预期发现

根据当前证据（bar 到达 + VM success + 0 信号），**最可能的断点是 ⑥ 或 ⑦**：
- ⑥ OnBar 没被 VM 调用（VM 事件 dispatch bug）。
- ⑦ OrderSend 没产生 StrategySignal（VM OrderSend handler bug）。

测试会**精确区分**这两个（以及 ④ compile 静默退化）。

---

## 对抗证明（测试本身的有效性）

- 用一个**已知能产信号**的策略（eager OrderSend）→ 如果测试 0 信号 = **测试有效（抓到了 bug）**。
- 修复后 → 测试出信号 = GREEN。
- 删测试的任何一步断言 → 测试变弱（RED 对抗——证明每步断言都在守一个环节）。

## 红队自审
- [ ] mock barSource 要真正推到 barCh（模拟 broker.Subscribe 返回的 channel）。
- [ ] paperEngine 要真捕获 OrderSend（不是 no-op）。
- [ ] compile 用真 CompileMQL（不是 mock）——这样才能测到 VM 的 OnBar/OrderSend。
- [ ] 如果 CompileMQL 失败（eager 代码不兼容），测试要 log compile error（而非静默）。
- [ ] 超时要有诊断（哪步断），不是盲目 fail。

## 后续
- 测试暴露断点后 → Windsurf 修那个断点 → 测试 GREEN → 再加 MACD 策略测试（稀有信号策略的正常路径）。
- **这个测试是永久的"网"**——以后任何 live 路径改动，CI 跑这个测试，断了立刻知道。

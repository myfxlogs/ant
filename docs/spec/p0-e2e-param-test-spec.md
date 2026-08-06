# 施工 Spec：端到端测试——参数链 + 防线呈现（坑 10 修复）

> 坑 10 是 xianhua 所有坑的共同系统性根因——坑 1/7/8/9 全靠用户踩坑发现，没有测试提前抓。本 spec 建两个端到端集成测试，守护"参数链"和"防线呈现"两条最脆弱的链路。

## 测试 1：参数链端到端（compile → inject → VM → 撮合 → result）

**文件**：`backend/tools/mql2go/e2e_param_pipeline_test.go`

**验证链路**：MQL 源码（含浮点 default 参数）→ CompileMQL → cfg.Params（用户填的值）→ engine.Run → result.Trades.volume == 用户填的值（不是默认值，不是 0）。

```go
func TestE2E_ParamPipeline_FloatDefaultParam(t *testing.T) {
    // 用 MACD Sample（含 input double Lots=0.1 浮点 default）
    src := readMacdSample(t) // 读 testdata/macd_sample.mq4
    runner, err := CompileMQL(src) // 编译（findIdent/findType/findInitValue 全部走一遍）

    // 用户填 Lots=0.42（不是默认 0.1，验证注入用的是用户值不是默认）
    cfg := backtest.Config{
        InitialCapital: decimal.NewFromInt(10000),
        Leverage:       100,
        Params:         map[string]string{"Lots": "0.42"},
    }
    engine := backtest.New(cfg, runner, makeE2EBars(80))
    result, err := engine.Run(context.Background())

    // 断言：每笔交易 volume=0.42（用户填的值）
    for i, tr := range result.Trades {
        if !tr.Volume.Equal(decimal.NewFromFloat(0.42)) {
            t.Errorf("trade[%d] volume=%s, want 0.42 (param pipeline broken)", i, tr.Volume)
        }
    }

    // 断言：Lots global slot 注入了 0.42（不是 0/0.1/double）
    if v, ok := runner.GetGlobal("Lots"); ok {
        // v 应该是 0.42
    }
}
```

**对抗 case**（不预先公布，但 spec 要求覆盖）：
- Lots=0.42（非默认非0）→ volume=0.42
- Lots 不传（空 Params）→ volume=默认值（0.1，findInitValue 提取的）
- 杠杆=500 → 验证 cfg.Leverage 注入（可从 metrics 或 engine 行为推断）

## 测试 2：防线呈现端到端（detection → marking → status DEGRADED）

**文件**：`backend/internal/connect/strategy/e2e_defense_presentation_test.go`

**验证链路**：buildBacktestResponse（含 volume=0 的 trades）→ 防线 B 检测 → IsReliable=false + BlindSpot → hasInvariantBlindSpot → status=DEGRADED（不是 SUCCEEDED）。

```go
func TestE2E_DefensePresentation_DegradedStatus(t *testing.T) {
    // 构造 volume=0 的 trades（故意触发防线 B）
    trades := []backtest.Trade{{Volume: decimal.Zero, Side: sdk.SideBuy, ...}}

    result := &backtest.Result{Trades: trades, Metrics: makeMetrics(1), Config: ...}
    resp, _, _, _ := buildBacktestResponse(result, cfg, params, vmRunner)

    // 断言 1：防线 B 检测到 → BlindSpot 含 zero_volume_trade
    found := false
    for _, bs := range resp.BlindSpots {
        if bs.Id == "zero_volume_trade" { found = true }
    }
    if !found { t.Fatal("防线 B 未检测到 zero_volume_trade") }

    // 断言 2：IsReliable=false
    if resp.Risk.IsReliable { t.Error("IsReliable 应为 false") }

    // 断言 3：hasInvariantBlindSpot(resp)=true → status 应该 DEGRADED
    if !hasInvariantBlindSpot(resp) { t.Fatal("hasInvariantBlindSpot 应为 true") }

    // 断言 4：正常 trades（volume>0）→ 无 invariant BlindSpot → SUCCEEDED
    goodTrades := []backtest.Trade{{Volume: decimal.NewFromFloat(0.1), ...}}
    goodResult := &backtest.Result{Trades: goodTrades, ...}
    goodResp, _, _, _ := buildBacktestResponse(goodResult, cfg, params, vmRunner)
    if hasInvariantBlindSpot(goodResp) { t.Error("正常 trades 不应有 invariant BlindSpot") }
}
```

## 约束

- 后端 Go 测试，`go test ./tools/mql2go/ ./internal/connect/strategy/` 全绿。
- 最小改动（只加测试文件，不改生产代码）。
- 测试用真实 EA（testdata/macd_sample.mq4）+ 真实 engine.Run（不 mock）——这是端到端的意义。

## 硬性质量要求（不只"能跑"，必须最优）

1. 实现方法是最优解——测试验证的是**链路行为**（端到端），不是单个函数。
2. 代码干净无冗余——无死代码、无未使用 import。
3. 无技术债——无 TODO/FIXME。
4. 无违规——不用 nolint。
5. 符合第一性原则——测试名表达**验证的链路意图**（如 ParamPipeline_FloatDefaultParam），不是验证某个函数。

# P0 施工 Spec 02：资金守恒不变量（恒等类，Stage 1.1）

> ADR-0028 §4.2 防线 B 恒等类第二条。配合已落地的"成交手数>0"（`checkVolumeInvariant`），加"资金守恒"。**抓撮合/记账/净值曲线类 bug（账对不上）——恒等类、零误报、判决型。** 这是碰钱逻辑，评审会亲自 review。

## 现状代码（必读，自行定位）

- **文件**：`backend/internal/connect/strategy/backtest_worker_vm.go`
- `buildBacktestResponse` 已有 `checkVolumeInvariant(trades) *BlindSpot`（纯函数，成交手数>0；调用处 `if bs := ...; bs != nil { resp.Risk.IsReliable=false; resp.BlindSpots=append(...) }`）。**本任务加一个并列的同模式函数，不改 checkVolumeInvariant。**
- 相关类型（已核对，准确）：
  - `backtest.Result.Equity` 是 `[]EquityPoint`，`EquityPoint{Time time.Time; Equity decimal.Decimal; Bar int}` → **期末净值 = `result.Equity[len(result.Equity)-1].Equity`**
  - `backtest.Result.Trades` 是 `[]Trade`，每个 `Profit / Commission / Swap` 均 `decimal.Decimal`
  - `backtest.Result.Config.InitialCapital` 是 `decimal.Decimal` → **本金**

## 任务

1. **实现纯函数** `checkCapitalConservation(result *backtest.Result) *antv1.BlindSpot`：
   - **资金守恒等式**：`|期末净值 − (本金 + ΣProfit − ΣCommission − ΣSwap)| < 容差`
   - 期末净值 = `result.Equity` 末元素；**若 `len(Equity)==0`，vacuously true 返回 nil**
   - 本金 = `result.Config.InitialCapital`
   - ΣProfit / ΣCommission / ΣSwap = 遍历 `result.Trades` 求和
   - **容差** = `max(decimal.New(1, -2) /*0.01*/, InitialCapital.Mul(decimal.New(1, -4)) /*1e-4×本金*/)`（初始值，覆盖浮点累计 + swap/手续费模型小偏差）
   - 违反（差值 ≥ 容差）→ 返回 `&antv1.BlindSpot{Id:"capital_not_conserved", Category:"invariant", Severity:interp.SeverityFatal, Description:"资金不守恒：期末净值与 本金+Σ盈亏−Σ手续费−Σswap 对不上，回测结果不可信"}`
   - 通过 → 返回 `nil`
2. **接入** `buildBacktestResponse`（紧跟 `checkVolumeInvariant` 的 if 之后，并列）：
   ```go
   if bs := checkCapitalConservation(result); bs != nil {
       resp.Risk.IsReliable = false
       resp.BlindSpots = append(resp.BlindSpots, bs)
   }
   ```

## 测试（必须带）

- **正例**：守恒（构造 result：期末净值 = 本金 + ΣProfit − ΣCommission − ΣSwap）→ 返回 nil。
- **负例**：不守恒（期末净值偏离 > 容差）→ 返回 BlindSpot，字段正确。
- **边界**（自行充分补充）：空 Equity（vacuously true）、容差临界（差值刚好 = 容差，判定边界）、单笔交易、零交易。
- **集成测试（必须有）**：`buildBacktestResponse` 接入——守恒时 IsReliable 不被本规则置 false；不守恒时 IsReliable=false + BlindSpot 出现。

## 约束

- 纯函数（与 `checkVolumeInvariant` 同模式），**不改 `checkVolumeInvariant`、不重构无关代码、不自作主张加其他不变量**（最小改动）。
- 不准硬编码、不准 mock。必须 `go build ./...` 与 `go test ./internal/connect/strategy/` 全绿。
- 评审会做对抗验收（spec 未明说的输入 + 检查测试覆盖度），具体 case 不预先公布。

## 验收（评审标准）

- `go build` + `go test` 全绿。
- 读实现：公式对、容差对、空 Equity 处理对、纯函数、接入对（紧跟 volume 之后、不改 volume）。
- 集成测试必须有。
- 不过度设计、无无关改动、无 `//nolint`。

## 待校准（§8 诚实边界，本任务不解决）

容差初始值 `max(0.01, 1e-4×本金)` 是估计，**需基于真实回测数据校准**（如用 xianhua 的成功回测代入，看正常偏差范围）——太严误报、太松漏 bug。本任务先用初始值并标注，校准单独做。

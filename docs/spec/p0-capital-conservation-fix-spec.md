# P0 施工 Spec 02-fix：资金守恒改用已实现 Balance（修正期末未平仓误报）

> **修正 Spec 02 的缺陷**（负责人校准发现）：原 `checkCapitalConservation` 用 Equity（= balance + 未实现浮盈）对等式右边（ΣProfit 已实现），期末有未平仓浮盈时 `diff = 浮盈` → 误报"资金不守恒"。引擎 Run 结束不平仓，真实回测（期末留仓）会被冤枉。**这是原 spec 等式漏了未实现浮盈边界，GLM 实现无误，spec 错了。** 修正：查"已实现 Balance 守恒"，与浮盈无关。

## 改动

1. **`backtest.Result` 加字段**：`FinalBalance decimal.Decimal`（期末已实现余额，= broker.balance，**不含未实现浮盈**）。
   - 文件：`backend/strategy/backtest/types.go` 的 `Result` struct。

2. **`engine.Run` 结束填**：在 return Result 前，`result.FinalBalance = e.broker.balance`。
   - 文件：`backend/strategy/backtest/engine.go`（Run 函数末尾，构造 Result 处）。

3. **`checkCapitalConservation` 改用 FinalBalance**（不再用 Equity 末点）：
   - 等式：`|result.FinalBalance − (本金 + ΣProfit − ΣCommission − ΣSwap)| < 容差`
   - 容差不变：`max(0.01, 1e-4 × 本金)`
   - **移除"空 Equity vacuously true"**（不再依赖 Equity）；改为：始终检查 FinalBalance 守恒。
   - 违反 → 同样 BlindSpot（Id `capital_not_conserved`）。

4. **更新 `capital_conservation_test.go`**：所有 case 改用 FinalBalance（不再构造 Equity）。

## 测试（必须带）

- **正例**：FinalBalance 守恒（= 本金+ΣProfit−ΣCommission−ΣSwap）→ nil。
- **负例**：FinalBalance 偏离 > 容差 → BlindSpot。
- **边界**：空 Trades（FinalBalance 应 = 本金，守恒）、容差临界（= 容差判违反）、单笔、零本金。
- **【关键新增】期末未平仓浮盈仍守恒**：构造 FinalBalance = 本金+Σ已实现−Σ费用（守恒），**与"是否有未平仓浮盈"无关** → 必须返回 nil。这一条验证修正后不再误报期末留仓。
- **集成**：`buildBacktestResponse` 接入（守恒通过/不守恒覆盖 IsReliable）。

## 约束

- 纯函数（checkCapitalConservation）、最小改动（只加 FinalBalance 字段 + 填值 + 改用它 + 更新测试）、不硬编码、不 mock。
- `go build ./...` + `go test ./strategy/backtest/ ./internal/connect/strategy/` 全绿（改了 engine/Result，两个包都要过）。

## 验收（评审，碰钱亲自 review）

- build + 两个包 test 全绿。
- 读实现：FinalBalance 字段加对、engine 填对（=broker.balance）、checkCapitalConservation 用 FinalBalance、等式对、空 Trades 守恒。
- **对抗验收**：构造"期末有大额未平仓浮盈"的 case，确认修正后判守恒（不误报）——这是本次修正的核心，必测。
- 容差仍为初始估计，真实数据校准单独做（§8）。

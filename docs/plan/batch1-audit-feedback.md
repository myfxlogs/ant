# 批次 1 审计反馈 — 施工补漏清单

> **审计**: Claude Code (2026-08-05)
> **状态**: 主体验收通过，4 项补漏
> **参考**: `docs/plan/mql-ea-compatibility-proposal.md` §13.3 批次 1

---

## 审计结论

工程主体合格。47 文件，+894/-165 行，16/18 个 bug 完全修复，超额 7 项。CI 全绿。

以下 4 项需要补充。

---

## 补 1：`builtinIADXWilder` mode 参数忽略（同 IADX bug）

**文件**：`backend/tools/mql2go/vm_builtin_indicators_ext.go:289-295`

**当前**：
```go
func builtinIADXWilder(vm *VM, args []interp.Value) (interp.Value, error) {
    period := int(argI(args, 2))
    // mode = args[4] — IGNORED
    shift := int(argI(args, 5))
    return interp.DecimalVal(vm.ctx.Indicators().ADXWilder(period, shift)), nil
}
```

**要求**：与 `builtinIADX` 同模式——读 mode 参数，MODE_MAIN(0) 返回主线，MODE_PLUSDI(1)/MODE_MINUSDI(2) 返回 0 + 记录盲区。

---

## 补 2：`compile.go resolveVar` typo'd 变量静默注册

**文件**：`backend/tools/mql2go/compile.go:196-200`

**当前**：未知变量名直接注册为新全局 slot（值=0），不报错。

**要求**：加编译错误。逻辑：如果变量名不在 `MQLConstants`、不在 `ir.Enums`、不在 `isSeriesName`、不在用户函数参数中 → 这是拼写错误，不是新全局变量。参考 `compile_expr.go` 的处理方式，设 `c.err`，至少 warning 级别。

**注意**：需要排除 MQL4 的隐式声明模式——某些 MQL4 EA 不声明变量直接赋值。如果 `ir.Version == "mql4"`，可以降级为 warning + 盲区记录，不阻断编译。

---

## 补 3：MACD Sample 测试断言 trades > 0

**文件**：`backend/tools/mql2go/e2e_test.go` `TestE2E_MACD_Sample`

**当前**：只验证编译通过和无 iMACD 盲区，不验证 EA 实际产生了交易。

**要求**：在测试末尾加断言：
```go
// The MACD Sample EA should produce trades on oscillating data.
// If MODE_SIGNAL is broken, MacdCurrent == SignalCurrent and no trades open.
if len(result.Trades) == 0 {
    t.Error("MACD Sample EA produced 0 trades on oscillating data — MODE_SIGNAL may still be broken")
}
```

注意：`result.Trades` 记录的是已平仓交易，但至少应该有。如果数据确实没触发→用 `t.Log` 代替 `t.Error`，由人工判断。

---

## 补 4：补充回归测试

新增以下测试（可以在 `e2e_test.go` 或新建 `e2e_regression_test.go`）：

### 4a. `TestE2E_ClassicStartEntry` — 验证 `start()` 入口映射
- 构造经典 MQL4 源码：`int start() { OrderSend(...); return(0); }`
- 编译应成功，`start()` 被映射到 `OnTick`
- 回测应执行（产生 trades 或 equity 变化）

### 4b. `TestE2E_OrderSelectHistory` — 验证 MODE_HISTORY 池
- EA 先开仓再平仓，然后用 `OrderSelect(i, SELECT_BY_POS, MODE_HISTORY)` 遍历
- 验证 `OrdersHistoryTotal() > 0`
- 验证 `OrderClosePrice()` 和 `OrderCloseTime()` 返回非零值

### 4c. `TestE2E_IADX_Mode` — 验证 ADX mode 盲区记录
- EA 调用 `iADX(NULL,0,14,PRICE_CLOSE,MODE_PLUSDI,0)`
- 编译应成功（MODE_PLUSDI 已知常量）
- 运行后 `GetRuntimeBlindSpots()` 应包含 `"iADX:MODE_PLUSDI"` 盲区
- 同理测 `MODE_MAIN` → 无盲区，`MODE_MINUSDI` → 有盲区

---

## 修复后验证

```bash
cd backend
go build ./...
go test ./tools/mql2go/... -v -run "TestE2E"
go test ./strategy/backtest/... -v
go run ./tools/check-file-lines --strict
```

全部通过后 push，通知我复核。

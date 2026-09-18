# Builder Handoff — ACCOUNT-MARGIN-LEVEL-PCT-1

> `AccountInfoDouble(ACCOUNT_MARGIN_LEVEL)` 返回比率而非 MQL5 百分比——100× 语义偏差。
> 设计 SSOT：Devin CLI。施工只执行、不决策。完成报证据等独立复审。

## 0. 立项证据链（设计实查已核实）

**缺陷**：`backend/tools/mql2go/vm_builtin_mql5_info.go:60-64`：

```go
case 6: // ACCOUNT_MARGIN_LEVEL
    if vm.ctx.Account().Margin.IsZero() {
        return interp.DecimalVal(decimalZero), nil
    }
    return interp.DecimalVal(vm.ctx.Account().Equity.Div(vm.ctx.Account().Margin)), nil
```

返回 `Equity/Margin` **比率**（如 9.92）。

**官方语义实证**：MQL5 官方文档 `ACCOUNT_MARGIN_LEVEL` = "Account margin level **in percents**" = `equity/margin*100`——官方示例 equity=9921.24/margin=1000 → `ACCOUNT_MARGIN_LEVEL=992.12`。本实现得 9.92，**100× 偏差确认**（非疑似）。佐证：mtapi AccountSummary f7 `MarginLevel` 注释 "Margin percent"、`mt_accounts.margin_level` 亦按百分比存储——全栈约定一致，VM 是唯一异类。

**后果**：任何用 `AccountInfoDouble(ACCOUNT_MARGIN_LEVEL)` 做保证金阈值判断的策略（margin call/stop-out 风控）在 VM 下阈值错 100×——真 MQL5 语义下 `level > 500` 的判定在本 VM 永不触发。

**边界实况**（已核）：
- `Margin.IsZero()` → 返回 0：**保留**（无持仓时 MQL5 约定值；真 MT5 报 0）。
- `sdk.AccountInfo.MarginLevel` 字段存在但**无任何写入方/读取方**（死字段）——本债不动字段本身，builtin 本地重算 Equity/Margin 即权威（两输入字段已接）；死字段清理另债不阻。
- `AccountMarginLevel` MQL4 式 builtin 不存在于 api_registry——无需同步。
- 存量测试零 margin-level 值断言（`vm_api_truth_test.go` 只测 prop 存在性/编译层）——无 IR/行为断言波及。
- `PositionSnapshot.MarginLevel`/`mt_accounts.margin_level` 是 broker 直读值——**不改用直读**：VM 内 Equity/Margin 已是权威输入，本地重算与直读一致性等价且更简单（直读会引入第三条同步链路）。

## S1 — 修复（`tools/mql2go/vm_builtin_mql5_info.go:64`）

```go
// MQL5 ACCOUNT_MARGIN_LEVEL is a percentage (equity/margin*100, official
// doc: 9921.24/1000 → 992.12) — not a ratio. ACCOUNT-MARGIN-LEVEL-PCT-1.
return interp.DecimalVal(vm.ctx.Account().Equity.Div(vm.ctx.Account().Margin).Mul(decimalHundred)), nil
```

**坐标修正**：本文件**未 import** shopspring/decimal——`decimalZero` 是包级变量（`vm_builtin_math.go:97` `var decimalZero = safeDecimalFromFloat(0)`）。在同处加 `var decimalHundred = safeDecimalFromFloat(100)` 并用之（包级惯例一致）。

## S2 — pin 测试（`tools/mql2go/` 测试文件，沿用 vm_api_truth/harness 测试惯例）

**T1 百分比断言**：构造 VM ctx 使 `Account().Equity=9921.24, Margin=1000`（harness/测试 broker 任一可用通道）→ VM 执行 `AccountInfoDouble(ACCOUNT_MARGIN_LEVEL)` → 断言返回值 `==992.124`（DecimalVal 精确比较或 InexactFloat64 近似 1e-9）——**非** 9.92124。

**T2 zero-margin 边界**：`Margin=0` → 断言返回 `0`（现行语义保留 pin——防回归误改 error 或 inf）。

## 验收门

1. `cd backend && go build ./...`
2. `go test ./tools/mql2go/` 全绿（新增 T1/T2 + 存量回归）
3. `go test ./tools/mql2go/ -race -count=3`
4. `go vet ./tools/mql2go/` / `gofmt -l` 改动文件净 / `git diff --check`
5. `go run ./tools/check-file-lines --strict`
6. **先红后绿**：M1 摘除 `.Mul(100)` → T1 应 RED（得 9.92124）；T2 不受影响仍过（边界独立）。restore→GREEN。
7. **禁止**：动 `Margin.IsZero` 早退分支、动 `sdk.AccountInfo.MarginLevel` 死字段（另债）、动 SO_CALL/SO_SO 等其余 prop、引入 PositionSnapshot/mt_accounts 直读链路。

## 范围边界

只改 `vm_builtin_mql5_info.go` 一行 + 一个测试文件。registry/STATE.md 状态更新到 🟦open（施工完成待复审）后停手，勿部署勿 push。

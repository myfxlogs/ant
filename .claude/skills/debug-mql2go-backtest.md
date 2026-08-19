---
description: Debug mql2go backtest issues where the MQL4/MQL5 EA works on MT4/MT5 but fails in AlphaForge backtest
---

# Debug mql2go Backtest Issues

## When to use

- Strategy opens zero trades in backtest but works on MT4/MT5.
- Backtest intermittently produces `volume=0` trades.
- Indicator values look wrong or all zeros.
- Same code and same test command sometimes PASS, sometimes FAIL.

## Step-by-step

1. **Check blind spots first**
   - Look at `CoverageReport.BlindSpots` for `"unknown constant"` or `"unknown function"`.
   - Missing constants in `backend/tools/mql2go/interp/constants.go` cause the compiler to silently push `0`.

2. **Verify indicator mode constants**
   - `MODE_MAIN = 0`, `MODE_SIGNAL = 1`
   - `MODE_PLUSDI = 1`, `MODE_MINUSDI = 2` (iADX)
   - `MODE_UPPER = 1`, `MODE_LOWER = 2` (iBands, iEnvelopes)
   - `MODE_BASE = 0` (iAlligator, iGator)
   - `MODE_TENKAN = 1`, `MODE_KIJUN = 2`, `MODE_SENKOU_A = 3`, `MODE_SENKOU_B = 4`, `MODE_CHIKOU = 5` (iIchimoku)
   - If any are missing, `iMACD` / `iStochastic` / `iADX` etc. return the wrong line.

3. **Check `builtinOrderType` mapping**
   - MQL4 expects `OP_BUY = 0`, `OP_SELL = 1`.
   - Returning `PositionSide` (`SideBuy = 1`, `SideSell = -1`) breaks position management.

4. **Check for map-order flakiness**
   - `ir.Funcs` is a map; the compiler must use two-pass compilation.
   - If a caller is compiled before its callee, `compileCall` falls through to `"unknown function"` and the return value becomes `NoneVal` (`0`).
   - Reproduce with:

     ```bash
     for i in $(seq 1 50); do go test ./tools/mql2go/ -run <TestName> -count=1 -v 2>&1 | grep -E 'PASS|FAIL'; done
     ```

5. **Add temporary builtin logging**
   - In `backend/tools/mql2go/vm_builtin_*.go`, add:

     ```go
     fmt.Fprintf(os.Stderr, "[builtin] %s args=%v\n", name, args)
     ```

   - Re-run and compare PASS vs FAIL logs.

6. **Trace `compileCall`**
   - Path: user func → builtin → API registry → unknown fallback.
   - Each layer can silently swallow the call.

## Key files

- `backend/tools/mql2go/compile.go` — two-pass compiler
- `backend/tools/mql2go/compile_expr.go:compileCall` — call resolution
- `backend/tools/mql2go/interp/constants.go` — MQL constants
- `backend/tools/mql2go/vm_builtin_trade.go` — `OrderSend`, `OrderType`
- `backend/tools/mql2go/vm_builtin_indicators.go` — `iMACD` etc.
- `backend/tools/mql2go/vm.go` — `initGlobals`

## Reference

- `docs/runbook/mql2go-known-pitfalls.md`

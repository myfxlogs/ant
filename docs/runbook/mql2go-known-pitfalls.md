# MQL2GO VM Known Pitfalls

> **For all agents and developers**: Read this before debugging "backtest opens no trades" or "position management broken" issues with MQL4 EAs.

## Pitfall 1: Missing indicator mode constants → iMACD returns wrong line (CRITICAL)

### Symptom
MQL4 EA works correctly on MT4 client but **never opens any trades in backtest**. No errors, no warnings — strategy simply produces zero trades.

### Root Cause
`backend/tools/mql2go/interp/constants.go` must define **all** MQL4/MQL5 indicator mode constants. If a constant is missing, the compiler (`compile_expr.go:54-58`) silently pushes **0** and records a blind spot — no error, no warning.

The MACD Sample EA uses:
```mql4
MacdCurrent   = iMACD(NULL,0,12,26,9,PRICE_CLOSE,MODE_MAIN,0);
SignalCurrent = iMACD(NULL,0,12,26,9,PRICE_CLOSE,MODE_SIGNAL,0);
```

If `MODE_SIGNAL` is missing → resolved to 0 → `builtinIMACD` returns MACD line instead of signal line → `MacdCurrent == SignalCurrent` → entry condition `MacdCurrent > SignalCurrent` is always false → **zero trades**.

### Constants that MUST exist in `interp/constants.go`

**Indicator line selector modes (ENUM_INDEXBUFFER — used by iMACD, iStochastic, iADX, iBands, iEnvelopes, iFractals):**
```
MODE_MAIN    = 0
MODE_SIGNAL  = 1
MODE_PLUSDI  = 1   (iADX)
MODE_MINUSDI = 2   (iADX)
MODE_UPPER   = 1   (iBands, iEnvelopes)
MODE_LOWER   = 2   (iBands, iEnvelopes)
MODE_BASE    = 0   (iAlligator, iGator)
MODE_TENKAN  = 1   (iIchimoku)
MODE_KIJUN   = 2   (iIchimoku)
MODE_SENKOU_A= 3   (iIchimoku)
MODE_SENKOU_B= 4   (iIchimoku)
MODE_CHIKOU  = 5   (iIchimoku)
```

### How to verify
```bash
# Check if any indicator mode constant is missing
grep -E 'MODE_MAIN|MODE_SIGNAL|MODE_PLUSDI|MODE_MINUSDI|MODE_UPPER|MODE_LOWER|MODE_BASE|MODE_TENKAN|MODE_KIJUN|MODE_SENKOU|MODE_CHIKOU' backend/tools/mql2go/interp/constants.go
```

### Debugging tip
When a strategy opens zero trades in backtest but works on MT4:
1. Check `CoverageReport.BlindSpots` for "unknown constant" entries
2. Compare every constant used in the MQL source against `MQLConstants` map
3. The blind spot is **silent** — it does not cause an error, just pushes 0

---

## Pitfall 2: builtinOrderType returns wrong values (HIGH)

### Symptom
MQL4 EA opens trades in backtest but **position management fails** — trailing stops, close conditions, and position type checks never trigger correctly.

### Root Cause
`vm_builtin_trade.go:builtinOrderType` returns `vm.currentPos.Side` which is `PositionSide` (`SideBuy=1`, `SideSell=-1`), but MQL4 expects `OP_BUY=0`, `OP_SELL=1`.

```go
// WRONG — returns PositionSide enum (1 / -1)
func builtinOrderType(vm *VM, args []interp.Value) (interp.Value, error) {
    if vm.currentPos != nil {
        return interp.IntVal(int32(vm.currentPos.Side)), nil
    }
    return interp.IntVal(0), nil
}
```

MQL4 EA code like `if(OrderType()==OP_BUY)` compares against `OP_BUY=0`, but gets `1` for a buy position → condition never true.

### Fix
Map `SideBuy(1) → 0 (OP_BUY)`, `SideSell(-1) → 1 (OP_SELL)` before returning.

---

## Pitfall 3: Fixed-length live window freezes incremental indicator cache (CRITICAL)

### Symptom

Live strategy status remains Active, tick/bar evaluations and "last evaluated" continuously advance, the window has 500 bars, account state is fresh, but the strategy permanently produces zero signals. Runtime diagnostics show EMA/MACD/RSI/ATR/ADX values never change after startup.

### Root Cause

Live startup seeds exactly `maxContextBars=500`. Every new closed bar runs `appendDedupBar`, which drops the oldest bar and appends the newest, so the window length remains 500.

`indicatorSet.ensureCache()` updates `src.bars`, but `SeriesCache.EnsureUpdated()` detects changes only through `Len()`:

```go
n := c.src.Len()
for i := n - c.n - 1; i >= 0; i-- {
    c.processBar(i)
}
c.n = n
```

After the first evaluation, `c.n=500`. Every rolling update is also `n=500`, so the loop never executes. All cache-backed indicators remain frozen at the startup bar while VM evaluation counters continue increasing. This is a cache-invalidation protocol error: the cache assumes append-only input, but live execution provides a bounded rolling window.

Production evidence (LIVE-INDICATOR-1, 2026-08-20): schedule `599ddaa5`, run `3d2184b5` had 4817 evaluations (46 bar + 4771 tick), `Window Bars=500`, `Orders Total=0`; diagnostics `MACD.main=0.23718981101410463`, `MACD.signal=8.119201031042982`, `EMA26=69559.67407019278` matched the 00:44 startup frame exactly. The 00:51 SELL condition was true under the same algorithm but produced no signal.

### Required Fix and Test

- Establish an optional `RevisionedBarSource` protocol between runner and `SeriesCache`; normal append-only backtest sources remain unchanged.
- For revisioned sources, **any revision change** resets and lazily rebuilds all cached series. Do not infer “append-only” merely because length grew; revision means content changed.
- Runner advances revision only on `OnBar`; tick/trade/timer evaluations over the unchanged window must not rebuild.
- Adversarial test must mutate and reuse the **same source and same cache**: initialize 500 bars, query indicators, drop oldest + append newest (length stays 500), increment revision, query again, and compare all cache-backed indicators with stateless results. Creating a second cache makes the test pass even when production invalidation is broken.
- VMLiveSession integration must reproduce legacy MQL4 `start()` semantics: initial BAR → TICK, rolling BAR → next TICK emits the crossover signal. An OnBar-only fixture does not fully cover the production path.
- Each VMLiveSession owns one Runner/SeriesCache and dispatches events serially; verify this with `-race`. Do not add a partial mutex only around `EnsureUpdated`, because lazy indicator query methods also mutate the maps.

### Debugging Tip

When evaluations advance but no trades occur, compare runtime indicator values against values recomputed from the latest bars. If diagnostics match the startup frame rather than current bars, inspect cache invalidation before investigating strategy conditions, risk gates, or broker execution.

---

## General Lesson

The mql2go VM's most dangerous failures are silent: unknown constants become zero, enum mismatches disable branches, and stale caches return plausible but obsolete indicator values. When an EA works on MT4/MT5 but not on the platform, verify both coverage blind spots and that runtime indicator values advance with new bars.

The full list of MQL4/MQL5 constants is large and spread across many enum groups. The `interp/constants.go` file should be treated as a potential source of silent bugs. When adding support for a new indicator or MQL function, verify all constants it references exist in the map.

---

## Pitfall 3: Map iteration non-determinism → user function calls silently return 0 (CRITICAL)

### Symptom
Backtest **intermittently** produces `volume=0` trades. Same code, same command (`go test -count=1`), different results across runs. The flaky behavior masks the root cause — passing runs hide the bug.

### Root Cause
`ir.Funcs` is a `map[string]*FuncDef`. `compile.go` iterated this map to compile user functions. **Go map iteration order is non-deterministic** (randomized by runtime).

When a caller function (e.g. `CheckForOpen`) is compiled **before** its callee (e.g. `LotsOptimized`):
1. `compileCall("LotsOptimized")` checks `bc.Funcs["LotsOptimized"]` → not found (not yet compiled)
2. Falls through to "unknown function" blind spot
3. Arguments are compiled then **popped/discarded**
4. Return value silently replaced with `NoneVal` (=0)
5. `OrderSend(Symbol(), OP_BUY, LotsOptimized(), Ask, ...)` becomes `OrderSend(Symbol(), OP_BUY, 0, Ask, ...)` → **volume=0**

### Fix (applied 2026-08-07)
Two-pass compilation in `compile.go`:
1. **Pass 1**: Pre-register all user function entry PCs (emit `OP_ENTER_FUNC` + write `bc.Funcs` with placeholder `NumLocals`)
2. **Pass 2**: Compile each function body, updating `NumLocals` and `ParamName`

All forward references resolve after Pass 1 — `compileCall` always finds the callee in `bc.Funcs`.

### General Rule
**Never iterate a Go map when order matters.** In compilers, linkers, and any ordered pipeline:
- If entries reference each other (forward references), pre-register all entries in a first pass
- If order doesn't matter but determinism is required (e.g. tests), sort keys before iterating
- Map iteration randomness is per-invocation, not per-process — calling the same function twice in a row can produce different orderings

### How to verify
```bash
# Run the flaky test 50 times — should be 0 failures after fix
for i in $(seq 1 50); do go test ./tools/mql2go/ -run TestParamPipeline_FloatDefaultParam -count=1 -v 2>&1 | grep -E 'PASS|FAIL'; done
```

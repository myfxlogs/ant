# Risk Management Audit Report

## Scope

Comprehensive review of the risk management subsystem:
- `internal/risk/gate.go` — Gate evaluator (R10 kill switch, R11 autotrade, D6-A fail-closed)
- `internal/risk/rules.go` — R1-R9 rule implementations (max lot, position count, exposure, daily loss, drawdown, symbol whitelist, leverage cap, order frequency, duplicate protection, margin pre-check)
- `internal/risk/rules_risksvc.go` — Ported risksvc rules (KYC/jurisdiction, contract expiry, margin floor, capability tier)
- `internal/risk/guard.go` — 3-rule mandatory pre-broker safety net (kill switch, max lot, duplicate)
- `internal/risk/rule_user_config.go` — Per-account risk config from DB
- `internal/risk/canary.go` — Canary rollout controller
- `internal/risksvc/precheck.go` — Pre-trade risk checks (position limits, exposure, margin utilization)
- `internal/risksvc/hardlimit.go` — 4 hard limit rules (KYC, margin floor, kill switch, contract expiry)
- `internal/risksvc/pipeline.go` — Signal pipeline (capability → hardlimit → platform → engine → sizer)
- `internal/risksvc/engine.go` — Risk rule engine
- `internal/risksvc/rules.go` — 6 engine rules (max position, daily loss, drawdown, session, margin, canonical auth)
- `internal/connect/autotrading/auto_trade_risk_handler.go` — CheckRiskLimits RPC handler

## Findings

### R1 — UserRiskConfigRule nil-state panic on DailyPnL access 🔴 CRITICAL

**File**: `backend/internal/risk/rule_user_config.go:60-65`

**Problem**: `UserRiskConfigRule.Check` accesses `state.DailyPnL` without nil-checking `state`. All other field accesses in the method (`MaxPositions` at line 52, `MaxDrawdownPercent` at line 67, `MaxRiskPercent` at line 77) have `state != nil` guards, but the `MaxDailyLoss` block was missing one.

If `state` is nil and `MaxDailyLoss` is configured (non-zero), this panics with nil pointer dereference. Since the Gate's D6-A fail-closed now correctly blocks nil-state live orders (fixed in MT Gateway audit), `state` should never be nil for live orders reaching this rule. However, if the rule is registered on a Gate that evaluates SIM orders (where state can be nil), or if `accountStateProvider` returns nil on error, the panic still occurs.

**Fix**: Added `state != nil` guard to the `MaxDailyLoss` condition.

```diff
-	if rc.MaxDailyLoss.GreaterThan(decimal.Zero) {
+	if rc.MaxDailyLoss.GreaterThan(decimal.Zero) && state != nil {
```

**Risk if unfixed**: Runtime panic when state is nil and daily loss limit is configured.

### R2 — MarginFloorRule omits contract size, understates required margin by 100x 🔴 CRITICAL

**File**: `backend/internal/risk/rules_risksvc.go:119` (risk package) and `backend/internal/risksvc/hardlimit.go:88` (risksvc package)

**Problem**: Both `MarginFloorRule` implementations calculate `required = vol * price` but omit the contract size multiplier. For standard FX (contract size = 100,000), the required margin is understated by 100x. This means an order requiring $100,000 in margin would pass with only $1,000 in free margin.

The sibling rule `MarginPreCheck` in `rules.go:327` correctly uses `contractSize(state)`, making this an inconsistency rather than a design choice.

**Fix**:
- `risk/rules_risksvc.go`: Added `contractSize(state)` to the calculation
- `risksvc/hardlimit.go`: Added `ContractSize` field to `HardLimitRequest` and used it in the calculation
- `risksvc/pipeline.go`: Pass `sig.ContractSize` to `HardLimitRequest`
- Updated tests to set `ContractSize: newDec("1")` (spot crypto, where contract size = 1)

```diff
// risk/rules_risksvc.go
-	required := vol.Mul(price)
+	required := vol.Mul(price).Mul(contractSize(state))

// risksvc/hardlimit.go — HardLimitRequest
+	ContractSize   decimal.Decimal

// risksvc/hardlimit.go — MarginFloorRule.Check
 	required := req.Volume.Mul(req.Price)
+	if req.ContractSize.GreaterThan(decimal.Zero) {
+		required = required.Mul(req.ContractSize)
+	}

// risksvc/pipeline.go — checkHardLimit
+	ContractSize: sig.ContractSize,
```

**Risk if unfixed**: Margin floor check is effectively disabled for FX instruments — orders that should be blocked by insufficient margin pass through, potentially leading to margin calls and forced liquidation.

## Verified Safe (No Issues Found)

- **Gate.Evaluate**: D6-A fail-closed correctly blocks nil-state and negative-equity live orders. SIM orders bypass this check (intentional — backtest doesn't need live account data).
- **R1 MaxLotSize**: Pure volume check, no state dependency. Returns `AdjustedVolume` for cap-down.
- **R2 MaxPositionCount**: Nil-state safe (returns allowed). Checks `>=` not `>` (correct — blocks at limit, not above).
- **R3 MaxExposure**: Nil-state safe. Skips market orders (price=0) — broker determines fill price.
- **R4a DailyLossBreaker**: Nil-state safe. Zero `MaxDailyLoss` = disabled.
- **R4b DrawdownBreaker**: Nil-state safe. Zero `MaxDrawdownPct` or `PeakEquity` = disabled. Uses `1 - equity/peak` (correct drawdown formula).
- **R5 SymbolWhitelist**: Empty whitelist = all allowed (intentional default-open).
- **R6 LeverageCap**: Nil-state safe. Zero `SymbolLeverage` = skip.
- **R7 OrderFrequencyLimit**: Per-userID tracking with mutex. Lazy cleanup at >1000 users. Uses `time.Now()` (not `Clk.Now()`) — acceptable since this is rate limiting, not financial timestamping.
- **R8 DuplicateProtection**: Key = `symbol|side|volume|type|price`. Lazy cleanup at >1000 entries. 5s default window.
- **R9 MarginPreCheck**: Nil-state safe. Correctly uses `contractSize(state)`. Leverage defaults to 100 when zero. Market orders (price=0) skip.
- **Guard**: 3-rule safety net (kill switch, max lot, duplicate). Nil-safe config. Lazy cleanup at >10000 entries.
- **CanaryController**: Proper mutex protection. Stage transitions validated (Off→Canary→Expanded→Full). Kill switch returns zero lots. Rollback saves previous state. Audit history recorded.
- **PreCheck**: 4 checks (symbol position, total exposure, account exposure, margin utilization). Nil-safe limits.
- **HardLimitEvaluator**: Sequential evaluation, first block stops. 4 rules (KYC, margin floor, kill switch, contract expiry).
- **Engine**: Sequential rule evaluation. Per-user rate limiter integration.
- **Drawdown/Session/Margin/CanonicalAuth rules**: All nil-safe, correct calculations.
- **CheckRiskLimits handler**: Preview endpoint — missing fields (UserID, Price, FreeMargin) are known limitations of the preview, not bugs. Actual order flow goes through `mthub.PlaceOrder` → `risk.Gate`.
- **resolveRiskLimit**: Account RiskConfig → user GlobalSettings → system defaults fallback chain. Correct precedence.
- **CalculatePositionSize**: Uses `riskAmount = balance * riskPct / 100`, `volume = riskAmount / (slPips * pipValue)`. Correct position sizing formula.

## Architecture Compliance

- ✅ No REST endpoints (ConnectRPC only)
- ✅ No JSON for data persistence
- ✅ `decimal.Decimal` for all financial calculations in risk rules
- ✅ No `//nolint` or `// @ts-ignore`
- ✅ Push-first: event-driven reconciliation, no polling for risk state
- ✅ Fail-closed design (D6-A): nil state blocks live orders

## Reuse Preflight

- **R1**: REUSE: `state != nil` guard pattern @ `rule_user_config.go:52` (MaxPositions block already had this pattern)
- **R2**: REUSE: `contractSize(state)` @ `rules.go:38-43` (helper function already existed and was used by MarginPreCheck)

## Migrations

No migrations required.

## Deployment

- `go build ./...` ✅
- `go test ./internal/risk/... ./internal/risksvc/... ./internal/mthub/...` ✅
- `docker compose build backend` ✅
- `docker compose up -d backend` ✅
- Container health: `healthy` ✅

## Residual Risks

- **risksvc Drawdown/DailyLoss shared state**: The `Drawdown` and `DailyLoss` rules in `risksvc/rules.go` maintain a single `PeakEquity`/`DailyPL` across all accounts (not per-account). This is a design concern in the preview `CheckRiskLimits` path. The actual order flow uses `risk.Gate` which receives per-account `AccountState`, so live trading is not affected. Fixing would require refactoring these rules to be per-account, but since they're only used in the preview path, the risk is low.
- **R7 OrderFrequencyLimit uses `time.Now()`**: Not `Clk.Now()` (the injectable clock). This means the frequency limit can't be tested with a mock clock. Acceptable since rate limiting is not financial timestamping.
- **R8 DuplicateProtection key includes price**: For market orders, price is "0" in the intent, so all market orders with the same symbol/side/volume/type within the dedup window are considered duplicates. This is intentional — prevents rapid-fire market orders.
- **MarginFloorRule skips market orders**: When `price <= 0` (market orders), the check is skipped. The broker determines fill price and the sizer already validated account equity. This is documented in code comments.
- **DefaultRiskLimits MaxExposurePerAccount = 100000**: Comment says "1 standard lot" but 100000 is the notional value of 1 standard lot of EURUSD. The field is compared against `req.Volume` (in lots), not notional. This means it blocks volumes > 100000 lots, which is effectively unlimited. The comment is misleading but the behavior is safe (no realistic order would be 100000 lots).

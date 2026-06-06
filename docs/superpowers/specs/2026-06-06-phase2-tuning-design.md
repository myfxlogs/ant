# Phase 2 — Parameter Optimization Gap-Closing Design

**Date:** 2026-06-06
**Status:** Design complete — audited for optimality
**Scope:** 4 critical gaps to make Phase 2 production-ready
**Reference:** Phase 1 design: `docs/superpowers/specs/2026-06-06-ai-prompt-context-design.md`

---

## Executive Summary

Phase 2 (Smart Tuning parameter optimization) is ~85% complete. Four critical gaps prevent production use:

| # | Gap | Severity | Files affected | LOC |
|---|-----|----------|---------------|-----|
| 1 | searchMethod always sends `random` for DE/TPE/AGS/AI | 🔴 Blocker | 1 (frontend) | 1 |
| 2 | OOS validation exists but never wired | 🔴 Blocker | 4 (backend + DB) | ~100 |
| 3 | Market regime detected but never stored | 🟡 Major | 3 (backend + DB) | ~45 |
| 4 | OOS proto fields defined but Go/DB never populated | 🟡 Major | 3 (backend + DB) | ~55 |

**Total estimated change:** ~200 lines across 8 files + 1 DB migration.

---

## Architecture Overview

```
┌──────────────────────────────────────────────────────────────────────────┐
│                        Smart Tuning Pipeline                              │
│                                                                          │
│  Frontend                    Backend                             Storage  │
│  ────────                    ───────                             ───────  │
│                                                                          │
│  SmartTuningPanel  ──SSE──▶  ExperimentWorker  ──SQL──▶  strategy_       │
│  │                           │                                experiments│
│  │ tuneMethod ──────────▶    │ processOne()                            │
│  │ (grid|random|             │  │                                      │
│  │  de|tpe|ags|ai)    ◀──FIX│  ├─ 1. Claim experiment + detect regime  │
│  │                           │  ├─ 2. runOptimizer() (IS backtests)    │
│  │                           │  │    ├─ GridSearch     (one-shot)      │
│  │                           │  │    ├─ RandomSearch   (one-shot)      │
│  │                           │  │    ├─ DEOptimizer    (ask/tell)      │
│  │                           │  │    ├─ TPEOptimizer   (ask/tell)      │
│  │                           │  │    ├─ AGS Optimizer  (ask/tell)      │
│  │                           │  │    └─ AI Proposal    (multi-round)   │
│  │                           │  ├─ 3. Select top-K by IS score    ◀NEW │
│  │                           │  ├─ 4. OOS backtests for top-K     ◀NEW │
│  │                           │  │    ├─ Split 70/30 time window        │
│  │                           │  │    ├─ Run OOS backtest               │
│  │                           │  │    └─ Validate() degradation         │
│  │                           │  └─ 5. Store all + OOS on top-K    ◀NEW │
│  │                           │                                      │
│  │                           │  detectRegime()                      │
│  │                           │    └─ Once per experiment  ◀──FIX    │
│  │                           │       stored in market_regime_ref    │
│  └───────────────────────────┘                                      │
└──────────────────────────────────────────────────────────────────────┘
```

---

## §1 — searchMethod Fix (Frontend)

### Problem

`frontend/src/pages/strategy/hooks/useBacktestParams.ts:167`:

```ts
searchMethod: tuneMethod === 'grid' ? 'grid' : 'random'
```

User selects DE/TPE/AGS/AI → backend receives `random`. The advanced optimizers are unreachable from the UI.

### Root Cause

Defensive coding pattern — the ternary was written when only `grid` and `random` existed, then never updated when 4 more methods were added.

### Fix (1 line)

```ts
// BEFORE
searchMethod: tuneMethod === 'grid' ? 'grid' : 'random',

// AFTER
searchMethod: tuneMethod,
```

`TuneMethod` type is `'grid' | 'random' | 'de' | 'tpe' | 'ags' | 'ai'`. The backend's `runOptimizer()` switch-case (worker.go:159-172) already handles all 6 values:

```go
case "de":     return w.runIterative(ctx, ai.NewDEOptimizer(...))
case "ai":     return w.runAIProposal(...)
case "tpe":    return w.runIterative(ctx, ai.NewTPEOptimizer(...))
case "ags":    return w.runIterative(ctx, ai.NewAnnealedGaussianOptimizer(...))
case "random": return w.runOneShot(ctx, ai.RandomSearch(...))
default:       return w.runOneShot(ctx, ai.GridSearch(...))   // "grid"
```

### Optimality assessment

| Approach | Verdict | Reason |
|----------|---------|--------|
| Direct pass-through `tuneMethod` | ✅ **Optimal** | Zero transformation, union type already constrains valid values, backend switch-case is the single dispatch point |
| Add enum mapping layer | ❌ | Redundant indirection — no value add |
| Validate on both sides | ❌ | Violates single source of truth (backend is authoritative) |

---

## §2 — OOS Validation Wiring

### Problem

`oos_validator.go` is complete and correct — `DefaultOOSValidator()`, `ComputeWindows()`, and `Validate()` have 100% coverage of the OOS validation logic. But no code path calls them.

`backtestAndScore()` runs ONE backtest on the full time window, scores it, and returns. No out-of-sample check exists.

### Design

#### 2.1 Core Insight: Top-K OOS (not per-candidate)

Naive approach: run IS + OOS backtest for every candidate → 2N backtests. For N=20, that's 40 backtests. But:

- **Iterative optimizers** (DE/TPE/AGS): OOS scores are never fed back via `Tell()`. OOS data for non-winning candidates is discarded.
- **One-shot optimizers** (Grid/Random): Users only care about top-ranked candidates. OOS data for rank #15 is noise.
- **Correctness**: OOS is a diagnostic, not an optimization target (§2.3).

**Optimal solution: IS backtests for all N candidates (needed for scoring/ranking), OOS backtests only for top-K (K=5).**

Cost: N + K backtests instead of 2N. For N=20: 25 vs 40 (37.5% fewer).

#### 2.2 Data Flow

```
processOne(ctx)
  │
  ├─ 1. Compute OOS windows (once, shared by all top-K)
  │     v := DefaultOOSValidator()
  │     windows := v.ComputeWindows(exp.FromTs, exp.ToTs)
  │     hasOOS := windows != nil
  │
  ├─ 2. runOptimizer() — IS backtests only
  │     for each candidate:
  │       isResult := runSingleBacktest(params, [full_start, full_end], regime)
  │       results ← candidateResult{Score: isResult.Score, ...}
  │       // Iterative optimizers receive IS score via Tell()
  │
  ├─ 3. Select top-K by IS score
  │     topK := selectTopK(results, 5)
  │
  ├─ 4. OOS backtests for top-K only
  │     for each candidate in topK:
  │       oosResult := runSingleBacktest(params, [windows.OOSStart, windows.OOSEnd], regime)
  │       validation := v.Validate(candidate.Score, oosResult.Score)
  │       candidate.OOSScore = &oosResult.Score
  │       candidate.DegradationPct = &validation.Degradation
  │       candidate.IsOverfit = validation.IsOverfit
  │
  └─ 5. Store all candidates (top-K have OOS data, rest have nil)
       for each candidate in results:
         CreateCandidate(...)
```

**No OOS data feeds into the optimizer loop.** The iterative optimizers receive IS scores only via `Tell()`, maintaining the strict separation between optimization signal and diagnostic metadata.

#### 2.3 Key Decision: IS score for ranking, OOS for metadata

The IS score determines candidate ranking. OOS score, degradation %, and is_overfit flag are **metadata** — displayed in the UI but not used for ranking.

**Why this is the only correct approach:**

| Approach | Problem |
|----------|---------|
| Rank by IS, OOS as metadata | ✅ No data leakage, correct separation |
| Rank by OOS score | 🔴 Data leakage: optimizer sees OOS data → overfit to "test" set |
| Combined IS+OOS score | 🔴 Arbitrary weighting + partial leakage |
| Penalize IS by degradation | 🟡 Changes search landscape for iterative optimizers |

If we ranked by OOS score, the optimizer would effectively optimize on the full dataset (IS + OOS), defeating the purpose of the train/test split. OOS exists purely as a diagnostic: "here's how this parameter set performed on unseen data."

#### 2.4 Why K=5?

- Covers the candidates a user would reasonably inspect and potentially apply
- ~37% fewer backtests vs per-candidate OOS (at N=20)
- UI typically shows top-5 in the results card; users rarely scroll past #5
- Configurable via `oosTopK` constant for future tuning

#### 2.5 Degradation Threshold

```go
MaxDegradation: 0.4  // 40% score drop → flagged as overfit
```

QuantDinger-derived default. A candidate whose OOS score drops >40% from IS score is marked `is_overfit = true`.

#### 2.6 Fallback: Short Time Windows

`ComputeWindows()` returns `nil` when `totalDays < MinTrainDays + MinOOSDays` (default: 37 days). In this case, the entire OOS phase is skipped — all candidates get nil OOS fields.

Frontend interprets "no OOS fields" as "OOS not available for this time window" → shows muted "N/A."

#### 2.7 Code Restructuring

To support IS-only backtests (needed by the optimizer loop) and OOS-only backtests (needed for top-K), extract the backtest execution into a shared helper:

```go
// runSingleBacktest executes one backtest with the given time window,
// polls for completion, scores with regime-aware weights, and returns metrics.
func (w *ExperimentWorker) runSingleBacktest(
    ctx context.Context, code string, overrides map[string]interface{},
    symbol, timeframe string, fromTs, toTs time.Time,
    initialCapital float64, regime ai.MarketRegime,
) (*ai.ScoredResult, *antv1.ExecuteBacktestResponse, error) {
    // Create BacktestRun with specified time window
    // Poll for completion (reuse existing polling logic)
    // Parse proto response
    // Score with ai.Score(metrics, regime)
    // Return scored result + raw metrics for OOS field extraction
}
```

This eliminates code duplication between IS and OOS paths. The existing `backtestAndScore` is refactored to call `runSingleBacktest` with the full experiment time window.

#### 2.8 Optimizer Interface Impact

The `Optimizer.Tell()` method receives IS scores only. This is the correct contract — OOS is invisible to the optimizer:

```go
// In runIterative: Tell() receives IS score (unchanged behavior)
r, err := w.runSingleBacktest(ctx, code, overrides, symbol, tf, fromTs, toTs, capital, regime)
opt.Tell([]ai.OptimizerResult{{Indices: indices, Score: r.Score}})
```

### Optimality assessment

| Approach | Backtests (N=20) | Data leakage | Complexity |
|----------|-----------------|--------------|------------|
| Per-candidate OOS (naive) | 40 | None | Low |
| **Top-K OOS (this design)** | **25** | **None** | **Medium** |
| No OOS at all | 20 | None (but no overfit detection) | Low |
| Walk-forward (rolling) | 100+ | None | High |
| Combined IS+OOS score | 40 | 🔴 Yes | Low |

**Top-K OOS is optimal**: 37.5% fewer backtests than per-candidate, zero data leakage, acceptable complexity increase.

---

## §3 — Regime Storage

### Problem

`scoreFromBacktest()` calls `detectRegime()` and uses the result for scoring, but the regime value is discarded after scoring. It's never stored in the database or returned via proto.

The `strategy_experiments` table already has a `market_regime_ref TEXT` column (from migration 064), but it's always empty.

### Design

#### 3.1 Store regime on the experiment (not per candidate)

Add a step in `processOne()` to detect and persist regime before the optimizer loop:

```go
func (w *ExperimentWorker) processOne(ctx context.Context) error {
    // ... claim experiment ...

    // Detect and store regime once per experiment
    regime := w.detectRegimeForExperiment(ctx, exp)
    if exp.MarketRegimeRef == "" {
        if err := w.repo.UpdateMarketRegime(ctx, exp.ID, regime.String()); err != nil {
            w.log.Warn("failed to persist regime", zap.Error(err))
        }
    }

    // ... run optimizer (pass regime to all backtests) ...
}
```

#### 3.2 `detectRegimeForExperiment` helper

Extract from `scoreFromBacktest`'s `detectRegime` — uses experiment's symbol/timeframe/time window:

```go
func (w *ExperimentWorker) detectRegimeForExperiment(
    ctx context.Context, exp *repository.StrategyExperiment,
) ai.MarketRegime {
    if w.marketDataRepo == nil || exp.Symbol == "" || exp.Timeframe == "" {
        return ai.RegimeTransition
    }
    fromTs := time.UnixMilli(exp.FromTsUnixMs)
    toTs := time.UnixMilli(exp.ToTsUnixMs)
    bars, err := w.marketDataRepo.GetKlines(
        ctx, exp.Symbol, "", exp.Timeframe, &fromTs, &toTs, 2000,
    )
    if err != nil || len(bars) < 30 {
        return ai.RegimeTransition
    }
    ohlc := make([]ai.OHLCBar, len(bars))
    for i := 0; i < len(bars); i++ {
        b := bars[len(bars)-1-i]
        ohlc[i] = ai.OHLCBar{Open: b.Open, High: b.High, Low: b.Low, Close: b.Close, Volume: b.Volume}
    }
    return ai.DetectRegime(ohlc).Regime
}
```

#### 3.3 Repository method

```go
func (r *StrategyExperimentRepository) UpdateMarketRegime(ctx context.Context, id uuid.UUID, regime string) error {
    _, err := r.db.Exec(ctx,
        `UPDATE strategy_experiments SET market_regime_ref = $2 WHERE id = $1`,
        id, regime)
    return err
}
```

#### 3.4 Proto flow

`expToProto()` already maps `MarketRegimeRef` — no handler changes needed for regime:

```go
// Existing code (handler.go:48):
MarketRegimeRef: e.MarketRegimeRef,
```

Frontend receives regime via `StrategyExperiment.market_regime_ref` in both `GetStrategyExperiment` and `WatchExperiment` responses.

### Optimality assessment

| Approach | Verdict | Reason |
|----------|---------|--------|
| **Per-experiment string** | ✅ **Optimal** | One fetch per experiment, reuses existing column, string maps 1:1 to `MarketRegime.String()` |
| Per-candidate storage | ❌ | Regime is a market property — same for all candidates; storing per-candidate is redundant |
| FK to `market_regimes` table | 🟡 Overengineered | The `market_regimes` table is for historical cross-strategy tracking; inserting a row per experiment adds I/O without benefit at this stage |
| Global cache by symbol+tf | ❌ | Incorrect: regime is time-window dependent (Jan–Mar bull ≠ Apr–Jun bear) |

---

## §4 — OOS DB Schema + Proto Wiring

### Problem

Proto `StrategyExperimentCandidate` defines 5 OOS fields (15-19):
```proto
optional double oos_score = 15;
optional double oos_total_return = 16;
optional double oos_sharpe_ratio = 17;
optional double degradation_pct = 18;
bool is_overfit = 19;
```

But:
- Go `StrategyExperimentCandidate` struct lacks these fields
- DB `strategy_experiment_candidates` table lacks OOS columns
- `CreateCandidate` SQL doesn't insert OOS data
- `ListCandidates` / `GetCandidate` scan doesn't read OOS data
- `candidateToProto` doesn't populate proto OOS fields

### Design

#### 4.1 DB Migration (new: `142_`)

```sql
-- 142_candidate_oos_fields.up.sql
ALTER TABLE strategy_experiment_candidates
    ADD COLUMN IF NOT EXISTS oos_score DOUBLE PRECISION,
    ADD COLUMN IF NOT EXISTS oos_total_return DOUBLE PRECISION,
    ADD COLUMN IF NOT EXISTS oos_sharpe_ratio DOUBLE PRECISION,
    ADD COLUMN IF NOT EXISTS degradation_pct DOUBLE PRECISION,
    ADD COLUMN IF NOT EXISTS is_overfit BOOLEAN NOT NULL DEFAULT FALSE;
```

```sql
-- 142_candidate_oos_fields.down.sql
ALTER TABLE strategy_experiment_candidates
    DROP COLUMN IF EXISTS oos_score,
    DROP COLUMN IF EXISTS oos_total_return,
    DROP COLUMN IF EXISTS oos_sharpe_ratio,
    DROP COLUMN IF EXISTS degradation_pct,
    DROP COLUMN IF EXISTS is_overfit;
```

#### 4.2 Go Struct Update

```go
type StrategyExperimentCandidate struct {
    // ... existing fields ...

    // OOS validation (nil when window too short or not in top-K)
    OOSScore        *float64 `db:"oos_score"`
    OOSTotalReturn  *float64 `db:"oos_total_return"`
    OOSSharpeRatio  *float64 `db:"oos_sharpe_ratio"`
    DegradationPct  *float64 `db:"degradation_pct"`
    IsOverfit       bool     `db:"is_overfit"`
}
```

`*float64` matches the proto3 `optional double` generated type exactly — no conversion needed between repo struct and proto struct.

#### 4.3 CreateCandidate SQL Update

Add OOS columns to INSERT:
```sql
INSERT INTO strategy_experiment_candidates
    (id, experiment_id, parameters, draft_code_ref, backtest_run_id,
     score, grade, score_components, rank, summary, recommendation, created_at,
     oos_score, oos_total_return, oos_sharpe_ratio, degradation_pct, is_overfit)
VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17)
```

#### 4.4 ListCandidates / GetCandidate Scan Update

Both use `SELECT *` / `SELECT c.*` — columns are auto-included. Only the `Scan()` call needs updating to read the 5 new trailing columns:

```go
// ListCandidates (line 174):
rows.Scan(&c.ID, &c.ExperimentID, &c.Parameters, &c.DraftCodeRef,
    &c.BacktestRunID, &c.Score, &c.Grade, &c.ScoreComponents,
    &c.Rank, &c.Summary, &c.Recommendation, &c.CreatedAt,
    &c.OOSScore, &c.OOSTotalReturn, &c.OOSSharpeRatio,
    &c.DegradationPct, &c.IsOverfit)

// Same for GetCandidate (line 188)
```

#### 4.5 candidateToProto Update

```go
func candidateToProto(c *repository.StrategyExperimentCandidate) *antv1.StrategyExperimentCandidate {
    p := &antv1.StrategyExperimentCandidate{
        // ... existing fields ...
    }
    // proto3 optional double → *float64, direct pointer assignment
    if c.OOSScore != nil {
        p.OosScore = c.OOSScore
    }
    if c.OOSTotalReturn != nil {
        p.OosTotalReturn = c.OOSTotalReturn
    }
    if c.OOSSharpeRatio != nil {
        p.OosSharpeRatio = c.OOSSharpeRatio
    }
    if c.DegradationPct != nil {
        p.DegradationPct = c.DegradationPct
    }
    p.IsOverfit = c.IsOverfit
    return p
}
```

#### 4.6 Worker: Populate OOS Fields

In `processOne()`, after OOS backtests complete:

```go
// Enrich top-K candidates with OOS data
for i, idx := range topKIndices {
    oosResult := oosResults[i] // from OOS backtest
    validation := v.Validate(candidates[idx].Score, oosResult.Score)
    candidates[idx].OOSScore = &oosResult.Score
    candidates[idx].OOSTotalReturn = &oosResult.TotalReturn
    candidates[idx].OOSSharpeRatio = &oosResult.SharpeRatio
    candidates[idx].DegradationPct = &validation.Degradation
    candidates[idx].IsOverfit = validation.IsOverfit
}

// Store all candidates (top-K have OOS, rest have nil)
for _, c := range candidates {
    record := &repository.StrategyExperimentCandidate{
        // ... existing fields ...
        OOSScore:        c.OOSScore,
        OOSTotalReturn:  c.OOSTotalReturn,
        OOSSharpeRatio:  c.OOSSharpeRatio,
        DegradationPct:  c.DegradationPct,
        IsOverfit:       c.IsOverfit,
    }
    w.repo.CreateCandidate(ctx, record)
}
```

### Optimality assessment

| Approach | Verdict | Reason |
|----------|---------|--------|
| **`*float64` (pointer)** | ✅ **Optimal** | Matches proto3 `optional` generated type exactly; nil = absent from wire; pgx handles natively |
| `sql.NullFloat64` | 🟡 Works but redundant | Requires conversion to `*float64` for proto; adds intermediate type |
| Sentinel values (e.g. -1) | ❌ | Ambiguous (negative scores are possible), no standard for "absent" |
| Separate OOS table | ❌ | Over-normalized; OOS is 1:1 with candidate |

---

## §5 — Complete File Change Map

```
Frontend (1 file)
├── frontend/src/pages/strategy/hooks/useBacktestParams.ts
│   └── Line 167: tuneMethod === 'grid' ? 'grid' : 'random' → tuneMethod

Backend (7 files)
├── backend/internal/connect/strategy/experiment_scoring.go
│   ├── Extract runSingleBacktest() from backtestAndScore (shared IS/OOS helper)
│   ├── Modify backtestAndScore: call runSingleBacktest with full window
│   ├── Add detectRegimeForExperiment() helper
│   └── Remove detectRegime() call from scoreFromBacktest (regime now passed in)
│
├── backend/internal/connect/strategy/strategy_experiment_worker.go
│   ├── candidateResult: add OOS fields (OOSScore, OOSTotalReturn, OOSSharpeRatio, DegradationPct, IsOverfit)
│   ├── processOne: detect regime once → pass to all backtests
│   ├── processOne: after runOptimizer, select top-K → run OOS backtests → enrich
│   └── processOne: populate OOS fields in CreateCandidate records
│
├── backend/internal/connect/strategy/strategy_experiment_handler.go
│   └── candidateToProto: populate OOS proto fields (15-19)
│
├── backend/internal/repository/strategy_experiment_repository.go
│   ├── StrategyExperimentCandidate: add 5 OOS fields
│   ├── CreateCandidate: add 5 OOS columns to INSERT
│   ├── ListCandidates: scan 5 OOS columns
│   ├── GetCandidate: scan 5 OOS columns
│   └── Add: UpdateMarketRegime(id, regime)
│
├── backend/internal/ai/oos_validator.go
│   └── No changes — already complete, just wiring

Database (1 new migration)
├── backend/migrations/142_candidate_oos_fields.up.sql
└── backend/migrations/142_candidate_oos_fields.down.sql
```

---

## §6 — Optimality Audit

### §6.1 — Per-design-decision audit

| # | Decision | Alternatives considered | Why optimal |
|---|----------|------------------------|-------------|
| 1 | Direct `tuneMethod` pass-through | Enum mapping, dual validation | Zero transformation; union type + switch-case already form complete contract |
| 2 | Top-K OOS (not per-candidate) | Per-candidate OOS, no OOS, walk-forward | 37.5% fewer backtests, zero data leakage, acceptable complexity; walk-forward is better but 5× cost |
| 3 | IS score for ranking, OOS for metadata | OOS ranking, combined score, penalty | Only approach without data leakage; OOS = diagnostic, not optimization target |
| 4 | K=5 for top candidates | K=3, K=10, K=N | Covers user inspection range; configurable constant |
| 5 | Regime per experiment (string) | Per-candidate, FK table, global cache | 1 fetch vs N; reuses existing column; time-window correct |
| 6 | `*float64` for nullable OOS | `sql.NullFloat64`, sentinel -1 | Matches proto3 `optional` exactly; nil = absent from wire |
| 7 | `SELECT *` kept (not changed) | Explicit column list | Consistent with repo pattern; OOS columns are trailing; changing would be inconsistent |
| 8 | `ADD COLUMN IF NOT EXISTS` | Plain `ADD COLUMN` | Idempotent — safe for CI re-runs |
| 9 | Degradation threshold 0.4 | 0.3, 0.5, adaptive | QuantDinger standard; 40% drop = clear overfit signal without false positives |
| 10 | 70/30 split ratio | 80/20, 60/40, rolling | Industry standard (scikit-learn, QuantDinger, Backtrader); adequate OOS with sufficient IS training data |

### §6.2 — Forward-compatibility check

| Future feature | How this design supports it |
|----------------|---------------------------|
| Walk-forward / rolling OOS | `runSingleBacktest` helper accepts arbitrary time windows; loop over folds |
| Per-round SSE progress | `processOne` structure has clear phase boundaries (IS → top-K → OOS → store) |
| Adaptive degradation threshold | `Validate()` already parameterized; change `MaxDegradation` in `OOSValidator` |
| Additional OOS metrics (max DD, profit factor) | Add fields to `candidateResult` + migration; same pattern as existing 5 fields |
| Regime FK migration | `market_regime_ref` string maps 1:1 to `market_regimes.regime`; ALTER + FK constraint |

### §6.3 — Edge cases

| Edge case | Handling |
|-----------|----------|
| Time window < 37 days | `ComputeWindows()` returns nil → skip OOS entirely, nil fields |
| N < K (fewer candidates than top-K limit) | OOS runs for all N candidates |
| All IS scores ≤ 0 | `Validate()` returns `IsOverfit: true` for score ≤ 0 (existing behavior) |
| OOS backtest fails | Log warning, skip OOS for that candidate, remaining OOS continue |
| Experiment cancelled mid-OOS | OOS loop checks `ctx.Done()`, exits gracefully |
| Empty `ParameterSpace` | GridSearch returns nil → 0 candidates → experiment completes with 0 candidates |
| Single parameter dimension | OOS still meaningful — overfitting is as likely with 1 param as 10 |

### §6.4 — Consistency verification

- ✅ Proto OOS fields (15-19) match Go struct OOS fields (same names, types, nullability)
- ✅ `optional double` = `*float64` in generated Go — repo struct uses same type
- ✅ `market_regime_ref` column already exists in both DB and Go struct
- ✅ `expToProto` already maps `MarketRegimeRef` — no handler change for regime display
- ✅ `WatchExperiment` sends candidates via `candidateToProto` — OOS fields flow automatically
- ✅ `SELECT *` in ListCandidates/GetCandidate auto-includes new columns
- ✅ `IsOverfit` is non-nullable `bool` (always has a value) vs 4 optional metrics (nullable)
- ✅ `processOne` has single `CreateCandidate` call site — only one place to update

### §6.5 — File size impact (post-change)

| File | Current | After | Limit | OK |
|------|---------|-------|-------|-----|
| `experiment_scoring.go` | 161 | ~230 | 250 | ✅ |
| `strategy_experiment_worker.go` | 213 | ~260 | 300 | ✅ |
| `strategy_experiment_repository.go` | 242 | ~285 | 300 | ✅ |
| `strategy_experiment_handler.go` | 260 | ~270 | 300 | ✅ |
| All others | — | <10 lines each | — | ✅ |

### §6.6 — Acceptance criteria

1. `searchMethod` sends all 6 values correctly → verify with DevTools network tab
2. OOS validation runs on top-K candidates when experiment has ≥37 days of data
3. `is_overfit = true` when OOS score drops >40% from IS score
4. `market_regime_ref` populated on experiment after completion
5. Non-top-K candidates have nil OOS fields in response
6. `go build ./...` passes
7. `python3 scripts/check-file-lines.py --strict` passes
8. Existing tests pass

### §6.7 — Not in scope (explicit exclusions)

| Exclusion | Reason |
|-----------|--------|
| Independent `market_regimes` table | Uses existing `market_regime_ref` column |
| Dedicated OOS RPC endpoint | OOS data embedded in candidate responses |
| Per-round SSE streaming during optimization | Uses existing `WatchExperiment` (push on completion) |
| New optimizer algorithms | DE, TPE, AGS, Grid, Random already cover the design space |
| Walk-forward / rolling OOS | 70/30 split is sufficient for overfit detection; walk-forward is future optimization |
| Parallel OOS backtests | Sequential is acceptable for K=5 (max ~2.5min OOS phase); parallel is future optimization |

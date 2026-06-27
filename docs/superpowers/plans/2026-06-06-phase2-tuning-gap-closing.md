# Phase 2 — Parameter Optimization Gap-Closing Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Wire 4 critical gaps to make Phase 2 parameter optimization production-ready: searchMethod pass-through, OOS validation with top-K backtests, regime storage, and OOS DB/proto fields.

**Architecture:** Extract `runSingleBacktest` as shared IS/OOS backtest runner accepting arbitrary time window + pre-detected regime. Compute regime once per experiment in `processOne`. After optimizer loop, run OOS backtests only for top-5 candidates (by IS score). Store OOS data via migration 142 with `*float64` pointer types matching proto3 `optional double`.

**Tech Stack:** Go (ConnectRPC + repository + pgx), TypeScript (React hook), PostgreSQL (migration), protobuf

**Spec:** `docs/superpowers/specs/2026-06-06-phase2-tuning-design.md`

---

### Task 1: DB Migration — Add OOS columns

**Files:**
- Create: `backend/migrations/142_candidate_oos_fields.up.sql`
- Create: `backend/migrations/142_candidate_oos_fields.down.sql`

- [ ] **Step 1: Write up migration**

```sql
ALTER TABLE strategy_experiment_candidates
    ADD COLUMN IF NOT EXISTS oos_score DOUBLE PRECISION,
    ADD COLUMN IF NOT EXISTS oos_total_return DOUBLE PRECISION,
    ADD COLUMN IF NOT EXISTS oos_sharpe_ratio DOUBLE PRECISION,
    ADD COLUMN IF NOT EXISTS degradation_pct DOUBLE PRECISION,
    ADD COLUMN IF NOT EXISTS is_overfit BOOLEAN NOT NULL DEFAULT FALSE;
```

- [ ] **Step 2: Write down migration**

```sql
ALTER TABLE strategy_experiment_candidates
    DROP COLUMN IF EXISTS oos_score,
    DROP COLUMN IF EXISTS oos_total_return,
    DROP COLUMN IF EXISTS oos_sharpe_ratio,
    DROP COLUMN IF EXISTS degradation_pct,
    DROP COLUMN IF EXISTS is_overfit;
```

- [ ] **Step 3: Apply to local DB**

```bash
docker exec ant-postgres psql -U ant -d ant -c "ALTER TABLE strategy_experiment_candidates ADD COLUMN IF NOT EXISTS oos_score DOUBLE PRECISION, ADD COLUMN IF NOT EXISTS oos_total_return DOUBLE PRECISION, ADD COLUMN IF NOT EXISTS oos_sharpe_ratio DOUBLE PRECISION, ADD COLUMN IF NOT EXISTS degradation_pct DOUBLE PRECISION, ADD COLUMN IF NOT EXISTS is_overfit BOOLEAN NOT NULL DEFAULT FALSE;"
```
Expected: `ALTER TABLE`

- [ ] **Step 4: Verify**

```bash
docker exec ant-postgres psql -U ant -d ant -c "\d strategy_experiment_candidates" | grep -E "oos_|is_overfit"
```
Expected: 5 rows with the new columns

- [ ] **Step 5: Commit**

```bash
git -C /opt/ant add backend/migrations/142_candidate_oos_fields.up.sql backend/migrations/142_candidate_oos_fields.down.sql
git -C /opt/ant commit -m "feat(db): add OOS validation columns to strategy_experiment_candidates

Migration 142: oos_score, oos_total_return, oos_sharpe_ratio,
degradation_pct, is_overfit.

Co-Authored-By: Claude Opus 4.8 <noreply@anthropic.com>"
```

---

### Task 2: Repository — Go struct + SQL updates

**Files:**
- Modify: `backend/internal/repository/strategy_experiment_repository.go`

- [ ] **Step 1: Add OOS fields to StrategyExperimentCandidate struct (line 40-53)**

Replace the existing struct:

```go
type StrategyExperimentCandidate struct {
	ID              uuid.UUID  `db:"id"`
	ExperimentID    uuid.UUID  `db:"experiment_id"`
	Parameters      []byte     `db:"parameters"`
	DraftCodeRef    string     `db:"draft_code_ref"`
	BacktestRunID   *uuid.UUID `db:"backtest_run_id"`
	Score           float64    `db:"score"`
	Grade           string     `db:"grade"`
	ScoreComponents []byte     `db:"score_components"`
	Rank            int        `db:"rank"`
	Summary         string     `db:"summary"`
	Recommendation  string     `db:"recommendation"`
	CreatedAt       time.Time  `db:"created_at"`
	// OOS validation (nil when window too short or not in top-K)
	OOSScore        *float64 `db:"oos_score"`
	OOSTotalReturn  *float64 `db:"oos_total_return"`
	OOSSharpeRatio  *float64 `db:"oos_sharpe_ratio"`
	DegradationPct  *float64 `db:"degradation_pct"`
	IsOverfit       bool     `db:"is_overfit"`
}
```

- [ ] **Step 2: Update CreateCandidate INSERT (lines 152-155)**

Replace the SQL + VALUES with 5 additional columns:

```go
_, err := r.db.Exec(ctx, `
	INSERT INTO strategy_experiment_candidates (id,experiment_id,parameters,draft_code_ref,backtest_run_id,score,grade,score_components,rank,summary,recommendation,created_at,oos_score,oos_total_return,oos_sharpe_ratio,degradation_pct,is_overfit)
	VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17)
`, candidate.ID, candidate.ExperimentID, candidate.Parameters, candidate.DraftCodeRef, candidate.BacktestRunID, candidate.Score, candidate.Grade, candidate.ScoreComponents, candidate.Rank, candidate.Summary, candidate.Recommendation, candidate.CreatedAt, candidate.OOSScore, candidate.OOSTotalReturn, candidate.OOSSharpeRatio, candidate.DegradationPct, candidate.IsOverfit)
```

- [ ] **Step 3: Update ListCandidates Scan (line 174)**

Replace the Scan line — add 5 trailing OOS columns:

```go
if err := rows.Scan(&c.ID, &c.ExperimentID, &c.Parameters, &c.DraftCodeRef, &c.BacktestRunID, &c.Score, &c.Grade, &c.ScoreComponents, &c.Rank, &c.Summary, &c.Recommendation, &c.CreatedAt, &c.OOSScore, &c.OOSTotalReturn, &c.OOSSharpeRatio, &c.DegradationPct, &c.IsOverfit); err != nil {
```

- [ ] **Step 4: Update GetCandidate Scan (line 188)**

Replace the Scan line — same 5 trailing fields:

```go
`, candidateID, userID).Scan(&row.ID, &row.ExperimentID, &row.Parameters, &row.DraftCodeRef, &row.BacktestRunID, &row.Score, &row.Grade, &row.ScoreComponents, &row.Rank, &row.Summary, &row.Recommendation, &row.CreatedAt, &row.OOSScore, &row.OOSTotalReturn, &row.OOSSharpeRatio, &row.DegradationPct, &row.IsOverfit)
```

- [ ] **Step 5: Add UpdateMarketRegime method (after line 233)**

```go
// UpdateMarketRegime persists the detected market regime on the experiment.
func (r *StrategyExperimentRepository) UpdateMarketRegime(ctx context.Context, id uuid.UUID, regime string) error {
	_, err := r.db.Exec(ctx,
		`UPDATE strategy_experiments SET market_regime_ref = $2 WHERE id = $1`,
		id, regime)
	return err
}
```

- [ ] **Step 6: Verify compilation**

```bash
cd /opt/ant/backend && go build ./internal/repository/...
```
Expected: exit 0

- [ ] **Step 7: Commit**

```bash
git -C /opt/ant add backend/internal/repository/strategy_experiment_repository.go
git -C /opt/ant commit -m "feat(repo): add OOS fields to candidate + UpdateMarketRegime

Adds 5 OOS columns to Go struct, CreateCandidate SQL, ListCandidates scan,
GetCandidate scan. Adds UpdateMarketRegime for storing detected regime
in existing market_regime_ref column.

Co-Authored-By: Claude Opus 4.8 <noreply@anthropic.com>"
```

---

### Task 3: Experiment Scoring — Add runSingleBacktest + detectRegimeForExperiment

**Files:**
- Modify: `backend/internal/connect/strategy/experiment_scoring.go`

- [ ] **Step 1: Add runSingleBacktest function (insert before backtestAndScore at line 16)**

This is the shared helper for both IS and OOS backtests — accepts an arbitrary time window and pre-detected regime:

```go
// runSingleBacktest executes one backtest with the given time window,
// polls for completion, scores with regime-aware weights, and returns the scored result.
// Used for both in-sample (full window) and out-of-sample (OOS window) backtests.
func (w *ExperimentWorker) runSingleBacktest(
	ctx context.Context, code string, overrides map[string]interface{},
	userID uuid.UUID, symbol, timeframe string, fromTs, toTs time.Time,
	regime ai.MarketRegime,
) (*ai.ScoredResult, error) {
	modifiedCode := code
	overridesBytes, err := marshalOverrides(overrides)
	if err != nil {
		return nil, fmt.Errorf("marshal overrides: %w", err)
	}
	run := &repository.BacktestRun{
		ID:                 uuid.New(),
		UserID:             userID,
		AccountID:          uuid.Nil,
		Symbol:             symbol,
		Timeframe:          timeframe,
		FromTs:             &fromTs,
		ToTs:               &toTs,
		Mode:               "KLINE_RANGE",
		Status:             StatusPending,
		StrategyCode:       &modifiedCode,
		InitialCapital:     f64Ptr(10000),
		Commission:         f64Ptr(0.001),
		Slippage:           f64Ptr(0),
		Leverage:           f64Ptr(1),
		TradeDirection:     strPtr("both"),
		StrictMode:         boolPtr(true),
		StrategyCodeHash:   "",
		Error:              "",
		ExtraSymbols:       []string{},
		ParameterOverrides: overridesBytes,
	}

	runID, err := w.backtestRepo.Create(ctx, run)
	if err != nil {
		return nil, fmt.Errorf("create backtest: %w", err)
	}

	// Poll for completion
	for i := 0; i < 120; i++ { // 10 minutes max timeout
		select {
		case <-ctx.Done():
			return nil, fmt.Errorf("backtest %s cancelled: %w", runID, ctx.Err())
		case <-time.After(5 * time.Second):
		}
		bt, err := w.backtestRepo.GetByID(ctx, userID, runID)
		if err != nil {
			return nil, fmt.Errorf("get backtest: %w", err)
		}
		if bt.Status == StatusSucceeded || bt.Status == StatusFailed {
			if bt.Status == StatusFailed {
				return nil, fmt.Errorf("backtest failed: %s", bt.Error)
			}
			btMetrics := extractBacktestMetrics(bt.ProtoResponse)
			scored := ai.Score(btMetrics, regime)
			return scored, nil
		}
	}
	return nil, fmt.Errorf("backtest %s timed out", runID)
}
```

- [ ] **Step 2: Add detectRegimeForExperiment function (insert after detectRegime at line 160)**

```go
// detectRegimeForExperiment fetches K-lines for the experiment's symbol/timeframe/time-window
// and classifies the market regime. Called once per experiment, not per candidate.
func (w *ExperimentWorker) detectRegimeForExperiment(
	ctx context.Context, exp *repository.StrategyExperiment,
) ai.MarketRegime {
	if w.marketDataRepo == nil || exp.Symbol == "" || exp.Timeframe == "" {
		return ai.RegimeTransition
	}
	fromTs := time.UnixMilli(exp.FromTsUnixMs)
	toTs := time.UnixMilli(exp.ToTsUnixMs)
	if exp.FromTsUnixMs == 0 {
		ft := time.Now().AddDate(0, -1, 0)
		fromTs = ft
	}
	if exp.ToTsUnixMs == 0 {
		toTs = time.Now()
	}
	bars, err := w.marketDataRepo.GetKlines(
		ctx, exp.Symbol, "", exp.Timeframe, &fromTs, &toTs, 2000,
	)
	if err != nil || len(bars) < 30 {
		return ai.RegimeTransition
	}
	ohlc := make([]ai.OHLCBar, len(bars))
	for i := 0; i < len(bars); i++ {
		b := bars[len(bars)-1-i] // reverse DESC→ASC
		ohlc[i] = ai.OHLCBar{Open: b.Open, High: b.High, Low: b.Low, Close: b.Close, Volume: b.Volume}
	}
	result := ai.DetectRegime(ohlc)
	return result.Regime
}
```

- [ ] **Step 3: Verify compilation**

```bash
cd /opt/ant/backend && go build ./internal/connect/strategy/...
```
Expected: exit 0 (new functions added, no callers yet — no breakage)

- [ ] **Step 4: Commit**

```bash
git -C /opt/ant add backend/internal/connect/strategy/experiment_scoring.go
git -C /opt/ant commit -m "feat(scoring): add runSingleBacktest + detectRegimeForExperiment

runSingleBacktest: shared IS/OOS backtest runner accepting time window + regime.
detectRegimeForExperiment: K-line regime detection once per experiment.

Co-Authored-By: Claude Opus 4.8 <noreply@anthropic.com>"
```

---

### Task 4: Worker — Refactor backtestAndScore + processOne for IS-only + top-K OOS

**Files:**
- Modify: `backend/internal/connect/strategy/strategy_experiment_worker.go`
- Modify: `backend/internal/connect/strategy/experiment_scoring.go`

- [ ] **Step 1: Add OOS fields to candidateResult struct (worker.go:147-153)**

Replace the struct:

```go
type candidateResult struct {
	Overrides       map[string]interface{}
	Score           float64
	Grade           string
	ScoreComponents map[string]float64
	Summary         string
	// OOS validation (nil when not in top-K or window too short)
	OOSScore        *float64
	OOSTotalReturn  *float64
	OOSSharpeRatio  *float64
	DegradationPct  *float64
	IsOverfit       bool
}
```

- [ ] **Step 2: Refactor backtestAndScore to use runSingleBacktest (scoring.go:16-97)**

Replace `backtestAndScore` (lines 16-97) with a version that accepts regime and delegates to `runSingleBacktest`:

```go
// backtestAndScore executes an in-sample backtest on the full experiment time window.
func (w *ExperimentWorker) backtestAndScore(
	ctx context.Context, code string, overrides map[string]interface{},
	exp *repository.StrategyExperiment, regime ai.MarketRegime,
) (candidateResult, error) {
	symbol := exp.Symbol
	if symbol == "" {
		symbol = "XAUUSDm"
	}
	tf := exp.Timeframe
	if tf == "" {
		tf = "1h"
	}
	fromTs := time.UnixMilli(exp.FromTsUnixMs)
	toTs := time.UnixMilli(exp.ToTsUnixMs)
	if exp.FromTsUnixMs == 0 {
		fromTs = time.Now().AddDate(0, -1, 0)
	}
	if exp.ToTsUnixMs == 0 {
		toTs = time.Now()
	}

	scored, err := w.runSingleBacktest(ctx, code, overrides, exp.UserID, symbol, tf, fromTs, toTs, regime)
	if err != nil {
		return candidateResult{}, err
	}

	summary := "param search"
	if scored.Trades < 5 {
		summary = fmt.Sprintf("only %d trades", scored.Trades)
	}

	return candidateResult{
		Overrides:       overrides,
		Score:           scored.Score,
		Grade:           scored.Grade,
		ScoreComponents: scored.Components,
		Summary:         summary,
	}, nil
}
```

- [ ] **Step 3: Refactor scoreFromBacktest to accept regime parameter (scoring.go:99-117)**

Replace `scoreFromBacktest` — remove internal `detectRegime` call:

```go
// scoreFromBacktest extracts metrics from proto binary and scores with the given regime.
func (w *ExperimentWorker) scoreFromBacktest(bt *repository.BacktestRun, overrides map[string]interface{}, regime ai.MarketRegime) candidateResult {
	btMetrics := extractBacktestMetrics(bt.ProtoResponse)
	scored := ai.Score(btMetrics, regime)

	summary := "param search"
	if scored.Trades < 5 {
		summary = fmt.Sprintf("only %d trades", scored.Trades)
	}

	return candidateResult{
		Overrides:       overrides,
		Score:           scored.Score,
		Grade:           scored.Grade,
		ScoreComponents: scored.Components,
		Summary:         summary,
	}
}
```

- [ ] **Step 4: Add selectTopK helper to experiment_scoring.go**

Insert after `detectRegimeForExperiment`:

```go
// selectTopK returns the indices of the top-K candidates sorted by score descending.
func selectTopK(candidates []candidateResult, k int) []int {
	if k > len(candidates) {
		k = len(candidates)
	}
	if k <= 0 {
		return nil
	}
	type indexed struct {
		idx   int
		score float64
	}
	items := make([]indexed, len(candidates))
	for i, c := range candidates {
		items[i] = indexed{idx: i, score: c.Score}
	}
	sort.Slice(items, func(i, j int) bool { return items[i].score > items[j].score })
	indices := make([]int, k)
	for i := 0; i < k; i++ {
		indices[i] = items[i].idx
	}
	return indices
}
```

**Also:** add `"sort"` to the imports in `experiment_scoring.go` (line 6, after `"time"`):

```go
import (
	"context"
	"fmt"
	"sort"
	"time"
	// ... rest unchanged
)
```

- [ ] **Step 5: Update runOptimizer to accept and thread regime (worker.go:155-173)**

Replace `runOptimizer` signature and all internal calls — add `regime ai.MarketRegime` parameter:

```go
func (w *ExperimentWorker) runOptimizer(
	ctx context.Context, exp *repository.StrategyExperiment,
	params []ai.TunableParam, space ai.ResolvedSpace, code string,
	regime ai.MarketRegime,
) ([]candidateResult, error) {
	switch exp.SearchMethod {
	case "de":
		return w.runIterative(ctx, ai.NewDEOptimizer(space, exp.MaxCandidates), space, code, exp, regime)
	case "ai":
		return w.runAIProposal(ctx, params, code, exp, regime)
	case "tpe":
		return w.runIterative(ctx, ai.NewTPEOptimizer(space, exp.MaxCandidates), space, code, exp, regime)
	case "ags":
		return w.runIterative(ctx, ai.NewAnnealedGaussianOptimizer(space, exp.MaxCandidates), space, code, exp, regime)
	case "random":
		return w.runOneShot(ctx, ai.RandomSearch(params, exp.MaxCandidates), code, exp, regime)
	default:
		return w.runOneShot(ctx, ai.GridSearch(params, exp.MaxCandidates), code, exp, regime)
	}
}
```

- [ ] **Step 6: Update runOneShot to accept + thread regime (worker.go:175-187)**

Add `regime ai.MarketRegime` parameter and pass to `backtestAndScore`:

```go
func (w *ExperimentWorker) runOneShot(ctx context.Context, overridesList []map[string]interface{}, code string, exp *repository.StrategyExperiment, regime ai.MarketRegime) ([]candidateResult, error) {
	var results []candidateResult
	for _, overrides := range overridesList {
		r, err := w.backtestAndScore(ctx, code, overrides, exp, regime)
		if err != nil {
			w.log.Warn("backtest failed", zap.Error(err))
			continue
		}
		results = append(results, r)
	}
	return results, nil
}
```

- [ ] **Step 7: Update runIterative to accept + thread regime (worker.go:189-210)**

Add `regime ai.MarketRegime` parameter and pass to `backtestAndScore`:

```go
func (w *ExperimentWorker) runIterative(
	ctx context.Context, opt ai.Optimizer, space ai.ResolvedSpace,
	code string, exp *repository.StrategyExperiment, regime ai.MarketRegime,
) ([]candidateResult, error) {
	var results []candidateResult
	for !opt.Done() {
		batch := opt.Ask(0)
		for _, indices := range batch {
			overrides := ai.IndexToOverrides(indices, space)
			r, err := w.backtestAndScore(ctx, code, overrides, exp, regime)
			if err != nil {
				w.log.Warn("backtest failed", zap.Error(err))
				opt.Tell([]ai.OptimizerResult{{Indices: indices, Score: 0}})
				continue
			}
			opt.Tell([]ai.OptimizerResult{{Indices: indices, Score: r.Score}})
			results = append(results, r)
		}
	}
	return results, nil
}
```

- [ ] **Step 8: Update processOne — add regime detection, top-K OOS, OOS enrichment (worker.go:66-145)**

Replace the entire `processOne` method body (lines 66-145). Key changes:
1. Detect regime once before optimizer
2. Persist regime to DB
3. Pass regime to runOptimizer
4. After optimizer: compute OOS windows, select top-5, run OOS backtests
5. Enrich top-K candidates with OOS data before storage

```go
func (w *ExperimentWorker) processOne(ctx context.Context) error {
	exp, err := w.repo.ClaimPendingExperiment(ctx)
	if err != nil {
		return err
	}
	if exp == nil {
		return nil
	}

	w.log.Info("Processing experiment",
		zap.String("expID", exp.ID.String()),
		zap.String("userID", exp.UserID.String()),
		zap.String("method", exp.SearchMethod),
		zap.Int("maxCandidates", exp.MaxCandidates))

	code := exp.StrategyCode
	if code == "" {
		if err := w.repo.UpdateExperimentStatus(ctx, exp.ID, StatusFailed); err != nil {
			w.log.Error("update experiment status to FAILED failed", zap.Error(err), zap.String("expID", exp.ID.String()))
		}
		ExperimentRunsTotal.WithLabelValues(StatusFailed).Inc()
		return fmt.Errorf("experiment %s has no strategy_code", exp.ID)
	}

	params := ai.ExtractParams(code)
	if len(params) == 0 {
		if err := w.repo.UpdateExperimentStatus(ctx, exp.ID, StatusCompleted); err != nil {
			w.log.Error("update experiment status to COMPLETED failed", zap.Error(err), zap.String("expID", exp.ID.String()))
		}
		ExperimentRunsTotal.WithLabelValues(StatusCompleted).Inc()
		w.log.Info("No @params found", zap.String("id", exp.ID.String()))
		return nil
	}

	// Detect and persist regime once per experiment
	regime := w.detectRegimeForExperiment(ctx, exp)
	if exp.MarketRegimeRef == "" {
		if err := w.repo.UpdateMarketRegime(ctx, exp.ID, regime.String()); err != nil {
			w.log.Warn("failed to persist regime", zap.Error(err))
		}
	}

	space := ai.NormalizeSpace(params)
	candidates, err := w.runOptimizer(ctx, exp, params, space, code, regime)
	if err != nil {
		w.log.Error("optimizer failed", zap.Error(err))
		if err := w.repo.UpdateExperimentStatus(ctx, exp.ID, StatusFailed); err != nil {
			w.log.Error("update experiment status to FAILED failed", zap.Error(err), zap.String("expID", exp.ID.String()))
		}
		ExperimentRunsTotal.WithLabelValues(StatusFailed).Inc()
		return err
	}

	// ── OOS validation for top-K candidates ──
	const oosTopK = 5
	oosVal := ai.DefaultOOSValidator()
	symbol := exp.Symbol
	if symbol == "" {
		symbol = "XAUUSDm"
	}
	tf := exp.Timeframe
	if tf == "" {
		tf = "1h"
	}
	fromTs := time.UnixMilli(exp.FromTsUnixMs)
	toTs := time.UnixMilli(exp.ToTsUnixMs)
	if exp.FromTsUnixMs == 0 {
		fromTs = time.Now().AddDate(0, -1, 0)
	}
	if exp.ToTsUnixMs == 0 {
		toTs = time.Now()
	}
	windows := oosVal.ComputeWindows(fromTs, toTs)
	if windows != nil && len(candidates) > 0 {
		topIndices := selectTopK(candidates, oosTopK)
		for _, idx := range topIndices {
			c := &candidates[idx]
			oosScored, err := w.runSingleBacktest(
				ctx, code, c.Overrides, exp.UserID, symbol, tf,
				windows.OOSStart, windows.OOSEnd, regime,
			)
			if err != nil {
				w.log.Warn("OOS backtest failed",
					zap.Error(err),
					zap.Int("candidateIdx", idx),
					zap.Float64("isScore", c.Score))
				continue
			}
			validation := oosVal.Validate(c.Score, oosScored.Score)
			c.OOSScore = &oosScored.Score
			c.OOSTotalReturn = &oosScored.TotalReturn
			c.OOSSharpeRatio = &oosScored.SharpeRatio
			c.DegradationPct = &validation.Degradation
			c.IsOverfit = validation.IsOverfit
		}
	}

	// Create candidate records with scores (including OOS data for top-K)
	for i, c := range candidates {
		paramProto, err := proto.Marshal(paramsToProto(c.Overrides))
		if err != nil {
			w.log.Error("marshal candidate params failed", zap.Error(err), zap.String("expID", exp.ID.String()), zap.Int("idx", i))
			continue
		}
		scoreProto, err := proto.Marshal(scoreComponentsToProto(c.ScoreComponents))
		if err != nil {
			w.log.Error("marshal candidate score failed", zap.Error(err), zap.String("expID", exp.ID.String()), zap.Int("idx", i))
			continue
		}
		record := &repository.StrategyExperimentCandidate{
			ID:              uuid.New(),
			ExperimentID:    exp.ID,
			Parameters:      paramProto,
			Rank:            i + 1,
			Score:           c.Score,
			Grade:           c.Grade,
			ScoreComponents: scoreProto,
			Summary:         fmt.Sprintf("%s score=%.1f grade=%s", c.Summary, c.Score, c.Grade),
			OOSScore:        c.OOSScore,
			OOSTotalReturn:  c.OOSTotalReturn,
			OOSSharpeRatio:  c.OOSSharpeRatio,
			DegradationPct:  c.DegradationPct,
			IsOverfit:       c.IsOverfit,
		}
		if err := w.repo.CreateCandidate(ctx, record); err != nil {
			w.log.Warn("create candidate failed", zap.Error(err), zap.Int("idx", i))
		}
	}

	if err := w.repo.UpdateExperimentStatus(ctx, exp.ID, StatusCompleted); err != nil {
		w.log.Error("update experiment status to COMPLETED failed", zap.Error(err), zap.String("expID", exp.ID.String()))
	}
	w.log.Info("Experiment completed", zap.String("id", exp.ID.String()),
		zap.Int("candidates", len(candidates)))
	return nil
}
```

- [ ] **Step 9: Update runAIProposal signature (experiment_utils.go:96)**

Replace the signature and the `backtestAndScore` call:

```go
func (w *ExperimentWorker) runAIProposal(ctx context.Context, params []ai.TunableParam, code string, exp *repository.StrategyExperiment, regime ai.MarketRegime) ([]candidateResult, error) {
	if w.systemAISvc == nil {
		return nil, fmt.Errorf("AI proposer not configured")
	}
	proposer := &systemAIAdapter{svc: w.systemAISvc, userID: exp.UserID}
	var results []candidateResult
	maxRounds := 3
	for round := 1; round <= maxRounds; round++ {
		req := &ai.ProposeRequest{
			IndicatorCode: code,
			Params:        params,
			Round:         round,
			MaxCandidates: exp.MaxCandidates / maxRounds,
			PrevResults:   make([]ai.ProposePrevResult, len(results)),
		}
		for i, r := range results {
			req.PrevResults[i] = ai.ProposePrevResult{
				Params: r.Overrides, Score: r.Score, Grade: r.Grade,
			}
		}
		proposed, err := ai.ProposeParams(ctx, proposer, req)
		if err != nil {
			w.log.Warn("AI proposal failed", zap.Error(err), zap.Int("round", round))
			continue
		}
		for _, overrides := range proposed {
			r, err := w.backtestAndScore(ctx, code, overrides, exp, regime)
			if err != nil {
				w.log.Warn("AI backtest failed", zap.Error(err))
				continue
			}
			results = append(results, r)
		}
	}
	return results, nil
}
```

- [ ] **Step 10: Verify compilation**

```bash
cd /opt/ant/backend && go build ./internal/connect/strategy/...
```
Expected: exit 0

- [ ] **Step 11: Commit**

```bash
git -C /opt/ant add backend/internal/connect/strategy/strategy_experiment_worker.go backend/internal/connect/strategy/experiment_scoring.go
git -C /opt/ant commit -m "feat(worker): wire OOS validation + regime storage into processOne

- Refactor backtestAndScore to use runSingleBacktest (IS only)
- Add regime parameter threading through runOptimizer/runOneShot/runIterative
- Detect regime once per experiment, persist via UpdateMarketRegime
- Run OOS backtests for top-5 candidates after optimizer completes
- Enrich top-K candidates with OOS fields (score, return, sharpe, degradation, is_overfit)

Co-Authored-By: Claude Opus 4.8 <noreply@anthropic.com>"
```

---

### Task 5: Handler — Populate OOS proto fields in candidateToProto

**Files:**
- Modify: `backend/internal/connect/strategy/strategy_experiment_handler.go`

- [ ] **Step 1: Update candidateToProto (lines 67-85)**

Add OOS field population after line 83 (`p.ScoreComponents = ...`):

```go
func candidateToProto(c *repository.StrategyExperimentCandidate) *antv1.StrategyExperimentCandidate {
	p := &antv1.StrategyExperimentCandidate{
		Id:            c.ID.String(),
		ExperimentId:  c.ExperimentID.String(),
		DraftCodeRef:  c.DraftCodeRef,
		Score:         c.Score,
		Grade:         c.Grade,
		Rank:          int32(c.Rank),
		Summary:       c.Summary,
		Recommendation: c.Recommendation,
		CreatedAt:     timestamppb.New(c.CreatedAt),
	}
	if c.BacktestRunID != nil {
		p.BacktestRunId = c.BacktestRunID.String()
	}
	p.Parameters = paramsProtoToStruct(c.Parameters)
	p.ScoreComponents = scoreProtoToStruct(c.ScoreComponents)
	// OOS validation fields (proto3 optional double → *float64, direct pointer assignment)
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

- [ ] **Step 2: Verify compilation**

```bash
cd /opt/ant/backend && go build ./internal/connect/strategy/...
```
Expected: exit 0

- [ ] **Step 3: Commit**

```bash
git -C /opt/ant add backend/internal/connect/strategy/strategy_experiment_handler.go
git -C /opt/ant commit -m "feat(handler): populate OOS proto fields in candidateToProto

Maps repository OOS fields (oos_score, oos_total_return, oos_sharpe_ratio,
degradation_pct, is_overfit) to proto StrategyExperimentCandidate fields 15-19.

Co-Authored-By: Claude Opus 4.8 <noreply@anthropic.com>"
```

---

### Task 6: Frontend — Fix searchMethod pass-through

**Files:**
- Modify: `frontend/src/pages/strategy/hooks/useBacktestParams.ts`

- [ ] **Step 1: Fix searchMethod at line 167**

Replace:
```ts
searchMethod: tuneMethod === 'grid' ? 'grid' : 'random',
```
With:
```ts
searchMethod: tuneMethod,
```

- [ ] **Step 2: Verify compilation**

```bash
cd /opt/ant/frontend && npx tsc --noEmit --pretty 2>&1 | head -5
```
Expected: no new errors from this file

- [ ] **Step 3: Commit**

```bash
git -C /opt/ant add frontend/src/pages/strategy/hooks/useBacktestParams.ts
git -C /opt/ant commit -m "fix(frontend): searchMethod direct pass-through for all 6 methods

Replace ternary tuneMethod === 'grid' ? 'grid' : 'random' with direct
tuneMethod pass-through. DE, TPE, AGS, and AI methods were unreachable.

Co-Authored-By: Claude Opus 4.8 <noreply@anthropic.com>"
```

---

### Task 7: Full Build Verification

- [ ] **Step 1: Go build**

```bash
cd /opt/ant/backend && go build ./...
```
Expected: exit 0

- [ ] **Step 2: File size check**

```bash
cd /opt/ant/backend && go run ./tools/check-file-lines --strict
```
Expected: exit 0 (no 🔴 failures)

- [ ] **Step 3: Frontend typecheck**

```bash
cd /opt/ant/frontend && npx tsc --noEmit --pretty
```
Expected: exit 0 (pre-existing errors only, no new ones)

- [ ] **Step 4: Run existing Go tests**

```bash
cd /opt/ant/backend && go test ./internal/ai/... ./internal/connect/strategy/... ./internal/repository/... -count=1 -timeout 60s 2>&1 | tail -20
```
Expected: PASS for all packages

---

## Self-Review

### Spec coverage
- §1 searchMethod fix → Task 6 ✅
- §2 OOS validation wiring (runSingleBacktest, top-K, OOS flow) → Tasks 3+4 ✅
- §3 Regime storage (detectRegimeForExperiment, UpdateMarketRegime, processOne) → Tasks 3+4 ✅
- §4 OOS DB schema + proto wiring (migration, struct, SQL, candidateToProto) → Tasks 1+2+5 ✅
- §5 File change map → All 8 files covered across 7 tasks ✅
- §6 Acceptance criteria → Task 7 verification ✅

### Placeholder check
- No "TBD", "TODO", or vague references ✅
- All code steps contain exact code ✅
- All commands have expected output ✅
- No "add error handling" without actual code ✅

### Type consistency
- `candidateResult.OOSScore` defined in Task 4 Step 1 as `*float64` → used in Task 4 Step 8 as `&oosScored.Score` → mapped in Task 5 as `c.OOSScore` → matches proto `*float64` ✅
- `runSingleBacktest(userID uuid.UUID, ..., regime ai.MarketRegime)` → called with `exp.UserID` and `regime` in Task 4 Step 2 + Step 8 ✅
- `detectRegimeForExperiment` returns `ai.MarketRegime` → stored as `regime.String()` in `UpdateMarketRegime` → consistent with TEXT column ✅

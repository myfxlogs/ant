package strategy

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"go.uber.org/zap"
	"google.golang.org/protobuf/proto"

	antv1 "anttrader/gen/proto/ant/v1"
	"anttrader/internal/repository"
	"anttrader/strategy/backtest"
	"anttrader/strategy/sdk"
	"anttrader/tools/mql2go"
)

const (
	listenTimeout  = 30 * time.Second // fallback poll interval when no PG notification received
	leaseFor       = 60 * time.Second
	defaultWorkers = 3
)

// backtestWorker waits for new PENDING backtest runs via PG LISTEN/NOTIFY,
// with a 30s fallback ticker for safety (Push-First pattern). This replaces
// the previous 3-second polling, which violated the project's push-first rule.
func (s *StrategyExecutionServer) backtestWorker(ctx context.Context, workerID int) {
	// Subscribe to PG notifications for new pending runs.
	notifCh, listenCancel, _ := s.pgListen.Listen(ctx, "backtest_pending")
	if listenCancel != nil {
		defer listenCancel()
	}

	ticker := time.NewTicker(listenTimeout)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-notifCh:
			// New pending run inserted — wake up immediately.
		case <-ticker.C:
			// Safety fallback in case notification was missed.
		}

		// Claim next pending run (atomic SKIP LOCKED — concurrent-safe).
		leaseUntil := time.Now().Add(leaseFor)
		run, err := s.backtestRepo.ClaimNextForWork(ctx, leaseUntil)
		if err != nil {
			s.log.Warn("backtest worker: ClaimNextForWork",
				zap.Int("worker", workerID), zap.Error(err))
			continue
		}
		if run == nil {
			continue // no pending work
		}

		s.log.Info("backtest worker: claimed run",
			zap.Int("worker", workerID),
			zap.String("run_id", run.ID.String()),
			zap.String("symbol", run.Symbol),
			zap.String("timeframe", run.Timeframe))

		s.executeBacktestRun(ctx, run, leaseFor)
	}
}

// backtestParams holds extracted parameters from a BacktestRun for execution.
type backtestParams struct {
	code           string
	initialCapital string
	commission     string
	slippage       string
	leverage       string
	tradeDir       antv1.TradeDirection
	strictMode     bool
	strategyCfg    *antv1.StrategyConfig
}

// extractBacktestParams extracts and validates parameters from a BacktestRun.
func extractBacktestParams(run *repository.BacktestRun) (backtestParams, error) {
	p := backtestParams{
		initialCapital: "10000", commission: "0.001", slippage: "0", leverage: "1",
		tradeDir: antv1.TradeDirection_TRADE_DIRECTION_BOTH, strictMode: true,
	}
	if run.StrategyCode != nil {
		p.code = *run.StrategyCode
	}
	if p.code == "" {
		return p, fmt.Errorf("strategy code is empty")
	}
	if run.InitialCapital != nil {
		p.initialCapital = run.InitialCapital.String()
	}
	if run.Commission != nil {
		p.commission = run.Commission.String()
	}
	if run.Slippage != nil {
		p.slippage = run.Slippage.String()
	}
	if run.Leverage != nil && run.Leverage.GreaterThan(decimal.Zero) {
		p.leverage = run.Leverage.String()
	}
	if run.TradeDirection != nil {
		p.tradeDir = stringToTradeDirection(*run.TradeDirection)
	}
	if run.StrictMode != nil {
		p.strictMode = *run.StrictMode
	}
	if len(run.ConfigSnapshot) > 0 {
		var ec antv1.BacktestExecutionConfig
		opts := proto.UnmarshalOptions{DiscardUnknown: true}
		if err := opts.Unmarshal(run.ConfigSnapshot, &ec); err == nil {
			p.strategyCfg = ec.GetStrategyConfig()
		}
	}
	return p, nil
}

func (s *StrategyExecutionServer) executeBacktestRun(ctx context.Context, run *repository.BacktestRun, leaseFor time.Duration) {
	defer func() {
		if r := recover(); r != nil {
			s.log.Error("backtest worker: panic recovered",
				zap.String("runID", run.ID.String()), zap.Any("panic", r))
			s.failRun(ctx, run, fmt.Sprintf("internal panic: %v", r))
		}
	}()

	params, err := extractBacktestParams(run)
	if err != nil { s.failRun(ctx, run, err.Error()); return }
	execCtx, execCancel := s.startBacktestWatchers(ctx, run, leaseFor)
	defer execCancel()

	klines, err := s.fetchBars(ctx, run)
	if err != nil {
		s.failRun(ctx, run, fmt.Sprintf("fetch bars: %v", err))
		return
	}

	// Go-native backtest engine execution.
	result, err := s.executeGoBacktest(execCtx, run, params, klines)
	if err != nil {
		s.handleBacktestError(ctx, run, execCtx, err)
		return
	}
	s.saveBacktestResult(ctx, run, result)
}

// executeGoBacktest runs a backtest using the Go-native engine.
// MQL source → VMRunner (in-process Bytecode VM) + backtest.Engine.
// Generated Go strategy → GoExecutor (subprocess go run).
func (s *StrategyExecutionServer) executeGoBacktest(ctx context.Context, run *repository.BacktestRun, params backtestParams, klines []*antv1.ExecuteKlineBar) (*antv1.ExecuteBacktestResponse, error) {
	// MQL path: in-process Bytecode VM execution.
	if isMQLStrategy(params.code) {
		return s.executeVMBacktest(ctx, params, klines, run)
	}

	// Go-native compilation path: generated Go strategy via go run.
	if s.goExecutor == nil {
		return nil, fmt.Errorf("GoExecutor not configured — cannot run Go-native backtest")
	}
	symbolInfo := s.fetchSymbolInfo(ctx, run)
	req := buildBacktestRequest(run, params, klines, symbolInfo)
	return s.goExecutor.RunBacktest(ctx, params.code, req)
}

// executeVMBacktest runs a backtest via the in-process Bytecode VM:
// MQL source → CompileMQL → VMRunner → backtest.Engine → ExecuteBacktestResponse.
func (s *StrategyExecutionServer) executeVMBacktest(ctx context.Context, params backtestParams, klines []*antv1.ExecuteKlineBar, run *repository.BacktestRun) (*antv1.ExecuteBacktestResponse, error) {
	s.log.Info("executeVMBacktest: starting",
		zap.Int("klines", len(klines)),
		zap.String("symbol", run.Symbol),
		zap.String("timeframe", run.Timeframe))

	runner, err := mql2go.CompileMQL(params.code)
	if err != nil {
		s.log.Error("executeVMBacktest: compile failed", zap.Error(err))
		return nil, fmt.Errorf("compile MQL: %w", err)
	}
	s.log.Info("executeVMBacktest: compiled successfully")

	// Convert klines to sdk.Bar
	bars := make([]sdk.Bar, len(klines))
	for i, k := range klines {
		bars[i] = sdk.Bar{
			Open:      parseDecimal(k.Open),
			High:      parseDecimal(k.High),
			Low:       parseDecimal(k.Low),
			Close:     parseDecimal(k.Close),
			Volume:    parseInt64(k.Volume),
			Timestamp: k.OpenTimeMs,
		}
	}

	// Build backtest config
	cfg := backtest.Config{
		Symbol:         run.Symbol,
		Timeframe:      run.Timeframe,
		InitialCapital: parseDecimal(params.initialCapital),
		Leverage:       parseInt32(params.leverage),
		Commission:     parseDecimal(params.commission),
		Slippage:       parseDecimal(params.slippage),
		SwapRate:       decimal.NewFromFloat(0.00001),
		StrictMode:     params.strictMode,
		Params:         paramsProtoToMap(run.ParameterOverrides),
	}
	if run.FromTs != nil {
		cfg.StartDate = *run.FromTs
	}
	if run.ToTs != nil {
		cfg.EndDate = *run.ToTs
	}

	// Fetch symbol info for digits/point
	symbolInfo := s.fetchSymbolInfo(ctx, run)
	if symbolInfo != nil {
		cfg.SymbolDigits = symbolInfo.Digits
		if p, err := decimal.NewFromString(symbolInfo.Point); err == nil {
			cfg.SymbolPoint = p
		}
		if v, err := decimal.NewFromString(symbolInfo.VolumeMin); err == nil {
			cfg.VolumeMin = v
		}
		if v, err := decimal.NewFromString(symbolInfo.VolumeMax); err == nil {
			cfg.VolumeMax = v
		}
		if v, err := decimal.NewFromString(symbolInfo.VolumeStep); err == nil {
			cfg.VolumeStep = v
		}
		if v, err := decimal.NewFromString(symbolInfo.ContractSize); err == nil {
			cfg.ContractSize = v
		}
	} else if len(bars) > 0 {
		// Derive digits/point from K-line data when MT gateway is unavailable.
		// Count decimal places in the first bar's close price.
		closeStr := bars[0].Close.String()
		dotIdx := -1
		for i, c := range closeStr {
			if c == '.' {
				dotIdx = i
				break
			}
		}
		digits := int32(0)
		if dotIdx >= 0 {
			digits = int32(len(closeStr) - dotIdx - 1)
		}
		if digits > 8 {
			digits = 8
		}
		cfg.SymbolDigits = digits
		point := decimal.NewFromFloat(1)
		for i := int32(0); i < digits; i++ {
			point = point.Div(decimal.NewFromInt(10))
		}
		cfg.SymbolPoint = point
		// Derive a reasonable spread from point (typical: 10-20 points for forex, 1-2 for crypto)
		cfg.Spread = point.Mul(decimal.NewFromInt(10))
		s.log.Info("executeVMBacktest: derived symbol info from K-lines",
			zap.Int32("digits", digits),
			zap.String("point", point.String()),
			zap.String("spread", cfg.Spread.String()))
	}

	engine := backtest.New(cfg, runner, bars)
	result, err := engine.Run(ctx)
	if err != nil {
		s.log.Error("executeVMBacktest: engine.Run failed", zap.Error(err), zap.Int("bars", len(bars)))
		return &antv1.ExecuteBacktestResponse{Success: false, Error: err.Error()}, nil
	}
	s.log.Info("executeVMBacktest: engine.Run completed",
		zap.Int("trades", len(result.Trades)),
		zap.Int("equity_points", len(result.Equity)),
		zap.Int("bars_processed", len(bars)))

	// Convert backtest.Result → ExecuteBacktestResponse
	resp := &antv1.ExecuteBacktestResponse{
		Success: true,
		Metrics: &antv1.ExecuteBacktestMetrics{
			TotalReturn:   result.Metrics.TotalReturn,
			AnnualReturn:  result.Metrics.AnnualReturn,
			MaxDrawdown:   result.Metrics.MaxDrawdown,
			SharpeRatio:   result.Metrics.SharpeRatio,
			WinRate:       result.Metrics.WinRate,
			ProfitFactor:  result.Metrics.ProfitFactor,
			TotalTrades:   result.Metrics.TotalTrades,
			WinningTrades: result.Metrics.WinningTrades,
			LosingTrades:  result.Metrics.LosingTrades,
		},
	}
	if result.Metrics != nil {
		totalPnl := cfg.InitialCapital.Mul(decimal.NewFromFloat(result.Metrics.TotalReturn))
		resp.Metrics.TotalPnlAbsolute = totalPnl.String()
	}
	for _, ep := range result.Equity {
		resp.EquityCurve = append(resp.EquityCurve, ep.Equity.String())
		resp.EquityTimesMs = append(resp.EquityTimesMs, ep.Time.UnixMilli())
	}
	for i, t := range result.Trades {
		side := "BUY"
		if t.Side == sdk.SideSell {
			side = "SELL"
		}
		resp.Trades = append(resp.Trades, &antv1.ExecuteBacktestTrade{
			Ticket:     int64(i + 1),
			Side:       side,
			Volume:     t.Volume.String(),
			OpenTsMs:   t.EntryTime.UnixMilli(),
			OpenPrice:  t.EntryPrice.String(),
			CloseTsMs:  t.ExitTime.UnixMilli(),
			ClosePrice: t.ExitPrice.String(),
			Pnl:        t.Profit.String(),
			Commission: t.Commission.String(),
			Reason:     t.Comment,
		})
	}

	// ADR-0023 §5.5 #14: ExecutionAssumptions — transparency panel.
	resp.ExecutionAssumptions = &antv1.ExecutionAssumptions{
		SimulationMode:  "KLINE_RANGE",
		SignalTiming:    "next_bar_open",
		FillRule:        "bar_close",
		ActualCommission: cfg.Commission.String(),
		ActualSlippage:  cfg.Slippage.String(),
		ActualLeverage:  fmt.Sprintf("%d", cfg.Leverage),
		TradeDirection:  "both",
	}
	if cfg.StrictMode {
		resp.ExecutionAssumptions.SignalTiming = "next_bar_open"
	} else {
		resp.ExecutionAssumptions.SignalTiming = "same_bar_close"
		resp.ExecutionAssumptions.MtfFallbackReason = "strict_mode disabled"
	}

	// ADR-0023 §5.5 #14: RiskAssessment — basic heuristic from metrics.
	resp.Risk = assessRisk(result.Metrics)

	return resp, nil
}

// StartBacktestWorker launches a pool of background workers that poll for PENDING backtest
// runs and execute them via the Go-native backtest engine. Call this once during server startup.
// Defaults to 3 concurrent workers; each claims a run via SKIP LOCKED.
func (s *StrategyExecutionServer) StartBacktestWorker(ctx context.Context) {
	s.log.Info("backtest worker: starting pool", zap.Int("workers", defaultWorkers))

	// Push-first: shared LISTEN for cancel notifications (single connection).
	go s.startCancelListener(ctx)

	for i := 0; i < defaultWorkers; i++ {
		go s.backtestWorker(ctx, i)
	}
}

// startCancelListener subscribes to backtest_cancel PG notifications and
// cancels the corresponding active run's context. Single shared listener
// for all workers — no per-run LISTEN needed.
func (s *StrategyExecutionServer) startCancelListener(ctx context.Context) {
	if s.pgListen == nil {
		s.log.Warn("backtest cancel listener: pgListen unavailable, cancel relies on ticker fallback")
		return
	}
	notifCh, listenCancel, _ := s.pgListen.Listen(ctx, "backtest_cancel")
	if listenCancel != nil {
		defer listenCancel()
	}
	for {
		select {
		case <-ctx.Done():
			return
		case runIDStr, ok := <-notifCh:
			if !ok {
				return
			}
			runID, err := uuid.Parse(runIDStr)
			if err != nil {
				continue
			}
			s.activeCancelsMu.Lock()
			cancelFn, exists := s.activeCancels[runID]
			s.activeCancelsMu.Unlock()
			if exists {
				s.log.Info("backtest cancel listener: cancelling run", zap.String("runID", runIDStr))
				cancelFn()
			}
		}
	}
}

// paramsProtoToMap converts proto binary StrategyParams to a map for ExecuteBacktestRequest.
func paramsProtoToMap(raw []byte) map[string]string {
	if len(raw) == 0 {
		return nil
	}
	var sp antv1.StrategyParams
	if err := proto.Unmarshal(raw, &sp); err != nil {
		return nil
	}
	return sp.GetValues()
}

func parseFloat64(s string) float64 {
	f, _ := strconv.ParseFloat(s, 64)
	return f
}

func parseDecimal(s string) decimal.Decimal {
	d, err := decimal.NewFromString(s)
	if err != nil {
		return decimal.Zero
	}
	return d
}

func parseInt64(s string) int64 {
	n, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		return 0
	}
	return n
}

func parseInt32(s string) int32 {
	n, err := strconv.ParseInt(s, 10, 32)
	if err != nil {
		return 0
	}
	return int32(n)
}

// assessRisk produces a basic ExecuteRiskAssessment from backtest metrics.
// ADR-0023 §5.5 #14: transparency for the user.
func assessRisk(m *antv1.BacktestMetrics) *antv1.ExecuteRiskAssessment {
	if m == nil {
		return &antv1.ExecuteRiskAssessment{Score: 50, Level: "unknown", IsReliable: false}
	}

	score := 50
	var reasons, warnings []string

	// Max drawdown penalty (0-25 points)
	switch {
	case m.MaxDrawdown > 0.5:
		score -= 25
		warnings = append(warnings, "Max drawdown exceeds 50%")
	case m.MaxDrawdown > 0.3:
		score -= 15
		reasons = append(reasons, "High drawdown (30-50%)")
	case m.MaxDrawdown > 0.15:
		score -= 8
		reasons = append(reasons, "Moderate drawdown (15-30%)")
	default:
		reasons = append(reasons, "Low drawdown (<15%)")
	}

	// Sharpe ratio (0-25 points)
	switch {
	case m.SharpeRatio >= 2.0:
		score += 20
		reasons = append(reasons, "Excellent Sharpe ratio (≥2.0)")
	case m.SharpeRatio >= 1.0:
		score += 10
		reasons = append(reasons, "Good Sharpe ratio (≥1.0)")
	case m.SharpeRatio < 0:
		score -= 15
		warnings = append(warnings, "Negative Sharpe ratio")
	default:
		score -= 5
		reasons = append(reasons, "Low Sharpe ratio (<1.0)")
	}

	// Win rate (0-15 points)
	if m.WinRate >= 0.6 {
		score += 10
	} else if m.WinRate < 0.3 {
		score -= 10
		warnings = append(warnings, "Low win rate (<30%)")
	}

	// Trade count reliability
	if m.TotalTrades < 10 {
		warnings = append(warnings, "Insufficient trades for reliable assessment")
	}

	// Profit factor
	if m.ProfitFactor > 0 && m.ProfitFactor < 1.0 {
		score -= 10
		warnings = append(warnings, "Profit factor below 1.0 (unprofitable)")
	}

	// Clamp 0-100
	if score < 0 {
		score = 0
	}
	if score > 100 {
		score = 100
	}

	level := "medium"
	switch {
	case score >= 70:
		level = "low"
	case score < 40:
		level = "high"
	}

	return &antv1.ExecuteRiskAssessment{
		Score:       int32(score),
		Level:       level,
		Reasons:     reasons,
		Warnings:    warnings,
		IsReliable:  m.TotalTrades >= 10,
	}
}

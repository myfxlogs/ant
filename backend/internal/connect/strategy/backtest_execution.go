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

	antv1 "alphaforge/gen/proto/ant/v1"
	"alphaforge/internal/pkg/ptr"
	"alphaforge/internal/repository"
)

// BacktestExecutor executes a backtest in-process without creating a backtest_runs record.
// Implemented by StrategyExecutionServer.
type BacktestExecutor interface {
	ExecuteBacktestDirect(ctx context.Context, code string, overrides []byte, symbol, timeframe string, fromTs, toTs time.Time, backtestRunID string, userID uuid.UUID) (*antv1.ExecuteBacktestResponse, error)
}

// ExecuteBacktestDirect runs a backtest in-process: fetch bars → compile → engine → return response.
// No DB record created, no polling, no persistence. Used by experiment worker for candidate scoring.
// When backtestRunID is non-empty, config (leverage/commission/slippage/swapRate/marginCallLevel/strictMode)
// is inherited from the originating backtest_runs record instead of using hardcoded defaults.
func (s *StrategyExecutionServer) ExecuteBacktestDirect(
	ctx context.Context, code string, overrides []byte, symbol, timeframe string, fromTs, toTs time.Time,
	backtestRunID string, userID uuid.UUID,
) (*antv1.ExecuteBacktestResponse, error) {
	if code == "" {
		return nil, fmt.Errorf("strategy code is empty")
	}
	params := backtestParams{
		code:           code,
		initialCapital: "10000",
		commission:     "0.001",
		slippage:       "0",
		leverage:       "1",
		tradeDir:       antv1.TradeDirection_TRADE_DIRECTION_BOTH,
		strictMode:     true,
	}
	run := &repository.BacktestRun{
		ID:                 uuid.Nil,
		UserID:             uuid.Nil,
		Symbol:             symbol,
		Timeframe:          timeframe,
		FromTs:             &fromTs,
		ToTs:               &toTs,
		Mode:               "KLINE_RANGE",
		StrategyCode:       &code,
		InitialCapital:     ptr.Decimal(decimal.NewFromInt(10000)),
		Commission:         ptr.Decimal(decimal.NewFromFloat(0.001)),
		Slippage:           ptr.Decimal(decimal.Zero),
		Leverage:           ptr.Decimal(decimal.NewFromInt(1)),
		TradeDirection:     ptr.Str("both"),
		StrictMode:         ptr.Bool(true),
		ParameterOverrides: overrides,
	}
	// Inherit config from originating backtest run when available.
	inheritBacktestRunConfig(ctx, s, &params, run, backtestRunID, userID)
	s.log.Info("ExecuteBacktestDirect final params",
		zap.String("leverage", params.leverage),
		zap.String("commission", params.commission),
		zap.String("initialCapital", params.initialCapital),
		zap.String("backtestRunID", backtestRunID),
		zap.String("symbol", symbol),
		zap.String("timeframe", timeframe),
		zap.String("accountID", run.AccountID.String()),
		zap.Int("codeLen", len(params.code)),
		zap.String("codeHead", safeHead(params.code, 80)),
		zap.Int("overridesLen", len(overrides)))
	klines, err := s.fetchBars(ctx, run)
	if err != nil {
		return nil, fmt.Errorf("fetch bars: %w", err)
	}
	if len(klines) == 0 {
		return nil, fmt.Errorf("no K-line data available for %s %s", symbol, timeframe)
	}
	return s.executeGoBacktest(ctx, run, params, klines)
}

func safeHead(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n]
}

// inheritBacktestRunConfig loads config from the originating backtest_runs record
// and applies it to params/run, replacing hardcoded defaults.
func inheritBacktestRunConfig(ctx context.Context, s *StrategyExecutionServer, params *backtestParams, run *repository.BacktestRun, backtestRunID string, userID uuid.UUID) {
	if backtestRunID == "" || s.backtestRepo == nil {
		return
	}
	rid, err := uuid.Parse(backtestRunID)
	if err != nil {
		return
	}
	srcRun, err := s.backtestRepo.GetByID(ctx, userID, rid)
	if err != nil {
		s.log.Warn("failed to load backtest run config for experiment, falling back to defaults",
			zap.String("backtestRunID", backtestRunID),
			zap.Error(err))
		return
	}
	if srcRun == nil {
		return
	}
	s.log.Info("experiment config inheritance",
		zap.String("backtestRunID", backtestRunID),
		zap.String("leverage", params.leverage),
		zap.String("commission", params.commission),
		zap.String("slippage", params.slippage),
		zap.Bool("strictMode", params.strictMode))
	run.AccountID = srcRun.AccountID
	if srcRun.InitialCapital != nil {
		params.initialCapital = srcRun.InitialCapital.String()
		run.InitialCapital = srcRun.InitialCapital
	}
	if srcRun.Commission != nil {
		params.commission = srcRun.Commission.String()
		run.Commission = srcRun.Commission
	}
	if srcRun.Slippage != nil {
		params.slippage = srcRun.Slippage.String()
		run.Slippage = srcRun.Slippage
	}
	if srcRun.Leverage != nil && srcRun.Leverage.GreaterThan(decimal.Zero) {
		params.leverage = strconv.FormatInt(srcRun.Leverage.IntPart(), 10)
		run.Leverage = srcRun.Leverage
	}
	if srcRun.TradeDirection != nil {
		params.tradeDir = stringToTradeDirection(*srcRun.TradeDirection)
		run.TradeDirection = srcRun.TradeDirection
	}
	if srcRun.StrictMode != nil {
		params.strictMode = *srcRun.StrictMode
		run.StrictMode = srcRun.StrictMode
	}
	if len(srcRun.ConfigSnapshot) > 0 {
		run.ConfigSnapshot = srcRun.ConfigSnapshot
		var ec antv1.BacktestExecutionConfig
		opts := proto.UnmarshalOptions{DiscardUnknown: true}
		if err := opts.Unmarshal(srcRun.ConfigSnapshot, &ec); err == nil {
			params.strategyCfg = ec.GetStrategyConfig()
			if ec.GetSwapRate() != "" {
				params.swapRate = ec.GetSwapRate()
			}
			if ec.GetMarginCallLevel() != "" {
				params.marginCallLevel = ec.GetMarginCallLevel()
			}
			if ec.GetSignalTiming() != "" {
				params.signalTiming = ec.GetSignalTiming()
			}
			if ec.GetFillRule() != "" {
				params.fillRule = ec.GetFillRule()
			}
			if ec.GetSimulationMode() != "" {
				params.simulationMode = ec.GetSimulationMode()
			}
		}
	}
}

// startBacktestWatchers starts lease heartbeat and cancel watcher goroutines.
// Returns a derived context that is cancelled when the user requests cancellation.
// Cancel detection uses push-first: the shared LISTEN listener calls the cancel
// function when a backtest_cancel notification arrives. A 30s safety ticker
// provides fallback in case a notification is missed.
func (s *StrategyExecutionServer) startBacktestWatchers(ctx context.Context, run *repository.BacktestRun, leaseFor time.Duration) (context.Context, context.CancelFunc) {
	execCtx, execCancel := context.WithCancel(ctx)

	// Register cancel function for push-based cancel via shared LISTEN listener.
	s.activeCancelsMu.Lock()
	s.activeCancels[run.ID] = execCancel
	s.activeCancelsMu.Unlock()

	go func() {
		ticker := time.NewTicker(leaseFor / 2)
		defer ticker.Stop()
		for {
			select {
			case <-execCtx.Done():
				return
			case <-ticker.C:
				newLease := time.Now().Add(leaseFor)
				if err := s.backtestRepo.ExtendLease(ctx, run.UserID, run.ID, newLease); err != nil {
					s.log.Warn("extend lease failed", zap.String("runID", run.ID.String()), zap.Error(err))
				}
			}
		}
	}()
	// Safety fallback: check cancel status every 30s in case LISTEN notification is missed.
	go func() {
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-execCtx.Done():
				return
			case <-ticker.C:
				status, _, err := s.backtestRepo.GetStatusAndCancelRequestedAt(ctx, run.UserID, run.ID)
				if err != nil {
					s.log.Warn("cancel fallback check failed", zap.String("runID", run.ID.String()), zap.Error(err))
					continue
				}
				if status == StatusCancelRequested {
					s.log.Info("cancelling run per fallback check", zap.String("runID", run.ID.String()))
					execCancel()
					return
				}
			}
		}
	}()
	// Cleanup: unregister from active cancels map when context is done.
	go func() {
		<-execCtx.Done()
		s.activeCancelsMu.Lock()
		delete(s.activeCancels, run.ID)
		s.activeCancelsMu.Unlock()
	}()
	return execCtx, execCancel
}

// fetchBars retrieves K-line data via BarSource when available,
// falling back to direct PG fetch for backward compatibility.
// Auto-fetches missing data from the connected MT broker when PG data is stale.
func (s *StrategyExecutionServer) fetchBars(ctx context.Context, run *repository.BacktestRun) ([]*antv1.ExecuteKlineBar, error) {
	// Ensure PG has data covering the requested range before querying.
	if err := s.ensureBarData(ctx, run.Symbol, run.Timeframe, run.FromTs, run.ToTs, run.AccountID.String()); err != nil {
		s.log.Warn("fetchBars: ensureBarData failed, proceeding with PG data only",
			zap.String("symbol", run.Symbol), zap.Error(err))
	}

	if s.barSource != nil {
		klines, err := s.barSource.Fetch(ctx, run.Symbol, run.Timeframe, run.FromTs, run.ToTs)
		if err != nil {
			return nil, fmt.Errorf("fetch bars from barSource: %w", err)
		}
		return klines, nil
	}
	return s.fetchBacktestKlines(ctx, run)
}

// fetchBacktestKlines retrieves K-line data from PG and converts to proto format.
func (s *StrategyExecutionServer) fetchBacktestKlines(ctx context.Context, run *repository.BacktestRun) ([]*antv1.ExecuteKlineBar, error) {
	if s.marketDataRepo == nil || run.Symbol == "" || run.Timeframe == "" {
		return nil, fmt.Errorf("cannot fetch klines: marketDataRepo=%v symbol=%q timeframe=%q", s.marketDataRepo != nil, run.Symbol, run.Timeframe)
	}
	chBars, err := s.marketDataRepo.GetKlines(ctx, run.Symbol, "", run.Timeframe, run.FromTs, run.ToTs, 100000)
	if err != nil {
		return nil, fmt.Errorf("fetch klines from marketDataRepo: %w", err)
	}
	return klineBarsToProto(chBars), nil
}

// fetchExtraSymbolKlines retrieves K-line data for a secondary symbol in multi-symbol backtests.
// Uses the same timeframe and time range as the primary symbol.
func (s *StrategyExecutionServer) fetchExtraSymbolKlines(ctx context.Context, symbol string, run *repository.BacktestRun) ([]*antv1.ExecuteKlineBar, error) {
	if s.marketDataRepo == nil || symbol == "" || run.Timeframe == "" {
		return nil, nil
	}
	chBars, err := s.marketDataRepo.GetKlines(ctx, symbol, "", run.Timeframe, run.FromTs, run.ToTs, 100000)
	if err != nil {
		return nil, fmt.Errorf("fetch klines for %s: %w", symbol, err)
	}
	return klineBarsToProto(chBars), nil
}

// fetchSymbolInfo queries the MT gateway for live contract metadata.
// Returns nil when no MT session is connected — the engine will
// fall back to K-line data derivation.
func (s *StrategyExecutionServer) fetchSymbolInfo(ctx context.Context, run *repository.BacktestRun) *antv1.SymbolInfo {
	if s.mtHub == nil {
		return nil
	}
	params, err := s.mtHub.SymbolParams(ctx, run.AccountID.String(), []string{run.Symbol})
	if err != nil || len(params) == 0 {
		s.log.Info("symbol info not available from MT gateway — will derive from K-lines",
			zap.String("symbol", run.Symbol), zap.Error(err))
		return nil
	}
	p := params[0]
	// Compute point as decimal string to preserve precision (no float64).
	point := decimal.New(1, -p.Digits).String()

	info := &antv1.SymbolInfo{
		Digits:       p.Digits,
		Point:        point,
		ContractSize: p.LotSize.String(),
		StopsLevel:   p.StopLevel,
		TickValue:    p.PointValue.String(),
	}
	// Only set volume fields when the broker provides non-zero values
	// (MT4 may not have these; use sensible defaults).
	if p.LotMin.IsPositive() {
		info.VolumeMin = p.LotMin.String()
	}
	if p.LotMax.IsPositive() {
		info.VolumeMax = p.LotMax.String()
	}
	if p.LotStep.IsPositive() {
		info.VolumeStep = p.LotStep.String()
	}
	return info
}

func (s *StrategyExecutionServer) handleBacktestError(ctx context.Context, run *repository.BacktestRun, execCtx context.Context, err error) {
	if execCtx.Err() != nil {
		s.log.Info("backtest worker: run cancelled", zap.String("runID", run.ID.String()))
		now := time.Now()
		if uerr := s.backtestRepo.UpdateAsyncFields(ctx, run.UserID, run.ID, StatusCanceled, "cancelled by user", nil, &now, nil, nil); uerr != nil {
			s.log.Error("update backtest run to CANCELED failed", zap.Error(uerr), zap.String("runID", run.ID.String()))
		}
		return
	}
	s.log.Error("backtest worker: backtest execution failed", zap.String("runID", run.ID.String()), zap.Error(err))
	s.failRun(ctx, run, fmt.Sprintf("backtest execution failed: %v", err))
}

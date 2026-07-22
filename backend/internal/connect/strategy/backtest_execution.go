package strategy

import (
	"context"
	"fmt"
	"time"

	"github.com/shopspring/decimal"
	"go.uber.org/zap"

	antv1 "alphaforge/gen/proto/ant/v1"
	"alphaforge/internal/repository"
)

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
// falling back to direct ClickHouse fetch for backward compatibility.
func (s *StrategyExecutionServer) fetchBars(ctx context.Context, run *repository.BacktestRun) ([]*antv1.ExecuteKlineBar, error) {
	if s.barSource != nil {
		klines, err := s.barSource.Fetch(ctx, run.Symbol, run.Timeframe, run.FromTs, run.ToTs)
		if err != nil {
			return nil, fmt.Errorf("fetch bars from barSource: %w", err)
		}
		return klines, nil
	}
	return s.fetchBacktestKlines(ctx, run)
}

// fetchBacktestKlines retrieves K-line data from ClickHouse and converts to proto format.
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

func buildBacktestRequest(run *repository.BacktestRun, params backtestParams, klines []*antv1.ExecuteKlineBar, symbolInfo *antv1.SymbolInfo) *antv1.ExecuteBacktestRequest {
	fromMs, toMs := int64(0), int64(0)
	if run.FromTs != nil { fromMs = run.FromTs.UnixMilli() }
	if run.ToTs != nil { toMs = run.ToTs.UnixMilli() }
	return &antv1.ExecuteBacktestRequest{
		StrategyId: run.ID.String(), StrategyCode: params.code,
		Symbol: run.Symbol, Timeframe: run.Timeframe,
		StartDateMs: fromMs, EndDateMs: toMs,
		InitialCapital: params.initialCapital, Commission: params.commission,
		SlippageRate: params.slippage, SlippageMode: "fixed", SlippageSeed: 42,
		SwapRate: "0.00001", // standard FX overnight swap rate
		Leverage: params.leverage, TradeDirection: params.tradeDir,
		StrictMode: params.strictMode, StrategyConfig: params.strategyCfg,
		Klines: klines, StrategyParams: paramsProtoToMap(run.ParameterOverrides),
		SymbolInfo: symbolInfo,
	}
}

func (s *StrategyExecutionServer) handleBacktestError(ctx context.Context, run *repository.BacktestRun, execCtx context.Context, err error) {
	if execCtx.Err() != nil {
		s.log.Info("backtest worker: run cancelled", zap.String("runID", run.ID.String()))
		now := time.Now()
		if uerr := s.backtestRepo.UpdateAsyncFields(ctx, run.UserID, run.ID, StatusCanceled, "cancelled by user", nil, &now, nil); uerr != nil {
			s.log.Error("update backtest run to CANCELED failed", zap.Error(uerr), zap.String("runID", run.ID.String()))
		}
		return
	}
	s.log.Error("backtest worker: backtest execution failed", zap.String("runID", run.ID.String()), zap.Error(err))
	s.failRun(ctx, run, fmt.Sprintf("backtest execution failed: %v", err))
}

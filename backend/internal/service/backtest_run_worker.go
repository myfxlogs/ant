package service

import (
	"context"
	"encoding/json"
	"errors"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"anttrader/internal/repository"
	"anttrader/pkg/logger"
)

type BacktestRunWorker struct {
	backtestRun *BacktestRunService
	pythonSvc   *PythonStrategyService
	klineSvc    *KlineService
	dynamicCfg  *DynamicConfigService
	datasetSvc  *BacktestDatasetService
	tickDataset *TickDatasetService

	pollInterval  time.Duration
	leaseDuration time.Duration
}

func NewBacktestRunWorker(backtestRun *BacktestRunService, pythonSvc *PythonStrategyService, klineSvc *KlineService, dynamicCfg *DynamicConfigService, datasetSvc *BacktestDatasetService, tickDataset *TickDatasetService) *BacktestRunWorker {
	return &BacktestRunWorker{
		backtestRun:    backtestRun,
		pythonSvc:      pythonSvc,
		klineSvc:       klineSvc,
		dynamicCfg:     dynamicCfg,
		datasetSvc:     datasetSvc,
		tickDataset:    tickDataset,
		pollInterval:   1 * time.Second,
		leaseDuration:  45 * time.Second,
	}
}

func timeframeToMinutes(tf string) int {
	tf = strings.ToUpper(strings.TrimSpace(tf))
	if tf == "" {
		return 60
	}
	unit := tf[len(tf)-1:]
	numStr := tf[:len(tf)-1]
	n, err := strconv.Atoi(numStr)
	if err != nil || n <= 0 {
		return 60
	}
	switch unit {
	case "M":
		return n
	case "H":
		return n * 60
	case "D":
		return n * 60 * 24
	default:
		return 60
	}
}

func (w *BacktestRunWorker) Start(ctx context.Context) error {
	if w == nil || w.backtestRun == nil {
		return errors.New("backtest run worker not initialized")
	}
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		leaseUntil := time.Now().Add(w.leaseDuration)
		run, err := w.backtestRun.ClaimNextForWork(ctx, leaseUntil)
		if err != nil {
			logger.Warn("backtest-run-worker: claim failed", zap.Error(err))
			// If DB is temporarily unavailable, avoid tight-loop.
			t := time.NewTimer(w.pollInterval)
			select {
			case <-ctx.Done():
				t.Stop()
				return ctx.Err()
			case <-t.C:
			}
			continue
		}
		if run == nil {
			t := time.NewTimer(w.pollInterval)
			select {
			case <-ctx.Done():
				t.Stop()
				return ctx.Err()
			case <-t.C:
			}
			continue
		}

		_ = w.executeOne(ctx, run)
	}
}

func (w *BacktestRunWorker) ExecuteOne(ctx context.Context, run *repository.BacktestRun) error {
	return w.executeOne(ctx, run)
}

func (w *BacktestRunWorker) executeOne(ctx context.Context, run *repository.BacktestRun) error {
	if run == nil {
		return nil
	}
	userID := run.UserID
	runID := run.ID

	execCtx, cancel := context.WithTimeout(ctx, 15*time.Minute)
	defer cancel()

	leaseTicker := time.NewTicker(15 * time.Second)
	defer leaseTicker.Stop()

	// Fast-path: cancel requested before we start heavy work.
	_, cancelAt, sErr := w.backtestRun.GetStatusAndCancelRequestedAt(execCtx, userID, runID)
	if sErr == nil && cancelAt != nil {
		return w.backtestRun.MarkCanceled(execCtx, userID, runID, "canceled")
	}

	doneCh := make(chan error, 1)
	go func() {
		doneCh <- w.runBacktest(execCtx, run)
	}()

	for {
		select {
		case <-execCtx.Done():
			_ = w.backtestRun.MarkFailed(context.Background(), userID, runID, execCtx.Err().Error())
			return execCtx.Err()
		case err := <-doneCh:
			return err
		case <-leaseTicker.C:
			_ = w.backtestRun.ExtendLease(context.Background(), userID, runID, time.Now().Add(w.leaseDuration))
			_, cancelAt, err := w.backtestRun.GetStatusAndCancelRequestedAt(context.Background(), userID, runID)
			if err == nil && cancelAt != nil {
				cancel()
			}
		}
	}
}

func (w *BacktestRunWorker) runBacktest(ctx context.Context, run *repository.BacktestRun) error {
	if w.pythonSvc == nil {
		_ = w.backtestRun.MarkFailed(ctx, run.UserID, run.ID, "python service not available")
		return nil
	}
	if w.klineSvc == nil {
		_ = w.backtestRun.MarkFailed(ctx, run.UserID, run.ID, "kline service not available")
		return nil
	}
	if run.StrategyCode == nil || *run.StrategyCode == "" {
		_ = w.backtestRun.MarkFailed(ctx, run.UserID, run.ID, "strategy code not found")
		return nil
	}
	if run.InitialCapital == nil || *run.InitialCapital <= 0 {
		_ = w.backtestRun.MarkFailed(ctx, run.UserID, run.ID, "initial capital invalid")
		return nil
	}

	// Prepare inputs.
	klines, ticks, costPtr, extraKlines, prepErr := w.prepareBacktestInputs(ctx, run)
	if prepErr != nil {
		_ = w.backtestRun.MarkFailed(ctx, run.UserID, run.ID, prepErr.Error())
		return nil
	}

	resp, err := w.pythonSvc.RunBacktest(ctx, uuid.New(), *run.StrategyCode, klines, ticks, run.Symbol, run.Timeframe, *run.InitialCapital, costPtr, extraKlines)
	if err != nil {
		if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			_, cancelAt, sErr := w.backtestRun.GetStatusAndCancelRequestedAt(context.Background(), run.UserID, run.ID)
			if sErr == nil && cancelAt != nil {
				_ = w.backtestRun.MarkCanceled(context.Background(), run.UserID, run.ID, "canceled")
				return nil
			}
		}
		_ = w.backtestRun.MarkFailed(ctx, run.UserID, run.ID, err.Error())
		return nil
	}
	if resp == nil || !resp.Success {
		errMsg := "backtest failed"
		if resp != nil && resp.Error != "" {
			errMsg = resp.Error
		}
		_ = w.backtestRun.MarkFailed(ctx, run.UserID, run.ID, errMsg)
		return nil
	}

	metricsJSON := MarshalMetricsWithTrades(resp.Metrics, resp.Trades)
	equityJSON, _ := json.Marshal(resp.EquityCurve)
	_ = w.backtestRun.MarkSucceeded(ctx, run.UserID, run.ID, metricsJSON, equityJSON)
	return nil
}

func (w *BacktestRunWorker) prepareBacktestInputs(ctx context.Context, run *repository.BacktestRun) ([]*KlineResponse, []QuoteTickPython, *BacktestCostModel, map[string][]*KlineResponse, error) {
	var (
		klines      []*KlineResponse
		ticks       []QuoteTickPython
		costPtr     *BacktestCostModel
		extraKlines map[string][]*KlineResponse
	)

	switch run.Mode {
	case "DATASET":
		if w.datasetSvc == nil {
			return nil, nil, nil, nil, errors.New("dataset service not available")
		}
		if run.DatasetID == nil || *run.DatasetID == uuid.Nil {
			return nil, nil, nil, nil, errors.New("dataset_id is required")
		}
		k, err := w.datasetSvc.GetFrozenDatasetKlines(ctx, run.UserID, *run.DatasetID, 0)
		if err != nil {
			return nil, nil, nil, nil, err
		}
		klines = k
		if c, ok, _ := w.datasetSvc.GetFrozenDatasetCostModel(ctx, run.UserID, *run.DatasetID); ok {
			costPtr = c
		}
		// DATASET mode does not currently carry extra-symbol klines; secondary
		// symbols are ignored in this mode. (Future work: extend datasets to
		// snapshot multi-symbol bars alongside the primary.)

	case "KLINE_RANGE":
		if run.FromTs == nil || run.ToTs == nil {
			return nil, nil, nil, nil, errors.New("from/to are required")
		}
		fromUTC := run.FromTs.UTC()
		toUTC := run.ToTs.UTC()
		nowUTC := time.Now().UTC()
		if toUTC.After(nowUTC) {
			toUTC = nowUTC
		}
		if !fromUTC.Before(toUTC) {
			return nil, nil, nil, nil, errors.New("from must be < to (after normalization)")
		}

		tfMin := timeframeToMinutes(run.Timeframe)
		if tfMin <= 0 {
			tfMin = 60
		}
		durMin := int(toUTC.Sub(fromUTC).Minutes())
		estBars := durMin/tfMin + 50
		if estBars < 200 {
			estBars = 200
		}
		if estBars > 20000 {
			estBars = 20000
		}

		k, err := w.klineSvc.GetKlines(ctx, run.UserID, run.AccountID, &KlineRequest{
			AccountID: run.AccountID.String(),
			Symbol:    run.Symbol,
			Timeframe: run.Timeframe,
			From:      fromUTC.Format(time.RFC3339),
			To:        toUTC.Format(time.RFC3339),
			Count:     estBars,
		})
		if err != nil {
			return nil, nil, nil, nil, err
		}
		if len(k) == 0 {
			return nil, nil, nil, nil, errors.New("no kline data returned for the given range")
		}
		klines = k

		// Phase B2: fetch secondary symbols' klines over the same range. Each
		// missing / empty secondary symbol is silently skipped; the Python
		// engine treats an empty map as "single-symbol mode". We don't fail
		// the whole run when a single extra symbol has no data \u2014 the strategy
		// can detect that via the zero-length ndarray in ``closes_by_symbol``.
		for _, sym := range run.ExtraSymbols {
			if sym == "" || sym == run.Symbol {
				continue
			}
			ek, eerr := w.klineSvc.GetKlines(ctx, run.UserID, run.AccountID, &KlineRequest{
				AccountID: run.AccountID.String(),
				Symbol:    sym,
				Timeframe: run.Timeframe,
				From:      fromUTC.Format(time.RFC3339),
				To:        toUTC.Format(time.RFC3339),
				Count:     estBars,
			})
			if eerr != nil {
				logger.Warn("backtest-run-worker: extra-symbol klines fetch failed",
					zap.String("symbol", sym), zap.Error(eerr))
				continue
			}
			if len(ek) == 0 {
				logger.Warn("backtest-run-worker: extra-symbol returned no klines",
					zap.String("symbol", sym))
				continue
			}
			if extraKlines == nil {
				extraKlines = make(map[string][]*KlineResponse, len(run.ExtraSymbols))
			}
			extraKlines[sym] = ek
		}

	default:
		return nil, nil, nil, nil, errors.New("unknown mode")
	}

	if w.tickDataset != nil {
		fromTick, toTick := time.Now().Add(-24*time.Hour), time.Now()
		if run.FromTs != nil {
			fromTick = run.FromTs.UTC()
		}
		if run.ToTs != nil {
			toTick = run.ToTs.UTC()
		}
		dsid, err := w.tickDataset.CreateFrozenTickDatasetFromRedis(ctx, run.UserID, run.AccountID, run.Symbol, fromTick, toTick, 200000)
		if err == nil && dsid != uuid.Nil {
			rows, rerr := w.tickDataset.GetFrozenTickDatasetTicks(ctx, run.UserID, dsid, 200000)
			if rerr == nil {
				ticks = make([]QuoteTickPython, 0, len(rows))
				for _, r := range rows {
					if r == nil {
						continue
					}
					ticks = append(ticks, QuoteTickPython{Time: r.Time, Bid: r.Bid, Ask: r.Ask, Symbol: run.Symbol})
				}
			}
		}
	}

	if costPtr == nil {
		cost := ResolveBacktestCostModel(ctx, w.dynamicCfg)
		costPtr = &cost
	}

	return klines, ticks, costPtr, extraKlines, nil
}

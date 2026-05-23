package connect

import (
	"context"
	"strconv"
	"strings"
	"time"

	"connectrpc.com/connect"
	"github.com/google/uuid"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/timestamppb"

	v1 "anttrader/gen/proto"
	"anttrader/internal/interceptor"
	"anttrader/internal/service"
)

type PythonStrategyService struct {
	pythonSvc   *service.PythonStrategyService
	klineSvc    *service.KlineService
	dynamicCfg  *service.DynamicConfigService
	datasetSvc  *service.BacktestDatasetService
	tickDataset *service.TickDatasetService
	backtestRun *service.BacktestRunService
	streamSvc   *StreamService
}

func NewPythonStrategyService(pythonSvc *service.PythonStrategyService, klineSvc *service.KlineService, dynamicCfg *service.DynamicConfigService, datasetSvc *service.BacktestDatasetService, tickDataset *service.TickDatasetService, backtestRun *service.BacktestRunService, streamSvc *StreamService) *PythonStrategyService {
	return &PythonStrategyService{
		pythonSvc:   pythonSvc,
		klineSvc:    klineSvc,
		dynamicCfg:  dynamicCfg,
		datasetSvc:  datasetSvc,
		tickDataset: tickDataset,
		backtestRun: backtestRun,
		streamSvc:   streamSvc,
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

func (s *PythonStrategyService) getIntConfig(ctx context.Context, key string, defaultVal int) int {
	if s.dynamicCfg == nil {
		return defaultVal
	}
	val, enabled, _ := s.dynamicCfg.GetInt(ctx, key, defaultVal)
	if !enabled {
		return defaultVal
	}
	if val <= 0 {
		return defaultVal
	}
	return val
}

func (s *PythonStrategyService) Execute(ctx context.Context, req *connect.Request[v1.ExecuteStrategyRequest]) (*connect.Response[v1.ExecuteStrategyResponse], error) {
	if err := interceptor.RequireScopes(ctx, interceptor.ScopeStrategyRun); err != nil {
		return nil, err
	}

	userID, err := getUserIDFromContext(ctx)
	if err != nil {
		return nil, err
	}

	accountID, err := uuid.Parse(req.Msg.AccountId)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}

	// Preview (single-step) semantics: always run with recent N klines.
	// Keep the request light to avoid slow UI.
	previewBars := s.getIntConfig(ctx, "strategy.preview_bars", 500)
	tfMin := timeframeToMinutes(req.Msg.Timeframe)
	// Safety multiplier: request a slightly wider time window to ensure enough bars.
	from := time.Now().Add(-time.Duration(previewBars*tfMin*2) * time.Minute).Format("2006-01-02T15:04:05")
	klines, err := s.klineSvc.GetKlines(ctx, userID, accountID, &service.KlineRequest{
		AccountID: req.Msg.AccountId,
		Symbol:    req.Msg.Symbol,
		Timeframe: req.Msg.Timeframe,
		From:      from,
		Count:     previewBars,
	})
	if err != nil {
		return connect.NewResponse(&v1.ExecuteStrategyResponse{
			Success: false,
			Error:   err.Error(),
		}), nil
	}

	resp, err := s.pythonSvc.ExecuteStrategy(ctx, uuid.New(), req.Msg.Code, klines, req.Msg.Symbol, req.Msg.Timeframe, nil)
	if err != nil {
		return connect.NewResponse(&v1.ExecuteStrategyResponse{
			Success: false,
			Error:   err.Error(),
		}), nil
	}

	response := &v1.ExecuteStrategyResponse{
		Success: resp.Success,
		Logs:    resp.Logs,
		Error:   resp.Error,
	}

	if resp.Signal != nil {
		response.Signal = &v1.StrategySignal{
			Id:         uuid.New().String(),
			StrategyId: resp.Signal.Symbol,
			AccountId:  req.Msg.AccountId,
			Symbol:     resp.Signal.Symbol,
			SignalType: resp.Signal.Signal,
			Volume:     resp.Signal.Volume,
			Price:      resp.Signal.Price,
			StopLoss:   resp.Signal.StopLoss,
			TakeProfit: resp.Signal.TakeProfit,
			Reason:     resp.Signal.Reason,
			CreatedAt:  timestamppb.Now(),
		}
	}

	return connect.NewResponse(response), nil
}

func (s *PythonStrategyService) Validate(ctx context.Context, req *connect.Request[v1.ValidateStrategyRequest]) (*connect.Response[v1.ValidateStrategyResponse], error) {
	if err := interceptor.RequireScopes(ctx, interceptor.ScopeStrategyRun); err != nil {
		return nil, err
	}

	_, err := getUserIDFromContext(ctx)
	if err != nil {
		return nil, err
	}

	resp, err := s.pythonSvc.ValidateStrategy(ctx, req.Msg.Code)
	if err != nil {
		return connect.NewResponse(&v1.ValidateStrategyResponse{
			Valid:    false,
			Errors:   []string{err.Error()},
			Warnings: []string{},
		}), nil
	}

	return connect.NewResponse(&v1.ValidateStrategyResponse{
		Valid:    resp.Valid,
		Errors:   resp.Errors,
		Warnings: resp.Warnings,
	}), nil
}

func (s *PythonStrategyService) Backtest(ctx context.Context, req *connect.Request[v1.BacktestStrategyRequest]) (*connect.Response[v1.BacktestStrategyResponse], error) {
	if err := interceptor.RequireScopes(ctx, interceptor.ScopeStrategyRun); err != nil {
		return nil, err
	}

	userID, err := getUserIDFromContext(ctx)
	if err != nil {
		return nil, err
	}

	accountID, err := uuid.Parse(req.Msg.AccountId)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}

	// Dataset semantics:
	// - if dataset_id provided: read frozen snapshot only (reproducible)
	// - else: fetch klines (DB-first with optional backfill) and freeze as a dataset
	var (
		klines    []*service.KlineResponse
		datasetID string
		costPtr   *service.BacktestCostModel
	)
	if req.Msg.GetDatasetId() != "" {
		if s.datasetSvc == nil {
			return connect.NewResponse(&v1.BacktestStrategyResponse{Success: false, Error: "dataset service not available"}), nil
		}
		dsid, err := uuid.Parse(req.Msg.GetDatasetId())
		if err != nil {
			return connect.NewResponse(&v1.BacktestStrategyResponse{Success: false, Error: "invalid dataset_id"}), nil
		}
		klines, err = s.datasetSvc.GetFrozenDatasetKlines(ctx, userID, dsid, 0)
		if err != nil {
			return connect.NewResponse(&v1.BacktestStrategyResponse{Success: false, Error: err.Error()}), nil
		}
		if c, ok, _ := s.datasetSvc.GetFrozenDatasetCostModel(ctx, userID, dsid); ok {
			costPtr = c
		}
		datasetID = dsid.String()
	} else {
		// Backtest semantics: use a longer kline window.
		backtestMonths := s.getIntConfig(ctx, "strategy.backtest_window_months", 3)
		from := time.Now().AddDate(0, -backtestMonths, 0).Format("2006-01-02T15:04:05")

		tfMin := timeframeToMinutes(req.Msg.Timeframe)
		// Estimate bars for the window; cap to avoid overly heavy calls.
		windowMinutes := backtestMonths * 30 * 24 * 60
		estBars := windowMinutes/tfMin + 50
		if estBars < 500 {
			estBars = 500
		}
		if estBars > 10000 {
			estBars = 10000
		}
		k, kErr := s.klineSvc.GetKlines(ctx, userID, accountID, &service.KlineRequest{
			AccountID: req.Msg.AccountId,
			Symbol:    req.Msg.Symbol,
			Timeframe: req.Msg.Timeframe,
			From:      from,
			Count:     estBars,
		})
		if kErr != nil {
			return connect.NewResponse(&v1.BacktestStrategyResponse{Success: false, Error: kErr.Error()}), nil
		}
		klines = k
		if s.datasetSvc != nil {
			fromT, _ := time.Parse("2006-01-02T15:04:05", from)
			cost := service.ResolveBacktestCostModel(ctx, s.dynamicCfg)
			dsid, err := s.datasetSvc.CreateFrozenDatasetFromKlines(ctx, userID, accountID, req.Msg.Symbol, req.Msg.Timeframe, &fromT, nil, estBars, klines, &cost)
			if err == nil {
				datasetID = dsid.String()
				costPtr = &cost
			}
		}
	}

	// Best-effort tick dataset:
	// - freeze quote_tick events into DB for reproducibility
	// - replay ticks and send to python for tick-driven matching
	var ticks []service.QuoteTickPython
	if s.tickDataset != nil {
		fromT, toT := time.Now().Add(-24*time.Hour), time.Now()
		if len(klines) > 0 {
			if t, err := time.Parse(time.RFC3339, klines[0].OpenTime); err == nil {
				fromT = t
			}
			if t, err := time.Parse(time.RFC3339, klines[len(klines)-1].CloseTime); err == nil {
				toT = t
			}
		}
		dsid, err := s.tickDataset.CreateFrozenTickDatasetFromRedis(ctx, userID, accountID, req.Msg.Symbol, fromT, toT, 200000)
		if err == nil && dsid != uuid.Nil {
			rows, rerr := s.tickDataset.GetFrozenTickDatasetTicks(ctx, userID, dsid, 200000)
			if rerr == nil {
				ticks = make([]service.QuoteTickPython, 0, len(rows))
				for _, r := range rows {
					if r == nil {
						continue
					}
					ticks = append(ticks, service.QuoteTickPython{Time: r.Time, Bid: r.Bid, Ask: r.Ask, Symbol: req.Msg.Symbol})
				}
			}
		}
	}

	if costPtr == nil {
		cost := service.ResolveBacktestCostModel(ctx, s.dynamicCfg)
		costPtr = &cost
	}
	resp, err := s.pythonSvc.RunBacktest(ctx, uuid.New(), req.Msg.Code, klines, ticks, req.Msg.Symbol, req.Msg.Timeframe, 10000.0, costPtr, nil)
	if err != nil {
		return connect.NewResponse(&v1.BacktestStrategyResponse{
			Success: false,
			Error:   err.Error(),
		}), nil
	}

	if resp != nil && resp.Success && s.backtestRun != nil {
		var dsUUID *uuid.UUID
		if datasetID != "" {
			if id, perr := uuid.Parse(datasetID); perr == nil {
				dsUUID = &id
			}
		}
		_, _ = s.backtestRun.Record(ctx, &service.BacktestRunRecordRequest{
			UserID:       userID,
			AccountID:    accountID,
			Symbol:       req.Msg.Symbol,
			Timeframe:    req.Msg.Timeframe,
			DatasetID:    dsUUID,
			StrategyCode: req.Msg.Code,
			CostModel:    costPtr,
			Metrics:      resp.Metrics,
			EquityCurve:  resp.EquityCurve,
		})
	}

	// 回测成功后异步写入 BM25 记忆（best-effort，不阻塞主流程）
	if resp != nil && resp.Success && resp.Metrics != nil {
		metricsMap := map[string]interface{}{
			"total_return":  resp.Metrics.TotalReturn,
			"annual_return": resp.Metrics.AnnualReturn,
			"max_drawdown":  resp.Metrics.MaxDrawdown,
			"sharpe_ratio":  resp.Metrics.SharpeRatio,
			"win_rate":      resp.Metrics.WinRate,
			"profit_factor": resp.Metrics.ProfitFactor,
			"total_trades":  resp.Metrics.TotalTrades,
		}
		go func() {
			bgCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			_ = s.pythonSvc.RecordMemory(bgCtx, req.Msg.Symbol, req.Msg.Timeframe, req.Msg.Code, metricsMap, "")
		}()
	}

	// Publish backtest events (best-effort) to Redis Streams as position_update events.
	if s.streamSvc != nil && len(resp.Events) > 0 {
		for _, ev := range resp.Events {
			typ, _ := ev["type"].(string)
			if typ != "position_open" && typ != "position_close" {
				continue
			}
			ticketF, _ := ev["ticket"].(float64)
			ticket := int64(ticketF)
			side, _ := ev["side"].(string)
			vol, _ := ev["volume"].(float64)
			price, _ := ev["price"].(float64)
			openTime := int64(0)
			closeTime := int64(0)
			if typ == "position_close" {
				closeTime = time.Now().Unix()
			}
			pe := &v1.PositionUpdateEvent{
				AccountId:      req.Msg.AccountId,
				PositionTicket: ticket,
				Symbol:         req.Msg.Symbol,
				Action:         side,
				Volume:         vol,
				OpenPrice:      price,
				ClosePrice:     price,
				OpenTime:       openTime,
				CloseTime:      closeTime,
				Comment:        "backtest",
			}
			s.streamSvc.publishPositionEvent(req.Msg.AccountId, pe)
		}
	}

	response := &v1.BacktestStrategyResponse{
		Success:     resp.Success,
		Error:       resp.Error,
		EquityCurve: resp.EquityCurve,
		DatasetId:   nil,
	}
	if datasetID != "" {
		response.DatasetId = &datasetID
	}

	if resp.Metrics != nil {
		response.Metrics = &v1.BacktestMetrics{}
	}

	return connect.NewResponse(response), nil
}

func (s *PythonStrategyService) GetTemplates(ctx context.Context, req *connect.Request[emptypb.Empty]) (*connect.Response[v1.GetPythonTemplatesResponse], error) {
	_, err := getUserIDFromContext(ctx)
	if err != nil {
		return nil, err
	}

	return connect.NewResponse(&v1.GetPythonTemplatesResponse{
		Templates: []*v1.PythonTemplate{
			{
				Name:        "移动平均交叉策略",
				Description: "基于快慢移动平均线交叉的交易策略",
				Code:        "# 移动平均交叉策略示例\nfast_ma = sma(close, 10)\nslow_ma = sma(close, 20)\nif crossover(fast_ma, slow_ma):\n    buy()",
			},
			{
				Name:        "RSI超买超卖策略",
				Description: "基于RSI指标的超买超卖交易策略",
				Code:        "# RSI策略示例\nrsi_value = rsi(close, 14)\nif rsi_value < 30:\n    buy()\nelif rsi_value > 70:\n    sell()",
			},
		},
	}), nil
}

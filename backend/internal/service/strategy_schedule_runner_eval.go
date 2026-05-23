package service

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"anttrader/internal/model"
	"anttrader/pkg/logger"
)

func (r *StrategyScheduleRunner) evalOnceWithSource(ctx context.Context, schedule *model.StrategySchedule, source string) {
	if r == nil || schedule == nil {
		return
	}
	if strings.TrimSpace(source) == "" {
		source = "unknown"
	}
	throttledScheduleRunnerInfo(
		"schedule.trigger."+schedule.ID.String()+"."+source,
		scheduleLogThrottleWindow(),
		"schedule trigger observed",
		zap.String("schedule_id", schedule.ID.String()),
		zap.String("trigger_source", source),
		zap.Any("counters", snapshotScheduleMetrics(schedule.ID)),
	)
	r.evalOnce(ctx, schedule)
}

func (r *StrategyScheduleRunner) evalOnce(ctx context.Context, schedule *model.StrategySchedule) {
	if r == nil || schedule == nil {
		return
	}
	if r.klineSvc == nil || r.pythonSvc == nil {
		return
	}

	// reload schedule to honor disable quickly
	s, err := r.scheduleRepo.GetByID(ctx, schedule.ID)
	if err != nil || s == nil || !s.IsActive {
		return
	}
	schedule = s

	tpl, err := r.templateRepo.GetByID(ctx, schedule.TemplateID)
	if err != nil || tpl == nil {
		r.persistLastRun(schedule.ID, err)
		return
	}

	// Pull recent klines for indicator calculation.
	klines, err := r.klineSvc.GetKlines(ctx, schedule.UserID, schedule.AccountID, &KlineRequest{
		Symbol:    schedule.Symbol,
		Timeframe: schedule.Timeframe,
		Count:     500,
	})
	if err != nil {
		r.persistLastRun(schedule.ID, err)
		return
	}

	// EA-like: only evaluate once per new bar. If latest bar open_time unchanged, skip.
	latestBarOpen := ""
	if len(klines) > 0 && klines[len(klines)-1] != nil {
		latestBarOpen = klines[len(klines)-1].OpenTime
	}
	if latestBarOpen != "" {
		r.stateMu.Lock()
		st := r.states[schedule.ID]
		prev := ""
		if st != nil {
			prev = st.LastBarOpenTime
		}
		r.stateMu.Unlock()
		if prev != "" && prev == latestBarOpen {
			incrementScheduleMetric(schedule.ID, "duplicate_bar_skips")
			return
		}
		incrementScheduleMetric(schedule.ID, "bar_close_detections")
		throttledScheduleRunnerInfo(
			"schedule.trigger."+schedule.ID.String()+".bar_close",
			scheduleLogThrottleWindow(),
			"schedule trigger observed",
			zap.String("schedule_id", schedule.ID.String()),
			zap.String("trigger_source", "bar_close"),
			zap.String("bar_open_time", latestBarOpen),
		)
	}

	r.updateState(schedule.ID, func(st *ScheduleRuntimeState) {
		if latestBarOpen != "" {
			st.LastBarOpenTime = latestBarOpen
		}
		st.LastEvalAt = time.Now()
	})
	execCtx := r.snapshotExecContext(schedule.ID)
	if execCtx == nil {
		execCtx = map[string]interface{}{}
	}
	// Inject schedule parameters so strategy code can access user config (e.g. lot).
	if p, pErr := schedule.GetParameters(); pErr == nil && p != nil {
		execCtx["params"] = p
		if v, ok := p["lot"]; ok {
			execCtx["lot"] = v
		}
	}

	resp, err := r.pythonSvc.ExecuteStrategy(ctx, schedule.TemplateID, tpl.Code, klines, schedule.Symbol, schedule.Timeframe, execCtx)
	if err != nil {
		r.persistLastRun(schedule.ID, err)
		return
	}
	// Persist strategy runtime state (context['runtime']) if the python service returned it.
	if resp != nil && resp.Runtime != nil {
		r.updateState(schedule.ID, func(st *ScheduleRuntimeState) {
			if st.Data == nil {
				st.Data = map[string]interface{}{}
			}
			for k, v := range resp.Runtime {
				st.Data[k] = v
			}
		})
	}

	var runErr error
	if resp == nil || !resp.Success || resp.Signal == nil {
		r.persistLastRun(schedule.ID, nil)
		return
	}

	sig := resp.Signal
	r.updateState(schedule.ID, func(st *ScheduleRuntimeState) {
		st.LastSignal = sig.Signal
		st.LastSignalAt = time.Now()
	})
	orderType := scheduleOrderTypeFromSignal(sig.Signal)
	if orderType == "" {
		r.persistLastRun(schedule.ID, nil)
		return
	}
	// Special: close positions on CLOSE signal (records execution + order_history like opens).
	if stringsToUpperTrim(orderType) == "CLOSE" {
		if r.tradingSvc == nil || r.gateway == nil {
			runErr = errors.New("close requires trading service and gateway")
			r.persistLastRun(schedule.ID, runErr)
			return
		}
		runErr = r.closeSchedulePositions(ctx, schedule, sig)
		r.persistLastRun(schedule.ID, runErr)
		return
	}

	// Idempotency: prevent repeated execution for the same bar and same signal payload.
	// Key intentionally only depends on the latest bar open_time and signal core fields.
	signalKey := fmt.Sprintf("%s|%s|%s|%.6f|%.6f|%.6f|%.6f", latestBarOpen, schedule.ID.String(), orderType, sig.Volume, sig.Price, sig.StopLoss, sig.TakeProfit)
	r.stateMu.Lock()
	st := r.states[schedule.ID]
	prevKey := ""
	if st != nil {
		prevKey = st.LastSignalKey
	}
	r.stateMu.Unlock()
	if prevKey != "" && prevKey == signalKey {
		incrementScheduleMetric(schedule.ID, "duplicate_signal_skips")
		r.persistLastRun(schedule.ID, nil)
		return
	}

	if r.gateway == nil {
		runErr = errors.New("execution gateway not available")
		r.persistLastRun(schedule.ID, runErr)
		return
	}

	// --- Risk gate ---
	// Parse __risk.* 参数并按需填默认值、检查限额。
	rp := riskParams{}
	if p, pErr := schedule.GetParameters(); pErr == nil && p != nil {
		rp = parseRiskParams(p)
	}
	gate := &riskGate{
		userID:    schedule.UserID,
		accountID: schedule.AccountID,
		symbol:    schedule.Symbol,
		params:    rp,
		countPositions: func(ctx context.Context) (int, error) {
			if r.tradingSvc == nil {
				return 0, nil
			}
			positions, pErr := r.tradingSvc.GetPositions(ctx, schedule.UserID, schedule.AccountID)
			if pErr != nil {
				return 0, pErr
			}
			n := 0
			for _, p := range positions {
				if p != nil && stringsToUpperTrim(p.Symbol) == stringsToUpperTrim(schedule.Symbol) {
					n++
				}
			}
			return n, nil
		},
		readEquity: func(ctx context.Context) (float64, error) {
			acc, aerr := r.accountRepo.GetByID(ctx, schedule.AccountID)
			if aerr != nil || acc == nil {
				return 0, aerr
			}
			return acc.Equity, nil
		},
		peakEquityState: func(current float64) float64 {
			// 读-改-写 peak equity in runtime state.
			peak := current
			r.updateState(schedule.ID, func(st *ScheduleRuntimeState) {
				if st.PeakEquity < current {
					st.PeakEquity = current
				}
				peak = st.PeakEquity
			})
			return peak
		},
		autoDisable: func(ctx context.Context, reason string) {
			r.toggleScheduleInactive(ctx, schedule, reason)
		},
	}
	if ok, reason := gate.allow(ctx); !ok {
		r.persistLastRun(schedule.ID, errors.New("risk gate: "+reason))
		return
	}

	start := time.Now()
	execLog := (*model.StrategyExecutionLog)(nil)
	if r.logSvc != nil {
		execLog = model.NewStrategyExecutionLog(schedule.UserID, schedule.Symbol, schedule.Timeframe)
		execLog.Status = model.StrategyExecutionStatusRunning
		execLog.AccountID = &schedule.AccountID
		execLog.ScheduleID = &schedule.ID
		execLog.TemplateID = &schedule.TemplateID
		execLog.SignalType = model.StrategyExecutionLogSignalType(orderType)
		execLog.SignalPrice = sig.Price
		execLog.SignalVolume = sig.Volume
		execLog.SignalStopLoss = sig.StopLoss
		execLog.SignalTakeProfit = sig.TakeProfit
		if p, pErr := schedule.GetParameters(); pErr == nil && p != nil {
			execLog.StrategyParams = p
		}
		if err := r.logSvc.LogExecution(ctx, execLog); err != nil {
			logger.Warn("schedule execution log insert failed", zap.String("schedule_id", schedule.ID.String()), zap.Error(err))
		}
	}

	req := &OrderSendRequest{
		AccountID:  schedule.AccountID.String(),
		Symbol:     schedule.Symbol,
		Type:       orderType,
		Volume:     sig.Volume,
		Price:      sig.Price,
		StopLoss:   sig.StopLoss,
		TakeProfit: sig.TakeProfit,
		Comment:    fmt.Sprintf("schedule:%s|%s", schedule.ID.String(), sig.Reason),
	}
	// Apply __risk.* 默认值（仅填补信号没给的字段）。
	rp.applySignalDefaults(orderType, req)

	op := model.NewSystemOperationLog(schedule.UserID, model.OperationTypeCreate, "execution", "schedule_v2_execute")
	op.ResourceType = "schedule"
	op.ResourceID = schedule.ID
	op.Status = model.OperationStatusRunning
	callCtx := WithTradeTriggerSource(ctx, TriggerSourceStrategy)
	orderResp, runErr := r.gateway.OrderSend(callCtx, schedule.UserID, req, op)
	if execLog != nil {
		execLog.ExecutionTimeMs = time.Since(start).Milliseconds()
		if runErr != nil {
			execLog.Status = model.StrategyExecutionStatusFailed
			execLog.ErrorMessage = runErr.Error()
		} else {
			execLog.Status = model.StrategyExecutionStatusCompleted
			if orderResp != nil {
				execLog.ExecutedOrderID = fmt.Sprintf("%d", orderResp.Ticket)
				execLog.ExecutedPrice = orderResp.Price
				execLog.ExecutedVolume = orderResp.Volume
				execLog.Profit = orderResp.Profit
			}
		}
		if err := r.logSvc.UpdateExecution(ctx, execLog); err != nil {
			logger.Warn("schedule execution log update failed", zap.String("schedule_id", schedule.ID.String()), zap.Error(err))
		}
	}
	if runErr == nil && r.logSvc != nil && orderResp != nil {
		openTime := time.Now()
		if orderResp.OpenTime != "" {
			if t, perr := time.Parse(time.RFC3339, orderResp.OpenTime); perr == nil {
				openTime = t
			}
		}
		oh := &model.OrderHistory{
			ID:          uuid.New(),
			UserID:      schedule.UserID,
			AccountID:   schedule.AccountID,
			Ticket:      orderResp.Ticket,
			OrderType:   model.OrderHistoryType(orderResp.Type),
			Symbol:      orderResp.Symbol,
			Volume:      orderResp.Volume,
			OpenPrice:   orderResp.Price,
			ClosePrice:  0,
			OpenTime:    openTime,
			CloseTime:   nil,
			StopLoss:    orderResp.StopLoss,
			TakeProfit:  orderResp.TakeProfit,
			Profit:      orderResp.Profit,
			Commission:  0,
			Swap:        0,
			Comment:     orderResp.Comment,
			MagicNumber: orderResp.Magic,
			IsAutoTrade: true,
			ScheduleID:  schedule.ID,
			CreatedAt:   time.Now(),
		}
		if err := r.logSvc.LogOrder(ctx, oh); err != nil {
			logger.Warn("schedule order history insert failed", zap.String("schedule_id", schedule.ID.String()), zap.Error(err))
		}
	}
	if runErr == nil {
		r.updateState(schedule.ID, func(st *ScheduleRuntimeState) {
			st.LastOrderAt = time.Now()
			st.LastSignalKey = signalKey
		})
	}
	r.persistLastRun(schedule.ID, runErr)
}

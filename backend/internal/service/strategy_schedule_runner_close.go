package service

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"anttrader/internal/model"
	"anttrader/pkg/logger"
)

// scheduleOrderTypeFromSignal maps Python strategy.signal to engine order type strings (lowercase).
func scheduleOrderTypeFromSignal(raw string) string {
	s := strings.TrimSpace(raw)
	if s == "" {
		return ""
	}
	u := stringsToUpperTrim(strings.ReplaceAll(s, " ", ""))
	switch u {
	case "BUY", "BUYMARKET", "LONG":
		return "buy"
	case "SELL", "SELLMARKET", "SHORT":
		return "sell"
	case "CLOSE":
		return "close"
	default:
		return strings.ToLower(strings.TrimSpace(s))
	}
}

func parsePositionOpenTime(p *PositionResponse) time.Time {
	if p == nil {
		return time.Now()
	}
	s := strings.TrimSpace(p.OpenTime)
	if s == "" {
		return time.Now()
	}
	if t, err := time.Parse(time.RFC3339, s); err == nil {
		return t
	}
	if t, err := time.Parse("2006-01-02T15:04:05Z", strings.ReplaceAll(s, " ", "T")); err == nil {
		return t
	}
	return time.Now()
}

func orderHistoryTypeFromPosition(pos *PositionResponse) model.OrderHistoryType {
	if pos == nil {
		return model.OrderHistoryTypeBuy
	}
	t := strings.ToLower(strings.TrimSpace(pos.Type))
	if t == "" {
		return model.OrderHistoryTypeBuy
	}
	return model.OrderHistoryType(t)
}

// closeSchedulePositions closes all positions for schedule.Symbol and writes execution + order_history rows.
func (r *StrategyScheduleRunner) closeSchedulePositions(ctx context.Context, schedule *model.StrategySchedule, sig *TradeSignalPython) error {
	if r == nil || schedule == nil || r.tradingSvc == nil || r.gateway == nil {
		return nil
	}
	positions, pErr := r.tradingSvc.GetPositions(ctx, schedule.UserID, schedule.AccountID)
	if pErr != nil {
		return pErr
	}
	var runErr error
	for _, p := range positions {
		if p == nil {
			continue
		}
		if stringsToUpperTrim(p.Symbol) != stringsToUpperTrim(schedule.Symbol) {
			continue
		}
		vol := p.Volume
		if !(vol > 0) {
			continue
		}
		start := time.Now()
		var execLog *model.StrategyExecutionLog
		if r.logSvc != nil {
			execLog = model.NewStrategyExecutionLog(schedule.UserID, schedule.Symbol, schedule.Timeframe)
			execLog.Status = model.StrategyExecutionStatusRunning
			execLog.AccountID = &schedule.AccountID
			execLog.ScheduleID = &schedule.ID
			execLog.TemplateID = &schedule.TemplateID
			execLog.SignalType = model.StrategySignalTypeClose
			execLog.SignalVolume = vol
			execLog.SignalPrice = p.OpenPrice
			params := map[string]interface{}{}
			if sig != nil && sig.Reason != "" {
				params["reason"] = sig.Reason
			}
			if schedParams, spErr := schedule.GetParameters(); spErr == nil && schedParams != nil {
				params["schedule_params"] = schedParams
			}
			if len(params) > 0 {
				execLog.StrategyParams = params
			}
			if err := r.logSvc.LogExecution(ctx, execLog); err != nil {
				logger.Warn("schedule close execution log insert failed", zap.String("schedule_id", schedule.ID.String()), zap.Error(err))
			}
		}

		op := model.NewSystemOperationLog(schedule.UserID, model.OperationTypeCreate, "execution", "schedule_v2_close")
		op.ResourceType = "schedule"
		op.ResourceID = schedule.ID
		op.Status = model.OperationStatusRunning
		callCtx := WithTradeTriggerSource(ctx, TriggerSourceStrategy)
		closeResp, cerr := r.gateway.OrderClose(callCtx, schedule.UserID, &OrderCloseRequest{
			AccountID:   schedule.AccountID.String(),
			Ticket:      p.Ticket,
			Volume:      vol,
			CloseReason: "strategy_exit",
		}, op)

		if execLog != nil {
			execLog.ExecutionTimeMs = time.Since(start).Milliseconds()
			if cerr != nil {
				execLog.Status = model.StrategyExecutionStatusFailed
				execLog.ErrorMessage = cerr.Error()
			} else {
				execLog.Status = model.StrategyExecutionStatusCompleted
				execLog.ExecutedOrderID = fmt.Sprintf("%d", p.Ticket)
				if closeResp != nil {
					execLog.ExecutedPrice = closeResp.Price
					execLog.ExecutedVolume = closeResp.Volume
					execLog.Profit = closeResp.Profit
				}
			}
			if err := r.logSvc.UpdateExecution(ctx, execLog); err != nil {
				logger.Warn("schedule close execution log update failed", zap.String("schedule_id", schedule.ID.String()), zap.Error(err))
			}
		}

		if cerr != nil {
			runErr = cerr
			continue
		}
		if closeResp == nil || r.logSvc == nil {
			continue
		}
		closeTime := time.Now()
		if closeResp.CloseTime != "" {
			if t, err := time.Parse(time.RFC3339, closeResp.CloseTime); err == nil {
				closeTime = t
			}
		}
		n, uerr := r.logSvc.UpdateOrderHistoryClose(ctx, schedule.UserID, schedule.AccountID, schedule.ID, p.Ticket,
			closeResp.Price, closeResp.Profit, closeResp.Swap, closeResp.Commission, closeTime)
		if uerr != nil {
			logger.Warn("schedule order_history close update failed", zap.String("schedule_id", schedule.ID.String()), zap.Error(uerr))
			continue
		}
		if n == 0 {
			oh := &model.OrderHistory{
				ID:          uuid.New(),
				UserID:      schedule.UserID,
				AccountID:   schedule.AccountID,
				Ticket:      p.Ticket,
				OrderType:   orderHistoryTypeFromPosition(p),
				Symbol:      strings.TrimSpace(p.Symbol),
				Volume:      closeResp.Volume,
				OpenPrice:   p.OpenPrice,
				ClosePrice:  closeResp.Price,
				OpenTime:    parsePositionOpenTime(p),
				CloseTime:   &closeTime,
				StopLoss:    p.StopLoss,
				TakeProfit:  p.TakeProfit,
				Profit:      closeResp.Profit,
				Commission:  closeResp.Commission,
				Swap:        closeResp.Swap,
				Comment:     closeResp.Comment,
				MagicNumber: p.Magic,
				IsAutoTrade: true,
				ScheduleID:  schedule.ID,
				CreatedAt:   time.Now(),
			}
			if err := r.logSvc.LogOrder(ctx, oh); err != nil {
				logger.Warn("schedule close order_history insert failed", zap.String("schedule_id", schedule.ID.String()), zap.Error(err))
			}
		}
	}
	return runErr
}

package service

import (
	"context"
	"time"

	"github.com/google/uuid"

	"anttrader/internal/model"
	"anttrader/pkg/faultisol"
	"anttrader/pkg/logger"

	"go.uber.org/zap"
)

func (s *TradingService) OrderSend(ctx context.Context, userID uuid.UUID, req *OrderSendRequest) (*OrderResponse, error) {
	if !tradingWriteEnabled() {
		return nil, ErrTradeWriteDisabled
	}
	var result *OrderResponse

	err := s.executor.Execute(ctx, func(ctx context.Context) error {
		accountID, err := uuid.Parse(req.AccountID)
		if err != nil {
			return err
		}

		account, err := s.getAccountForTrade(ctx, userID, accountID)
		if err != nil {
			return err
		}

		openCount, _ := s.getOpenPositionsCount(ctx, userID, accountID)
		resolver := newRiskRuleResolver()
		riskCtx, err := resolver.resolve(account, req.Symbol, openCount)
		if err != nil {
			recordRiskValidateMetric("error", riskCodeFromError(err), account.MTType, TradeTriggerSourceFromContext(ctx), 0)
			return err
		}
		riskStart := time.Now()
		decision := s.risk().CheckManualOrderSend(ctx, riskCtx, req)
		if decision != nil && !decision.Allowed {
			err := RiskErrorFromDecision(decision)
			recordRiskValidateMetric("reject", riskCodeFromError(err), account.MTType, TradeTriggerSourceFromContext(ctx), time.Since(riskStart))
			recordOrderSendMetric("rejected", riskCodeFromError(err))
			return err
		}
		recordRiskValidateMetric("pass", "OK", account.MTType, TradeTriggerSourceFromContext(ctx), time.Since(riskStart))

		if account.MTType == "MT4" {
			result, err = s.orderSendMT4(ctx, accountID, account, req)
		} else {
			result, err = s.orderSendMT5(ctx, accountID, account, req)
		}

		if err != nil {
			recordOrderSendMetric("failed", err.Error())
			return err
		}
		recordOrderSendMetric("success", "OK")

		s.logTrade(ctx, userID, accountID, "ORDER_SEND", req.Symbol, req.Type, req.Volume, result.Price, result.Ticket, 0, "")

		return nil
	}, faultisol.WithOperation("order_send"), faultisol.WithTimeout(30*time.Second))

	return result, err
}

func (s *TradingService) orderSendMT4(ctx context.Context, accountID uuid.UUID, account *model.MTAccount, req *OrderSendRequest) (*OrderResponse, error) {
	conn, err := s.getMT4Connection(accountID)
	if err != nil {
		if connectErr := s.connManager.Connect(ctx, account); connectErr != nil {
			logger.Error("MT4 reconnect failed",
				zap.String("account_id", accountID.String()),
				zap.Error(connectErr))
			return nil, connectErr
		}
		conn, err = s.getMT4Connection(accountID)
		if err != nil {
			logger.Error("MT4 get connection after reconnect failed",
				zap.String("account_id", accountID.String()),
				zap.Error(err))
			return nil, err
		}
	}

	op, err := ParseOrderTypeMT4(req.Type)
	if err != nil {
		logger.Error("Invalid MT4 order type",
			zap.String("type", req.Type),
			zap.Error(err))
		return nil, err
	}

	order, err := conn.OrderSend(ctx, req.Symbol, op, req.Volume, req.Price, req.StopLoss, req.TakeProfit, req.Slippage, req.Comment, int32(req.Magic))
	if err != nil {
		logger.Error("MT4 order send failed",
			zap.String("account_id", accountID.String()),
			zap.String("symbol", req.Symbol),
			zap.Error(err))
		return nil, err
	}

	return &OrderResponse{
		Ticket:     int64(order.GetTicket()),
		Symbol:     order.GetSymbol(),
		Type:       OrderTypeToString(int32(order.GetType())),
		Volume:     order.GetLots(),
		Price:      order.GetOpenPrice(),
		StopLoss:   order.GetStopLoss(),
		TakeProfit: order.GetTakeProfit(),
		Comment:    order.GetComment(),
		Magic:      int64(order.GetMagicNumber()),
		OpenTime:   order.GetOpenTime().AsTime().Format("2006-01-02T15:04:05Z"),
		Profit:     order.GetProfit(),
	}, nil
}

func (s *TradingService) orderSendMT5(ctx context.Context, accountID uuid.UUID, account *model.MTAccount, req *OrderSendRequest) (*OrderResponse, error) {
	conn, err := s.getMT5Connection(accountID)
	if err != nil {
		if connectErr := s.connManager.Connect(ctx, account); connectErr != nil {
			logger.Error("MT5 reconnect failed",
				zap.String("account_id", accountID.String()),
				zap.Error(connectErr))
			return nil, connectErr
		}
		conn, err = s.getMT5Connection(accountID)
		if err != nil {
			logger.Error("MT5 get connection after reconnect failed",
				zap.String("account_id", accountID.String()),
				zap.Error(err))
			return nil, err
		}
	}

	op, err := ParseOrderTypeMT5(req.Type)
	if err != nil {
		logger.Error("Invalid order type",
			zap.String("type", req.Type),
			zap.Error(err))
		return nil, err
	}

	order, err := conn.OrderSend(ctx, req.Symbol, op, req.Volume, req.Price, req.StopLoss, req.TakeProfit, req.Slippage, req.Comment, req.Magic)
	if err != nil {
		logger.Error("MT5 order send failed",
			zap.String("account_id", accountID.String()),
			zap.String("symbol", req.Symbol),
			zap.Error(err))
		return nil, err
	}

	return &OrderResponse{
		Ticket:     order.GetTicket(),
		Symbol:     order.GetSymbol(),
		Type:       OrderTypeToString(int32(order.GetOrderType())),
		Volume:     order.GetLots(),
		Price:      order.GetOpenPrice(),
		StopLoss:   order.GetStopLoss(),
		TakeProfit: order.GetTakeProfit(),
		Comment:    order.GetComment(),
		Magic:      order.GetExpertId(),
		OpenTime:   order.GetOpenTime().AsTime().Format("2006-01-02T15:04:05Z"),
		Profit:     order.GetProfit(),
	}, nil
}

func (s *TradingService) OrderModify(ctx context.Context, userID uuid.UUID, req *OrderModifyRequest) (*OrderResponse, error) {
	if !tradingWriteEnabled() {
		return nil, ErrTradeWriteDisabled
	}
	var result *OrderResponse

	err := s.executor.Execute(ctx, func(ctx context.Context) error {
		accountID, err := uuid.Parse(req.AccountID)
		if err != nil {
			return err
		}

		account, err := s.getAccountForTrade(ctx, userID, accountID)
		if err != nil {
			return err
		}
		positions, _ := s.GetPositions(ctx, userID, accountID)
		openCount := len(positions)
		pos := findPositionByTicket(positions, req.Ticket)
		symbol := ""
		if pos != nil {
			symbol = pos.Symbol
		}
		resolver := newRiskRuleResolver()
		riskCtx, err := resolver.resolve(account, symbol, openCount)
		if err != nil {
			recordRiskValidateMetric("error", riskCodeFromError(err), account.MTType, TradeTriggerSourceFromContext(ctx), 0)
			return err
		}
		riskStart := time.Now()
		decision := s.risk().CheckManualOrderModify(ctx, riskCtx, req)
		if decision != nil && !decision.Allowed {
			err := RiskErrorFromDecision(decision)
			recordRiskValidateMetric("reject", riskCodeFromError(err), account.MTType, TradeTriggerSourceFromContext(ctx), time.Since(riskStart))
			return err
		}
		recordRiskValidateMetric("pass", "OK", account.MTType, TradeTriggerSourceFromContext(ctx), time.Since(riskStart))

		if account.MTType == "MT4" {
			result, err = s.orderModifyMT4(ctx, accountID, req)
		} else {
			result, err = s.orderModifyMT5(ctx, accountID, req)
		}

		if err != nil {
			return err
		}

		s.logTrade(ctx, userID, accountID, "ORDER_MODIFY", result.Symbol, "", 0, 0, req.Ticket, 0, "")

		return nil
	}, faultisol.WithOperation("order_modify"), faultisol.WithTimeout(30*time.Second))

	return result, err
}

func (s *TradingService) orderModifyMT4(ctx context.Context, accountID uuid.UUID, req *OrderModifyRequest) (*OrderResponse, error) {
	conn, err := s.getMT4Connection(accountID)
	if err != nil {
		return nil, ErrAccountNotConnected
	}

	order, err := conn.OrderModify(ctx, int32(req.Ticket), req.StopLoss, req.TakeProfit, req.Price, "")
	if err != nil {
		return nil, err
	}

	return &OrderResponse{
		Ticket:     int64(order.GetTicket()),
		Symbol:     order.GetSymbol(),
		Type:       OrderTypeToString(int32(order.GetType())),
		Volume:     order.GetLots(),
		Price:      order.GetOpenPrice(),
		StopLoss:   order.GetStopLoss(),
		TakeProfit: order.GetTakeProfit(),
		Comment:    order.GetComment(),
		Magic:      int64(order.GetMagicNumber()),
		OpenTime:   order.GetOpenTime().AsTime().Format("2006-01-02T15:04:05Z"),
		Profit:     order.GetProfit(),
	}, nil
}

func (s *TradingService) orderModifyMT5(ctx context.Context, accountID uuid.UUID, req *OrderModifyRequest) (*OrderResponse, error) {
	conn, err := s.getMT5Connection(accountID)
	if err != nil {
		return nil, ErrAccountNotConnected
	}

	order, err := conn.OrderModify(ctx, req.Ticket, req.StopLoss, req.TakeProfit, req.Price, "")
	if err != nil {
		return nil, err
	}

	return &OrderResponse{
		Ticket:     order.GetTicket(),
		Symbol:     order.GetSymbol(),
		Type:       OrderTypeToString(int32(order.GetOrderType())),
		Volume:     order.GetLots(),
		Price:      order.GetOpenPrice(),
		StopLoss:   order.GetStopLoss(),
		TakeProfit: order.GetTakeProfit(),
		Comment:    order.GetComment(),
		Magic:      order.GetExpertId(),
		OpenTime:   order.GetOpenTime().AsTime().Format("2006-01-02T15:04:05Z"),
		Profit:     order.GetProfit(),
	}, nil
}

func (s *TradingService) OrderClose(ctx context.Context, userID uuid.UUID, req *OrderCloseRequest) (*OrderResponse, error) {
	if !tradingWriteEnabled() {
		return nil, ErrTradeWriteDisabled
	}
	var result *OrderResponse

	err := s.executor.Execute(ctx, func(ctx context.Context) error {
		accountID, err := uuid.Parse(req.AccountID)
		if err != nil {
			return err
		}

		account, err := s.getAccountForTrade(ctx, userID, accountID)
		if err != nil {
			return err
		}
		positions, _ := s.GetPositions(ctx, userID, accountID)
		openCount := len(positions)
		pos := findPositionByTicket(positions, req.Ticket)
		symbol := ""
		if pos != nil {
			symbol = pos.Symbol
		}
		resolver := newRiskRuleResolver()
		riskCtx, err := resolver.resolve(account, symbol, openCount)
		if err != nil {
			recordRiskValidateMetric("error", riskCodeFromError(err), account.MTType, TradeTriggerSourceFromContext(ctx), 0)
			return err
		}
		riskStart := time.Now()
		decision := s.risk().CheckManualOrderClose(riskCtx, req, pos)
		if decision != nil && !decision.Allowed {
			err := RiskErrorFromDecision(decision)
			recordRiskValidateMetric("reject", riskCodeFromError(err), account.MTType, TradeTriggerSourceFromContext(ctx), time.Since(riskStart))
			recordOrderCloseMetric("rejected", riskCodeFromError(err))
			return err
		}
		recordRiskValidateMetric("pass", "OK", account.MTType, TradeTriggerSourceFromContext(ctx), time.Since(riskStart))

		if account.MTType == "MT4" {
			result, err = s.orderCloseMT4(ctx, accountID, req)
		} else {
			result, err = s.orderCloseMT5(ctx, accountID, req)
		}

		if err != nil {
			recordOrderCloseMetric("failed", err.Error())
			return err
		}
		recordOrderCloseMetric("success", "OK")

		s.logTrade(ctx, userID, accountID, "ORDER_CLOSE", result.Symbol, "", result.Volume, result.Price, req.Ticket, result.Profit, req.CloseReason)

		// Best-effort: sync recent history so analytics/history list updates without manual Sync.
		// Run async to avoid impacting the close latency.
		if s.tradeRecordRepo != nil {
			go func() {
				syncCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
				defer cancel()
				from := time.Now().Add(-30 * 24 * time.Hour)
				to := time.Now().Add(10 * time.Minute)
				if _, syncErr := s.SyncOrderHistory(syncCtx, uuid.Nil, accountID, from, to); syncErr != nil {
					logger.Warn("sync order history after close failed",
						zap.String("account_id", accountID.String()),
						zap.Error(syncErr))
				}
			}()
		}

		return nil
	}, faultisol.WithOperation("order_close"), faultisol.WithTimeout(30*time.Second))

	return result, err
}

func (s *TradingService) getOpenPositionsCount(ctx context.Context, userID uuid.UUID, accountID uuid.UUID) (int, error) {
	positions, err := s.GetPositions(ctx, userID, accountID)
	if err != nil {
		return 0, err
	}
	return len(positions), nil
}

func findPositionByTicket(positions []*PositionResponse, ticket int64) *PositionResponse {
	for _, p := range positions {
		if p != nil && p.Ticket == ticket {
			return p
		}
	}
	return nil
}

func (s *TradingService) risk() *RiskEngine {
	if s != nil && s.riskEngine != nil {
		return s.riskEngine
	}
	return NewRiskEngine()
}

func riskCodeFromError(err error) string {
	if re, ok := AsRiskError(err); ok && re != nil {
		return string(re.Code)
	}
	if err == nil {
		return "OK"
	}
	return err.Error()
}

func (s *TradingService) orderCloseMT4(ctx context.Context, accountID uuid.UUID, req *OrderCloseRequest) (*OrderResponse, error) {
	conn, err := s.getMT4Connection(accountID)
	if err != nil {
		return nil, ErrAccountNotConnected
	}

	order, err := conn.OrderClose(ctx, int32(req.Ticket), req.Volume, 0, 100)
	if err != nil {
		return nil, err
	}

	closePx := order.GetClosePrice()
	if closePx <= 0 {
		closePx = order.GetOpenPrice()
	}
	closeTime := ""
	if ct := order.GetCloseTime(); ct != nil {
		closeTime = ct.AsTime().Format(time.RFC3339)
	}
	return &OrderResponse{
		Ticket:     int64(order.GetTicket()),
		Symbol:     order.GetSymbol(),
		Type:       OrderTypeToString(int32(order.GetType())),
		Volume:     order.GetLots(),
		Price:      closePx,
		StopLoss:   order.GetStopLoss(),
		TakeProfit: order.GetTakeProfit(),
		Comment:    order.GetComment(),
		Magic:      int64(order.GetMagicNumber()),
		OpenTime:   order.GetOpenTime().AsTime().Format("2006-01-02T15:04:05Z"),
		CloseTime:  closeTime,
		Profit:     order.GetProfit(),
		Swap:       order.GetSwap(),
		Commission: order.GetCommission(),
	}, nil
}

func (s *TradingService) orderCloseMT5(ctx context.Context, accountID uuid.UUID, req *OrderCloseRequest) (*OrderResponse, error) {
	conn, err := s.getMT5Connection(accountID)
	if err != nil {
		return nil, ErrAccountNotConnected
	}

	order, err := conn.OrderClose(ctx, req.Ticket, req.Volume, 0, 100)
	if err != nil {
		return nil, err
	}

	closePx := order.GetClosePrice()
	if closePx <= 0 {
		closePx = order.GetOpenPrice()
	}
	closeTime := ""
	if ct := order.GetCloseTime(); ct != nil {
		closeTime = ct.AsTime().Format(time.RFC3339)
	}
	return &OrderResponse{
		Ticket:     order.GetTicket(),
		Symbol:     order.GetSymbol(),
		Type:       OrderTypeToString(int32(order.GetOrderType())),
		Volume:     order.GetLots(),
		Price:      closePx,
		StopLoss:   order.GetStopLoss(),
		TakeProfit: order.GetTakeProfit(),
		Comment:    order.GetComment(),
		Magic:      order.GetExpertId(),
		OpenTime:   order.GetOpenTime().AsTime().Format("2006-01-02T15:04:05Z"),
		CloseTime:  closeTime,
		Profit:     order.GetProfit(),
		Swap:       order.GetSwap(),
		Commission: order.GetCommission(),
	}, nil
}

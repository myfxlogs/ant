package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"

	"anttrader/pkg/faultisol"
	"anttrader/pkg/logger"

	"go.uber.org/zap"
)

func (s *TradingService) GetPositions(ctx context.Context, userID, accountID uuid.UUID) ([]*PositionResponse, error) {
	var result []*PositionResponse

	// Do not inherit short Connect/client deadlines; isolation executor timeout is the budget.
	execCtx := context.WithoutCancel(ctx)
	err := s.executor.Execute(execCtx, func(ctx context.Context) error {
		account, err := s.getAccountAndVerify(ctx, userID, accountID)
		if err != nil {
			return err
		}

		if account.MTType == "MT4" {
			result, err = s.getPositionsMT4(ctx, accountID)
		} else {
			result, err = s.getPositionsMT5(ctx, accountID)
		}

		return err
	}, faultisol.WithOperation("get_positions"), faultisol.WithTimeout(120*time.Second))

	return result, err
}

func (s *TradingService) getPositionsMT4(ctx context.Context, accountID uuid.UUID) ([]*PositionResponse, error) {
	conn, err := s.getMT4Connection(accountID)
	if err != nil {
		if s.connManager == nil {
			logger.Warn("MT4 get positions: connection manager unavailable",
				zap.String("account_id", accountID.String()))
			return nil, ErrAccountNotConnected
		}

		account, accErr := s.accountRepo.GetByID(ctx, accountID)
		if accErr != nil {
			return nil, ErrAccountNotConnected
		}

		logger.Info("MT4 get positions: attempting reconnect",
			zap.String("account_id", accountID.String()))
		if connectErr := s.connManager.Connect(ctx, account); connectErr != nil {
			logger.Error("MT4 reconnect failed for positions",
				zap.String("account_id", accountID.String()),
				zap.Error(connectErr))
			return nil, ErrAccountNotConnected
		}

		conn, err = s.getMT4Connection(accountID)
		if err != nil {
			logger.Warn("MT4 get positions: connection still unavailable after reconnect",
				zap.String("account_id", accountID.String()))
			return nil, ErrAccountNotConnected
		}
	}

	orders, err := conn.OpenedOrders(ctx)
	if err != nil {
		return nil, err
	}

	var positions []*PositionResponse
	for _, order := range orders {
		currentPrice := order.GetClosePrice()
		if currentPrice == 0 {
			currentPrice = order.GetOpenPrice()
		}

		positions = append(positions, &PositionResponse{
			Ticket:       int64(order.GetTicket()),
			Symbol:       order.GetSymbol(),
			Type:         OrderTypeToString(int32(order.GetType())),
			Volume:       order.GetLots(),
			OpenPrice:    order.GetOpenPrice(),
			CurrentPrice: currentPrice,
			StopLoss:     order.GetStopLoss(),
			TakeProfit:   order.GetTakeProfit(),
			Profit:       order.GetProfit(),
			Swap:         order.GetSwap(),
			Commission:   order.GetCommission(),
			OpenTime:     order.GetOpenTime().AsTime().Format("2006-01-02T15:04:05Z"),
			Comment:      order.GetComment(),
			Magic:        int64(order.GetMagicNumber()),
		})
	}

	return positions, nil
}

func (s *TradingService) getPositionsMT5(ctx context.Context, accountID uuid.UUID) ([]*PositionResponse, error) {
	startAll := time.Now()
	logger.Info("MT5 get positions phase",
		zap.String("account_id", accountID.String()),
		zap.String("phase", "start"),
		zap.Int64("elapsed_ms", 0))
	logger.Info("MT5 get positions phase",
		zap.String("account_id", accountID.String()),
		zap.String("phase", "pre_get_conn"),
		zap.Int64("elapsed_ms", time.Since(startAll).Milliseconds()))
	conn, err := s.getMT5Connection(accountID)
	logger.Info("MT5 get positions phase",
		zap.String("account_id", accountID.String()),
		zap.String("phase", "post_get_conn"),
		zap.Bool("has_err", err != nil),
		zap.Int64("elapsed_ms", time.Since(startAll).Milliseconds()))
	if err != nil {
		connLookupCost := time.Since(startAll)
		if s.connManager == nil {
			logger.Warn("MT5 get positions: connection manager unavailable",
				zap.String("account_id", accountID.String()))
			return nil, ErrAccountNotConnected
		}

		account, accErr := s.accountRepo.GetByID(ctx, accountID)
		if accErr != nil {
			return nil, ErrAccountNotConnected
		}

		// Keep position query responsive: reconnect with a short bounded timeout.
		logger.Info("MT5 get positions: attempting reconnect",
			zap.String("account_id", accountID.String()),
			zap.Int64("lookup_elapsed_ms", connLookupCost.Milliseconds()))
		reconnectCtx, cancelReconnect := context.WithTimeout(context.Background(), 12*time.Second)
		reconnectStart := time.Now()
		connectErr := s.connManager.Connect(reconnectCtx, account)
		reconnectCost := time.Since(reconnectStart)
		cancelReconnect()
		if connectErr != nil {
			logger.Error("MT5 reconnect failed for positions",
				zap.String("account_id", accountID.String()),
				zap.Int64("reconnect_elapsed_ms", reconnectCost.Milliseconds()),
				zap.Error(connectErr))
			return nil, fmt.Errorf("mt5_reconnect_failed: %w", ErrAccountNotConnected)
		}

		lookupAfterReconnectStart := time.Now()
		conn, err = s.getMT5Connection(accountID)
		lookupAfterReconnectCost := time.Since(lookupAfterReconnectStart)
		if err != nil {
			logger.Warn("MT5 get positions: connection still unavailable after reconnect",
				zap.String("account_id", accountID.String()),
				zap.Int64("lookup_after_reconnect_ms", lookupAfterReconnectCost.Milliseconds()))
			return nil, fmt.Errorf("mt5_connection_unavailable_after_reconnect: %w", ErrAccountNotConnected)
		}
	}
	logger.Info("MT5 get positions phase",
		zap.String("account_id", accountID.String()),
		zap.String("phase", "connection_ready"),
		zap.Int64("elapsed_ms", time.Since(startAll).Milliseconds()))

	// Independent of request ctx: OpenedOrders must not be clipped by frontend RPC timeouts.
	openCtx, cancelOpen := context.WithTimeout(context.Background(), 30*time.Second)
	openedOrdersStart := time.Now()
	logger.Info("MT5 get positions phase",
		zap.String("account_id", accountID.String()),
		zap.String("phase", "opened_orders_primary"))
	orders, err := conn.OpenedOrders(openCtx)
	openedOrdersCost := time.Since(openedOrdersStart)
	cancelOpen()
	if err != nil {
		logger.Error("获取 MT5 持仓订单失败",
			zap.String("account_id", accountID.String()),
			zap.Int64("opened_orders_elapsed_ms", openedOrdersCost.Milliseconds()),
			zap.Error(err))
		if s.connManager != nil && (errors.Is(err, context.DeadlineExceeded) || containsTimeoutText(err.Error())) {
			logger.Info("MT5 get positions phase",
				zap.String("account_id", accountID.String()),
				zap.String("phase", "opened_orders_retry_prepare"),
				zap.Int64("elapsed_ms", time.Since(startAll).Milliseconds()))
			// One synchronous reconnect+retry before giving up. This avoids repeated 30s user-facing failures
			// when the connection object is stale but stream path is still alive.
			acc, accErr := s.accountRepo.GetByID(context.Background(), accountID)
			if accErr == nil && acc != nil {
				reconnectCtx, cancel := context.WithTimeout(context.Background(), 12*time.Second)
				if reconnectErr := s.connManager.Connect(reconnectCtx, acc); reconnectErr != nil {
					logger.Warn("MT5 positions reconnect retry failed",
						zap.String("account_id", accountID.String()),
						zap.Error(reconnectErr))
				}
				cancel()
				if freshConn, getErr := s.getMT5Connection(accountID); getErr == nil && freshConn != nil {
					logger.Info("MT5 get positions phase",
						zap.String("account_id", accountID.String()),
						zap.String("phase", "opened_orders_secondary"))
					retryCtx, cancelRetry := context.WithTimeout(context.Background(), 30*time.Second)
					retryStart := time.Now()
					retryOrders, retryErr := freshConn.OpenedOrders(retryCtx)
					retryCost := time.Since(retryStart)
					cancelRetry()
					if retryErr == nil {
						orders = retryOrders
						err = nil
						logger.Info("MT5 positions retry succeeded",
							zap.String("account_id", accountID.String()),
							zap.Int64("retry_elapsed_ms", retryCost.Milliseconds()),
							zap.Int("orders", len(orders)))
					} else {
						logger.Warn("MT5 positions retry still failed",
							zap.String("account_id", accountID.String()),
							zap.Int64("retry_elapsed_ms", retryCost.Milliseconds()),
							zap.Error(retryErr))
					}
				}
			}
		}
		if err != nil && s.connManager != nil && (errors.Is(err, context.DeadlineExceeded) || containsTimeoutText(err.Error())) {
			// Best-effort recovery path:
			// 1) force one async reconnect attempt
			// 2) trigger history backfill for MT5 trade records
			go func() {
				acc, accErr := s.accountRepo.GetByID(context.Background(), accountID)
				if accErr == nil && acc != nil {
					reconnectCtx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
					_ = s.connManager.Connect(reconnectCtx, acc)
					cancel()
				}
			}()
			s.connManager.SyncOrderHistory(accountID, "MT5")
		}
		if err != nil {
			return nil, fmt.Errorf("mt5_opened_orders_failed: %w", err)
		}
	}

	var positions []*PositionResponse
	for _, order := range orders {
		currentPrice := order.GetClosePrice()
		if currentPrice == 0 {
			currentPrice = order.GetOpenPrice()
		}

		positions = append(positions, &PositionResponse{
			Ticket:       order.GetTicket(),
			Symbol:       order.GetSymbol(),
			Type:         OrderTypeToString(int32(order.GetOrderType())),
			Volume:       order.GetLots(),
			OpenPrice:    order.GetOpenPrice(),
			CurrentPrice: currentPrice,
			StopLoss:     order.GetStopLoss(),
			TakeProfit:   order.GetTakeProfit(),
			Profit:       order.GetProfit(),
			Swap:         order.GetSwap(),
			Commission:   order.GetCommission(),
			OpenTime:     order.GetOpenTime().AsTime().Format("2006-01-02T15:04:05Z"),
			Comment:      order.GetComment(),
			Magic:        order.GetExpertId(),
		})
	}

	logger.Info("MT5 get positions succeeded",
		zap.String("account_id", accountID.String()),
		zap.Int("positions", len(positions)),
		zap.Int64("total_elapsed_ms", time.Since(startAll).Milliseconds()))
	return positions, nil
}

func containsTimeoutText(msg string) bool {
	if msg == "" {
		return false
	}
	b := []byte(msg)
	for i := range b {
		if b[i] >= 'A' && b[i] <= 'Z' {
			b[i] += 32
		}
	}
	s := string(b)
	return stringsContainsLocal(s, "timed out") || stringsContainsLocal(s, "timeout")
}

func stringsContainsLocal(s string, sub string) bool {
	if len(sub) == 0 {
		return true
	}
	if len(sub) > len(s) {
		return false
	}
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}

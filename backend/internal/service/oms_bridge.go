package service

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"anttrader/internal/broker"
	"anttrader/internal/model"
	"anttrader/internal/oms"
)

// ── OMS Bridge (M2-4) ────────────────────────────────────────────────
// TradingService.OrderSend now delegates to the OMS BrokerAdapter when available.
// Legacy direct conn.OrderSend calls remain as fallback (orderSendMT4/orderSendMT5).
// After M2-6 full regression, the fallback paths will be removed.

// getBrokerAdapter returns an oms.BrokerAdapter for the given account.
// Uses the existing ConnectionManager to obtain an MT4/MT5 connection,
// then wraps it in the broker adapter from internal/broker/.
func (s *TradingService) getBrokerAdapter(ctx context.Context, accountID uuid.UUID, account *model.MTAccount) (oms.BrokerAdapter, error) {
	switch account.MTType {
	case "MT4":
		conn, err := s.getMT4Connection(accountID)
		if err != nil {
			if connectErr := s.connManager.Connect(ctx, account); connectErr == nil {
				conn, err = s.getMT4Connection(accountID)
			}
		}
		if err != nil {
			return nil, fmt.Errorf("mt4 adapter: %w", err)
		}
		if conn == nil {
			return nil, fmt.Errorf("mt4 adapter: connection is nil")
		}
		return broker.NewMT4Adapter(conn), nil

	case "MT5":
		conn, err := s.getMT5Connection(accountID)
		if err != nil {
			if connectErr := s.connManager.Connect(ctx, account); connectErr == nil {
				conn, err = s.getMT5Connection(accountID)
			}
		}
		if err != nil {
			return nil, fmt.Errorf("mt5 adapter: %w", err)
		}
		if conn == nil {
			return nil, fmt.Errorf("mt5 adapter: connection is nil")
		}
		return broker.NewMT5Adapter(conn), nil

	default:
		return nil, fmt.Errorf("unsupported platform: %s", account.MTType)
	}
}

// orderSendViaOMS sends an order through the OMS BrokerAdapter interface.
// This is the new canonical path (M2-4). Falls back to legacy orderSendMT4/MT5
// if the adapter is not available.
func (s *TradingService) orderSendViaOMS(ctx context.Context, accountID uuid.UUID, account *model.MTAccount, accountIDStr string, req *OrderSendRequest) (*OrderResponse, error) {
	adapter, err := s.getBrokerAdapter(ctx, accountID, account)
	if err != nil {
		// Fallback to legacy direct path
		if account.MTType == "MT4" {
			return s.orderSendMT4(ctx, accountID, account, req)
		}
		return s.orderSendMT5(ctx, accountID, account, req)
	}

	omsReq := &oms.OrderRequest{
		AccountID:  accountIDStr,
		Symbol:     req.Symbol,
		Side:       req.Type,
		Volume:     req.Volume,
		Price:      req.Price,
		StopLoss:   req.StopLoss,
		TakeProfit: req.TakeProfit,
		StrategyID: req.Comment,
		Comment:    req.Comment,
	}

	resp, err := adapter.Submit(ctx, omsReq)
	if err != nil {
		return nil, fmt.Errorf("oms submit: %w", err)
	}

	return &OrderResponse{
		Ticket: parseTicket(resp.Ticket),
		Symbol: req.Symbol,
		Type:   req.Type,
		Volume: resp.FilledQty,
		Price:  resp.FillPrice,
	}, nil
}

// parseTicket converts a string ticket to int64 (best-effort).
func parseTicket(t string) int64 {
	var v int64
	fmt.Sscanf(t, "%d", &v)
	return v
}

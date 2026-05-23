package broker

import (
	"context"
	"fmt"

	pb "anttrader/mt5"
	"anttrader/internal/mt5client"
	"anttrader/internal/oms"
)

// MT5Adapter wraps the existing mt5client connection with the oms.BrokerAdapter interface.
type MT5Adapter struct {
	baseAdapter
	conn *mt5client.MT5Connection
}

// NewMT5Adapter creates an MT5 broker adapter.
func NewMT5Adapter(conn *mt5client.MT5Connection) *MT5Adapter {
	return &MT5Adapter{
		baseAdapter: baseAdapter{platform: "mt5"},
		conn:        conn,
	}
}

// Submit sends an order via the MT5 gateway.
func (a *MT5Adapter) Submit(ctx context.Context, req *oms.OrderRequest) (*oms.BrokerResp, error) {
	if a.conn == nil {
		return nil, fmt.Errorf("mt5 adapter: not connected")
	}
	symbol := req.BrokerSymbolRaw
	if symbol == "" {
		symbol = req.Symbol
	}
	op := pb.OrderType_OrderType_Buy
	if req.Side == "sell" {
		op = pb.OrderType_OrderType_Sell
	}
	resp, err := a.conn.OrderSend(ctx, symbol, op, req.Volume, req.Price, req.StopLoss, req.TakeProfit, 10, req.StrategyID, 0)
	if err != nil {
		return nil, fmt.Errorf("mt5 OrderSend: %w", err)
	}
	return &oms.BrokerResp{
		Ticket:    fmt.Sprintf("%d", resp.Ticket),
		State:     oms.StateSubmitted,
		FilledQty: float64(resp.Volume),
		FillPrice: resp.OpenPrice,
	}, nil
}

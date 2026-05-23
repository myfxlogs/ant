package broker

import (
	"context"
	"fmt"

	pb "anttrader/mt4"
	"anttrader/internal/mt4client"
	"anttrader/internal/oms"
)

// MT4Adapter wraps the existing mt4client connection with the oms.BrokerAdapter interface.
type MT4Adapter struct {
	baseAdapter
	conn *mt4client.MT4Connection
}

// NewMT4Adapter creates an MT4 broker adapter.
func NewMT4Adapter(conn *mt4client.MT4Connection) *MT4Adapter {
	return &MT4Adapter{
		baseAdapter: baseAdapter{platform: "mt4"},
		conn:        conn,
	}
}

// Submit sends an order via the MT4 gateway.
func (a *MT4Adapter) Submit(ctx context.Context, req *oms.OrderRequest) (*oms.BrokerResp, error) {
	if a.conn == nil {
		return nil, fmt.Errorf("mt4 adapter: not connected")
	}
	symbol := req.BrokerSymbolRaw
	if symbol == "" {
		symbol = req.Symbol
	}
	op := pb.Op_Op_Buy
	if req.Side == "sell" {
		op = pb.Op_Op_Sell
	}
	resp, err := a.conn.OrderSend(ctx, symbol, op, req.Volume, req.Price, req.StopLoss, req.TakeProfit, 10, req.StrategyID, 0)
	if err != nil {
		return nil, fmt.Errorf("mt4 OrderSend: %w", err)
	}
	return &oms.BrokerResp{
		Ticket:    fmt.Sprintf("%d", resp.Ticket),
		State:     oms.StateSubmitted,
		FilledQty: resp.Lots,
		FillPrice: resp.OpenPrice,
	}, nil
}

// Package mthub — OrderExecutor implementation bridging broker adapters
// to the mthub service. Wraps the existing oms.BrokerAdapter interface.
package mthub

import (
	"context"
	"fmt"

	"anttrader/internal/oms"
)

// BrokerOrderExecutor implements OrderExecutor by delegating to
// an oms.BrokerAdapter (e.g. broker.MT4Adapter, broker.MT5Adapter).
type BrokerOrderExecutor struct {
	adapter oms.BrokerAdapter
}

// NewBrokerOrderExecutor creates an OrderExecutor backed by a broker adapter.
func NewBrokerOrderExecutor(adapter oms.BrokerAdapter) *BrokerOrderExecutor {
	return &BrokerOrderExecutor{adapter: adapter}
}

func (e *BrokerOrderExecutor) PlaceOrder(ctx context.Context, conn interface{}, platform, sessionID string, req *OrderRequest) (int64, error) {
	if e.adapter == nil {
		return 0, fmt.Errorf("broker executor: no adapter configured")
	}

	resp, err := e.adapter.Submit(ctx, &oms.OrderRequest{
		Symbol:  req.Symbol,
		Side:    req.Side,
		Volume:  req.Volume,
		Price:   req.Price,
		Comment: req.Comment,
	})
	if err != nil {
		return 0, err
	}
	return parseTicket(resp.Ticket), nil
}

func (e *BrokerOrderExecutor) CloseOrder(ctx context.Context, conn interface{}, platform, sessionID string, ticket int64, lots float64) error {
	if e.adapter == nil {
		return fmt.Errorf("broker executor: no adapter configured")
	}
	return e.adapter.Cancel(ctx, fmt.Sprintf("%d", ticket))
}

func (e *BrokerOrderExecutor) FetchOrderHistory(ctx context.Context, conn interface{}, platform, sessionID, from, to string) ([]*HistoryOrderInfo, error) {
	return nil, fmt.Errorf("broker executor: fetch history not implemented")
}

func (e *BrokerOrderExecutor) FetchOpenedOrders(ctx context.Context, conn interface{}, platform, sessionID string) ([]*HistoryOrderInfo, error) {
	return nil, fmt.Errorf("broker executor: fetch opened orders not implemented")
}

func (e *BrokerOrderExecutor) FetchSymbolParamsMany(ctx context.Context, conn interface{}, platform, sessionID string, symbols []string) ([]*SymbolParam, error) {
	return nil, fmt.Errorf("broker executor: fetch symbol params not implemented")
}

func (e *BrokerOrderExecutor) FetchPriceHistoryToday(ctx context.Context, conn interface{}, platform, sessionID, symbol string) ([]*PriceBar, error) {
	return nil, fmt.Errorf("broker executor: fetch price history not implemented")
}

func parseTicket(s string) int64 {
	var n int64
	for _, c := range s {
		if c >= '0' && c <= '9' {
			n = n*10 + int64(c-'0')
		}
	}
	return n
}

package runner

import (
	"context"

	"github.com/shopspring/decimal"

	"anttrader/strategy/sdk"
)

// brokerImpl implements sdk.Broker by delegating to OrderExecutor.
type brokerImpl struct {
	runner   *Runner
	executor OrderExecutor // set by LiveRunner
}

func (b *brokerImpl) OrderSend(req sdk.OrderRequest) (sdk.OrderResult, error) {
	if b.executor == nil {
		return sdk.OrderResult{RetCode: sdk.RetRejected}, nil
	}
	ticket, err := b.executor.PlaceOrder(
		context.Background(),
		req.Symbol, req.Side, req.Type,
		req.Volume, req.Price, req.StopLoss, req.TakeProfit,
		req.Comment, req.Magic,
	)
	if err != nil {
		return sdk.OrderResult{RetCode: sdk.RetRejected}, err
	}
	return sdk.OrderResult{
		RetCode: sdk.RetDone,
		Ticket:  ticket,
		Volume:  req.Volume,
		Price:   req.Price,
	}, nil
}

func (b *brokerImpl) PositionClose(ticket int64, volume decimal.Decimal) (sdk.OrderResult, error) {
	if b.executor == nil {
		return sdk.OrderResult{RetCode: sdk.RetRejected}, nil
	}
	if err := b.executor.CloseOrder(context.Background(), ticket, volume); err != nil {
		return sdk.OrderResult{RetCode: sdk.RetRejected}, err
	}
	return sdk.OrderResult{RetCode: sdk.RetDone, Ticket: ticket}, nil
}

func (b *brokerImpl) PositionModify(ticket int64, sl, tp decimal.Decimal) (sdk.OrderResult, error) {
	if b.executor == nil {
		return sdk.OrderResult{RetCode: sdk.RetRejected}, nil
	}
	if err := b.executor.ModifyOrder(context.Background(), ticket, sl, tp); err != nil {
		return sdk.OrderResult{RetCode: sdk.RetRejected}, err
	}
	return sdk.OrderResult{RetCode: sdk.RetDone, Ticket: ticket}, nil
}

func (b *brokerImpl) OrderDelete(ticket int64) (sdk.OrderResult, error) {
	if b.executor == nil {
		return sdk.OrderResult{RetCode: sdk.RetRejected}, nil
	}
	if err := b.executor.CancelOrder(context.Background(), ticket); err != nil {
		return sdk.OrderResult{RetCode: sdk.RetRejected}, err
	}
	return sdk.OrderResult{RetCode: sdk.RetDone, Ticket: ticket}, nil
}

func (b *brokerImpl) Positions(magic int32) []sdk.Position {
	if b.executor == nil {
		return nil
	}
	positions, _ := b.executor.OpenedOrders(context.Background())
	if magic == 0 {
		return positions
	}
	var filtered []sdk.Position
	for _, p := range positions {
		if p.Magic == magic {
			filtered = append(filtered, p)
		}
	}
	return filtered
}

func (b *brokerImpl) Orders(magic int32) []sdk.PendingOrder {
	if b.executor == nil {
		return nil
	}
	orders, _ := b.executor.PendingOrders(context.Background())
	if magic == 0 {
		return orders
	}
	var filtered []sdk.PendingOrder
	for _, o := range orders {
		if o.Magic == magic {
			filtered = append(filtered, o)
		}
	}
	return filtered
}

func (b *brokerImpl) HistoryOrders(from, to int64) []sdk.Position {
	if b.executor == nil {
		return nil
	}
	return nil
}

func (b *brokerImpl) Deals(from, to int64, magic int32) []sdk.Deal {
	if b.executor == nil {
		return nil
	}
	return nil
}

func (b *brokerImpl) SymbolInfo(symbol string) (sdk.SymbolInfo, error) {
	if b.executor == nil {
		return sdk.SymbolInfo{}, nil
	}
	return b.executor.SymbolInfo(symbol)
}

func (b *brokerImpl) Account() sdk.AccountInfo {
	if b.executor == nil {
		return sdk.AccountInfo{}
	}
	return b.executor.Account()
}

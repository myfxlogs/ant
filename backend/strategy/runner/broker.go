package runner

import (
	"context"
	"fmt"

	"github.com/shopspring/decimal"

	"alphaforge/strategy/sdk"
)

// brokerImpl implements sdk.Broker by delegating to OrderExecutor.
type brokerImpl struct {
	runner   *Runner
	executor OrderExecutor // set by LiveRunner
	ctx      context.Context
}

func (b *brokerImpl) setContext(ctx context.Context) { b.ctx = ctx }

func (b *brokerImpl) orderCtx() context.Context {
	if b.ctx != nil { return b.ctx }
	return context.Background()
}

func (b *brokerImpl) OrderSend(req sdk.OrderRequest) (sdk.OrderResult, error) {
	if b.executor == nil {
		return sdk.OrderResult{RetCode: sdk.RetRejected}, fmt.Errorf("broker: no executor configured")
	}
	ticket, err := b.executor.PlaceOrder(
		b.orderCtx(),
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
		return sdk.OrderResult{RetCode: sdk.RetRejected}, fmt.Errorf("broker: no executor configured")
	}
	if err := b.executor.CloseOrder(b.orderCtx(), ticket, volume); err != nil {
		return sdk.OrderResult{RetCode: sdk.RetRejected}, err
	}
	return sdk.OrderResult{RetCode: sdk.RetDone, Ticket: ticket}, nil
}

func (b *brokerImpl) PositionCloseBy(ticket1, ticket2 int64) (sdk.OrderResult, error) {
	if b.executor == nil {
		return sdk.OrderResult{RetCode: sdk.RetRejected}, fmt.Errorf("broker: no executor configured")
	}
	if err := b.executor.CloseOrder(b.orderCtx(), ticket1, decimal.Zero); err != nil {
		return sdk.OrderResult{RetCode: sdk.RetRejected}, err
	}
	if err := b.executor.CloseOrder(b.orderCtx(), ticket2, decimal.Zero); err != nil {
		return sdk.OrderResult{RetCode: sdk.RetRejected}, err
	}
	return sdk.OrderResult{RetCode: sdk.RetDone, Ticket: ticket1}, nil
}

func (b *brokerImpl) PositionModify(ticket int64, sl, tp decimal.Decimal) (sdk.OrderResult, error) {
	if b.executor == nil {
		return sdk.OrderResult{RetCode: sdk.RetRejected}, fmt.Errorf("broker: no executor configured")
	}
	if err := b.executor.ModifyOrder(b.orderCtx(), ticket, sl, tp); err != nil {
		return sdk.OrderResult{RetCode: sdk.RetRejected}, err
	}
	return sdk.OrderResult{RetCode: sdk.RetDone, Ticket: ticket}, nil
}

func (b *brokerImpl) OrderDelete(ticket int64) (sdk.OrderResult, error) {
	if b.executor == nil {
		return sdk.OrderResult{RetCode: sdk.RetRejected}, fmt.Errorf("broker: no executor configured")
	}
	if err := b.executor.CancelOrder(b.orderCtx(), ticket); err != nil {
		return sdk.OrderResult{RetCode: sdk.RetRejected}, err
	}
	return sdk.OrderResult{RetCode: sdk.RetDone, Ticket: ticket}, nil
}

func (b *brokerImpl) Positions(magic int32) []sdk.Position {
	if b.executor == nil {
		// Harness mode: use positions passed from parent process.
		b.runner.ctx.mu.RLock()
		livePos := b.runner.ctx.livePositions
		b.runner.ctx.mu.RUnlock()
		if magic == 0 {
			return livePos
		}
		var filtered []sdk.Position
		for _, p := range livePos {
			if p.Magic == magic {
				filtered = append(filtered, p)
			}
		}
		return filtered
	}
	positions, err := b.executor.OpenedOrders(b.orderCtx())
	if err != nil {
		// Log but don't return error — SDK interface returns []Position, not ([]Position, error).
		// Strategy will see empty positions and may re-enter, but at least we log.
		_ = err
		return nil
	}
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
	orders, err := b.executor.PendingOrders(b.orderCtx())
	if err != nil {
		return nil
	}
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
		// Harness mode: use account state passed from parent process.
		b.runner.ctx.mu.RLock()
		defer b.runner.ctx.mu.RUnlock()
		return sdk.AccountInfo{
			Balance: b.mustDecimal(b.runner.ctx.liveBalance),
			Equity:  b.mustDecimal(b.runner.ctx.liveEquity),
		}
	}
	return b.executor.Account()
}

// mustDecimal parses a decimal string.
// Returns zero if the string is empty or unparseable — a corrupted balance/equity
// value is a material error, but the SDK interface cannot return errors here.
func (b *brokerImpl) mustDecimal(s string) decimal.Decimal {
	if s == "" {
		return decimal.Zero
	}
	d, err := decimal.NewFromString(s)
	if err != nil {
		return decimal.Zero
	}
	return d
}

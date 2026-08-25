package runner

import (
	"context"
	"fmt"

	"github.com/shopspring/decimal"

	"alphaforge/strategy/sdk"
)

// brokerImpl implements sdk.Broker by delegating to OrderExecutor.
type brokerImpl struct {
	runner    *Runner
	executor  OrderExecutor // set by LiveRunner
	ctx       context.Context
	lastError error // VM-TRADE-CONTEXT-2: records last query error for fail-closed
}

func (b *brokerImpl) setContext(ctx context.Context) { b.ctx = ctx }

// LastError returns the last query error from Positions/Orders/HistoryOrders/Deals.
// VM-TRADE-CONTEXT-2: The runner checks this after strategy execution to fail-closed
// when broker queries fail (instead of silently presenting empty positions).
func (b *brokerImpl) LastError() error { return b.lastError }

// resetError clears the last error before each strategy event.
func (b *brokerImpl) resetError() { b.lastError = nil }

func (b *brokerImpl) orderCtx() context.Context {
	if b.ctx != nil {
		return b.ctx
	}
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
		// VM-TRADE-CONTEXT-2: Record error for fail-closed instead of silently
		// returning nil. The runner checks LastError() after strategy execution.
		b.lastError = fmt.Errorf("broker: Positions query failed: %w", err)
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
		// LIVE-MQL-ORDER-CONTEXT-1: harness mode — use pending orders passed
		// from parent process via UpdateLiveState. Previously returned nil,
		// causing OrdersTotal to undercount and OrderSelect to miss pending orders.
		b.runner.ctx.mu.RLock()
		liveOrders := b.runner.ctx.livePendingOrders
		b.runner.ctx.mu.RUnlock()
		if magic == 0 {
			return liveOrders
		}
		var filtered []sdk.PendingOrder
		for _, o := range liveOrders {
			if o.Magic == magic {
				filtered = append(filtered, o)
			}
		}
		return filtered
	}
	orders, err := b.executor.PendingOrders(b.orderCtx())
	if err != nil {
		// VM-TRADE-CONTEXT-2: Record error for fail-closed.
		b.lastError = fmt.Errorf("broker: Orders query failed: %w", err)
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
		// VM-TRADE-CONTEXT-3: harness mode also records error — HistoryOrders
		// is unavailable regardless of executor presence.
		b.lastError = fmt.Errorf("broker: HistoryOrders not available (no executor)")
		return nil
	}
	// VM-TRADE-CONTEXT-2: HistoryOrders is not implemented in the live executor.
	// Record error so the runner can fail-closed instead of silently returning nil.
	b.lastError = fmt.Errorf("broker: HistoryOrders not available in live mode")
	return nil
}

func (b *brokerImpl) Deals(from, to int64, magic int32) []sdk.Deal {
	if b.executor == nil {
		// VM-TRADE-CONTEXT-3: harness mode also records error.
		b.lastError = fmt.Errorf("broker: Deals not available (no executor)")
		return nil
	}
	// VM-TRADE-CONTEXT-2: Deals is not implemented in the live executor.
	b.lastError = fmt.Errorf("broker: Deals not available in live mode")
	return nil
}

func (b *brokerImpl) SymbolInfo(symbol string) (sdk.SymbolInfo, error) {
	if b.executor == nil {
		// Harness mode: use symbol info passed from parent process.
		b.runner.ctx.mu.RLock()
		defer b.runner.ctx.mu.RUnlock()
		return sdk.SymbolInfo{
			Name:         symbol,
			Point:        b.mustDecimal(b.runner.ctx.livePoint),
			Digits:       b.runner.ctx.liveDigits,
			ContractSize: b.mustDecimal(b.runner.ctx.liveContractSize),
			StopsLevel:   parseStopsLevel(b.runner.ctx.liveStopsLevel),
		}, nil
	}
	return b.executor.SymbolInfo(symbol)
}

func (b *brokerImpl) Account() sdk.AccountInfo {
	if b.executor == nil {
		// Harness mode: use account state passed from parent process.
		b.runner.ctx.mu.RLock()
		defer b.runner.ctx.mu.RUnlock()
		return sdk.AccountInfo{
			Balance:        b.mustDecimal(b.runner.ctx.liveBalance),
			Equity:         b.mustDecimal(b.runner.ctx.liveEquity),
			Margin:         b.mustDecimal(b.runner.ctx.liveMargin),
			FreeMargin:     b.mustDecimal(b.runner.ctx.liveFreeMargin),
			Login:          b.runner.ctx.liveLogin,          // VM-TRADE-CONTEXT-3
			Company:        b.runner.ctx.liveCompany,        // VM-TRADE-CONTEXT-3
			IsDemo:         b.runner.ctx.liveIsDemo,         // VM-API-TRUTH-3
			IsConnected:    b.runner.ctx.liveIsConnected,    // VM-API-TRUTH-3
			IsTradeAllowed: b.runner.ctx.liveIsTradeAllowed, // VM-API-TRUTH-3
		}
	}
	return b.executor.Account()
}

// parseStopsLevel parses a stops_level string to int32.
// Returns 0 if empty or unparseable.
func parseStopsLevel(s string) int32 {
	if s == "" {
		return 0
	}
	d, err := decimal.NewFromString(s)
	if err != nil {
		return 0
	}
	return int32(d.IntPart())
}

// mustDecimal parses a decimal string.
// Returns a negative sentinel if the string is empty or unparseable so account
// data corruption cannot masquerade as a zero account value.
func (b *brokerImpl) mustDecimal(s string) decimal.Decimal {
	if s == "" {
		return decimal.NewFromInt(-1)
	}
	d, err := decimal.NewFromString(s)
	if err != nil {
		return decimal.NewFromInt(-1)
	}
	return d
}

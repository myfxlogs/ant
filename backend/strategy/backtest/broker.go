package backtest

import (
	"fmt"
	"time"

	"github.com/shopspring/decimal"

	"anttrader/strategy/sdk"
)

// SimBroker implements sdk.Broker for backtesting.
// It simulates order fills at the next bar's open price.
type SimBroker struct {
	config      Config
	positions   []*OrderRecord
	pending     []*OrderRecord
	history     []*OrderRecord
	trades      []Trade
	ticketSeq   int64
	equity      decimal.Decimal
	balance     decimal.Decimal
	currentBar  int
}

// NewSimBroker creates a simulated broker for backtesting.
func NewSimBroker(cfg Config) *SimBroker {
	return &SimBroker{
		config:    cfg,
		equity:    cfg.InitialCapital,
		balance:   cfg.InitialCapital,
		ticketSeq: 1000,
	}
}

// SetBar updates the current bar index for position matching.
func (b *SimBroker) SetBar(index int) { b.currentBar = index }

// ── Order execution ────────────────────────────────────────────────

func (b *SimBroker) OrderSend(req sdk.OrderRequest) (sdk.OrderResult, error) {
	b.ticketSeq++
	ticket := b.ticketSeq

	rec := &OrderRecord{
		Ticket:     ticket,
		Symbol:     req.Symbol,
		Side:       req.Side,
		OrderType:  req.Type,
		Volume:     req.Volume,
		Price:      req.Price,
		StopLoss:   req.StopLoss,
		TakeProfit: req.TakeProfit,
		OpenTime:   time.Now(),
		State:      OrderOpen,
		Comment:    req.Comment,
		Magic:      req.Magic,
	}

	// Apply commission
	commission := req.Volume.Mul(b.config.Commission)
	rec.Commission = commission
	b.equity = b.equity.Sub(commission)

	// Market orders fill immediately at current price
	if req.Type == sdk.OrderMarket {
		b.positions = append(b.positions, rec)
	} else {
		// Pending orders wait for price trigger
		rec.State = OrderPending
		b.pending = append(b.pending, rec)
	}

	return sdk.OrderResult{
		RetCode: sdk.RetDone,
		Ticket:  ticket,
		Volume:  req.Volume,
		Price:   req.Price,
	}, nil
}

func (b *SimBroker) PositionClose(ticket int64, volume decimal.Decimal) (sdk.OrderResult, error) {
	for i, pos := range b.positions {
		if pos.Ticket == ticket {
			if volume.IsZero() || volume.Equal(pos.Volume) {
				// Close full position
				pos.State = OrderClosed
				pos.CloseTime = time.Now()
				b.history = append(b.history, pos)
				b.positions = append(b.positions[:i], b.positions[i+1:]...)
			} else {
				// Partial close
				pos.Volume = pos.Volume.Sub(volume)
			}
			return sdk.OrderResult{RetCode: sdk.RetDone, Ticket: ticket}, nil
		}
	}
	return sdk.OrderResult{RetCode: sdk.RetRejected}, fmt.Errorf("ticket %d not found", ticket)
}

func (b *SimBroker) PositionModify(ticket int64, sl, tp decimal.Decimal) (sdk.OrderResult, error) {
	for _, pos := range b.positions {
		if pos.Ticket == ticket {
			pos.StopLoss = sl
			pos.TakeProfit = tp
			return sdk.OrderResult{RetCode: sdk.RetDone, Ticket: ticket}, nil
		}
	}
	return sdk.OrderResult{RetCode: sdk.RetRejected}, nil
}

func (b *SimBroker) OrderDelete(ticket int64) (sdk.OrderResult, error) {
	for i, ord := range b.pending {
		if ord.Ticket == ticket {
			ord.State = OrderCancelled
			b.history = append(b.history, ord)
			b.pending = append(b.pending[:i], b.pending[i+1:]...)
			return sdk.OrderResult{RetCode: sdk.RetDone, Ticket: ticket}, nil
		}
	}
	return sdk.OrderResult{RetCode: sdk.RetRejected}, nil
}

// ── Query ──────────────────────────────────────────────────────────

func (b *SimBroker) Positions(magic int32) []sdk.Position {
	var out []sdk.Position
	for _, p := range b.positions {
		if magic == 0 || p.Magic == magic {
			out = append(out, sdk.Position{
				Ticket:     p.Ticket,
				Symbol:     p.Symbol,
				Side:       p.Side,
				Volume:     p.Volume,
				OpenPrice:  p.Price,
				StopLoss:   p.StopLoss,
				TakeProfit: p.TakeProfit,
				Profit:     p.Profit,
				Comment:    p.Comment,
				Magic:      p.Magic,
				OpenTime:   p.OpenTime,
			})
		}
	}
	return out
}

func (b *SimBroker) Orders(magic int32) []sdk.PendingOrder {
	var out []sdk.PendingOrder
	for _, p := range b.pending {
		if magic == 0 || p.Magic == magic {
			out = append(out, sdk.PendingOrder{
				Ticket:     p.Ticket,
				Symbol:     p.Symbol,
				Type:       p.OrderType,
				Side:       p.Side,
				Volume:     p.Volume,
				Price:      p.Price,
				StopLoss:   p.StopLoss,
				TakeProfit: p.TakeProfit,
				Comment:    p.Comment,
				Magic:      p.Magic,
			})
		}
	}
	return out
}

func (b *SimBroker) HistoryOrders(from, to int64) []sdk.Position { return nil }
func (b *SimBroker) Deals(from, to int64, magic int32) []sdk.Deal { return nil }
func (b *SimBroker) SymbolInfo(symbol string) (sdk.SymbolInfo, error) {
	return sdk.SymbolInfo{Name: symbol, Digits: 5, Point: decimal.NewFromFloat(0.00001)}, nil
}

func (b *SimBroker) Account() sdk.AccountInfo {
	return sdk.AccountInfo{
		Balance:    b.balance,
		Equity:     b.equity,
		Margin:     decimal.Zero,
		FreeMargin: b.equity,
		Leverage:   b.config.Leverage,
		Currency:   "USD",
	}
}

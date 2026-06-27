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
		OpenBar:    b.currentBar,
	}

	// Apply commission (basis points of volume)
	b.applyCommission(rec)

	// Apply slippage for market orders
	if req.Type == sdk.OrderMarket {
		slip := b.config.Slippage.Mul(decimal.NewFromInt(10)) // pips to points
		if req.Side == sdk.SideBuy {
			rec.Price = rec.Price.Add(slip)
		} else {
			rec.Price = rec.Price.Sub(slip)
		}
	}

	// Check margin before opening
	margin := req.Volume.Mul(rec.Price).Div(decimal.NewFromInt(int64(b.config.Leverage)))
	if b.equity.LessThan(margin) {
		return sdk.OrderResult{RetCode: sdk.RetNoMoney}, fmt.Errorf("insufficient margin")
	}

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
			closeVol := volume
			if closeVol.IsZero() || closeVol.Equal(pos.Volume) {
				closeVol = pos.Volume
			}
			// Calculate P&L for the closed volume.
			closePrice := pos.ClosePrice
			if closePrice.IsZero() {
				closePrice = pos.Price
			}
			profit := closePrice.Sub(pos.Price).Mul(closeVol)
			if pos.Side == sdk.SideSell {
				profit = profit.Neg()
			}
			b.equity = b.equity.Add(profit)
			b.balance = b.balance.Add(profit)

			if closeVol.Equal(pos.Volume) {
				// Close full position
				pos.State = OrderClosed
				pos.CloseTime = time.Now()
				pos.Profit = profit
				b.history = append(b.history, pos)
				b.positions = append(b.positions[:i], b.positions[i+1:]...)
			} else {
				// Partial close — reduce volume, keep position open
				pos.Volume = pos.Volume.Sub(closeVol)
			}
			return sdk.OrderResult{RetCode: sdk.RetDone, Ticket: ticket, Volume: closeVol, Price: closePrice}, nil
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

// applyCommission deducts commission from equity and records it.
func (b *SimBroker) applyCommission(rec *OrderRecord) {
	if b.config.Commission.IsZero() {
		return
	}
	// Commission as percentage of notional value
	notional := rec.Volume.Mul(rec.Price)
	commission := notional.Mul(b.config.Commission)
	rec.Commission = commission
	b.equity = b.equity.Sub(commission)
}

// applySwap calculates overnight swap for a position.
func (b *SimBroker) applySwap(rec *OrderRecord, days int) {
	swapRate := b.config.SwapRate
	if swapRate.IsZero() {
		swapRate = decimal.NewFromFloat(0.00001) // fallback default
	}
	swap := rec.Volume.Mul(swapRate).Mul(decimal.NewFromInt(int64(days)))
	rec.Swap = rec.Swap.Add(swap)
	b.equity = b.equity.Sub(swap)
}

// expirePending removes pending orders that have waited longer than maxBars bars.
func (b *SimBroker) expirePending(currentBar int, maxBars int) {
	for i := 0; i < len(b.pending); i++ {
		if currentBar-b.pending[i].OpenBar > maxBars {
			b.pending[i].State = OrderCancelled
			b.history = append(b.history, b.pending[i])
			b.pending = append(b.pending[:i], b.pending[i+1:]...)
			i--
		}
	}
}

func (b *SimBroker) Account() sdk.AccountInfo {
	equity := b.equity
	for _, pos := range b.positions {
		equity = equity.Add(pos.Profit)
	}
	return sdk.AccountInfo{
		Balance:    b.balance,
		Equity:     equity,
		Margin:     decimal.Zero,
		FreeMargin: equity,
		Leverage:   b.config.Leverage,
		Currency:   "USD",
	}
}

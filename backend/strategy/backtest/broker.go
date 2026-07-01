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
	deals       []sdk.Deal
	trades      []Trade
	ticketSeq   int64
	dealSeq     int64
	equity      decimal.Decimal
	balance     decimal.Decimal
	currentBar  int
	currentBarTime time.Time // timestamp of the bar being processed
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

// SetBarTime updates the current bar timestamp for order time tracking.
func (b *SimBroker) SetBarTime(t time.Time) { b.currentBarTime = t }

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
		OpenTime:   b.currentBarTime,
		State:      OrderOpen,
		Comment:    req.Comment,
		Magic:      req.Magic,
		OpenBar:    b.currentBar,
	}

	// Apply commission (basis points of volume)
	b.applyCommission(rec)

	// Apply slippage for market orders
	if req.Type == sdk.OrderMarket {
		slip := b.config.Slippage
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
				pos.CloseTime = b.currentBarTime
				pos.Profit = profit
				b.history = append(b.history, pos)
				b.recordDeal(pos, closeVol, profit, pos.CloseTime)
				b.positions = append(b.positions[:i], b.positions[i+1:]...)
			} else {
				// Partial close — reduce volume, keep position open
				pos.Volume = pos.Volume.Sub(closeVol)
				b.recordDealPartial(pos, closeVol, profit, b.currentBarTime)
			}
			return sdk.OrderResult{RetCode: sdk.RetDone, Ticket: ticket, Volume: closeVol, Price: closePrice}, nil
		}
	}
	return sdk.OrderResult{RetCode: sdk.RetRejected}, fmt.Errorf("ticket %d not found", ticket)
}

func (b *SimBroker) PositionCloseBy(ticket1, ticket2 int64) (sdk.OrderResult, error) {
	var pos1, pos2 *OrderRecord
	var idx1, idx2 int = -1, -1
	for i, p := range b.positions {
		if p.Ticket == ticket1 {
			pos1 = p
			idx1 = i
		}
		if p.Ticket == ticket2 {
			pos2 = p
			idx2 = i
		}
	}
	if pos1 == nil || pos2 == nil {
		return sdk.OrderResult{RetCode: sdk.RetRejected}, fmt.Errorf("one or both tickets not found")
	}
	if pos1.Side == pos2.Side {
		return sdk.OrderResult{RetCode: sdk.RetRejected}, fmt.Errorf("positions must be opposite sides")
	}
	if pos1.Symbol != pos2.Symbol {
		return sdk.OrderResult{RetCode: sdk.RetRejected}, fmt.Errorf("positions must be same symbol")
	}

	closePrice := pos1.ClosePrice
	if closePrice.IsZero() {
		closePrice = pos1.Price
	}

	closeVol := pos1.Volume
	if pos2.Volume.LessThan(closeVol) {
		closeVol = pos2.Volume
	}

	profit1 := closePrice.Sub(pos1.Price).Mul(closeVol)
	if pos1.Side == sdk.SideSell {
		profit1 = profit1.Neg()
	}
	profit2 := closePrice.Sub(pos2.Price).Mul(closeVol)
	if pos2.Side == sdk.SideSell {
		profit2 = profit2.Neg()
	}
	netProfit := profit1.Add(profit2)

	b.equity = b.equity.Add(netProfit)
	b.balance = b.balance.Add(netProfit)

	now := b.currentBarTime

	if closeVol.Equal(pos1.Volume) {
		pos1.State = OrderClosed
		pos1.CloseTime = now
		pos1.ClosePrice = closePrice
		pos1.Profit = profit1
		b.history = append(b.history, pos1)
		b.recordDeal(pos1, closeVol, profit1, now)
	} else {
		pos1.Volume = pos1.Volume.Sub(closeVol)
		b.recordDealPartial(pos1, closeVol, profit1, now)
	}

	if closeVol.Equal(pos2.Volume) {
		pos2.State = OrderClosed
		pos2.CloseTime = now
		pos2.ClosePrice = closePrice
		pos2.Profit = profit2
		b.history = append(b.history, pos2)
		b.recordDeal(pos2, closeVol, profit2, now)
	} else {
		pos2.Volume = pos2.Volume.Sub(closeVol)
		b.recordDealPartial(pos2, closeVol, profit2, now)
	}

	// Remove closed positions from active list (process higher index first)
	if idx2 > idx1 {
		if pos2.State == OrderClosed {
			b.positions = append(b.positions[:idx2], b.positions[idx2+1:]...)
		}
		if pos1.State == OrderClosed {
			b.positions = append(b.positions[:idx1], b.positions[idx1+1:]...)
		}
	} else {
		if pos1.State == OrderClosed {
			b.positions = append(b.positions[:idx1], b.positions[idx1+1:]...)
		}
		if pos2.State == OrderClosed {
			b.positions = append(b.positions[:idx2], b.positions[idx2+1:]...)
		}
	}

	return sdk.OrderResult{RetCode: sdk.RetDone, Ticket: ticket1, Volume: closeVol, Price: closePrice}, nil
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

func (b *SimBroker) HistoryOrders(from, to int64) []sdk.Position {
	var out []sdk.Position
	for _, h := range b.history {
		if h.State != OrderClosed {
			continue
		}
		t := h.CloseTime.UnixMilli()
		if from > 0 && t < from {
			continue
		}
		if to > 0 && t > to {
			continue
		}
		out = append(out, sdk.Position{
			Ticket:     h.Ticket,
			Symbol:     h.Symbol,
			Side:       h.Side,
			Volume:     h.Volume,
			OpenPrice:  h.Price,
			StopLoss:   h.StopLoss,
			TakeProfit: h.TakeProfit,
			Profit:     h.Profit,
			Swap:       h.Swap,
			Commission: h.Commission,
			Comment:    h.Comment,
			Magic:      h.Magic,
			OpenTime:   h.OpenTime,
		})
	}
	return out
}

func (b *SimBroker) Deals(from, to int64, magic int32) []sdk.Deal {
	var out []sdk.Deal
	for _, d := range b.deals {
		t := d.CloseTime.UnixMilli()
		if from > 0 && t < from {
			continue
		}
		if to > 0 && t > to {
			continue
		}
		if magic != 0 && d.Magic != magic {
			continue
		}
		out = append(out, d)
	}
	return out
}

func (b *SimBroker) SymbolInfo(symbol string) (sdk.SymbolInfo, error) {
	digits := int32(5)
	point := decimal.NewFromFloat(0.00001)
	if b.config.SymbolDigits > 0 {
		digits = b.config.SymbolDigits
	}
	if !b.config.SymbolPoint.IsZero() {
		point = b.config.SymbolPoint
	}
	volMin := b.config.VolumeMin
	if volMin.IsZero() {
		volMin = decimal.NewFromFloat(0.01)
	}
	volMax := b.config.VolumeMax
	if volMax.IsZero() {
		volMax = decimal.NewFromInt(1000)
	}
	volStep := b.config.VolumeStep
	if volStep.IsZero() {
		volStep = decimal.NewFromFloat(0.01)
	}
	contractSize := b.config.ContractSize
	if contractSize.IsZero() {
		contractSize = decimal.NewFromInt(100000)
	}
	return sdk.SymbolInfo{
		Name:         symbol,
		Digits:       digits,
		Point:        point,
		VolumeMin:    volMin,
		VolumeMax:    volMax,
		VolumeStep:   volStep,
		ContractSize: contractSize,
		StopsLevel:   b.config.StopsLevel,
		TickValue:    b.config.TickValue,
		TickSize:     point,
	}, nil
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

func (b *SimBroker) recordDeal(rec *OrderRecord, vol decimal.Decimal, profit decimal.Decimal, now time.Time) {
	b.dealSeq++
	b.deals = append(b.deals, sdk.Deal{
		Ticket:      b.dealSeq,
		OrderTicket: rec.Ticket,
		Symbol:      rec.Symbol,
		Side:        rec.Side,
		Volume:      vol,
		Price:       rec.ClosePrice,
		Profit:      profit,
		Commission:  rec.Commission,
		Swap:        rec.Swap,
		Comment:     rec.Comment,
		Magic:       rec.Magic,
		OpenTime:    rec.OpenTime,
		CloseTime:   now,
	})
}

func (b *SimBroker) recordDealPartial(rec *OrderRecord, vol decimal.Decimal, profit decimal.Decimal, now time.Time) {
	b.dealSeq++
	b.deals = append(b.deals, sdk.Deal{
		Ticket:      b.dealSeq,
		OrderTicket: rec.Ticket,
		Symbol:      rec.Symbol,
		Side:        rec.Side,
		Volume:      vol,
		Price:       rec.ClosePrice,
		Profit:      profit,
		Comment:     rec.Comment,
		Magic:       rec.Magic,
		OpenTime:    rec.OpenTime,
		CloseTime:   now,
	})
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
	equity := b.balance
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

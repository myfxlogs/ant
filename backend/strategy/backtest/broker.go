package backtest

import (
	"fmt"
	"time"

	"github.com/shopspring/decimal"

	"alphaforge/strategy/sdk"
)

// SimBroker implements sdk.Broker for backtesting.
// It simulates order fills at the next bar's open price.
type SimBroker struct {
	config         Config
	positions      []*OrderRecord
	pending        []*OrderRecord
	history        []*OrderRecord
	deals          []sdk.Deal
	trades         []Trade
	ticketSeq      int64
	dealSeq        int64
	equity         decimal.Decimal
	balance        decimal.Decimal
	currentBar     int
	currentBarTime time.Time       // timestamp of the bar being processed
	currentPrice   decimal.Decimal // current bar's close price for floating P&L
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

// SetBarPrice sets the current bar's close price for floating P&L calculation.
func (b *SimBroker) SetBarPrice(price decimal.Decimal) { b.currentPrice = price }

// Trades returns all completed round-trip trades recorded by the broker.
func (b *SimBroker) Trades() []Trade { return b.trades }

// recordTrade creates a Trade record from a closed OrderRecord.
func (b *SimBroker) recordTrade(rec *OrderRecord) {
	var profitPct float64
	if rec.Price.IsPositive() {
		diff := rec.ClosePrice.Sub(rec.Price)
		if rec.Side == sdk.SideSell {
			diff = diff.Neg()
		}
		profitPct, _ = diff.Div(rec.Price).Float64()
		profitPct *= 100
	}
	b.trades = append(b.trades, Trade{
		Symbol:     rec.Symbol,
		Side:       rec.Side,
		EntryTime:  rec.OpenTime,
		ExitTime:   rec.CloseTime,
		EntryPrice: rec.Price,
		ExitPrice:  rec.ClosePrice,
		Volume:     rec.Volume,
		Profit:     rec.Profit,
		ProfitPct:  profitPct,
		Commission: rec.Commission,
		Swap:       rec.Swap,
		Comment:    rec.Comment,
	})
}

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

	// Market orders with price=0 fill at current market price (Python SDK convention).
	if req.Type == sdk.OrderMarket && rec.Price.IsZero() {
		rec.Price = b.currentPrice
	}

	// Apply spread to market order fills: buys pay ask (price + spread),
	// sells receive bid (price). Pending orders fill at their specified price.
	if req.Type == sdk.OrderMarket {
		rec.Price = b.applySpreadToFill(rec.Price, req.Side == sdk.SideBuy)
	}

	// Apply commission (basis points of volume)
	b.applyCommission(rec)

	// Slippage in MQL4 is the maximum acceptable deviation from requested price,
	// not an additive cost. Spread is applied as a fill cost above.
	// Slippage only matters for rejection checks when a separate fill price
	// differs from the requested price (e.g. gaps).

	// Check margin before opening — use equity including floating P&L
	// (consistent with checkMarginCall's computeEquity, not just realized equity).
	contractSize := b.config.ContractSize
	if contractSize.IsZero() {
		contractSize = decimal.NewFromInt(100000)
	}
	notional := req.Volume.Mul(contractSize).Mul(rec.Price)
	margin := notional.Div(decimal.NewFromInt(int64(b.config.Leverage)))
	equityWithFloating := b.Account().Equity
	if equityWithFloating.LessThan(margin) {
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
			// Use SL/TP close price if set, otherwise use current market price.
			closePrice := pos.ClosePrice
			if closePrice.IsZero() {
				closePrice = b.currentPrice
			}
			if closePrice.IsZero() {
				closePrice = pos.Price
			}
			// Apply spread: closing a sell position = buy (pay ask),
			// closing a buy position = sell (receive bid).
			closePrice = b.applySpreadToFill(closePrice, pos.Side == sdk.SideSell)
			// Charge swap for the holding period.
			days := b.swapDays(pos.OpenTime)
			if closeVol.Equal(pos.Volume) {
				b.applySwap(pos, closeVol, days)
			} else {
				b.chargeSwap(closeVol, days)
			}
			contractSize := b.config.ContractSize
			if contractSize.IsZero() {
				contractSize = decimal.NewFromInt(100000)
			}
			profit := closePrice.Sub(pos.Price).Mul(closeVol).Mul(contractSize)
			if pos.Side == sdk.SideSell {
				profit = profit.Neg()
			}
			b.equity = b.equity.Add(profit)
			b.balance = b.balance.Add(profit)

			if closeVol.Equal(pos.Volume) {
				// Close full position
				pos.State = OrderClosed
				pos.CloseTime = b.currentBarTime
				pos.ClosePrice = closePrice
				pos.Profit = profit
				b.history = append(b.history, pos)
				b.recordDeal(pos, closeVol, profit, pos.CloseTime)
				b.recordTrade(pos)
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
	var idx1, idx2 = -1, -1
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

	baseClosePrice := pos1.ClosePrice
	if baseClosePrice.IsZero() {
		baseClosePrice = pos1.Price
	}
	// Apply spread per position: closing a sell = buy (pay ask), closing a buy = sell (receive bid).
	closePrice1 := b.applySpreadToFill(baseClosePrice, pos1.Side == sdk.SideSell)
	closePrice2 := b.applySpreadToFill(baseClosePrice, pos2.Side == sdk.SideSell)

	closeVol := pos1.Volume
	if pos2.Volume.LessThan(closeVol) {
		closeVol = pos2.Volume
	}

	// Charge swap for both positions based on their holding periods.
	days1 := b.swapDays(pos1.OpenTime)
	days2 := b.swapDays(pos2.OpenTime)

	contractSize := b.config.ContractSize
	if contractSize.IsZero() {
		contractSize = decimal.NewFromInt(100000)
	}
	profit1 := closePrice1.Sub(pos1.Price).Mul(closeVol).Mul(contractSize)
	if pos1.Side == sdk.SideSell {
		profit1 = profit1.Neg()
	}
	profit2 := closePrice2.Sub(pos2.Price).Mul(closeVol).Mul(contractSize)
	if pos2.Side == sdk.SideSell {
		profit2 = profit2.Neg()
	}

	// Charge swap for both positions.
	if closeVol.Equal(pos1.Volume) {
		b.applySwap(pos1, closeVol, days1)
	} else {
		b.chargeSwap(closeVol, days1)
	}
	if closeVol.Equal(pos2.Volume) {
		b.applySwap(pos2, closeVol, days2)
	} else {
		b.chargeSwap(closeVol, days2)
	}

	netProfit := profit1.Add(profit2)

	b.equity = b.equity.Add(netProfit)
	b.balance = b.balance.Add(netProfit)

	now := b.currentBarTime

	if closeVol.Equal(pos1.Volume) {
		pos1.State = OrderClosed
		pos1.CloseTime = now
		pos1.ClosePrice = closePrice1
		pos1.Profit = profit1
		b.history = append(b.history, pos1)
		b.recordDeal(pos1, closeVol, profit1, now)
		b.recordTrade(pos1)
	} else {
		pos1.Volume = pos1.Volume.Sub(closeVol)
		b.recordDealPartial(pos1, closeVol, profit1, now)
	}

	if closeVol.Equal(pos2.Volume) {
		pos2.State = OrderClosed
		pos2.CloseTime = now
		pos2.ClosePrice = closePrice2
		pos2.Profit = profit2
		b.history = append(b.history, pos2)
		b.recordDeal(pos2, closeVol, profit2, now)
		b.recordTrade(pos2)
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

	return sdk.OrderResult{RetCode: sdk.RetDone, Ticket: ticket1, Volume: closeVol, Price: closePrice1}, nil
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
				Swap:       p.Swap,
				Commission: p.Commission,
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
			ClosePrice: h.ClosePrice,
			CloseTime:  h.CloseTime,
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

// applySpreadToFill adjusts fill price for spread: buys pay ask (price + spread),
// sells receive bid (price). Returns original price if spread is zero.
func (b *SimBroker) applySpreadToFill(price decimal.Decimal, isBuy bool) decimal.Decimal {
	spread := b.config.Spread
	if spread.IsZero() {
		spread = b.config.Slippage // fallback: use slippage as spread if spread not set
	}
	if spread.IsZero() {
		return price
	}
	if isBuy {
		return price.Add(spread)
	}
	return price
}

// applyCommission deducts commission from equity and records it.
func (b *SimBroker) applyCommission(rec *OrderRecord) {
	if b.config.Commission.IsZero() {
		return
	}
	// Commission as percentage of notional value (volume * contractSize * price)
	contractSize := b.config.ContractSize
	if contractSize.IsZero() {
		contractSize = decimal.NewFromInt(100000)
	}
	notional := rec.Volume.Mul(contractSize).Mul(rec.Price)
	commission := notional.Mul(b.config.Commission)
	rec.Commission = commission
	b.equity = b.equity.Sub(commission)
	b.balance = b.balance.Sub(commission)
}

// computeSwap calculates the swap charge for a given volume and holding duration in days.
func (b *SimBroker) computeSwap(volume decimal.Decimal, days int) decimal.Decimal {
	swapRate := b.config.SwapRate
	if swapRate.IsZero() {
		swapRate = decimal.NewFromFloat(0.00001) // fallback default
	}
	contractSize := b.config.ContractSize
	if contractSize.IsZero() {
		contractSize = decimal.NewFromInt(100000)
	}
	return volume.Mul(contractSize).Mul(swapRate).Mul(decimal.NewFromInt(int64(days)))
}

// applySwap charges swap to balance/equity and records it on the position.
func (b *SimBroker) applySwap(rec *OrderRecord, volume decimal.Decimal, days int) {
	swap := b.computeSwap(volume, days)
	rec.Swap = rec.Swap.Add(swap)
	b.equity = b.equity.Sub(swap)
	b.balance = b.balance.Sub(swap)
}

// chargeSwap deducts swap from balance/equity without recording on a position.
// Used for partial closes where the position remains open.
func (b *SimBroker) chargeSwap(volume decimal.Decimal, days int) {
	swap := b.computeSwap(volume, days)
	b.equity = b.equity.Sub(swap)
	b.balance = b.balance.Sub(swap)
}

// swapDays calculates the number of overnight holding days for a position.
func (b *SimBroker) swapDays(openTime time.Time) int {
	if b.currentBarTime.IsZero() || openTime.IsZero() {
		return 0
	}
	heldDuration := b.currentBarTime.Sub(openTime)
	days := int64(heldDuration.Hours() / 24)
	if days < 0 {
		return 0
	}
	return int(days)
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

func (b *SimBroker) Account() sdk.AccountInfo {
	contractSize := b.config.ContractSize
	if contractSize.IsZero() {
		contractSize = decimal.NewFromInt(100000)
	}
	floatingProfit := decimal.Zero
	marginUsed := decimal.Zero
	for _, pos := range b.positions {
		// Calculate unrealized P&L using current market price
		closePrice := pos.ClosePrice
		if closePrice.IsZero() {
			closePrice = b.currentPrice
			if closePrice.IsZero() {
				closePrice = pos.Price
			}
		}
		profit := closePrice.Sub(pos.Price).Mul(pos.Volume).Mul(contractSize)
		if pos.Side == sdk.SideSell {
			profit = profit.Neg()
		}
		floatingProfit = floatingProfit.Add(profit)
		// Margin = notional / leverage
		notional := pos.Volume.Mul(contractSize).Mul(pos.Price)
		marginUsed = marginUsed.Add(notional.Div(decimal.NewFromInt(int64(b.config.Leverage))))
	}
	// equity = balance + floating P&L (commission already deducted from balance at open)
	equity := b.balance.Add(floatingProfit)
	return sdk.AccountInfo{
		Balance:    b.balance,
		Equity:     equity,
		Margin:     marginUsed,
		FreeMargin: equity.Sub(marginUsed),
		Leverage:   b.config.Leverage,
		Currency:   "USD",
	}
}

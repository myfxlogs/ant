package interp

import (
	"anttrader/strategy/sdk"
)

// MQL4OrderPool simulates the MQL4 OrdersTotal()/OrderSelect() model.
// It is rebuilt from SDK positions/orders at the start of each bar.
type MQL4OrderPool struct {
	orders   []OrderRecord
	history  []OrderRecord
	current *OrderRecord
	total   int
}

// OrderRecord represents a single order in the MQL4 order pool.
type OrderRecord struct {
	Ticket      int64
	Symbol      string
	Type        int32 // OP_BUY=0, OP_SELL=1, OP_BUYLIMIT=2, ...
	Lots        Value
	OpenPrice   Value
	StopLoss    Value
	TakeProfit  Value
	ClosePrice  Value
	OpenTime    int64
	CloseTime   int64
	Profit      Value
	Commission  Value
	Swap        Value
	MagicNumber int32
	Comment     string
}

// Reset rebuilds the order pool from SDK broker state.
func (p *MQL4OrderPool) Reset(ctx sdk.Context) {
	if ctx == nil {
		return
	}
	broker := ctx.Broker()
	if broker == nil {
		return
	}

	positions := broker.Positions(0)
	orders := broker.Orders(0)

	p.orders = make([]OrderRecord, 0, len(positions)+len(orders))
	for _, pos := range positions {
		rec := OrderRecord{
			Ticket:      pos.Ticket,
			Symbol:      pos.Symbol,
			Type:        int32(pos.Side), // OP_BUY=0(→SideBuy=1?), OP_SELL=1
			Lots:        DecimalVal(pos.Volume),
			OpenPrice:   DecimalVal(pos.OpenPrice),
			StopLoss:    DecimalVal(pos.StopLoss),
			TakeProfit:  DecimalVal(pos.TakeProfit),
			Profit:      DecimalVal(pos.Profit),
			Commission:  DecimalVal(pos.Commission),
			Swap:        DecimalVal(pos.Swap),
			MagicNumber: pos.Magic,
			Comment:     pos.Comment,
			OpenTime:    pos.OpenTime.UnixMilli(),
		}
		// Map SDK side to MQL4 order type
		if pos.Side == sdk.SideBuy {
			rec.Type = 0 // OP_BUY
		} else {
			rec.Type = 1 // OP_SELL
		}
		p.orders = append(p.orders, rec)
	}
	for _, ord := range orders {
		rec := OrderRecord{
			Ticket:      ord.Ticket,
			Symbol:      ord.Symbol,
			Lots:        DecimalVal(ord.Volume),
			OpenPrice:   DecimalVal(ord.Price),
			StopLoss:    DecimalVal(ord.StopLoss),
			TakeProfit:  DecimalVal(ord.TakeProfit),
			MagicNumber: ord.Magic,
			Comment:     ord.Comment,
			OpenTime:    ord.OpenTime.UnixMilli(),
		}
		// Map SDK order type to MQL4 pending order type
		switch ord.Type {
		case sdk.OrderLimit:
			if ord.Side == sdk.SideBuy {
				rec.Type = 2 // OP_BUYLIMIT
			} else {
				rec.Type = 3 // OP_SELLLIMIT
			}
		case sdk.OrderStop:
			if ord.Side == sdk.SideBuy {
				rec.Type = 4 // OP_BUYSTOP
			} else {
				rec.Type = 5 // OP_SELLSTOP
			}
		}
		p.orders = append(p.orders, rec)
	}
	p.total = len(p.orders)
	p.current = nil

	// Load history orders
	histPositions := broker.HistoryOrders(0, 0)
	p.history = make([]OrderRecord, 0, len(histPositions))
	for _, h := range histPositions {
		rec := OrderRecord{
			Ticket:      h.Ticket,
			Symbol:      h.Symbol,
			Lots:        DecimalVal(h.Volume),
			OpenPrice:   DecimalVal(h.OpenPrice),
			StopLoss:    DecimalVal(h.StopLoss),
			TakeProfit:  DecimalVal(h.TakeProfit),
			Profit:      DecimalVal(h.Profit),
			Commission:  DecimalVal(h.Commission),
			Swap:        DecimalVal(h.Swap),
			MagicNumber: h.Magic,
			Comment:     h.Comment,
			OpenTime:    h.OpenTime.UnixMilli(),
		}
		if h.Side == sdk.SideBuy {
			rec.Type = 0 // OP_BUY
		} else {
			rec.Type = 1 // OP_SELL
		}
		p.history = append(p.history, rec)
	}
}

// Select implements OrderSelect(index, SELECT_BY_POS, MODE_TRADES).
func (p *MQL4OrderPool) Select(index int) bool {
	if index < 0 || index >= p.total {
		return false
	}
	p.current = &p.orders[index]
	return true
}

// SelectHistory implements OrderSelect(index, SELECT_BY_POS, MODE_HISTORY).
func (p *MQL4OrderPool) SelectHistory(index int) bool {
	if index < 0 || index >= len(p.history) {
		return false
	}
	p.current = &p.history[index]
	return true
}

// SelectByTicket implements OrderSelect(ticket, SELECT_BY_TICKET, MODE_TRADES).
func (p *MQL4OrderPool) SelectByTicket(ticket int64) bool {
	for i := range p.orders {
		if p.orders[i].Ticket == ticket {
			p.current = &p.orders[i]
			return true
		}
	}
	// Also search history
	for i := range p.history {
		if p.history[i].Ticket == ticket {
			p.current = &p.history[i]
			return true
		}
	}
	return false
}

// HistoryTotal returns the number of closed orders in history.
func (p *MQL4OrderPool) HistoryTotal() int {
	return len(p.history)
}

// Total returns the number of orders in the pool.
func (p *MQL4OrderPool) Total() int {
	return p.total
}

// ── Order* property functions (MQL4) ────────────────────────────────

func (p *MQL4OrderPool) Ticket() int64 {
	if p.current == nil {
		return 0
	}
	return p.current.Ticket
}

func (p *MQL4OrderPool) Symbol() string {
	if p.current == nil {
		return ""
	}
	return p.current.Symbol
}

func (p *MQL4OrderPool) Type() int32 {
	if p.current == nil {
		return -1
	}
	return p.current.Type
}

func (p *MQL4OrderPool) Lots() Value {
	if p.current == nil {
		return NoneVal()
	}
	return p.current.Lots
}

func (p *MQL4OrderPool) OpenPrice() Value {
	if p.current == nil {
		return NoneVal()
	}
	return p.current.OpenPrice
}

func (p *MQL4OrderPool) StopLoss() Value {
	if p.current == nil {
		return NoneVal()
	}
	return p.current.StopLoss
}

func (p *MQL4OrderPool) TakeProfit() Value {
	if p.current == nil {
		return NoneVal()
	}
	return p.current.TakeProfit
}

func (p *MQL4OrderPool) Profit() Value {
	if p.current == nil {
		return NoneVal()
	}
	return p.current.Profit
}

func (p *MQL4OrderPool) Commission() Value {
	if p.current == nil {
		return NoneVal()
	}
	return p.current.Commission
}

func (p *MQL4OrderPool) Swap() Value {
	if p.current == nil {
		return NoneVal()
	}
	return p.current.Swap
}

func (p *MQL4OrderPool) MagicNumber() int32 {
	if p.current == nil {
		return 0
	}
	return p.current.MagicNumber
}

func (p *MQL4OrderPool) Comment() string {
	if p.current == nil {
		return ""
	}
	return p.current.Comment
}

func (p *MQL4OrderPool) OpenTime() int64 {
	if p.current == nil {
		return 0
	}
	return p.current.OpenTime
}

func (p *MQL4OrderPool) CloseTime() int64 {
	if p.current == nil {
		return 0
	}
	return p.current.CloseTime
}

func (p *MQL4OrderPool) ClosePrice() Value {
	if p.current == nil {
		return NoneVal()
	}
	return p.current.ClosePrice
}

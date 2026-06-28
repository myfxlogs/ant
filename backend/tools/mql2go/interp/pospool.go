package interp

import (
	"anttrader/strategy/sdk"
)

// MQL5PositionPool simulates the MQL5 PositionsTotal()/PositionSelect() model.
type MQL5PositionPool struct {
	positions []PositionRecord
	current   *PositionRecord
	total     int
}

// PositionRecord represents a single position in the MQL5 position pool.
type PositionRecord struct {
	Ticket      int64
	Symbol      string
	Side        int32 // 0=buy, 1=sell
	Volume      Value
	OpenPrice   Value
	StopLoss    Value
	TakeProfit  Value
	Profit      Value
	Commission  Value
	Swap        Value
	MagicNumber int32
	Comment     string
	OpenTime    int64
}

// Reset rebuilds the position pool from SDK broker state.
func (p *MQL5PositionPool) Reset(ctx sdk.Context) {
	if ctx == nil {
		return
	}
	broker := ctx.Broker()
	if broker == nil {
		return
	}

	positions := broker.Positions(0)
	p.positions = make([]PositionRecord, 0, len(positions))
	for _, pos := range positions {
		rec := PositionRecord{
			Ticket:      pos.Ticket,
			Symbol:      pos.Symbol,
			Volume:      DecimalVal(pos.Volume),
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
		if pos.Side == sdk.SideBuy {
			rec.Side = 0
		} else {
			rec.Side = 1
		}
		p.positions = append(p.positions, rec)
	}
	p.total = len(p.positions)
	p.current = nil
}

// Total returns the number of positions.
func (p *MQL5PositionPool) Total() int {
	return p.total
}

// GetTicket returns the ticket at the given index and selects it.
func (p *MQL5PositionPool) GetTicket(index int) int64 {
	if index < 0 || index >= p.total {
		return 0
	}
	p.current = &p.positions[index]
	return p.current.Ticket
}

// SelectByTicket selects a position by ticket.
func (p *MQL5PositionPool) SelectByTicket(ticket int64) bool {
	for i := range p.positions {
		if p.positions[i].Ticket == ticket {
			p.current = &p.positions[i]
			return true
		}
	}
	return false
}

// ── PositionGet* functions (MQL5) ───────────────────────────────────

func (p *MQL5PositionPool) Symbol() string {
	if p.current == nil {
		return ""
	}
	return p.current.Symbol
}

func (p *MQL5PositionPool) GetDouble(prop int32) Value {
	if p.current == nil {
		return NoneVal()
	}
	switch prop {
	case 0: // POSITION_VOLUME
		return p.current.Volume
	case 1: // POSITION_OPEN_PRICE
		return p.current.OpenPrice
	case 2: // POSITION_SL
		return p.current.StopLoss
	case 3: // POSITION_TP
		return p.current.TakeProfit
	case 4: // POSITION_PROFIT
		return p.current.Profit
	case 5: // POSITION_SWAP
		return p.current.Swap
	default:
		return NoneVal()
	}
}

func (p *MQL5PositionPool) GetInteger(prop int32) int64 {
	if p.current == nil {
		return 0
	}
	switch prop {
	case 0: // POSITION_TICKET
		return p.current.Ticket
	case 1: // POSITION_MAGIC
		return int64(p.current.MagicNumber)
	case 2: // POSITION_TIME
		return p.current.OpenTime
	default:
		return 0
	}
}

func (p *MQL5PositionPool) GetString(prop int32) string {
	if p.current == nil {
		return ""
	}
	switch prop {
	case 0: // POSITION_SYMBOL
		return p.current.Symbol
	case 1: // POSITION_COMMENT
		return p.current.Comment
	default:
		return ""
	}
}

package service

import (
	"context"
	"errors"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
)

var (
	ErrOrderNotFound  = errors.New("order not found")
	ErrNoCurrentTick  = errors.New("no current tick available")
	ErrInvalidRequest = errors.New("invalid request")
)

type Tick struct {
	Time   time.Time
	Bid    float64
	Ask    float64
	Symbol string
}

type BacktestCostConfig struct {
	CommissionRate float64
	SlippageRate   float64
}

type btPosition struct {
	Ticket     int64
	Symbol     string
	Type       string
	Volume     float64
	OpenPrice  float64
	OpenTime   time.Time
	StopLoss   float64
	TakeProfit float64
	Comment    string
	Magic      int64
}

type btPendingOrder struct {
	Ticket       int64
	Symbol       string
	Type         string
	Volume       float64
	TriggerPrice float64
	StopLoss     float64
	TakeProfit   float64
	Comment      string
	Magic        int64
	PlacedTime   time.Time
}

type btClosedOrder struct {
	Ticket     int64
	Symbol     string
	Type       string
	Volume     float64
	OpenPrice  float64
	ClosePrice float64
	OpenTime   time.Time
	CloseTime  time.Time
	Profit     float64
	Commission float64
	StopLoss   float64
	TakeProfit float64
	Comment    string
	Magic      int64
}

type TickBacktestEngine struct {
	mu           sync.Mutex
	nextTicket   int64
	positions    map[int64]*btPosition
	pending      map[int64]*btPendingOrder
	history      []*btClosedOrder
	currentTicks map[string]*Tick
	cost         BacktestCostConfig
}

func NewTickBacktestEngine(cost BacktestCostConfig) *TickBacktestEngine {
	return &TickBacktestEngine{
		nextTicket:   1,
		positions:    make(map[int64]*btPosition),
		pending:      make(map[int64]*btPendingOrder),
		currentTicks: make(map[string]*Tick),
		cost:         cost,
	}
}

func (e *TickBacktestEngine) allocTicket() int64 {
	t := e.nextTicket
	e.nextTicket++
	return t
}

func (e *TickBacktestEngine) Feed(tick Tick) []btClosedOrder {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.currentTicks[tick.Symbol] = &tick
	var fills []btClosedOrder
	fills = append(fills, e.checkPendingOrders(tick)...)
	fills = append(fills, e.checkSLTP(tick)...)
	return fills
}

func (e *TickBacktestEngine) checkPendingOrders(tick Tick) []btClosedOrder {
	var fills []btClosedOrder
	for id, po := range e.pending {
		if po.Symbol != tick.Symbol {
			continue
		}
		triggered := false
		fillPrice := 0.0
		switch strings.ToLower(po.Type) {
		case "buy_limit":
			if tick.Ask <= po.TriggerPrice {
				triggered = true
				fillPrice = tick.Ask
			}
		case "sell_limit":
			if tick.Bid >= po.TriggerPrice {
				triggered = true
				fillPrice = tick.Bid
			}
		case "buy_stop":
			if tick.Ask >= po.TriggerPrice {
				triggered = true
				fillPrice = tick.Ask
			}
		case "sell_stop":
			if tick.Bid <= po.TriggerPrice {
				triggered = true
				fillPrice = tick.Bid
			}
		}
		if triggered {
			fillPrice = e.applySlippage(fillPrice, po.Type)
			pos := &btPosition{
				Ticket:     po.Ticket,
				Symbol:     po.Symbol,
				Type:       directionFromPending(po.Type),
				Volume:     po.Volume,
				OpenPrice:  fillPrice,
				OpenTime:   tick.Time,
				StopLoss:   po.StopLoss,
				TakeProfit: po.TakeProfit,
				Comment:    po.Comment,
				Magic:      po.Magic,
			}
			e.positions[pos.Ticket] = pos
			delete(e.pending, id)
		}
	}
	return fills
}

func (e *TickBacktestEngine) checkSLTP(tick Tick) []btClosedOrder {
	var fills []btClosedOrder
	for id, pos := range e.positions {
		if pos.Symbol != tick.Symbol {
			continue
		}
		isBuy := strings.ToLower(pos.Type) == "buy"
		closePrice := 0.0
		reason := ""
		if isBuy {
			if pos.StopLoss > 0 && tick.Bid <= pos.StopLoss {
				closePrice = tick.Bid
				reason = "sl"
			} else if pos.TakeProfit > 0 && tick.Bid >= pos.TakeProfit {
				closePrice = tick.Bid
				reason = "tp"
			}
		} else {
			if pos.StopLoss > 0 && tick.Ask >= pos.StopLoss {
				closePrice = tick.Ask
				reason = "sl"
			} else if pos.TakeProfit > 0 && tick.Ask <= pos.TakeProfit {
				closePrice = tick.Ask
				reason = "tp"
			}
		}
		if reason != "" {
			co := e.closePosition(pos, closePrice, tick.Time, reason)
			fills = append(fills, co)
			delete(e.positions, id)
		}
	}
	return fills
}

func (e *TickBacktestEngine) closePosition(pos *btPosition, closePrice float64, closeTime time.Time, reason string) btClosedOrder {
	profit := 0.0
	if strings.ToLower(pos.Type) == "buy" {
		profit = (closePrice - pos.OpenPrice) * pos.Volume
	} else {
		profit = (pos.OpenPrice - closePrice) * pos.Volume
	}
	commission := pos.Volume * e.cost.CommissionRate
	co := btClosedOrder{
		Ticket:     pos.Ticket,
		Symbol:     pos.Symbol,
		Type:       pos.Type,
		Volume:     pos.Volume,
		OpenPrice:  pos.OpenPrice,
		ClosePrice: closePrice,
		OpenTime:   pos.OpenTime,
		CloseTime:  closeTime,
		Profit:     profit - commission,
		Commission: commission,
		StopLoss:   pos.StopLoss,
		TakeProfit: pos.TakeProfit,
		Comment:    pos.Comment + " [" + reason + "]",
		Magic:      pos.Magic,
	}
	e.history = append(e.history, &co)
	return co
}

func (e *TickBacktestEngine) applySlippage(price float64, orderType string) float64 {
	if e.cost.SlippageRate <= 0 {
		return price
	}
	t := strings.ToLower(orderType)
	if strings.HasPrefix(t, "buy") {
		return price * (1 + e.cost.SlippageRate)
	}
	return price * (1 - e.cost.SlippageRate)
}

func directionFromPending(pendingType string) string {
	t := strings.ToLower(pendingType)
	if strings.HasPrefix(t, "buy") {
		return "buy"
	}
	return "sell"
}

func (e *TickBacktestEngine) OrderSend(_ context.Context, _ uuid.UUID, req *OrderSendRequest) (*OrderResponse, error) {
	if req == nil || req.Symbol == "" || req.Volume <= 0 {
		return nil, ErrInvalidRequest
	}
	e.mu.Lock()
	defer e.mu.Unlock()

	t := strings.ToLower(req.Type)
	isPending := strings.Contains(t, "limit") || strings.Contains(t, "stop")

	if isPending {
		ticket := e.allocTicket()
		po := &btPendingOrder{
			Ticket:       ticket,
			Symbol:       req.Symbol,
			Type:         t,
			Volume:       req.Volume,
			TriggerPrice: req.Price,
			StopLoss:     req.StopLoss,
			TakeProfit:   req.TakeProfit,
			Comment:      req.Comment,
			Magic:        req.Magic,
		}
		e.pending[ticket] = po
		return &OrderResponse{
			Ticket: ticket, Symbol: req.Symbol, Type: t,
			Volume: req.Volume, Price: req.Price,
			StopLoss: req.StopLoss, TakeProfit: req.TakeProfit,
			Comment: req.Comment, Magic: req.Magic,
		}, nil
	}

	tick, ok := e.currentTicks[req.Symbol]
	if !ok {
		return nil, ErrNoCurrentTick
	}
	fillPrice := 0.0
	if t == "buy" {
		fillPrice = tick.Ask
	} else {
		fillPrice = tick.Bid
	}
	fillPrice = e.applySlippage(fillPrice, t)
	ticket := e.allocTicket()
	pos := &btPosition{
		Ticket:     ticket,
		Symbol:     req.Symbol,
		Type:       t,
		Volume:     req.Volume,
		OpenPrice:  fillPrice,
		OpenTime:   tick.Time,
		StopLoss:   req.StopLoss,
		TakeProfit: req.TakeProfit,
		Comment:    req.Comment,
		Magic:      req.Magic,
	}
	e.positions[ticket] = pos
	return &OrderResponse{
		Ticket: ticket, Symbol: req.Symbol, Type: t,
		Volume: req.Volume, Price: fillPrice,
		StopLoss: req.StopLoss, TakeProfit: req.TakeProfit,
		Comment: req.Comment, Magic: req.Magic,
		OpenTime: tick.Time.Format(time.RFC3339),
	}, nil
}

func (e *TickBacktestEngine) OrderModify(_ context.Context, _ uuid.UUID, req *OrderModifyRequest) (*OrderResponse, error) {
	if req == nil {
		return nil, ErrInvalidRequest
	}
	e.mu.Lock()
	defer e.mu.Unlock()

	if pos, ok := e.positions[req.Ticket]; ok {
		pos.StopLoss = req.StopLoss
		pos.TakeProfit = req.TakeProfit
		return &OrderResponse{
			Ticket: pos.Ticket, Symbol: pos.Symbol, Type: pos.Type,
			Volume: pos.Volume, Price: pos.OpenPrice,
			StopLoss: pos.StopLoss, TakeProfit: pos.TakeProfit,
		}, nil
	}
	if po, ok := e.pending[req.Ticket]; ok {
		po.StopLoss = req.StopLoss
		po.TakeProfit = req.TakeProfit
		if req.Price > 0 {
			po.TriggerPrice = req.Price
		}
		return &OrderResponse{
			Ticket: po.Ticket, Symbol: po.Symbol, Type: po.Type,
			Volume: po.Volume, Price: po.TriggerPrice,
			StopLoss: po.StopLoss, TakeProfit: po.TakeProfit,
		}, nil
	}
	return nil, ErrOrderNotFound
}

func (e *TickBacktestEngine) OrderClose(_ context.Context, _ uuid.UUID, req *OrderCloseRequest) (*OrderResponse, error) {
	if req == nil {
		return nil, ErrInvalidRequest
	}
	e.mu.Lock()
	defer e.mu.Unlock()

	pos, ok := e.positions[req.Ticket]
	if !ok {
		if _, pok := e.pending[req.Ticket]; pok {
			delete(e.pending, req.Ticket)
			return &OrderResponse{Ticket: req.Ticket, Comment: "cancelled"}, nil
		}
		return nil, ErrOrderNotFound
	}
	tick, ok := e.currentTicks[pos.Symbol]
	if !ok {
		return nil, ErrNoCurrentTick
	}
	closePrice := tick.Bid
	if strings.ToLower(pos.Type) == "sell" {
		closePrice = tick.Ask
	}
	co := e.closePosition(pos, closePrice, tick.Time, "manual")
	delete(e.positions, req.Ticket)
	return &OrderResponse{
		Ticket: co.Ticket, Symbol: co.Symbol, Type: co.Type,
		Volume: co.Volume, Price: co.ClosePrice, Profit: co.Profit,
	}, nil
}

func (e *TickBacktestEngine) GetPositions(_ context.Context, _, _ uuid.UUID) ([]*PositionResponse, error) {
	e.mu.Lock()
	defer e.mu.Unlock()
	out := make([]*PositionResponse, 0, len(e.positions))
	for _, pos := range e.positions {
		out = append(out, &PositionResponse{
			Ticket:    pos.Ticket,
			Symbol:    pos.Symbol,
			Type:      pos.Type,
			Volume:    pos.Volume,
			OpenPrice: pos.OpenPrice,
			StopLoss:  pos.StopLoss,
			TakeProfit: pos.TakeProfit,
			OpenTime:  pos.OpenTime.Format(time.RFC3339),
			Comment:   pos.Comment,
			Magic:     pos.Magic,
		})
	}
	return out, nil
}

func (e *TickBacktestEngine) GetOrderHistory(_ context.Context, _, _ uuid.UUID, from, to time.Time) ([]*HistoryOrderResponse, error) {
	e.mu.Lock()
	defer e.mu.Unlock()
	var out []*HistoryOrderResponse
	for _, co := range e.history {
		if !co.CloseTime.Before(from) && !co.CloseTime.After(to) {
			out = append(out, &HistoryOrderResponse{
				Ticket:     co.Ticket,
				Symbol:     co.Symbol,
				Type:       co.Type,
				Volume:     co.Volume,
				OpenPrice:  co.OpenPrice,
				ClosePrice: co.ClosePrice,
				Profit:     co.Profit,
				Commission: co.Commission,
				OpenTime:   co.OpenTime.Format(time.RFC3339),
				CloseTime:  co.CloseTime.Format(time.RFC3339),
				StopLoss:   co.StopLoss,
				TakeProfit: co.TakeProfit,
				Comment:    co.Comment,
				Magic:      co.Magic,
			})
		}
	}
	return out, nil
}

func (e *TickBacktestEngine) GetHistory() []*btClosedOrder {
	e.mu.Lock()
	defer e.mu.Unlock()
	out := make([]*btClosedOrder, len(e.history))
	copy(out, e.history)
	return out
}

var _ MatchingEngine = (*TickBacktestEngine)(nil)

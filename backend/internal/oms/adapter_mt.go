package oms

import (
	"context"
	"fmt"
	"strconv"

	"github.com/shopspring/decimal"

	"anttrader/internal/mthub"
)

// MTBrokerAdapter adapts mthub.OrderExecutor to the OMS BrokerAdapter interface.
// It translates OMS OrderRequest into mthub.OrderRequest and maps broker responses back.
type MTBrokerAdapter struct {
	executor mthub.OrderExecutor
}

// NewMTBrokerAdapter creates an MT broker adapter wrapping a session executor.
func NewMTBrokerAdapter(exec mthub.OrderExecutor) *MTBrokerAdapter {
	return &MTBrokerAdapter{executor: exec}
}

// Submit converts an OMS order request to a mthub order and sends it to the broker.
func (a *MTBrokerAdapter) Submit(ctx context.Context, req *OrderRequest) (*BrokerResp, error) {
	side := mthub.SideBuy
	if req.Side == "sell" {
		side = mthub.SideSell
	}

	mreq := &mthub.OrderRequest{
		AccountID:  req.AccountID,
		Canonical:  req.Symbol,
		Side:       side,
		OrderType:  mthub.OrderMarket,
		Volume:     decimal.NewFromFloat(req.Volume),
		Price:      decimal.NewFromFloat(req.Price),
		StopLoss:   decimal.NewFromFloat(req.StopLoss),
		TakeProfit: decimal.NewFromFloat(req.TakeProfit),
		Comment:    req.Comment,
	}

	ticket, err := a.executor.PlaceOrder(ctx, mreq)
	if err != nil {
		return nil, err
	}

	return &BrokerResp{
		Ticket: strconv.FormatInt(ticket, 10),
		State:  StateSubmitted,
	}, nil
}

// Cancel requests cancellation of an order by ticket.
func (a *MTBrokerAdapter) Cancel(ctx context.Context, ticket string) error {
	t, err := strconv.ParseInt(ticket, 10, 64)
	if err != nil {
		return fmt.Errorf("adapter_mt: invalid ticket %q: %w", ticket, err)
	}
	// Look up the order to get its full volume before closing.
	orders, err := a.executor.FetchOpenedOrders(ctx)
	if err != nil {
		return fmt.Errorf("adapter_mt: cancel fetch orders: %w", err)
	}
	var vol decimal.Decimal
	for _, o := range orders {
		if o.Ticket == t {
			vol = o.Volume
			break
		}
	}
	if vol.IsZero() {
		return fmt.Errorf("adapter_mt: order %d not found in opened orders", t)
	}
	return a.executor.CloseOrder(ctx, t, vol)
}

// Modify adjusts price and/or stop-loss of an order by ticket.
func (a *MTBrokerAdapter) Modify(ctx context.Context, ticket string, price, stopPrice float64) error {
	t, err := strconv.ParseInt(ticket, 10, 64)
	if err != nil {
		return fmt.Errorf("adapter_mt: invalid ticket %q: %w", ticket, err)
	}
	return a.executor.ModifyOrder(ctx, t, decimal.NewFromFloat(stopPrice), decimal.Zero, decimal.NewFromFloat(price))
}

// Query retrieves the current broker-side state of an order by ticket.
func (a *MTBrokerAdapter) Query(ctx context.Context, ticket string) (*Order, error) {
	orders, err := a.executor.FetchOpenedOrders(ctx)
	if err != nil {
		return nil, err
	}
	for _, o := range orders {
		if fmt.Sprintf("%d", o.Ticket) == ticket {
			return &Order{
				Ticket:    ticket,
				Symbol:    o.Canonical,
				Volume:    o.Volume.InexactFloat64(),
				Price:     o.OpenPrice.InexactFloat64(),
				StopLoss:  decimal.Zero.InexactFloat64(),
				State:     mtStateToOMS(o.State),
				AccountID: o.AccountID,
			}, nil
		}
	}
	return nil, fmt.Errorf("adapter_mt: order %s not found", ticket)
}

func mtStateToOMS(s mthub.OrderState) OrderState {
	switch s {
	case mthub.OrderStatePending:
		return StateSubmitted
	case mthub.OrderStateOpen:
		return StateWorking
	case mthub.OrderStateClosed:
		return StateFilled
	case mthub.OrderStateCancelled:
		return StateCancelled
	case mthub.OrderStateRejected:
		return StateRejected
	default:
		return StateFailed
	}
}

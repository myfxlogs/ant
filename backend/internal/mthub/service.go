package mthub

import (
	"context"
	"time"

	"github.com/shopspring/decimal"
)

// MtHubService is the business-layer facade for order operations.
// All MT account interactions go through this service.
type MtHubService struct {
	hub    *Hub
	broker *OrderEventBroker
}

// NewMtHubService creates the service with a Hub and event broker.
func NewMtHubService(hub *Hub, broker *OrderEventBroker) *MtHubService {
	return &MtHubService{hub: hub, broker: broker}
}

// PlaceOrder places an order on the account's broker via the registered executor.
func (s *MtHubService) PlaceOrder(ctx context.Context, req *OrderRequest) (*OrderRecord, error) {
	exec := s.hub.Get(req.AccountID)
	if exec == nil { return nil, ErrSessionNotFound }
	ticket, err := exec.PlaceOrder(ctx, req)
	if err != nil { return nil, err }
	return &OrderRecord{Ticket: ticket, AccountID: req.AccountID, State: OrderStatePending}, nil
}

// CloseOrder closes an existing position.
func (s *MtHubService) CloseOrder(ctx context.Context, accountID string, ticket int64, lots decimal.Decimal) error {
	exec := s.hub.Get(accountID)
	if exec == nil { return ErrSessionNotFound }
	return exec.CloseOrder(ctx, ticket, lots)
}

// OpenedOrders returns currently open positions.
func (s *MtHubService) OpenedOrders(ctx context.Context, accountID string) ([]*OrderRecord, error) {
	exec := s.hub.Get(accountID)
	if exec == nil { return nil, ErrSessionNotFound }
	return exec.FetchOpenedOrders(ctx)
}

// OrderHistory returns historical orders for the account.
func (s *MtHubService) OrderHistory(ctx context.Context, accountID string, from, to time.Time) ([]*OrderRecord, error) {
	exec := s.hub.Get(accountID)
	if exec == nil { return nil, ErrSessionNotFound }
	return exec.FetchOrderHistory(ctx, from, to)
}

// SymbolParams returns trading parameters for the given symbols.
func (s *MtHubService) SymbolParams(ctx context.Context, accountID string, canonicals []string) ([]*SymbolParam, error) {
	exec := s.hub.Get(accountID)
	if exec == nil { return nil, ErrSessionNotFound }
	return exec.FetchSymbolParams(ctx, canonicals)
}

// SubscribeUserOrderEvents subscribes to all order events for a user.
func (s *MtHubService) SubscribeUserOrderEvents(ctx context.Context, userID string) (<-chan *OrderEvent, func()) {
	return s.broker.Subscribe(userID)
}

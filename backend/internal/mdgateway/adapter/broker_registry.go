// Package adapter provides broker abstraction and registry for multi-broker order routing.
// Implements ADR-0012: all broker implementations register via BrokerRegistry and are
// accessed through the BrokerAdapter interface defined in oms/broker.go.
package adapter

import (
	"context"
	"fmt"
	"strconv"
	"sync"

	"anttrader/internal/mdgateway/adapter/mt4"
	"anttrader/internal/mdgateway/adapter/mt5"
	"anttrader/internal/mthub"
	"anttrader/internal/oms"

	"github.com/shopspring/decimal"
)

// ── BrokerAdapter wrappers ──

// mt4Adapter wraps mt4.Gateway to implement oms.BrokerAdapter.
type mt4Adapter struct{ gw *mt4.Gateway }

func (a *mt4Adapter) Submit(ctx context.Context, req *oms.OrderRequest) (*oms.BrokerResp, error) {
	side := mthub.SideBuy
	if req.Side == "sell" {
		side = mthub.SideSell
	}
	mreq := &mthub.OrderRequest{
		Canonical: req.Symbol,
		AccountID: req.AccountID,
		Side:      side,
		OrderType: mthub.OrderMarket,
		Volume:    decimal.NewFromFloat(req.Volume),
		Price:     decimal.NewFromFloat(req.Price),
		StopLoss:  decimal.NewFromFloat(req.StopLoss),
		TakeProfit: decimal.NewFromFloat(req.TakeProfit),
		Comment:   req.Comment,
	}
	ticket, err := a.gw.PlaceOrder(ctx, mreq)
	if err != nil {
		return &oms.BrokerResp{ErrorMsg: err.Error(), ErrorCode: -1}, err
	}
	return &oms.BrokerResp{Ticket: strconv.FormatInt(ticket, 10), State: oms.StateSubmitted}, nil
}

func (a *mt4Adapter) Cancel(ctx context.Context, ticket string) error {
	tid, err := strconv.ParseInt(ticket, 10, 64)
	if err != nil {
		return fmt.Errorf("mt4 cancel: invalid ticket %q: %w", ticket, err)
	}
	return a.gw.CloseOrder(ctx, tid, decimal.Zero)
}

func (a *mt4Adapter) Modify(ctx context.Context, ticket string, price, stopPrice float64) error {
	tid, err := strconv.ParseInt(ticket, 10, 64)
	if err != nil {
		return fmt.Errorf("mt4 modify: invalid ticket %q: %w", ticket, err)
	}
	return a.gw.ModifyOrder(ctx, tid, decimal.NewFromFloat(stopPrice), decimal.Zero, decimal.NewFromFloat(price))
}

func (a *mt4Adapter) Query(ctx context.Context, ticket string) (*oms.Order, error) {
	orders, err := a.gw.FetchOpenedOrders(ctx)
	if err != nil {
		return nil, err
	}
	for _, o := range orders {
		if strconv.FormatInt(o.Ticket, 10) == ticket {
			return &oms.Order{
				Ticket:  ticket,
				Symbol:  o.SymbolRaw,
				Volume:  o.Volume.InexactFloat64(),
				Price:   o.OpenPrice.InexactFloat64(),
				State:   oms.StateWorking,
			}, nil
		}
	}
	return nil, fmt.Errorf("mt4 query: ticket %s not found", ticket)
}

// mt5Adapter wraps mt5.Gateway to implement oms.BrokerAdapter.
type mt5Adapter struct{ gw *mt5.Gateway }

func (a *mt5Adapter) Submit(ctx context.Context, req *oms.OrderRequest) (*oms.BrokerResp, error) {
	side := mthub.SideBuy
	if req.Side == "sell" {
		side = mthub.SideSell
	}
	mreq := &mthub.OrderRequest{
		Canonical: req.Symbol,
		AccountID: req.AccountID,
		Side:      side,
		OrderType: mthub.OrderMarket,
		Volume:    decimal.NewFromFloat(req.Volume),
		Price:     decimal.NewFromFloat(req.Price),
		StopLoss:  decimal.NewFromFloat(req.StopLoss),
		TakeProfit: decimal.NewFromFloat(req.TakeProfit),
		Comment:   req.Comment,
	}
	ticket, err := a.gw.PlaceOrder(ctx, mreq)
	if err != nil {
		return &oms.BrokerResp{ErrorMsg: err.Error(), ErrorCode: -1}, err
	}
	return &oms.BrokerResp{Ticket: strconv.FormatInt(ticket, 10), State: oms.StateSubmitted}, nil
}

func (a *mt5Adapter) Cancel(ctx context.Context, ticket string) error {
	tid, err := strconv.ParseInt(ticket, 10, 64)
	if err != nil {
		return fmt.Errorf("mt5 cancel: invalid ticket %q: %w", ticket, err)
	}
	return a.gw.CloseOrder(ctx, tid, decimal.Zero)
}

func (a *mt5Adapter) Modify(ctx context.Context, ticket string, price, stopPrice float64) error {
	tid, err := strconv.ParseInt(ticket, 10, 64)
	if err != nil {
		return fmt.Errorf("mt5 modify: invalid ticket %q: %w", ticket, err)
	}
	return a.gw.ModifyOrder(ctx, tid, decimal.NewFromFloat(stopPrice), decimal.Zero, decimal.NewFromFloat(price))
}

func (a *mt5Adapter) Query(ctx context.Context, ticket string) (*oms.Order, error) {
	orders, err := a.gw.FetchOpenedOrders(ctx)
	if err != nil {
		return nil, err
	}
	for _, o := range orders {
		if strconv.FormatInt(o.Ticket, 10) == ticket {
			return &oms.Order{
				Ticket:  ticket,
				Symbol:  o.SymbolRaw,
				Volume:  o.Volume.InexactFloat64(),
				Price:   o.OpenPrice.InexactFloat64(),
				State:   oms.StateWorking,
			}, nil
		}
	}
	return nil, fmt.Errorf("mt5 query: ticket %s not found", ticket)
}

// ── brokerExecutor adapts oms.BrokerAdapter.Submit to mthub.BrokerExecutor ──

// brokerExec wraps an oms.BrokerAdapter plus a platform string to implement
// mthub.BrokerExecutor. It translates the narrow SubmitOrder call into the
// full BrokerAdapter.Submit call.
type brokerExec struct {
	adapter  oms.BrokerAdapter
	platform string
}

func (b *brokerExec) SubmitOrder(ctx context.Context, req *mthub.OrderRequest) (int64, error) {
	side := "buy"
	if req.Side == mthub.SideSell {
		side = "sell"
	}
	omsReq := &oms.OrderRequest{
		AccountID:  req.AccountID,
		Symbol:     req.Canonical,
		Side:       side,
		Volume:     req.Volume.InexactFloat64(),
		Price:      req.Price.InexactFloat64(),
		StopLoss:   req.StopLoss.InexactFloat64(),
		TakeProfit: req.TakeProfit.InexactFloat64(),
		StrategyID: "",
		Comment:    req.Comment,
	}
	resp, err := b.adapter.Submit(ctx, omsReq)
	if err != nil {
		return 0, err
	}
	return strconv.ParseInt(resp.Ticket, 10, 64)
}

func (b *brokerExec) Platform() string { return b.platform }

// ── BrokerRegistry ──

// BrokerRegistry maps broker names to BrokerExecutor instances.
// Implements mthub.BrokerRegistry for multi-broker order routing (M12-C2).
// All broker implementations register at startup via RegisterDefaults.
type BrokerRegistry struct {
	mu       sync.RWMutex
	adapters map[string]mthub.BrokerExecutor
}

// NewBrokerRegistry creates an empty registry. Call Register() for each broker.
func NewBrokerRegistry() *BrokerRegistry {
	return &BrokerRegistry{adapters: make(map[string]mthub.BrokerExecutor)}
}

// Register adds a broker adapter under the given name (e.g. "mt4", "mt5").
// The adapter must implement oms.BrokerAdapter; it is wrapped as BrokerExecutor.
// Returns an error if the name is already registered.
func (r *BrokerRegistry) Register(name string, platform string, adapter oms.BrokerAdapter) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, exists := r.adapters[name]; exists {
		return fmt.Errorf("broker registry: %q already registered", name)
	}
	r.adapters[name] = &brokerExec{adapter: adapter, platform: platform}
	return nil
}

// Resolve returns the BrokerExecutor for the given broker name.
// Implements mthub.BrokerRegistry.
func (r *BrokerRegistry) Resolve(name string) (mthub.BrokerExecutor, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	a, ok := r.adapters[name]
	if !ok {
		return nil, fmt.Errorf("broker registry: %q not found", name)
	}
	return a, nil
}

// List returns all registered broker names.
// Implements mthub.BrokerRegistry.
func (r *BrokerRegistry) List() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	names := make([]string, 0, len(r.adapters))
	for name := range r.adapters {
		names = append(names, name)
	}
	return names
}

// Compile-time check: adapter.BrokerRegistry satisfies mthub.BrokerRegistry.
var _ mthub.BrokerRegistry = (*BrokerRegistry)(nil)

// RegisterDefaults registers MT4 and MT5 adapters in the registry.
// gateways should already be connected before registration.
func RegisterDefaults(reg *BrokerRegistry, mt4gw *mt4.Gateway, mt5gw *mt5.Gateway) error {
	if mt4gw != nil {
		if err := reg.Register("mt4", "mt4", &mt4Adapter{gw: mt4gw}); err != nil {
			return err
		}
	}
	if mt5gw != nil {
		if err := reg.Register("mt5", "mt5", &mt5Adapter{gw: mt5gw}); err != nil {
			return err
		}
	}
	return nil
}

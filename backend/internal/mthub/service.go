// V1-LEGACY: will be replaced by M7.1-7.4 cards. Do not extend; new code goes alongside.
package mthub

import (
	"context"
	"fmt"

	"go.uber.org/zap"
)

// OrderExecutor describes the minimal interface for placing, closing,
// and querying orders through an MT adapter. Wire a real implementation
// (e.g. mtapi) or a stub in tests.
type OrderExecutor interface {
	PlaceOrder(ctx context.Context, conn interface{}, platform, sessionID string, req *OrderRequest) (int64, error)
	CloseOrder(ctx context.Context, conn interface{}, platform, sessionID string, ticket int64, lots float64) error
	FetchOrderHistory(ctx context.Context, conn interface{}, platform, sessionID, from, to string) ([]*HistoryOrderInfo, error)
	FetchOpenedOrders(ctx context.Context, conn interface{}, platform, sessionID string) ([]*HistoryOrderInfo, error)
	FetchSymbolParamsMany(ctx context.Context, conn interface{}, platform, sessionID string, symbols []string) ([]*SymbolParam, error)
	FetchPriceHistoryToday(ctx context.Context, conn interface{}, platform, sessionID, symbol string) ([]*PriceBar, error)
}

// MtHubService provides session-scoped MT operations.
type MtHubService struct {
	hub     *Hub
	events  *OrderEventBroker
	exec    OrderExecutor // pluggable order executor; nil means stubbed
	log     *zap.Logger
}

// NewMtHubService creates a new MtHubService.
func NewMtHubService(hub *Hub, events *OrderEventBroker, exec OrderExecutor, log *zap.Logger) *MtHubService {
	return &MtHubService{hub: hub, events: events, exec: exec, log: log}
}

// ── Session management ──

// EnsureSession finds or creates a session for the given account.
func (s *MtHubService) EnsureSession(ctx context.Context, accountID string) (*EnsureSessionResult, error) {
	ses, err := s.hub.EnsureSession(accountID, "")
	if err != nil {
		return nil, err
	}
	if ses == nil {
		return nil, fmt.Errorf("mthub: session not found for account %s", accountID)
	}
	return &EnsureSessionResult{
		SessionID:     ses.SessionID(),
		AlreadyActive: ses.SessionID() != "",
	}, nil
}

// CloseSession removes a session from the registry.
func (s *MtHubService) CloseSession(ctx context.Context, accountID string) error {
	s.events.Unsubscribe(accountID)
	s.hub.CloseSession(accountID)
	return nil
}

// ── OMS ──

// OrderSend places a new order through the session's gateway.
func (s *MtHubService) OrderSend(ctx context.Context, accountID string, req *OrderRequest) (*OrderSendResult, error) {
	ses, err := s.hub.EnsureSession(accountID, "")
	if err != nil {
		return nil, fmt.Errorf("mthub: ensure session: %w", err)
	}
	if ses == nil {
		return nil, fmt.Errorf("mthub: no session for account %s", accountID)
	}
	conn := ses.Conn()
	if conn == nil {
		return nil, fmt.Errorf("mthub: account %s not connected", accountID)
	}
	if s.exec == nil {
		return nil, fmt.Errorf("mthub: no order executor configured")
	}

	ticket, err := s.exec.PlaceOrder(ctx, conn, ses.Platform(), ses.SessionID(), req)
	if err != nil {
		if s.log != nil {
			s.log.Warn("ordersend failed",
				zap.String("account_id", accountID),
				zap.String("symbol", req.Symbol),
				zap.Error(err),
			)
		}
		return &OrderSendResult{Error: err.Error()}, nil
	}

	if s.log != nil {
		s.log.Info("order placed",
			zap.String("account_id", accountID),
			zap.String("symbol", req.Symbol),
			zap.Int64("ticket", ticket),
		)
	}
	return &OrderSendResult{Ticket: ticket}, nil
}

// OrderClose closes an existing order.
func (s *MtHubService) OrderClose(ctx context.Context, accountID string, req *CloseRequest) (*CloseResult, error) {
	ses, err := s.hub.EnsureSession(accountID, "")
	if err != nil {
		return nil, fmt.Errorf("mthub: ensure session: %w", err)
	}
	if ses == nil {
		return nil, fmt.Errorf("mthub: no session for account %s", accountID)
	}
	conn := ses.Conn()
	if conn == nil {
		return nil, fmt.Errorf("mthub: not connected")
	}
	if s.exec == nil {
		return nil, fmt.Errorf("mthub: no order executor configured")
	}
	if err := s.exec.CloseOrder(ctx, conn, ses.Platform(), ses.SessionID(), req.Ticket, req.Lots); err != nil {
		if s.log != nil {
			s.log.Warn("orderclose failed", zap.Int64("ticket", req.Ticket), zap.Error(err))
		}
		return &CloseResult{Error: err.Error()}, nil
	}
	if s.log != nil {
		s.log.Info("order closed", zap.Int64("ticket", req.Ticket))
	}
	return &CloseResult{}, nil
}

// ── History ──

// OrderHistory fetches closed orders in the given time window.
func (s *MtHubService) OrderHistory(ctx context.Context, accountID string, req *HistoryRequest) ([]*OrderRecord, error) {
	ses, err := s.hub.EnsureSession(accountID, "")
	if err != nil {
		return nil, fmt.Errorf("mthub: ensure session: %w", err)
	}
	if ses == nil {
		return nil, fmt.Errorf("mthub: no session for account %s", accountID)
	}
	conn := ses.Conn()
	if conn == nil {
		return nil, fmt.Errorf("mthub: not connected")
	}
	if s.exec == nil {
		return nil, fmt.Errorf("mthub: no order executor configured")
	}
	orders, err := s.exec.FetchOrderHistory(ctx, conn, ses.Platform(), ses.SessionID(), req.From, req.To)
	if err != nil {
		return nil, err
	}
	out := make([]*OrderRecord, 0, len(orders))
	for _, o := range orders {
		out = append(out, toOrderRecord(o))
	}
	return out, nil
}

// OpenedOrders fetches currently open positions.
func (s *MtHubService) OpenedOrders(ctx context.Context, accountID string) ([]*OrderRecord, error) {
	ses, err := s.hub.EnsureSession(accountID, "")
	if err != nil {
		return nil, fmt.Errorf("mthub: ensure session: %w", err)
	}
	if ses == nil {
		return nil, fmt.Errorf("mthub: no session for account %s", accountID)
	}
	conn := ses.Conn()
	if conn == nil {
		return nil, fmt.Errorf("mthub: not connected")
	}
	if s.exec == nil {
		return nil, fmt.Errorf("mthub: no order executor configured")
	}
	positions, err := s.exec.FetchOpenedOrders(ctx, conn, ses.Platform(), ses.SessionID())
	if err != nil {
		return nil, err
	}
	if s.log != nil {
		s.log.Info("mthub fetched positions", zap.Int("count", len(positions)))
	}
	out := make([]*OrderRecord, 0, len(positions))
	for _, p := range positions {
		cp := p.CurrentPrice
		if cp == 0 || cp == p.OpenPrice {
			// Try raw symbol first, then strip suffix (EURUSDm → EURUSD)
			sym := p.Symbol
			lp := s.hub.LatestPriceForSide(sym, p.Type)
			if lp == 0 {
				if idx := len(sym) - 1; idx > 0 && (sym[idx] == 'm' || sym[idx] == 'x') {
					lp = s.hub.LatestPriceForSide(sym[:idx], p.Type)
				}
			}
			if lp > 0 {
				cp = lp
			}
		}
		out = append(out, &OrderRecord{
			Ticket: p.Ticket, Symbol: p.Symbol, Side: p.Type, Lots: p.Lots,
			OpenPrice: p.OpenPrice, Profit: p.Profit, Swap: p.Swap, Commission: p.Commission,
			OpenTimeMs: p.OpenTimeMs, CurrentPrice: cp,
		})
	}
	return out, nil
}

// ── Symbol + price ──

// SymbolParamsMany fetches contract parameters for the given symbols.
func (s *MtHubService) SymbolParamsMany(ctx context.Context, accountID string, symbols []string) ([]*SymbolParam, error) {
	ses, err := s.hub.EnsureSession(accountID, "")
	if err != nil {
		return nil, fmt.Errorf("mthub: ensure session: %w", err)
	}
	if ses == nil || ses.Conn() == nil {
		return nil, fmt.Errorf("mthub: not connected")
	}
	if s.exec == nil {
		return nil, fmt.Errorf("mthub: no order executor configured")
	}
	params, err := s.exec.FetchSymbolParamsMany(ctx, ses.Conn(), ses.Platform(), ses.SessionID(), symbols)
	if err != nil {
		return nil, err
	}
	out := make([]*SymbolParam, 0, len(params))
	for _, p := range params {
		out = append(out, &SymbolParam{
			Symbol: p.Symbol, Digits: p.Digits, Point: p.Point,
			ContractSize: p.ContractSize, MinLot: p.MinLot, MaxLot: p.MaxLot,
			LotStep: p.LotStep,
		})
	}
	return out, nil
}

// PriceHistory fetches today's price bars for a symbol.
func (s *MtHubService) PriceHistory(ctx context.Context, accountID, symbol string) ([]*PriceBar, error) {
	ses, err := s.hub.EnsureSession(accountID, "")
	if err != nil {
		return nil, fmt.Errorf("mthub: ensure session: %w", err)
	}
	if ses == nil || ses.Conn() == nil {
		return nil, fmt.Errorf("mthub: not connected")
	}
	if s.exec == nil {
		return nil, fmt.Errorf("mthub: no order executor configured")
	}
	bars, err := s.exec.FetchPriceHistoryToday(ctx, ses.Conn(), ses.Platform(), ses.SessionID(), symbol)
	if err != nil {
		return nil, err
	}
	out := make([]*PriceBar, 0, len(bars))
	for _, b := range bars {
		out = append(out, &PriceBar{
			OpenTsMs: b.OpenTsMs, Open: b.Open, High: b.High,
			Low: b.Low, Close: b.Close, Volume: b.Volume,
		})
	}
	return out, nil
}

// ── Stream ──

// SubscribeOrderEvents subscribes to order events for an account.
// The provided send function is called for each event.
// Blocks until ctx is cancelled.
func (s *MtHubService) SubscribeOrderEvents(ctx context.Context, accountID string, send func(*OrderEvent) error) error {
	ch := s.events.Subscribe(accountID)
	defer s.events.Unsubscribe(accountID)

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case ev, ok := <-ch:
			if !ok {
				return nil
			}
			if err := send(ev); err != nil {
				return err
			}
		}
	}
}

// ── helpers ──

func toOrderRecord(o *HistoryOrderInfo) *OrderRecord {
	return &OrderRecord{
		Ticket: o.Ticket, Symbol: o.Symbol, Side: o.Type, Lots: o.Lots,
		OpenPrice: o.OpenPrice, ClosePrice: o.ClosePrice,
		Profit: o.Profit, Swap: o.Swap, Commission: o.Commission,
		OpenTime: o.OpenTime, CloseTime: o.CloseTime,
	}
}

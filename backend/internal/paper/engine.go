// Package paper implements the virtual trading engine for paper/模拟 trading.
//
// PaperEngine maintains in-memory virtual portfolio state, simulates order fills
// at bar Bid/Ask prices, and pushes state updates via SSE subscribers.
//
// Architecture:
//   LiveStrategyRunner signal → PaperEngine.PlacePaperOrder()
//     → fetch symbol params → determine fill price (buy=Ask, sell=Bid)
//     → create paper order → update account balance → push SSE update

package paper

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/shopspring/decimal"
	"go.uber.org/zap"

	"alphaforge/internal/mthub"
	"alphaforge/internal/repository"
	"alphaforge/internal/risk"
)

// paperRepository is the local interface for paper account persistence.
type paperRepository interface {
	CreateOrder(ctx context.Context, o *repository.PaperOrder) error
	GetAccount(ctx context.Context, accountID string) (*repository.PaperAccount, error)
	ListAccounts(ctx context.Context, userID string) ([]*repository.PaperAccount, error)
	UpdateAccountBalance(ctx context.Context, accountID string, balance, equity decimal.Decimal) error
	GetOrder(ctx context.Context, orderID string) (*repository.PaperOrder, error)
	UpdateOrder(ctx context.Context, o *repository.PaperOrder) error
	FindOpenOrder(ctx context.Context, paperAccountID, symbol string) (*repository.PaperOrder, error)
}

// PaperEngine manages virtual paper trading accounts, simulated order fills, and SSE subscribers.
type PaperEngine struct {
	repo  paperRepository
	mtHub *mthub.MtHubService
	log   *zap.Logger
	guard *risk.Guard // mandatory safety net (nil = no-op, for tests)

	// SSE subscribers: paperAccountID → list of subscriber channels
	subscribers map[string][]chan *repository.PaperAccount
	mu          sync.RWMutex
}

// SetGuard injects the mandatory safety net for paper trading.
func (e *PaperEngine) SetGuard(g *risk.Guard) { e.guard = g }

// New creates a PaperEngine.
func New(repo paperRepository, mtHub *mthub.MtHubService, log *zap.Logger) *PaperEngine {
	return &PaperEngine{
		repo:        repo,
		mtHub:       mtHub,
		log:         log,
		subscribers: make(map[string][]chan *repository.PaperAccount),
	}
}

// PlacePaperOrder simulates a market order fill against current bar Bid/Ask prices.
// Called by LiveStrategyRunner when mode="paper".
func (e *PaperEngine) PlacePaperOrder(ctx context.Context, accountID, symbol, side string,
	volume, bid, ask decimal.Decimal) error {

	// Determine simulated fill price: buy → Ask, sell → Bid, fallback to mid.
	fillPrice := bid.Add(ask).Div(decimal.NewFromInt(2))
	if side == "buy" && ask.GreaterThan(decimal.Zero) {
		fillPrice = ask
	} else if side == "sell" && bid.GreaterThan(decimal.Zero) {
		fillPrice = bid
	}

	// Guard: mandatory safety net before any order reaches the broker.
	if e.guard != nil {
		if result := e.guard.Check(ctx, &risk.GuardRequest{
			Symbol: symbol, Side: side,
			Volume: volume, OrderType: "market", Price: fillPrice,
		}); !result.Allowed {
			e.log.Error("PaperEngine: PlacePaperOrder rejected by guard",
				zap.String("accountID", accountID), zap.String("symbol", symbol),
				zap.String("side", side), zap.String("volume", volume.String()),
				zap.String("reason", result.Reason))
			return fmt.Errorf("paper: guard blocked: %s", result.Reason)
		}
	}

	now := time.Now()
	order := &repository.PaperOrder{
		PaperAccountID: accountID,
		Symbol:         symbol,
		Side:           side,
		Volume:         volume,
		FillPrice:      fillPrice,
		SlippageBps:    0, // perfect fill in paper mode
		State:          "open",
		CreatedAt:      now,
	}

	if err := e.repo.CreateOrder(ctx, order); err != nil {
		e.log.Error("PaperEngine: CreateOrder failed",
			zap.String("accountID", accountID), zap.String("symbol", symbol),
			zap.String("side", side), zap.Error(err))
		return err
	}

	// Fetch the specific paper account for balance update.
	target, err := e.repo.GetAccount(ctx, accountID)
	if err != nil || target == nil {
		e.log.Error("PaperEngine: cannot fetch account for balance update",
			zap.String("accountID", accountID), zap.Error(err))
		return fmt.Errorf("paper: fetch account %s: %w", accountID, err)
	}

	marginPct := decimal.NewFromFloat(0.01)
	marginRequired := volume.Mul(fillPrice).Mul(marginPct)
	newBalance := target.CurrentBalance.Sub(marginRequired)

	// Simplified equity: balance (PnL tracking added when position closes).
	newEquity := newBalance

	if err := e.repo.UpdateAccountBalance(ctx, accountID, newBalance, newEquity); err != nil {
		e.log.Error("PaperEngine: balance update failed",
			zap.String("accountID", accountID), zap.String("symbol", symbol),
			zap.Error(err))
	}

	// Push SSE update to subscribers.
	e.broadcast(ctx, accountID)

	e.log.Info("PaperEngine: order executed",
		zap.String("accountID", accountID),
		zap.String("symbol", symbol),
		zap.String("side", side),
		zap.String("volume", volume.String()),
		zap.String("fillPrice", fillPrice.String()),
	)

	return nil
}

// PaperPnl returns the unrealized PnL of the single open paper position for
// the given account+symbol, using the latest bid/ask. Returns 0 if no open order.
func (e *PaperEngine) PaperPnl(ctx context.Context, accountID, symbol string, bid, ask decimal.Decimal) (decimal.Decimal, error) {
	order, err := e.repo.FindOpenOrder(ctx, accountID, symbol)
	if err != nil || order == nil || order.State != "open" {
		return decimal.Zero, nil
	}
	var exit decimal.Decimal
	switch order.Side {
	case "buy":
		exit = bid
	case "sell":
		exit = ask
	default:
		exit = bid.Add(ask).Div(decimal.NewFromInt(2))
	}
	diff := exit.Sub(order.FillPrice)
	if order.Side == "sell" {
		diff = order.FillPrice.Sub(exit)
	}
	return order.Volume.Mul(diff), nil
}

func (e *PaperEngine) ClosePaperOrder(ctx context.Context, accountID, symbol string) error {
	order, err := e.repo.FindOpenOrder(ctx, accountID, symbol)
	if err != nil {
		e.log.Error("PaperEngine: ClosePaperOrder find open order failed",
			zap.String("accountID", accountID), zap.String("symbol", symbol), zap.Error(err))
		return fmt.Errorf("find open order: %w", err)
	}
	if order == nil {
		e.log.Error("PaperEngine: ClosePaperOrder no open position",
			zap.String("accountID", accountID), zap.String("symbol", symbol))
		return fmt.Errorf("no open position for %s on account %s", symbol, accountID)
	}
	now := time.Now()
	order.State = "closed"
	order.ClosedAt = &now
	if err := e.repo.UpdateOrder(ctx, order); err != nil {
		e.log.Error("PaperEngine: ClosePaperOrder update failed",
			zap.String("accountID", accountID), zap.String("symbol", symbol), zap.Error(err))
		return fmt.Errorf("close order: %w", err)
	}
	e.log.Info("PaperEngine: position closed",
		zap.String("accountID", accountID), zap.String("symbol", symbol),
		zap.String("orderID", order.ID))
	e.broadcast(ctx, accountID)
	return nil
}

// ModifyPaperOrder updates SL/TP on an open paper position by symbol.
func (e *PaperEngine) ModifyPaperOrder(ctx context.Context, accountID, symbol string, sl, tp decimal.Decimal) error {
	order, err := e.repo.FindOpenOrder(ctx, accountID, symbol)
	if err != nil {
		e.log.Error("PaperEngine: ModifyPaperOrder find open order failed",
			zap.String("accountID", accountID), zap.String("symbol", symbol), zap.Error(err))
		return fmt.Errorf("find open order: %w", err)
	}
	if order == nil {
		e.log.Error("PaperEngine: ModifyPaperOrder no open position",
			zap.String("accountID", accountID), zap.String("symbol", symbol))
		return fmt.Errorf("no open position for %s on account %s", symbol, accountID)
	}
	order.StopLoss = sl
	order.TakeProfit = tp
	if err := e.repo.UpdateOrder(ctx, order); err != nil {
		e.log.Error("PaperEngine: ModifyPaperOrder update failed",
			zap.String("accountID", accountID), zap.String("symbol", symbol), zap.Error(err))
		return fmt.Errorf("modify order: %w", err)
	}
	e.log.Info("PaperEngine: position modified",
		zap.String("accountID", accountID), zap.String("symbol", symbol),
		zap.String("sl", sl.String()), zap.String("tp", tp.String()))
	return nil
}

// CancelPaperOrder cancels a pending paper order by symbol.
func (e *PaperEngine) CancelPaperOrder(ctx context.Context, accountID, symbol string) error {
	order, err := e.repo.FindOpenOrder(ctx, accountID, symbol)
	if err != nil {
		e.log.Error("PaperEngine: CancelPaperOrder find open order failed",
			zap.String("accountID", accountID), zap.String("symbol", symbol), zap.Error(err))
		return fmt.Errorf("find open order: %w", err)
	}
	if order == nil {
		return nil // nothing to cancel
	}
	order.State = "cancelled"
	if err := e.repo.UpdateOrder(ctx, order); err != nil {
		e.log.Error("PaperEngine: CancelPaperOrder update failed",
			zap.String("accountID", accountID), zap.String("symbol", symbol), zap.Error(err))
		return fmt.Errorf("cancel order: %w", err)
	}
	e.log.Info("PaperEngine: order cancelled",
		zap.String("accountID", accountID), zap.String("symbol", symbol))
	return nil
}

// ── SSE subscriber management ──

// Subscribe returns a channel that receives paper account state updates.
// The caller must call Unsubscribe when done.
func (e *PaperEngine) Subscribe(accountID string) (<-chan *repository.PaperAccount, func()) {
	e.mu.Lock()
	defer e.mu.Unlock()

	ch := make(chan *repository.PaperAccount, 8)
	e.subscribers[accountID] = append(e.subscribers[accountID], ch)

	unsubscribe := func() {
		e.mu.Lock()
		defer e.mu.Unlock()
		subs := e.subscribers[accountID]
		for i, s := range subs {
			if s == ch {
				e.subscribers[accountID] = append(subs[:i], subs[i+1:]...)
				close(ch)
				return
			}
		}
	}
	return ch, unsubscribe
}

// broadcast sends the current account state to all subscribers.
func (e *PaperEngine) broadcast(ctx context.Context, accountID string) {
	e.mu.RLock()
	subs := e.subscribers[accountID]
	e.mu.RUnlock()

	if len(subs) == 0 {
		return
	}

	// Fetch fresh account state.
	accounts, err := e.repo.ListAccounts(ctx, "")
	if err != nil {
		return
	}
	var target *repository.PaperAccount
	for _, a := range accounts {
		if a.ID == accountID {
			target = a
			break
		}
	}
	if target == nil {
		return
	}

	for _, ch := range subs {
		select {
		case ch <- target:
		default:
			// drop slow consumer
		}
	}
}

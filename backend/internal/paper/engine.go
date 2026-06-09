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
	"sync"
	"time"

	"github.com/shopspring/decimal"
	"go.uber.org/zap"

	"anttrader/internal/mthub"
	"anttrader/internal/repository"
)

// paperRepository is the local interface for paper account persistence.
// Defined on the consumer side per Go convention — repository package need not know about it.
type paperRepository interface {
	CreateOrder(ctx context.Context, o *repository.PaperOrder) error
	ListAccounts(ctx context.Context, userID string) ([]*repository.PaperAccount, error)
	UpdateAccountBalance(ctx context.Context, accountID string, balance, equity decimal.Decimal) error
}

// PaperEngine manages virtual paper trading accounts, simulated order fills, and SSE subscribers.
type PaperEngine struct {
	repo  paperRepository
	mtHub *mthub.MtHubService
	log   *zap.Logger

	// SSE subscribers: paperAccountID → list of subscriber channels
	subscribers map[string][]chan *repository.PaperAccount
	mu          sync.RWMutex
}

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
	volume decimal.Decimal, bid, ask float64) error {

	// Determine simulated fill price: buy → Ask, sell → Bid, fallback to mid.
	fillPrice := decimal.NewFromFloat((bid + ask) / 2.0)
	if side == "buy" && ask > 0 {
		fillPrice = decimal.NewFromFloat(ask)
	} else if side == "sell" && bid > 0 {
		fillPrice = decimal.NewFromFloat(bid)
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
		return err
	}

	// Update account balance: reduce by margin (simplified: volume * fillPrice * 1%)
	acct, err := e.repo.ListAccounts(ctx, "") // TODO: filter by accountID when repo supports it
	if err != nil || len(acct) == 0 {
		e.log.Warn("PaperEngine: cannot fetch account for balance update",
			zap.String("accountID", accountID))
		return nil
	}

	// Find the specific account.
	var target *repository.PaperAccount
	for _, a := range acct {
		if a.ID == accountID {
			target = a
			break
		}
	}
	if target == nil {
		return nil
	}

	marginPct := decimal.NewFromFloat(0.01)
	marginRequired := volume.Mul(fillPrice).Mul(marginPct)
	newBalance := target.CurrentBalance.Sub(marginRequired)

	// Simplified equity: balance (PnL tracking added when position closes).
	newEquity := newBalance

	if err := e.repo.UpdateAccountBalance(ctx, accountID, newBalance, newEquity); err != nil {
		e.log.Warn("PaperEngine: balance update failed", zap.Error(err))
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

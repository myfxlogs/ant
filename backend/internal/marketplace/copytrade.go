// Package marketplace — CopyTrade engine (M12-B2).
//
// The CopyTradeEngine listens for strategy signal events from publishers and
// replicates orders to all active subscribers with pro-rata volume allocation.
// Each subscriber order goes through the full mthub risk pipeline (kill switch,
// ownership verification, rate limiting, OMS state machine, pre-trade cost
// estimation) ensuring the same safety guarantees as manual orders.
package marketplace

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/shopspring/decimal"
	"go.uber.org/zap"

	"anttrader/internal/mthub"
	"anttrader/internal/risksvc"
)

// CopyTradeEngine replicates publisher strategy signals to subscriber accounts.
// It is designed for concurrent use — multiple signals can be processed
// simultaneously, each in its own goroutine.
type CopyTradeEngine struct {
	marketplace *Service
	mthub       OrderPlacer
	allocator   risksvc.BlockAllocator
	accountInfo AccountInfoProvider
	log         *zap.Logger

	// Concurrency control: max concurrent copy operations.
	sem chan struct{}
}

// OrderPlacer abstracts the order submission capability needed by the engine.
// MtHubService satisfies this interface directly.
type OrderPlacer interface {
	PlaceOrder(ctx context.Context, req *mthub.OrderRequest) (*mthub.OrderRecord, error)
}

// AccountInfoProvider fetches account state for volume allocation.
type AccountInfoProvider interface {
	GetAccountInfo(ctx context.Context, accountID string) (*AccountInfo, error)
}

// AccountInfo holds the data needed for copy-trade allocation decisions.
type AccountInfo struct {
	Equity     float64
	FreeMargin float64
	Balance    float64
	Status     string // "connected", "disconnected", etc.
}

// CopySignalEvent is the input to the copy-trade engine.
// It carries the publisher's signal that should be replicated.
type CopySignalEvent struct {
	// StrategyID identifies the published marketplace strategy.
	StrategyID string
	// PublisherUserID identifies the strategy owner.
	PublisherUserID string
	// Symbol is the canonical trading instrument (e.g. "EURUSD").
	Symbol string
	// Side is "buy" or "sell".
	Side string
	// Volume is the publisher's original order volume (lots).
	Volume float64
	// Price is the execution price (0 for market orders).
	Price float64
	// StopLoss and TakeProfit are optional protective levels.
	StopLoss   float64
	TakeProfit float64
	// Comment is appended to each subscriber's order.
	Comment string
	// SignalID is a unique identifier for idempotency.
	SignalID string
}

// CopyTradeResult summarizes the outcome of a copy-trade operation.
type CopyTradeResult struct {
	SignalID        string
	StrategyID      string
	SubscriberCount int
	SuccessCount    int
	FailureCount    int
	SkippedCount    int // accounts not connected or with insufficient margin
	Errors          []CopyTradeError
	Duration        time.Duration
}

// CopyTradeError records a per-subscriber failure.
type CopyTradeError struct {
	SubscriberID string
	AccountID    string
	Error        string
}

// NewCopyTradeEngine creates a copy-trade engine with default
// ProRataAllocator and max 8 concurrent copy operations.
func NewCopyTradeEngine(marketplace *Service, mthub OrderPlacer, accountInfo AccountInfoProvider, log *zap.Logger) *CopyTradeEngine {
	return &CopyTradeEngine{
		marketplace: marketplace,
		mthub:       mthub,
		allocator:   &risksvc.ProRataAllocator{},
		accountInfo: accountInfo,
		log:         log,
		sem:         make(chan struct{}, 8),
	}
}

// Process handles a publisher's signal event by replicating the order
// to all active subscribers. It runs asynchronously — the caller receives
// the result via the returned channel.
func (e *CopyTradeEngine) Process(ctx context.Context, signal CopySignalEvent) <-chan *CopyTradeResult {
	resultCh := make(chan *CopyTradeResult, 1)

	go func() {
		defer close(resultCh)
		started := time.Now()

		// Acquire concurrency slot.
		select {
		case e.sem <- struct{}{}:
			defer func() { <-e.sem }()
		case <-ctx.Done():
			resultCh <- &CopyTradeResult{
				SignalID: signal.SignalID, StrategyID: signal.StrategyID,
				Duration: time.Since(started),
				Errors:   []CopyTradeError{{Error: "context cancelled before processing"}},
			}
			return
		}

		result := e.processSync(ctx, signal, started)
		resultCh <- result
	}()

	return resultCh
}

// processSync is the synchronous implementation. It runs inside the goroutine
// spawned by Process, after acquiring the concurrency slot.
func (e *CopyTradeEngine) processSync(ctx context.Context, signal CopySignalEvent, started time.Time) *CopyTradeResult {
	result := &CopyTradeResult{
		SignalID:   signal.SignalID,
		StrategyID: signal.StrategyID,
	}

	// 1. Look up all active subscriptions for this strategy.
	subs, err := e.marketplace.ListSubscriptions(ctx, signal.PublisherUserID)
	// Filter to only this strategy's subscribers.
	var strategySubs []SubscriptionItem
	for _, sub := range subs {
		if sub.StrategyID == signal.StrategyID && sub.Active {
			strategySubs = append(strategySubs, sub)
		}
	}
	if err != nil {
		result.Errors = append(result.Errors, CopyTradeError{Error: fmt.Sprintf("list subscriptions: %v", err)})
		result.Duration = time.Since(started)
		return result
	}
	result.SubscriberCount = len(strategySubs)
	if len(strategySubs) == 0 {
		result.Duration = time.Since(started)
		return result
	}

	// 2. Fetch account info for all subscribers in parallel.
	type subAccount struct {
		sub     SubscriptionItem
		account *AccountInfo
	}
	accounts := make([]subAccount, 0, len(strategySubs))
	var mu sync.Mutex
	var wg sync.WaitGroup

	for _, sub := range strategySubs {
		wg.Add(1)
		go func(s SubscriptionItem) {
			defer wg.Done()
			info, err := e.accountInfo.GetAccountInfo(ctx, s.TargetUserID)
			if err != nil {
				e.log.Warn("copytrade: account info fetch failed",
					zap.String("account_id", s.TargetUserID),
					zap.Error(err))
			}
			mu.Lock()
			accounts = append(accounts, subAccount{sub: s, account: info})
			mu.Unlock()
		}(sub)
	}
	wg.Wait()

	// 3. Build allocator input: only include connected accounts with positive equity.
	var allocInput []risksvc.AllocAccount
	validAccounts := make([]subAccount, 0)
	for _, a := range accounts {
		if a.account == nil || a.account.Status != "connected" {
			result.SkippedCount++
			continue
		}
		if a.account.Equity <= 0 {
			result.SkippedCount++
			continue
		}
		allocInput = append(allocInput, risksvc.AllocAccount{
			AccountID:  a.sub.TargetUserID,
			Equity:     a.account.Equity,
			FreeMargin: a.account.FreeMargin,
		})
		validAccounts = append(validAccounts, a)
	}

	// 4. Allocate volume across valid subscribers.
	allocation := e.allocator.Allocate(ctx, signal.Volume, allocInput)

	// 5. Submit orders to each subscriber.
	var submitWg sync.WaitGroup
	var resultMu sync.Mutex

	for _, a := range validAccounts {
		vol, ok := allocation[a.sub.TargetUserID]
		if !ok || vol <= 0 {
			resultMu.Lock()
			result.SkippedCount++
			resultMu.Unlock()
			continue
		}

		submitWg.Add(1)
		go func(acc subAccount, volume float64) {
			defer submitWg.Done()

			side := mthub.SideBuy
			if signal.Side == "sell" {
				side = mthub.SideSell
			}

			req := &mthub.OrderRequest{
				AccountID:  acc.sub.TargetUserID,
				Canonical:  signal.Symbol,
				Side:       side,
				OrderType:  mthub.OrderMarket,
				Volume:     decimal.NewFromFloat(volume),
				Price:      decimal.NewFromFloat(signal.Price),
				StopLoss:   decimal.NewFromFloat(signal.StopLoss),
				TakeProfit: decimal.NewFromFloat(signal.TakeProfit),
				Comment:    fmt.Sprintf("copytrade:%s:%s", signal.StrategyID[:8], signal.Comment),
				ClientID:   fmt.Sprintf("copytrade-%s-%s", signal.SignalID, acc.sub.TargetUserID[:8]),
			}

			if signal.Price > 0 {
				req.OrderType = mthub.OrderLimit
			}

			_, err := e.mthub.PlaceOrder(ctx, req)
			resultMu.Lock()
			defer resultMu.Unlock()
			if err != nil {
				result.FailureCount++
				result.Errors = append(result.Errors, CopyTradeError{
					SubscriberID: acc.sub.SubscriptionID,
					AccountID:    acc.sub.TargetUserID,
					Error:        err.Error(),
				})
				e.log.Warn("copytrade: order submit failed",
					zap.String("account_id", acc.sub.TargetUserID),
					zap.String("strategy_id", signal.StrategyID),
					zap.Error(err))
			} else {
				result.SuccessCount++
			}
		}(a, vol)
	}
	submitWg.Wait()

	result.Duration = time.Since(started)
	e.log.Info("copytrade: signal processed",
		zap.String("signal_id", signal.SignalID),
		zap.String("strategy_id", signal.StrategyID),
		zap.Int("subscribers", result.SubscriberCount),
		zap.Int("success", result.SuccessCount),
		zap.Int("failed", result.FailureCount),
		zap.Int("skipped", result.SkippedCount),
		zap.Duration("duration", result.Duration))
	return result
}

// Concurrency returns the current number of in-flight copy operations.
func (e *CopyTradeEngine) Concurrency() int { return len(e.sem) }

// MaxConcurrency returns the maximum concurrent copy operations.
func (e *CopyTradeEngine) MaxConcurrency() int { return cap(e.sem) }

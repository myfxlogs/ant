package risk

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/shopspring/decimal"
)

// Guard is the mandatory pre-broker safety net. Every order — live, paper,
// backtest — MUST pass through Guard.Check before reaching a broker.
// The three checks are irreducible: kill switch, duplicate, max lot size.
// These protect against software bugs, not market judgment.
type Guard struct {
	killFn    func() bool          // nil → no kill switch
	maxLots   decimal.Decimal     // 0 → no limit
	dedup     map[string]time.Time // symbol|side|vol|type|price → last seen
	dedupMu   sync.Mutex
	dedupWin  time.Duration       // window to remember orders (default 5s)
}

// GuardConfig sets up the Guard.
type GuardConfig struct {
	KillSwitch     func() bool
	MaxLotSize     decimal.Decimal
	DedupWindow    time.Duration // 0 → default 5s
}

// NewGuard creates a Guard. Passing nil config is safe (all checks become no-ops).
func NewGuard(cfg *GuardConfig) *Guard {
	g := &Guard{
		maxLots:  decimal.Zero,
		dedup:    make(map[string]time.Time),
		dedupWin: 5 * time.Second,
	}
	if cfg == nil {
		return g
	}
	g.killFn = cfg.KillSwitch
	if cfg.MaxLotSize.IsPositive() {
		g.maxLots = cfg.MaxLotSize
	}
	if cfg.DedupWindow > 0 {
		g.dedupWin = cfg.DedupWindow
	}
	return g
}

// GuardResult is the outcome of Guard.Check.
type GuardResult struct {
	Allowed bool
	Reason  string
}

// Check evaluates all three guards. Returns the first denial reason.
func (g *Guard) Check(ctx context.Context, req *GuardRequest) *GuardResult {
	// 1. Kill switch.
	if g.killFn != nil && g.killFn() {
		return &GuardResult{Allowed: false, Reason: "kill switch active"}
	}

	// 2. Max lot size.
	if g.maxLots.IsPositive() && req.Volume.GreaterThan(g.maxLots) {
		return &GuardResult{Allowed: false, Reason: fmt.Sprintf(
			"volume %s exceeds max lot size %s", req.Volume, g.maxLots,
		)}
	}

	// 3. Duplicate protection: same (symbol|side|volume|type|price) within dedup window.
	g.dedupMu.Lock()
	now := time.Now()
	key := fmt.Sprintf("%s|%s|%s|%s|%s", req.Symbol, req.Side, req.Volume, req.OrderType, req.Price)
	if last, ok := g.dedup[key]; ok && now.Sub(last) < g.dedupWin {
		g.dedupMu.Unlock()
		return &GuardResult{Allowed: false, Reason: "duplicate order within dedup window"}
	}
	g.dedup[key] = now
	// Lazy cleanup: purge old entries when map grows large.
	if len(g.dedup) > 10000 {
		for k, v := range g.dedup {
			if now.Sub(v) > g.dedupWin*2 {
				delete(g.dedup, k)
			}
		}
	}
	g.dedupMu.Unlock()

	return &GuardResult{Allowed: true}
}

// GuardRequest is the input to Guard.Check.
type GuardRequest struct {
	Symbol    string
	Side      string
	Volume    decimal.Decimal
	OrderType string
	Price     decimal.Decimal
}

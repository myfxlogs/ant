package marketplace

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"alphaforge/internal/pglisten"
)

// DecayMonitor listens for trade_record_sync notifications (push-first) and
// triggers decay detection for the affected strategy. It throttles per-strategy
// to at most once per day. On startup it runs a full batch scan.
type DecayMonitor struct {
	svc      *Service
	pgListen *pglisten.Listener
	log      *zap.Logger

	// throttle tracks last scan time per strategy to enforce once-per-day.
	mu       sync.Mutex
	throttle map[string]time.Time
}

// NewDecayMonitor creates a decay monitor that reacts to trade_record_sync.
func NewDecayMonitor(svc *Service, pgListen *pglisten.Listener, log *zap.Logger) *DecayMonitor {
	return &DecayMonitor{
		svc:      svc,
		pgListen: pgListen,
		log:      log,
		throttle: make(map[string]time.Time),
	}
}

// Start runs the startup batch scan and begins listening for trade_record_sync.
// It blocks until ctx is cancelled; caller should run in a goroutine.
func (m *DecayMonitor) Start(ctx context.Context) {
	// Startup: full batch scan of all published strategies with linked accounts.
	m.log.Info("decay monitor: startup batch scan")
	m.batchScan(ctx)

	// Push-first: listen for trade_record_sync notifications.
	notifCh, cancelListen, err := m.pgListen.Listen(ctx, "trade_record_sync")
	if err != nil {
		m.log.Error("decay monitor: LISTEN failed", zap.Error(err))
		return
	}
	defer cancelListen()

	for {
		select {
		case <-ctx.Done():
			return
		case _, ok := <-notifCh:
			if !ok {
				return
			}
			m.handleTradeRecordSync(ctx)
		}
	}
}

// handleTradeRecordSync queries strategies that had recent trades and runs
// decay detection for each, throttled to once per day per strategy.
func (m *DecayMonitor) handleTradeRecordSync(ctx context.Context) {
	// Find published strategies with linked accounts that had trades recently.
	rows, err := m.svc.pg.Query(ctx,
		`SELECT DISTINCT ms.strategy_id::text
		 FROM marketplace_strategies ms
		 JOIN trade_records tr ON tr.account_id = ms.linked_account_id
		 WHERE ms.status = 'published'
		   AND ms.linked_account_id IS NOT NULL
		   AND tr.created_at > now() - INTERVAL '5 minutes'`)
	if err != nil {
		m.log.Warn("decay monitor: query recent trades failed", zap.Error(err))
		return
	}
	defer rows.Close()

	var strategyIDs []string
	for rows.Next() {
		var sid string
		if err := rows.Scan(&sid); err != nil {
			continue
		}
		strategyIDs = append(strategyIDs, sid)
	}
	rows.Close()

	for _, sid := range strategyIDs {
		if !m.shouldScan(sid) {
			continue
		}
		m.scanAndProcess(ctx, sid)
	}
}

// batchScan runs DetectDecayBatch and processes all decaying strategies.
func (m *DecayMonitor) batchScan(ctx context.Context) {
	results, err := m.svc.DetectDecayBatch(ctx)
	if err != nil {
		m.log.Warn("decay monitor: batch scan failed", zap.Error(err))
		return
	}
	for _, r := range results {
		m.processDecayResult(ctx, r)
		m.markScanned(r.StrategyID)
	}
}

// scanAndProcess runs decay detection for a single strategy and processes the result.
func (m *DecayMonitor) scanAndProcess(ctx context.Context, strategyID string) {
	result, err := m.svc.DetectDecay(ctx, strategyID)
	if err != nil {
		m.log.Warn("decay monitor: detect failed",
			zap.String("strategy_id", strategyID), zap.Error(err))
		return
	}
	m.markScanned(strategyID)

	if result.IsDecaying {
		m.processDecayResult(ctx, result)
	} else {
		// Strategy is healthy — clear decay status if it was previously set.
		_ = m.svc.updateDecayStatus(ctx, strategyID, "none")
	}
}

// processDecayResult updates decay status, notifies author and buyers.
// It does NOT auto-create optimization tasks (author must manually initiate).
func (m *DecayMonitor) processDecayResult(ctx context.Context, result *DecayResult) {
	status := "decaying"
	if result.DecayScore >= 3 {
		status = "decayed"
	}

	if err := m.svc.updateDecayStatus(ctx, result.StrategyID, status); err != nil {
		m.log.Warn("decay monitor: update status failed",
			zap.String("strategy_id", result.StrategyID), zap.Error(err))
	}

	m.svc.notifyDecayDetected(ctx, result, status)
}

// shouldScan returns true if the strategy has not been scanned in the last 24h.
func (m *DecayMonitor) shouldScan(strategyID string) bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	last, ok := m.throttle[strategyID]
	if !ok {
		return true
	}
	return time.Since(last) >= 24*time.Hour
}

// markScanned records that a strategy was scanned now.
func (m *DecayMonitor) markScanned(strategyID string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.throttle[strategyID] = time.Now()
}

// updateDecayStatus persists the decay status and last_decay_at timestamp.
func (s *Service) updateDecayStatus(ctx context.Context, strategyID, status string) error {
	sid, err := uuid.Parse(strategyID)
	if err != nil {
		return fmt.Errorf("marketplace: update decay status: invalid strategy_id: %w", err)
	}
	_, err = s.pg.Exec(ctx,
		`UPDATE marketplace_strategies
		 SET decay_status = $2, last_decay_at = now()
		 WHERE strategy_id = $1`,
		sid, status)
	return err
}

// session_registry.go — In-memory registry of active live strategy sessions.
// Tracks running LiveSession instances with metadata for monitoring and control.
package strategy

import (
	"context"
	"sync"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"alphaforge/internal/mdgateway/adapter/mdtick"
	"alphaforge/internal/mthub"
	"alphaforge/internal/repository"
)

// ActiveSession holds metadata about a running live strategy session.
type ActiveSession struct {
	RunID        uuid.UUID
	UserID       uuid.UUID
	AccountID    string
	Symbol       string
	Timeframe    string
	Mode         string
	ScheduleID   uuid.UUID
	MagicNumber  int32 // ARCH-4: deterministic magic for position attribution
	StartedAt    time.Time
	LastSignalAt time.Time
	LastTickAt   time.Time
	SignalCount  int
	ErrorCount   int
	LastError    string
	StderrTail   string
	PnL          string // running PnL for this run, empty when unknown
	circuitOpen  bool
	cancel       context.CancelFunc
	signalSubs   []chan *SignalEvent
	signalSubsMu sync.Mutex
	pnlMu        sync.RWMutex
	log          *zap.Logger      // optional logger for error recording
	registry     *SessionRegistry // back-pointer for watcher notification
	diag         *sessionDiag     // runtime diagnostics (L1 counters + L2 indicators)

	// LIVE-ORDER-REENTRY-1: session-scoped execution barrier.
	// Serializes broker mutations — at most one unconfirmed order in-flight.
	// Restores MT4 EA single-threaded OrderSend semantics.
	barrier *TradeBarrier
}

// SignalEvent is pushed to SSE subscribers when a signal is dispatched.
type SignalEvent struct {
	RunID      uuid.UUID
	AccountID  string
	Symbol     string
	SignalType string
	Volume     string
	Price      string
	StopLoss   string
	TakeProfit string
	Reason     string
	Timestamp  time.Time
}

// SessionRegistry tracks all active live strategy sessions in-memory.
// Thread-safe. Sessions are registered at RunLiveStrategy start and
// deregistered on exit.
type SessionRegistry struct {
	mu       sync.RWMutex
	sessions map[uuid.UUID]*ActiveSession // keyed by RunID
	watchers []chan struct{}              // notified on any session change
	log      *zap.Logger                  // optional logger for session errors
	logRepo  *repository.LogRepository    // optional schedule run log persistence
}

// NewSessionRegistry creates a new empty registry.
func NewSessionRegistry() *SessionRegistry {
	return &SessionRegistry{
		sessions: make(map[uuid.UUID]*ActiveSession),
	}
}

// SetLogger injects the logger for RecordError output.
func (r *SessionRegistry) SetLogger(log *zap.Logger) { r.log = log }

// SetLogRepository injects the repository for best-effort schedule run logging.
func (r *SessionRegistry) SetLogRepository(repo *repository.LogRepository) { r.logRepo = repo }

// SubscribeToMthub registers a cross-account listener on the mthub profit broker
// so running sessions receive per-magic PnL updates in real time.
func (r *SessionRegistry) SubscribeToMthub(mthubSvc *mthub.MtHubService) {
	if mthubSvc == nil {
		return
	}
	ch, cancel := mthubSvc.SubscribeAccountProfitAll()
	go func() {
		defer cancel()
		for ev := range ch {
			positions := make([]mdtick.ProfitPosition, 0, len(ev.Positions))
			for _, p := range ev.Positions {
				positions = append(positions, mdtick.ProfitPosition{
					Symbol: p.Symbol,
					Magic:  p.Magic,
					Profit: p.Profit,
				})
			}
			r.UpdatePnlFromPositions(ev.AccountID, positions)
		}
	}()
}

func (r *SessionRegistry) logger() *zap.Logger {
	if r.log != nil {
		return r.log
	}
	return zap.NewNop()
}

// notifyWatchers sends a non-blocking signal to all watchers.
func (r *SessionRegistry) notifyWatchers() {
	r.mu.RLock()
	for _, w := range r.watchers {
		select {
		case w <- struct{}{}:
		default:
		}
	}
	r.mu.RUnlock()
}

// Watch returns a channel that receives a signal whenever sessions change
// (register, deregister, signal, error). The cancel func unsubscribes.
func (r *SessionRegistry) Watch() (<-chan struct{}, func()) {
	ch := make(chan struct{}, 1)
	r.mu.Lock()
	r.watchers = append(r.watchers, ch)
	r.mu.Unlock()
	cancel := func() {
		r.mu.Lock()
		for i, w := range r.watchers {
			if w == ch {
				r.watchers = append(r.watchers[:i], r.watchers[i+1:]...)
				break
			}
		}
		r.mu.Unlock()
		close(ch)
	}
	return ch, cancel
}

// Register adds a new active session to the registry.
// Register creates a new ActiveSession and adds it to the registry.
// ARCH-4: Multiple sessions per account are allowed — position attribution
// is handled by Magic Numbers, not by session exclusivity.
// Returns the created ActiveSession.
func (r *SessionRegistry) Register(runID uuid.UUID, userID uuid.UUID, accountID, symbol, timeframe, mode string, scheduleID uuid.UUID, cancel context.CancelFunc) *ActiveSession {
	sess := &ActiveSession{
		RunID:       runID,
		UserID:      userID,
		AccountID:   accountID,
		Symbol:      symbol,
		Timeframe:   timeframe,
		Mode:        mode,
		ScheduleID:  scheduleID,
		MagicNumber: strategyMagic(scheduleID),
		StartedAt:   time.Now(),
		cancel:      cancel,
		log:         r.logger(),
		registry:    r,
		diag:        newSessionDiag(),
		barrier:     NewTradeBarrier(r.logger()),
	}
	r.mu.Lock()
	r.sessions[runID] = sess
	r.mu.Unlock()
	r.notifyWatchers()
	return sess
}

// Deregister removes a session from the registry. Returns the removed session.
func (r *SessionRegistry) Deregister(runID uuid.UUID) *ActiveSession {
	r.mu.Lock()
	defer r.mu.Unlock()
	sess, ok := r.sessions[runID]
	if !ok {
		return nil
	}
	delete(r.sessions, runID)
	sess.signalSubsMu.Lock()
	for _, sub := range sess.signalSubs {
		close(sub)
	}
	sess.signalSubs = nil
	sess.signalSubsMu.Unlock()
	// Notify watchers outside the lock to avoid deadlock.
	go r.notifyWatchers()
	return sess
}

// Get returns a session by RunID.
func (r *SessionRegistry) Get(runID uuid.UUID) (*ActiveSession, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	sess, ok := r.sessions[runID]
	return sess, ok
}

// ListByUser returns all active sessions for a user.
func (r *SessionRegistry) ListByUser(userID uuid.UUID) []*ActiveSession {
	r.mu.RLock()
	defer r.mu.RUnlock()
	var out []*ActiveSession
	for _, sess := range r.sessions {
		if sess.UserID == userID {
			out = append(out, sess)
		}
	}
	return out
}

// GetByScheduleID returns the active session for a schedule (if any).
func (r *SessionRegistry) GetByScheduleID(scheduleID uuid.UUID) (*ActiveSession, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	for _, sess := range r.sessions {
		if sess.ScheduleID == scheduleID {
			return sess, true
		}
	}
	return nil, false
}

// UpdatePnlFromPositions recomputes PnL for active sessions from a broker
// profit/position snapshot. It matches by account + symbol + magic number and
// updates each running session's PnL in-place.

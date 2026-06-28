// session_registry.go — In-memory registry of active live strategy sessions.
// Tracks running LiveSession instances with metadata for monitoring and control.
package strategy

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
)

// ActiveSession holds metadata about a running live strategy session.
type ActiveSession struct {
	RunID         uuid.UUID
	UserID        uuid.UUID
	AccountID     string
	Symbol        string
	Timeframe     string
	Mode          string
	StartedAt     time.Time
	LastSignalAt  time.Time
	SignalCount   int
	ErrorCount    int
	LastError     string
	StderrTail    string
	cancel        context.CancelFunc
	signalSubs    []chan *SignalEvent
	signalSubsMu  sync.Mutex
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
}

// NewSessionRegistry creates a new empty registry.
func NewSessionRegistry() *SessionRegistry {
	return &SessionRegistry{
		sessions: make(map[uuid.UUID]*ActiveSession),
	}
}

// Register adds a new active session to the registry.
// Returns the created ActiveSession.
func (r *SessionRegistry) Register(runID uuid.UUID, userID uuid.UUID, accountID, symbol, timeframe, mode string, cancel context.CancelFunc) *ActiveSession {
	sess := &ActiveSession{
		RunID:     runID,
		UserID:    userID,
		AccountID: accountID,
		Symbol:    symbol,
		Timeframe: timeframe,
		Mode:      mode,
		StartedAt: time.Now(),
		cancel:    cancel,
	}
	r.mu.Lock()
	r.sessions[runID] = sess
	r.mu.Unlock()
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

// ListByAccount returns all active sessions for an account.
func (r *SessionRegistry) ListByAccount(accountID string) []*ActiveSession {
	r.mu.RLock()
	defer r.mu.RUnlock()
	var out []*ActiveSession
	for _, sess := range r.sessions {
		if sess.AccountID == accountID {
			out = append(out, sess)
		}
	}
	return out
}

// ListAll returns all active sessions.
func (r *SessionRegistry) ListAll() []*ActiveSession {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]*ActiveSession, 0, len(r.sessions))
	for _, sess := range r.sessions {
		out = append(out, sess)
	}
	return out
}

// Stop cancels a session's context, causing RunLiveStrategy to exit.
func (r *SessionRegistry) Stop(runID uuid.UUID) error {
	r.mu.RLock()
	sess, ok := r.sessions[runID]
	r.mu.RUnlock()
	if !ok {
		return fmt.Errorf("session %s not found", runID)
	}
	sess.cancel()
	return nil
}

// RecordSignal updates session metadata when a signal is dispatched and
// publishes the signal event to all SSE subscribers.
func (s *ActiveSession) RecordSignal(event *SignalEvent) {
	s.signalSubsMu.Lock()
	s.SignalCount++
	s.LastSignalAt = event.Timestamp
	for _, sub := range s.signalSubs {
		select {
		case sub <- event:
		default:
		}
	}
	s.signalSubsMu.Unlock()
}

// SubscribeSignals returns a channel that receives signal events for this session.
// The channel is closed when the session is deregistered.
func (s *ActiveSession) SubscribeSignals() <-chan *SignalEvent {
	ch := make(chan *SignalEvent, 16)
	s.signalSubsMu.Lock()
	s.signalSubs = append(s.signalSubs, ch)
	s.signalSubsMu.Unlock()
	return ch
}

// RecordError increments the error count and stores the last error.
func (s *ActiveSession) RecordError(err string) {
	s.signalSubsMu.Lock()
	s.ErrorCount++
	s.LastError = err
	s.signalSubsMu.Unlock()
}

// SetStderrTail updates the captured stderr tail from the WASM session.
func (s *ActiveSession) SetStderrTail(tail string) {
	s.signalSubsMu.Lock()
	s.StderrTail = tail
	s.signalSubsMu.Unlock()
}

// session_registry_active.go — ActiveSession methods extracted from session_registry.go.
package strategy

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"go.uber.org/zap"
)

func (s *ActiveSession) RecordTick(t time.Time) {
	s.pnlMu.Lock()
	s.LastTickAt = t
	s.pnlMu.Unlock()
	if s.registry != nil {
		s.registry.notifyWatchers()
	}
}

// SetPnL updates the running PnL for this run.
func (s *ActiveSession) SetPnL(pnl string) {
	s.pnlMu.Lock()
	s.PnL = pnl
	s.pnlMu.Unlock()
	if s.registry != nil {
		s.registry.notifyWatchers()
	}
}

// RecordSignal updates session metadata when a signal is dispatched and
// publishes the signal event to all SSE subscribers.
// Also persists "signal_generated" to sessionDiag so diagnostics reflect
// the most recent signal even before any order submission attempt.
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
	// LIVE-DIAG-TRUTH-1: persist signal_generated lifecycle (rule 1: signal ≠ fill)
	if s.diag != nil {
		s.diag.RecordLifecycle("signal_generated", 0)
	}
	if s.registry != nil {
		s.registry.notifyWatchers()
	}
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

// InsertScheduleRunLog writes a best-effort schedule run log entry.
// It does not block the caller and swallows panics/timeout internally.
func (r *SessionRegistry) InsertScheduleRunLog(ctx context.Context, userID, scheduleID uuid.UUID, kind, action, status, errorMessage, signalType string, signalVolume decimal.Decimal) {
	if r.logRepo == nil {
		return
	}
	go func() {
		defer func() { _ = recover() }()
		ctx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
		defer cancel()
		_ = r.logRepo.InsertScheduleRunLog(ctx, userID, scheduleID, kind, action, status, errorMessage, signalType, signalVolume)
	}()
}

// RecordError increments the error count, stores the last error, and logs it.
func (s *ActiveSession) RecordError(err string) {
	s.signalSubsMu.Lock()
	s.ErrorCount++
	s.LastError = err
	s.signalSubsMu.Unlock()
	if s.log != nil {
		s.log.Error("ActiveSession: recorded error",
			zap.String("run", s.RunID.String()),
			zap.String("schedule", s.ScheduleID.String()),
			zap.String("error", err))
	}
	if s.registry != nil {
		s.registry.notifyWatchers()
		s.registry.InsertScheduleRunLog(context.Background(), s.UserID, s.ScheduleID,
			"error", "record", "failed", err, "", decimal.Zero)
	}
}

// SetCircuitOpen marks the session as having a tripped circuit breaker.
// When true, new order signals are suppressed to avoid flooding a broken broker.
func (s *ActiveSession) SetCircuitOpen(open bool) {
	s.signalSubsMu.Lock()
	s.circuitOpen = open
	s.signalSubsMu.Unlock()
}

// IsCircuitOpen returns whether the broker circuit breaker is currently open.
func (s *ActiveSession) IsCircuitOpen() bool {
	s.signalSubsMu.Lock()
	defer s.signalSubsMu.Unlock()
	return s.circuitOpen
}

// SetStderrTail updates the captured stderr tail from the live session.
func (s *ActiveSession) SetStderrTail(tail string) {
	s.signalSubsMu.Lock()
	s.StderrTail = tail
	s.signalSubsMu.Unlock()
}

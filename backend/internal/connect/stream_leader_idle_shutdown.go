package connect

import (
	"context"
	"time"

	"github.com/google/uuid"
)

func (s *StreamService) scheduleIdleShutdown(accountID string, seq uint64) {
	if s == nil {
		return
	}
	// Grace period to behave like an official terminal: keep the session warm briefly
	// to avoid flapping when user navigates.
	const idleGrace = 2 * time.Minute

	if s.goroutineMgr != nil {
		_, _ = s.goroutineMgr.Spawn("idle-shutdown-"+accountID, func(gctx context.Context) error {
			select {
			case <-gctx.Done():
				return gctx.Err()
			case <-time.After(idleGrace):
			}

			as, ok := s.getAccountStream(accountID)
			if !ok || as == nil {
				return nil
			}
			as.mu.RLock()
			stillIdle := len(as.Subscribers) == 0 && as.idleSeq.Load() == seq
			as.mu.RUnlock()
			if stillIdle && s.hasDemand(accountID) {
				stillIdle = false
			}
			if stillIdle && s.connManager != nil {
				uid, err := uuid.Parse(accountID)
				if err == nil && s.connManager.ShouldKeepAlive(uid) {
					stillIdle = false
				}
			}
			if !stillIdle {
				return nil
			}



			s.stopSupervisor(accountID)
			if s.connManager != nil {
				uid, err := uuid.Parse(accountID)
				if err == nil {
					ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
					_ = s.connManager.Disconnect(ctx, uid)
					cancel()
				}
			}
			return nil
		})
		return
	}

	go func() {
		time.Sleep(idleGrace)
		as, ok := s.getAccountStream(accountID)
		if !ok || as == nil {
			return
		}
		as.mu.RLock()
		stillIdle := len(as.Subscribers) == 0 && as.idleSeq.Load() == seq
		as.mu.RUnlock()
		if stillIdle && s.hasDemand(accountID) {
			stillIdle = false
		}
		if stillIdle && s.connManager != nil {
			uid, err := uuid.Parse(accountID)
			if err == nil && s.connManager.ShouldKeepAlive(uid) {
				stillIdle = false
			}
		}
		if !stillIdle {
			return
		}

		s.stopSupervisor(accountID)
		if s.connManager != nil {
			uid, err := uuid.Parse(accountID)
			if err == nil {
				ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
				_ = s.connManager.Disconnect(ctx, uid)
				cancel()
			}
		}
	}()
}

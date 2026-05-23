package connect

import (
	"context"

	"anttrader/internal/coordination"
	"anttrader/pkg/logger"

	"go.uber.org/zap"
)

func (s *StreamService) isLeaderForAccount(accountID string) bool {
	if s == nil || accountID == "" {
		return false
	}
	if s.sessionLeader == nil {
		// single-instance mode
		return true
	}
	s.supervisorsMu.Lock()
	lease := s.leaderLeases[accountID]
	s.supervisorsMu.Unlock()
	return lease != nil
}

func (s *StreamService) hasSupervisor(accountID string) bool {
	s.supervisorsMu.Lock()
	_, ok := s.supervisors[accountID]
	s.supervisorsMu.Unlock()
	return ok
}

func (s *StreamService) ensureSupervisor(accountID string) {
	if s == nil {
		return
	}

	// Multi-instance safety: only the Redis leader for this account may start the supervisor.
	// If sessionLeader is not configured, fall back to single-instance behavior.
	if s.sessionLeader != nil {
		lease, acquired, err := s.sessionLeader.TryAcquire(context.Background(), accountID)
		if err != nil {
			logger.Warn("leader: failed to acquire lease; skip supervisor start",
				zap.String("account_id", accountID),
				zap.Error(err))
			return
		}
		if !acquired {
			return
		}
		// Store lease; released when supervisor stops.
		s.supervisorsMu.Lock()
		s.leaderLeases[accountID] = lease
		s.supervisorsMu.Unlock()
		// If we become leader, stop any redis forwarder to avoid duplicate delivery.
		s.stopForwarder(accountID)

		// If lease is lost (renew fails), stop supervisor to avoid dual-session.
		if s.goroutineMgr != nil {
			_, _ = s.goroutineMgr.Spawn("leader-lease-watch-"+accountID, func(gctx context.Context) error {
				select {
				case <-gctx.Done():
					return gctx.Err()
				case <-lease.Done():
					logger.Warn("leader: lease lost; stopping supervisor",
						zap.String("account_id", accountID))
					s.stopSupervisor(accountID)
					as, ok := s.getAccountStream(accountID)
					if ok && as != nil {
						as.mu.RLock()
						hasSubs := len(as.Subscribers) > 0
						as.mu.RUnlock()
						if hasSubs {
							s.startForwarder(accountID)
							s.publishWakeup(accountID)
						}
					}
					return nil
				}
			})
		} else {
			go func() {
				<-lease.Done()
				logger.Warn("leader: lease lost; stopping supervisor",
					zap.String("account_id", accountID))
				s.stopSupervisor(accountID)
				as, ok := s.getAccountStream(accountID)
				if ok && as != nil {
					as.mu.RLock()
					hasSubs := len(as.Subscribers) > 0
					as.mu.RUnlock()
					if hasSubs {
						s.startForwarder(accountID)
						s.publishWakeup(accountID)
					}
				}
			}()
		}
	}

	s.supervisorsMu.Lock()
	if _, ok := s.supervisors[accountID]; ok {
		s.supervisorsMu.Unlock()
		// If we acquired a lease above but supervisor already exists (race), release lease.
		if s.sessionLeader != nil {
			s.supervisorsMu.Lock()
			lease := s.leaderLeases[accountID]
			delete(s.leaderLeases, accountID)
			s.supervisorsMu.Unlock()
			if lease != nil {
				s.sessionLeader.Release(context.Background(), lease)
			}
		}
		return
	}
	sup, err := newSessionAgent(s, accountID)
	if err != nil {
		s.supervisorsMu.Unlock()
		if s.sessionLeader != nil {
			s.supervisorsMu.Lock()
			lease := s.leaderLeases[accountID]
			delete(s.leaderLeases, accountID)
			s.supervisorsMu.Unlock()
			if lease != nil {
				s.sessionLeader.Release(context.Background(), lease)
			}
		}
		logger.Warn("failed to create account supervisor", zap.String("account_id", accountID), zap.Error(err))
		return
	}
	s.supervisors[accountID] = sup
	s.supervisorsMu.Unlock()

	if s.goroutineMgr != nil {
		_, err = s.goroutineMgr.Spawn("supervisor-main-"+accountID, func(gctx context.Context) error {
			go func() {
				<-gctx.Done()
				sup.Stop()
			}()
			sup.Run()
			return gctx.Err()
		})
		if err == nil {
			return
		}
		logger.Warn("failed to spawn supervisor via goroutine manager; falling back to raw goroutine",
			zap.String("account_id", accountID),
			zap.Error(err))
	}

	go sup.Run()
}

func (s *StreamService) StopAllSupervisors() {
	if s == nil {
		return
	}

	s.supervisorsMu.Lock()
	sups := make([]*SessionAgent, 0, len(s.supervisors))
	for _, sup := range s.supervisors {
		if sup != nil {
			sups = append(sups, sup)
		}
	}
	leases := make([]*coordination.Lease, 0, len(s.leaderLeases))
	for _, l := range s.leaderLeases {
		if l != nil {
			leases = append(leases, l)
		}
	}
	// Reset map so future ensureSupervisor can recreate if needed.
	s.supervisors = make(map[string]*SessionAgent)
	s.leaderLeases = make(map[string]*coordination.Lease)
	s.supervisorsMu.Unlock()

	for _, sup := range sups {
		sup.Stop()
	}
	if s.sessionLeader != nil {
		for _, l := range leases {
			s.sessionLeader.Release(context.Background(), l)
		}
	}
}

func (s *StreamService) stopSupervisor(accountID string) {
	if s == nil {
		return
	}
	var sup *SessionAgent
	var lease *coordination.Lease
	s.supervisorsMu.Lock()
	sup = s.supervisors[accountID]
	delete(s.supervisors, accountID)
	lease = s.leaderLeases[accountID]
	delete(s.leaderLeases, accountID)
	s.supervisorsMu.Unlock()
	if sup != nil {
		sup.Stop()
	}
	if s.sessionLeader != nil && lease != nil {
		s.sessionLeader.Release(context.Background(), lease)
	}
}

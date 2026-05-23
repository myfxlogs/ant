package connect

import (
	"context"
	"errors"
	"fmt"
	"math"
	"strings"

	"github.com/google/uuid"

	"anttrader/internal/model"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func isCanceledErr(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, context.Canceled) {
		return true
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return true
	}
	if st, ok := status.FromError(err); ok {
		return st.Code() == codes.Canceled || st.Code() == codes.DeadlineExceeded
	}
	errStr := strings.ToLower(err.Error())
	return strings.Contains(errStr, "context canceled") || strings.Contains(errStr, "grpc: the client connection is closing")
}

func (s *StreamService) ensureAccountStream(ctx context.Context, account *model.MTAccount) error {
	accountID := account.ID.String()
	s.streamMu.Lock()
	accountStream, exists := s.accountStreams[accountID]
	if !exists {
		accountStreamCtx, accountStreamCancel := context.WithCancel(context.Background())
		accountStream = &AccountStream{
			AccountID:   accountID,
			Subscribers: make(map[string]*StreamSubscriber),
			profitNotify: make(chan struct{}, 1),
			orderNotify: make(chan struct{}, 1),
			streamEnabled: map[string]bool{
				"quote":  false,  // QuoteStream 已禁用
				"order":  true,
				"profit": true,
			},
			Ctx:    accountStreamCtx,
			Cancel: accountStreamCancel,
		}
		s.accountStreams[accountID] = accountStream
	}
	s.streamMu.Unlock()

	return nil
}

func (s *StreamService) manageZeroBalanceStream(accountStream *AccountStream, account *model.MTAccount) {
	isZeroBalance := math.Abs(account.Balance) < 0.01 && math.Abs(account.Equity) < 0.01

	if isZeroBalance {
		s.closeNonEssentialStreams(accountStream.AccountID)
		s.zeroBalanceAccounts.Store(accountStream.AccountID, true)
	} else {
		s.restoreAllStreams(accountStream.AccountID)
		s.zeroBalanceAccounts.Delete(accountStream.AccountID)
	}
}

// broadcastQuoteUpdate 已禁用 - 不再需要
// func (s *StreamService) broadcastQuoteUpdate(accountStream *AccountStream, account *model.MTAccount) { ... }

func (s *StreamService) startQuoteStream(accountStream *AccountStream, account *model.MTAccount) {
	s.startQuoteStreamWithCtx(accountStream.Ctx, accountStream, account)
}

func (s *StreamService) registerSubscriber(subscriber *StreamSubscriber) error {
	accountStream, exists := s.getAccountStream(subscriber.AccountID)
	if !exists {
		return fmt.Errorf("账户流不存在: %s", subscriber.AccountID)
	}

	accountStream.mu.Lock()
	accountStream.Subscribers[subscriber.ID] = subscriber
	count := len(accountStream.Subscribers)
	needStart := count == 1
	accountStream.mu.Unlock()

	// Multi-instance demand is based on local subscriber count.
	// Only flip Redis key on transitions to reduce churn.
	if count == 1 {
		s.setDemand(subscriber.AccountID, true)
	}

	// Forwarder lifecycle: if we have local subscribers and we are not leader, ensure forwarder is running.
	// If we are leader, ensure forwarder is stopped.
	isLeader := s.isLeaderForAccount(subscriber.AccountID)
	if !isLeader {
		s.startForwarder(subscriber.AccountID)
		if count == 1 {
			s.publishWakeup(subscriber.AccountID)
		}
	} else {
		s.stopForwarder(subscriber.AccountID)
	}

	if needStart {
		// IMPORTANT (multi-instance): ConnectionManager subscription lifecycle triggers MT connect.
		// Only the cluster leader is allowed to touch connManager + supervisor for a given account.
		// Non-leaders rely on Redis event bus forwarding.
		if isLeader {
			if s.disableSupervisorForTest {
				// Test mode: do not start MT sessions or supervisors.
				return nil
			}
			if s.connManager != nil {
				if uid, err := uuid.Parse(subscriber.AccountID); err == nil {
					s.connManager.AddSubscription(uid, "events")
				}
			}
			s.ensureSupervisor(subscriber.AccountID)
		}
	}

	return nil
}

func (s *StreamService) unregisterSubscriber(subscriber *StreamSubscriber) {
	accountStream, exists := s.getAccountStream(subscriber.AccountID)
	if !exists {
		return
	}

	accountStream.mu.Lock()
	delete(accountStream.Subscribers, subscriber.ID)
	remaining := len(accountStream.Subscribers)
	idleSeq := accountStream.idleSeq.Add(1)
	accountStream.mu.Unlock()

	// Forwarder + demand lifecycle: any local subscribers => demand on; zero => demand off.
	if remaining == 0 {
		s.setDemand(subscriber.AccountID, false)
		s.stopForwarder(subscriber.AccountID)
	} else {
		if !s.isLeaderForAccount(subscriber.AccountID) {
			s.startForwarder(subscriber.AccountID)
		} else {
			s.stopForwarder(subscriber.AccountID)
		}
	}

	if remaining == 0 {
		// IMPORTANT (multi-instance): Only leader should drive connManager + idle shutdown.
		// Non-leaders just clear demand + stop forwarder (above).
		if s.isLeaderForAccount(subscriber.AccountID) {
			if s.disableSupervisorForTest {
				return
			}
			if s.connManager != nil {
				if uid, err := uuid.Parse(subscriber.AccountID); err == nil {
					s.connManager.RemoveSubscription(uid, "events")
				}
			}
			s.scheduleIdleShutdown(subscriber.AccountID, idleSeq)
		}
	}
}

func (s *StreamService) getAccountStream(accountID string) (*AccountStream, bool) {
	s.streamMu.RLock()
	defer s.streamMu.RUnlock()

	accountStream, exists := s.accountStreams[accountID]
	return accountStream, exists
}

func (s *StreamService) closeAccountStream(accountID string, reason string) {
	var accountStream *AccountStream
	var exists bool

	// MT-official-like: keep AccountStream object persistent (do not delete/cancel).
	// This method becomes a best-effort cleanup hook.
	accountStream, exists = s.getAccountStream(accountID)

	if !exists {
		return
	}

	accountStream.mu.Lock()
	// clear subscribers
	accountStream.Subscribers = make(map[string]*StreamSubscriber)
	accountStream.mu.Unlock()
	// NOTE: do not call accountStream.Cancel() here; supervisor keeps session alive.
}

func (s *StreamService) isStreamEnabled(accountStream *AccountStream, streamType string) bool {
	accountStream.streamMu.RLock()
	defer accountStream.streamMu.RUnlock()

	enabled, exists := accountStream.streamEnabled[streamType]
	if !exists {
		return true
	}
	return enabled
}

func (s *StreamService) closeNonEssentialStreams(accountID string) {
	accountStream, exists := s.getAccountStream(accountID)
	if !exists {
		return
	}

	accountStream.streamMu.Lock()
	defer accountStream.streamMu.Unlock()

	accountStream.streamEnabled["quote"] = false
	accountStream.streamEnabled["order"] = false
	accountStream.streamEnabled["profit"] = true

}

func (s *StreamService) restoreAllStreams(accountID string) {
	accountStream, exists := s.getAccountStream(accountID)
	if !exists {
		return
	}

	accountStream.streamMu.Lock()
	defer accountStream.streamMu.Unlock()

	accountStream.streamEnabled["quote"] = true
	accountStream.streamEnabled["order"] = true
	accountStream.streamEnabled["profit"] = true

}

package connect

import (
	"github.com/google/uuid"
)

func (s *StreamService) subscribeAccountEnabledChanges(userID string) (string, <-chan accountEnabledChange, func()) {
	if s == nil {
		return "", nil, func() {}
	}
	ch := make(chan accountEnabledChange, 32)
	subID := uuid.New().String()

	s.accountChangeMu.Lock()
	if s.accountChangeSubs == nil {
		s.accountChangeSubs = make(map[string]map[string]chan accountEnabledChange)
	}
	if s.accountChangeSubs[userID] == nil {
		s.accountChangeSubs[userID] = make(map[string]chan accountEnabledChange)
	}
	s.accountChangeSubs[userID][subID] = ch
	s.accountChangeMu.Unlock()

	unsubscribe := func() {
		s.accountChangeMu.Lock()
		userSubs := s.accountChangeSubs[userID]
		if userSubs != nil {
			delete(userSubs, subID)
			if len(userSubs) == 0 {
				delete(s.accountChangeSubs, userID)
			}
		}
		s.accountChangeMu.Unlock()
	}

	return subID, ch, unsubscribe
}

func (s *StreamService) NotifyAccountEnabledState(userID string, accountID string, enabled bool) {
	if s == nil || userID == "" || accountID == "" {
		return
	}

	s.accountChangeMu.Lock()
	userSubs := s.accountChangeSubs[userID]
	// copy to avoid holding lock while sending
	subs := make([]chan accountEnabledChange, 0, len(userSubs))
	for _, ch := range userSubs {
		if ch != nil {
			subs = append(subs, ch)
		}
	}
	s.accountChangeMu.Unlock()

	chg := accountEnabledChange{accountID: accountID, enabled: enabled}
	for _, ch := range subs {
		select {
		case ch <- chg:
		default:
			// best-effort; periodic reconcile will fix if dropped
		}
	}
}

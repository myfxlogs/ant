package mthub

import "context"

// PublishBar publishes a bar update to all subscribers for the given account.
func (s *MtHubService) PublishBar(ev *BarUpdate) {
	if s.barBroker != nil {
		s.barBroker.Publish(ev)
	}
}

// SubscribeBarUpdates returns a channel of bar updates for the given account.
func (s *MtHubService) SubscribeBarUpdates(accountID string) (<-chan *BarUpdate, func()) {
	if s.barBroker == nil {
		ch := make(chan *BarUpdate)
		close(ch)
		return ch, func() {}
	}
	return s.barBroker.Subscribe(accountID)
}

// PublishTick publishes a tick (Bid/Ask) update to all subscribers for the given account.
func (s *MtHubService) PublishTick(ev *TickUpdate) {
	if s.tickBroker != nil {
		s.tickBroker.Publish(ev)
	}
}

// SubscribeTickUpdates returns a channel of tick updates for the given account.
func (s *MtHubService) SubscribeTickUpdates(accountID string) (<-chan *TickUpdate, func()) {
	if s.tickBroker == nil {
		ch := make(chan *TickUpdate)
		close(ch)
		return ch, func() {}
	}
	return s.tickBroker.Subscribe(accountID)
}

// LatestTick returns the most recent tick for the given account+symbol.
// Returns nil if no tick has been received yet.
func (s *MtHubService) LatestTick(accountID, symbol string) *TickUpdate {
	if s.tickBroker == nil {
		return nil
	}
	return s.tickBroker.LatestTick(accountID, symbol)
}

// WatchAllTicks returns a channel that receives ALL tick updates across all accounts.
// Used by WatchActiveStrategies to push real-time prices to the Active Runs table.
func (s *MtHubService) WatchAllTicks() (<-chan *TickUpdate, func()) {
	if s.tickBroker == nil {
		ch := make(chan *TickUpdate)
		close(ch)
		return ch, func() {}
	}
	return s.tickBroker.WatchAll()
}

// PublishTradeEvent publishes a trade event to all subscribers for the given account.
func (s *MtHubService) PublishTradeEvent(ev *BrokerTradeEvent) {
	if s.tradeBroker != nil {
		s.tradeBroker.Publish(ev)
	}
}

// SubscribeTradeEvents returns a channel of trade events for the given account.
func (s *MtHubService) SubscribeTradeEvents(accountID string) (<-chan *BrokerTradeEvent, func()) {
	if s.tradeBroker == nil {
		ch := make(chan *BrokerTradeEvent)
		close(ch)
		return ch, func() {}
	}
	return s.tradeBroker.Subscribe(accountID)
}

// PublishAccountStatus publishes an account status event to all subscribers.
func (s *MtHubService) PublishAccountStatus(ev *AccountStatusEvent) {
	if s.statusBroker != nil {
		s.statusBroker.Publish(ev)
	}
}

// SubscribeAccountStatus returns a channel of account status events for the given account.
func (s *MtHubService) SubscribeAccountStatus(accountID string) (<-chan *AccountStatusEvent, func()) {
	if s.statusBroker == nil {
		ch := make(chan *AccountStatusEvent)
		close(ch)
		return ch, func() {}
	}
	return s.statusBroker.Subscribe(accountID)
}

// SubscribeUserOrderEvents subscribes to all order events for a user.
func (s *MtHubService) SubscribeUserOrderEvents(ctx context.Context, userID string) (<-chan *OrderEvent, func()) {
	return s.broker.Subscribe(userID)
}

// PublishPositionSnapshot publishes a full position snapshot to all subscribers.
func (s *MtHubService) PublishPositionSnapshot(ev *PositionSnapshot) {
	s.snapshotBroker.Publish(ev)
}

// SubscribePositionSnapshots returns a channel of full position snapshots for a single account.
func (s *MtHubService) SubscribePositionSnapshots(ctx context.Context, accountID string) (<-chan *PositionSnapshot, func()) {
	return s.snapshotBroker.Subscribe(accountID)
}

// SnapshotBroker returns the underlying PositionSnapshotBroker.
// Used by the execution barrier to publish read-after-write confirmation
// snapshots into the existing position pipeline (LIVE-ORDER-REENTRY-1).
func (s *MtHubService) SnapshotBroker() *PositionSnapshotBroker { return s.snapshotBroker }

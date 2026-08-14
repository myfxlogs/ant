package mthub

import "context"

// PublishAccountProfit publishes an account profit event to all subscribers.
func (s *MtHubService) PublishAccountProfit(ev *AccountProfitEvent) {
	s.accountBroker.Publish(ev)
}

// SubscribeAccountProfit returns a channel of account profit events for a single account.
func (s *MtHubService) SubscribeAccountProfit(ctx context.Context, accountID string) (<-chan *AccountProfitEvent, func()) {
	return s.accountBroker.Subscribe(accountID)
}

// SubscribeAccountProfitAll returns a channel of account profit events for all accounts.
func (s *MtHubService) SubscribeAccountProfitAll() (<-chan *AccountProfitEvent, func()) {
	return s.accountBroker.WatchAll()
}

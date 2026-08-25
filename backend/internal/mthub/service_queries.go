package mthub

import (
	"context"
	"time"
)

func (s *MtHubService) OpenedOrders(ctx context.Context, accountID string) ([]*OrderRecord, error) {
	exec := s.hub.Get(accountID)
	if exec == nil {
		return nil, ErrSessionNotFound
	}
	result, err := exec.FetchOpenedOrders(ctx)
	if err != nil && isSessionError(err) {
		if retryErr := s.reconnectAndRetry(accountID, func() error {
			exec = s.hub.Get(accountID)
			if exec == nil {
				return err
			}
			result, err = exec.FetchOpenedOrders(ctx)
			return err
		}); retryErr != nil {
			err = retryErr
		}
	}
	return result, err
}

// OrderHistory returns historical orders for the account.
func (s *MtHubService) OrderHistory(ctx context.Context, accountID string, from, to time.Time) ([]*OrderRecord, error) {
	exec := s.hub.Get(accountID)
	if exec == nil {
		return nil, ErrSessionNotFound
	}
	result, err := exec.FetchOrderHistory(ctx, from, to)
	if err != nil && isSessionError(err) {
		if retryErr := s.reconnectAndRetry(accountID, func() error {
			exec = s.hub.Get(accountID)
			if exec == nil {
				return err
			}
			result, err = exec.FetchOrderHistory(ctx, from, to)
			return err
		}); retryErr != nil {
			err = retryErr
		}
	}
	return result, err
}

// SymbolParams returns trading parameters for the given symbols.
func (s *MtHubService) SymbolParams(ctx context.Context, accountID string, canonicals []string) ([]*SymbolParam, error) {
	exec := s.hub.Get(accountID)
	if exec == nil {
		return nil, ErrSessionNotFound
	}
	result, err := exec.FetchSymbolParams(ctx, canonicals)
	if err != nil && isSessionError(err) {
		if retryErr := s.reconnectAndRetry(accountID, func() error {
			exec = s.hub.Get(accountID)
			if exec == nil {
				return err
			}
			result, err = exec.FetchSymbolParams(ctx, canonicals)
			return err
		}); retryErr != nil {
			err = retryErr
		}
	}
	return result, err
}

// ActiveAccountIDs returns the IDs of all currently connected MT accounts.
// Delegates to Hub.ActiveAccountIDs.
func (s *MtHubService) ActiveAccountIDs() []string {
	return s.hub.ActiveAccountIDs()
}

// PriceHistory fetches K-line bars from the connected broker.
func (s *MtHubService) PriceHistory(ctx context.Context, accountID, symbol, period string, from, to int64, count int) ([]*Bar, error) {
	exec := s.hub.Get(accountID)
	if exec == nil {
		return nil, ErrSessionNotFound
	}
	result, err := exec.FetchPriceHistory(ctx, symbol, period, from, to, count)
	if err != nil && isSessionError(err) {
		if retryErr := s.reconnectAndRetry(accountID, func() error {
			exec = s.hub.Get(accountID)
			if exec == nil {
				return err
			}
			result, err = exec.FetchPriceHistory(ctx, symbol, period, from, to, count)
			return err
		}); retryErr != nil {
			err = retryErr
		}
	}
	return result, err
}

// SymbolList returns all available symbol names for a connected MT account.
func (s *MtHubService) SymbolList(ctx context.Context, accountID string) ([]string, error) {
	exec := s.hub.Get(accountID)
	if exec == nil {
		return nil, ErrSessionNotFound
	}
	result, err := exec.FetchAllSymbols(ctx)
	if err != nil && isSessionError(err) {
		if retryErr := s.reconnectAndRetry(accountID, func() error {
			exec = s.hub.Get(accountID)
			if exec == nil {
				return err
			}
			result, err = exec.FetchAllSymbols(ctx)
			return err
		}); retryErr != nil {
			err = retryErr
		}
	}
	return result, err
}

// SubscribeSymbols dynamically subscribes the gateway to additional symbols
// for the given account. Newly added symbols start receiving ticks through
// the existing quote stream without reconnection.
func (s *MtHubService) SubscribeSymbols(ctx context.Context, accountID string, symbols []string) error {
	exec := s.hub.Get(accountID)
	if exec == nil {
		return ErrSessionNotFound
	}
	return exec.AddSymbols(ctx, symbols)
}

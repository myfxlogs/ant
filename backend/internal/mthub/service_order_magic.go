package mthub

import "context"

// GetOrderMagic looks up the magic_number from the orders table by (accountID, ticket).
// Returns 0 if the order is not found or magic is 0. Used by buildClosedTradeRecord
// as a fallback when broker OnOrderUpdate events don't carry magic (MT4/MT5 real-time
// stream omits magic, but the orders table has it from order placement).
func (s *MtHubService) GetOrderMagic(ctx context.Context, accountID string, ticket int64) (int32, error) {
	if s.omsWriter == nil {
		return 0, nil
	}
	pool := s.omsWriter.Pool()
	if pool == nil {
		return 0, nil
	}
	var magic int32
	err := pool.QueryRow(ctx,
		`SELECT magic_number FROM orders WHERE mt_account_id = $1::uuid AND ticket = $2
		 ORDER BY created_at DESC LIMIT 1`,
		accountID, ticket).Scan(&magic)
	if err != nil {
		return 0, nil // not found is fine — return 0, no error
	}
	return magic, nil
}

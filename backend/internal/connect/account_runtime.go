package connect

import (
	"context"
	"fmt"

	"github.com/google/uuid"
)

type AccountRuntime struct {
	s          *StreamService
	accountID  string
	accountUID uuid.UUID
}

func newAccountRuntime(s *StreamService, accountID string) (*AccountRuntime, error) {
	uid, err := uuid.Parse(accountID)
	if err != nil {
		return nil, fmt.Errorf("invalid account id: %s", accountID)
	}
	return &AccountRuntime{s: s, accountID: accountID, accountUID: uid}, nil
}

func (r *AccountRuntime) getStream() (*AccountStream, bool) {
	return r.s.getAccountStream(r.accountID)
}

func (r *AccountRuntime) Reconcile(ctx context.Context) {
	_ = ctx
	// Legacy runtime reconciliation is intentionally disabled.
	// The account supervisor owns profit/order worker lifecycles.
}

func (r *AccountRuntime) ForceClose(reason string) {
	_ = reason
	// Legacy runtime force-close is intentionally disabled.
}

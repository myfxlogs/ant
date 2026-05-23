package broker

import (
	"context"
	"fmt"

	"anttrader/internal/oms"
)

// baseAdapter provides default Cancel/Modify/Query implementations
// for broker adapters. Embed this in platform-specific adapters and
// override Submit with the actual implementation.
type baseAdapter struct {
	platform string
}

func (a *baseAdapter) Cancel(ctx context.Context, ticket string) error {
	return fmt.Errorf("%s cancel not implemented", a.platform)
}

func (a *baseAdapter) Modify(ctx context.Context, ticket string, price, stopPrice float64) error {
	return fmt.Errorf("%s modify not implemented", a.platform)
}

func (a *baseAdapter) Query(ctx context.Context, ticket string) (*oms.Order, error) {
	return nil, fmt.Errorf("%s query not implemented", a.platform)
}

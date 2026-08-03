package ai

import (
	"context"
	"fmt"

	"connectrpc.com/connect"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"

	"alphaforge/internal/interceptor"
)

// userIDFromCtx extracts the authenticated user ID from the ConnectRPC context.
func userIDFromCtx(ctx context.Context) (uuid.UUID, error) {
	id, err := uuid.Parse(interceptor.GetUserID(ctx))
	if err != nil {
		return uuid.Nil, connect.NewError(connect.CodeUnauthenticated, fmt.Errorf("invalid user id"))
	}
	return id, nil
}

func parseDecimal(s string) (decimal.Decimal, error) {
	return decimal.NewFromString(s)
}

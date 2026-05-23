package service

import (
	"context"
	"time"

	"github.com/google/uuid"
)

type MatchingEngine interface {
	OrderSend(ctx context.Context, userID uuid.UUID, req *OrderSendRequest) (*OrderResponse, error)
	OrderModify(ctx context.Context, userID uuid.UUID, req *OrderModifyRequest) (*OrderResponse, error)
	OrderClose(ctx context.Context, userID uuid.UUID, req *OrderCloseRequest) (*OrderResponse, error)

	GetPositions(ctx context.Context, userID, accountID uuid.UUID) ([]*PositionResponse, error)
	GetOrderHistory(ctx context.Context, userID, accountID uuid.UUID, from, to time.Time) ([]*HistoryOrderResponse, error)
}

var _ MatchingEngine = (*TradingService)(nil)

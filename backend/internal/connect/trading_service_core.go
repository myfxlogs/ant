package connect

import (
	"context"
	"crypto/sha256"
	"fmt"
	"time"

	"github.com/google/uuid"

	"anttrader/internal/model"
	"anttrader/internal/repository"
	"anttrader/internal/service"
)

type tradingService interface {
	OrderSend(ctx context.Context, userID uuid.UUID, req *service.OrderSendRequest) (*service.OrderResponse, error)
	OrderModify(ctx context.Context, userID uuid.UUID, req *service.OrderModifyRequest) (*service.OrderResponse, error)
	OrderClose(ctx context.Context, userID uuid.UUID, req *service.OrderCloseRequest) (*service.OrderResponse, error)
	GetPositions(ctx context.Context, userID uuid.UUID, accountID uuid.UUID) ([]*service.PositionResponse, error)
	GetOrderHistory(ctx context.Context, userID uuid.UUID, accountID uuid.UUID, from, to time.Time) ([]*service.HistoryOrderResponse, error)
	SyncOrderHistory(ctx context.Context, userID, accountID uuid.UUID, from, to time.Time) (int, error)
}

type TradingService struct {
	accountRepo *repository.AccountRepository
	tradingSvc  tradingService
	connManager interface {
		SyncOrderHistory(accountID uuid.UUID, mtType string)
	}
	logSvc interface {
		LogOperation(ctx context.Context, log *model.SystemOperationLog) error
	}
	streamSvc *StreamService
}

func NewTradingService(accountRepo *repository.AccountRepository, tradingSvc tradingService, connManager interface {
	SyncOrderHistory(accountID uuid.UUID, mtType string)
}, logSvc interface {
	LogOperation(ctx context.Context, log *model.SystemOperationLog) error
}, streamSvc *StreamService) *TradingService {
	return &TradingService{
		accountRepo: accountRepo,
		tradingSvc:  tradingSvc,
		connManager: connManager,
		logSvc:      logSvc,
		streamSvc:   streamSvc,
	}
}

func tradeIdempotencyKey(accountID string, operation string, idempotencyKey string) string {
	// Keep keys bounded in length while still deterministic.
	sum := sha256.Sum256([]byte(accountID + ":" + operation + ":" + idempotencyKey))
	return fmt.Sprintf("trade:idemp:%s:%x", accountID, sum)
}

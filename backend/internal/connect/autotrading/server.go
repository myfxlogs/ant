package autotrading

import (
	"context"

	"github.com/google/uuid"
	"go.uber.org/zap"

	antv1c "alphaforge/gen/proto/ant/v1/antv1connect"
	"alphaforge/internal/interceptor"
	"alphaforge/internal/repository"
	"alphaforge/internal/risksvc"
)

// AutoTradingServer implements the AutoTradingService ConnectRPC handler.
type AutoTradingServer struct {
	autoRepo *repository.AutoTradingRepository
	riskPipe *risksvc.SignalPipeline
	log      *zap.Logger
}

var _ antv1c.AutoTradingServiceHandler = (*AutoTradingServer)(nil)

// NewAutoTradingServer creates an AutoTradingService handler.
func NewAutoTradingServer(
	autoRepo *repository.AutoTradingRepository,
	riskPipe *risksvc.SignalPipeline,
	log *zap.Logger,
) *AutoTradingServer {
	return &AutoTradingServer{autoRepo: autoRepo, riskPipe: riskPipe, log: log}
}

// userID extracts the authenticated user from context.
// Returns uuid.Nil if not authenticated; callers should check and return
// CodeUnauthenticated for methods that require a valid user.
func (s *AutoTradingServer) userID(ctx context.Context) uuid.UUID {
	raw := interceptor.GetUserID(ctx)
	if raw == "" {
		s.log.Warn("autotrading: userID not in context")
		return uuid.Nil
	}
	id, err := uuid.Parse(raw)
	if err != nil {
		s.log.Warn("autotrading: userID parse failed", zap.String("raw", raw), zap.Error(err))
		return uuid.Nil
	}
	return id
}

package strategy

import (
	"context"

	"go.uber.org/zap"

	"github.com/google/uuid"

	antv1c "anttrader/gen/proto/ant/v1/antv1connect"
	"anttrader/internal/interceptor"
	"anttrader/internal/service"
	"anttrader/internal/pglisten"
	"anttrader/internal/strategysvc"
)

type StrategyServer struct {
	svc    *service.StrategySvc
	client *strategysvc.PythonClient // S2.5: real Python backtest
	log    *zap.Logger
	pgListen    *pglisten.Listener
}

var _ antv1c.StrategyServiceHandler = (*StrategyServer)(nil)

func NewStrategyServer(svc *service.StrategySvc, log *zap.Logger) *StrategyServer {
	return &StrategyServer{svc: svc, log: log}
}

// SetClient injects the Python strategy-service client (S2.5).
func (s *StrategyServer) SetClient(c *strategysvc.PythonClient) { s.client = c }

func (s *StrategyServer) userID(ctx context.Context) uuid.UUID {
	raw := interceptor.GetUserID(ctx)
	if raw == "" {
		s.log.Warn("strategy: userID called but no user in context")
		return uuid.Nil
	}
	id, err := uuid.Parse(raw)
	if err != nil {
		s.log.Warn("strategy: userID parse failed", zap.String("raw", raw), zap.Error(err))
		return uuid.Nil
	}
	return id
}

func (s *StrategyServer) SetPgListen(l *pglisten.Listener) { s.pgListen = l }

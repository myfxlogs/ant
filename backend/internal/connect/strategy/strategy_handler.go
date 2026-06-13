package strategy

import (
	"context"

	"go.uber.org/zap"

	"github.com/google/uuid"

	antv1c "anttrader/gen/proto/ant/v1/antv1connect"
	"anttrader/internal/interceptor"
	"anttrader/internal/repository"
	"anttrader/internal/service"
	"anttrader/internal/pglisten"
)

type StrategyServer struct {
	svc            *service.StrategySvc
	backtestClient antv1c.BacktestServiceClient   // ConnectRPC to Python BacktestService
	marketDataRepo repository.MarketDataStore
	log            *zap.Logger
	pgListen       *pglisten.Listener
	engine         *ScheduleEngine
}

var _ antv1c.StrategyServiceHandler = (*StrategyServer)(nil)

func NewStrategyServer(svc *service.StrategySvc, log *zap.Logger) *StrategyServer {
	return &StrategyServer{svc: svc, log: log}
}

func (s *StrategyServer) SetEngine(e *ScheduleEngine) { s.engine = e }

// SetBacktestClient injects the ConnectRPC backtest client for RunBacktest.
func (s *StrategyServer) SetBacktestClient(c antv1c.BacktestServiceClient) { s.backtestClient = c }

// SetMarketDataRepo injects the ClickHouse market data repo for fetching K-lines.
func (s *StrategyServer) SetMarketDataRepo(r repository.MarketDataStore) { s.marketDataRepo = r }

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

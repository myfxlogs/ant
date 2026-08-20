package autotrading

import (
	"context"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"go.uber.org/zap"

	antv1c "alphaforge/gen/proto/ant/v1/antv1connect"
	"alphaforge/internal/interceptor"
	"alphaforge/internal/model"
	"alphaforge/internal/repository"
	"alphaforge/internal/risksvc"
)

// autoTradeStore abstracts the AutoTradingRepository methods used by handlers.
// *repository.AutoTradingRepository satisfies this implicitly. Enables testing
// the callback wiring without a real PG connection (SCHEDULE-HOTLOOP-1).
type autoTradeStore interface {
	GetGlobalSettingsByUserID(ctx context.Context, userID uuid.UUID) (*model.GlobalSettings, error)
	CreateGlobalSettings(ctx context.Context, settings *model.GlobalSettings) error
	UpdateGlobalSettings(ctx context.Context, settings *model.GlobalSettings) error
	UpdateAutoTradeEnabled(ctx context.Context, userID uuid.UUID, enabled bool) error
	GetRiskConfigByAccountID(ctx context.Context, accountID uuid.UUID) (*model.RiskConfig, error)
	CreateRiskConfig(ctx context.Context, rc *model.RiskConfig) error
	UpdateRiskConfig(ctx context.Context, rc *model.RiskConfig) error
	CountActiveSchedules(ctx context.Context, userID uuid.UUID) (int, error)
	CountPendingExecutions(ctx context.Context, userID uuid.UUID) (int, error)
	CountTodayExecutionsByUser(ctx context.Context, userID uuid.UUID) (int, error)
	GetTodayProfitByUser(ctx context.Context, userID uuid.UUID) (decimal.Decimal, error)
	GetTradingLogs(ctx context.Context, userID uuid.UUID, params *model.LogListParams) ([]*model.TradingLog, int, error)
	GetRecentTradingLogs(ctx context.Context, userID uuid.UUID, limit int) ([]*model.TradingLog, error)
}

// AutoTradingServer implements the AutoTradingService ConnectRPC handler.
type AutoTradingServer struct {
	autoRepo           autoTradeStore
	riskPipe           *risksvc.SignalPipeline
	log                *zap.Logger
	onAutoTradeChanged func(userID uuid.UUID) // SCHEDULE-HOTLOOP-1: invalidate schedule engine cache
}

var _ antv1c.AutoTradingServiceHandler = (*AutoTradingServer)(nil)

// NewAutoTradingServer creates an AutoTradingService handler.
func NewAutoTradingServer(
	autoRepo *repository.AutoTradingRepository,
	riskPipe *risksvc.SignalPipeline,
	log *zap.Logger,
) *AutoTradingServer {
	var store autoTradeStore
	if autoRepo != nil {
		store = autoRepo
	}
	return &AutoTradingServer{autoRepo: store, riskPipe: riskPipe, log: log}
}

// SetOnAutoTradeChanged registers a callback invoked after autoTradeEnabled
// changes in the DB. The callback (wired by handlers_strategy_runtime.go)
// calls ScheduleEngine.InvalidateAutoTradeCache + Notify so the schedule
// engine immediately reflects the new autoTrade state — preventing a 30s
// TTL window where disabled autoTrade still dispatches (SCHEDULE-HOTLOOP-1).
// The autotrading package does not import strategy to avoid an import cycle.
func (s *AutoTradingServer) SetOnAutoTradeChanged(fn func(userID uuid.UUID)) {
	s.onAutoTradeChanged = fn
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

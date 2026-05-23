package service

import (
	"context"
	"time"

	"github.com/google/uuid"
)

type engineModeKey struct{}
type engineOverrideKey struct{}

type EngineMode string

const (
	EngineModeLive     EngineMode = "live"
	EngineModeBacktest EngineMode = "backtest"
)

func WithEngineMode(ctx context.Context, mode EngineMode) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	return context.WithValue(ctx, engineModeKey{}, mode)
}

func EngineModeFromContext(ctx context.Context) EngineMode {
	if ctx == nil {
		return EngineModeLive
	}
	v := ctx.Value(engineModeKey{})
	m, ok := v.(EngineMode)
	if !ok {
		return EngineModeLive
	}
	if m == EngineModeBacktest {
		return EngineModeBacktest
	}
	return EngineModeLive
}

func WithBacktestEngine(ctx context.Context, eng MatchingEngine) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	return context.WithValue(ctx, engineOverrideKey{}, eng)
}

func backtestEngineOverrideFromContext(ctx context.Context) (MatchingEngine, bool) {
	if ctx == nil {
		return nil, false
	}
	v := ctx.Value(engineOverrideKey{})
	if v == nil {
		return nil, false
	}
	eng, ok := v.(MatchingEngine)
	return eng, ok && eng != nil
}

type EngineSelector struct {
	live     MatchingEngine
	backtest MatchingEngine
}

func NewEngineSelector(live MatchingEngine, backtest MatchingEngine) *EngineSelector {
	return &EngineSelector{live: live, backtest: backtest}
}

func (s *EngineSelector) pick(ctx context.Context) MatchingEngine {
	if s == nil {
		return nil
	}
	if EngineModeFromContext(ctx) == EngineModeBacktest {
		if eng, ok := backtestEngineOverrideFromContext(ctx); ok {
			return eng
		}
		if s.backtest != nil {
			return s.backtest
		}
	}
	return s.live
}

func (s *EngineSelector) OrderSend(ctx context.Context, userID uuid.UUID, req *OrderSendRequest) (*OrderResponse, error) {
	return s.pick(ctx).OrderSend(ctx, userID, req)
}

func (s *EngineSelector) OrderModify(ctx context.Context, userID uuid.UUID, req *OrderModifyRequest) (*OrderResponse, error) {
	return s.pick(ctx).OrderModify(ctx, userID, req)
}

func (s *EngineSelector) OrderClose(ctx context.Context, userID uuid.UUID, req *OrderCloseRequest) (*OrderResponse, error) {
	return s.pick(ctx).OrderClose(ctx, userID, req)
}

func (s *EngineSelector) GetPositions(ctx context.Context, userID, accountID uuid.UUID) ([]*PositionResponse, error) {
	return s.pick(ctx).GetPositions(ctx, userID, accountID)
}

func (s *EngineSelector) GetOrderHistory(ctx context.Context, userID, accountID uuid.UUID, from, to time.Time) ([]*HistoryOrderResponse, error) {
	return s.pick(ctx).GetOrderHistory(ctx, userID, accountID, from, to)
}

var _ MatchingEngine = (*EngineSelector)(nil)

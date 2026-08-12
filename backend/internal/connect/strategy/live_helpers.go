package strategy

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"go.uber.org/zap"

	antv1 "alphaforge/gen/proto/ant/v1"
	"alphaforge/internal/model"
	"alphaforge/internal/mthub"
	"alphaforge/internal/repository"
)

// barOpenTimeForSignal extracts the bar open time for ClientID generation.
// For bar signals, returns bar.OpenTime (deterministic per bar — enables
// idempotency dedup for bar replays). For tick/trade signals (bar=nil),
// returns a per-run atomic counter value to ensure each tick order gets
// a unique ClientID (preventing idempotency guard from swallowing
// subsequent tick orders in the same run).
func barOpenTimeForSignal(bar *mthub.BarUpdate, cfg LiveStrategyConfig) int64 {
	if bar != nil {
		return bar.OpenTime
	}
	if cfg.TickSeq != nil {
		return cfg.TickSeq.Add(1)
	}
	return 0
}

// strategyOrderClientID generates a deterministic ClientID for strategy-submitted
// orders, enabling idempotency dedup. Same runID + bar open time + signal type
// always produces the same key, so duplicate dispatches (bar replay, VM retry,
// network glitch) are rejected by the idempotency guard in MtHubService.PlaceOrder.
// For tick/trade signals, barOpenTime is a per-run atomic counter (see barOpenTimeForSignal).
func strategyOrderClientID(runID uuid.UUID, barOpenTime int64, signalType string) string {
	return fmt.Sprintf("start-%s-%d-%s", runID, barOpenTime, signalType)
}

// strategyMagic delegates to model.StrategyMagic for a deterministic 32-bit magic number.
// Kept as a thin wrapper for call-site readability within the strategy package.
func strategyMagic(scheduleID uuid.UUID) int32 {
	return model.StrategyMagic(scheduleID)
}

// signalToSide maps a strategy signal action to mthub.Side. Returns 0 for non-directional signals.
// T3.1: expanded to cover all buy_* / sell_* variants (hard injury ②).
func signalToSide(action string) mthub.Side {
	switch action {
	case "buy", "buy_limit", "buy_stop", "buy_stop_limit":
		return mthub.SideBuy
	case sideSell, "sell_limit", "sell_stop", "sell_stop_limit":
		return mthub.SideSell
	default:
		return 0
	}
}

// sideToString returns a human-readable Side string for logging.
func sideToString(side mthub.Side) string {
	switch side {
	case mthub.SideBuy:
		return "buy"
	case mthub.SideSell:
		return sideSell
	default:
		return "unknown"
	}
}

// persistSignal writes the signal to strategy_signals and increments the run's signal count.
// Failures are logged but do not block dispatch — persistence is best-effort.
func (s *StrategyExecutionServer) persistSignal(ctx context.Context, cfg LiveStrategyConfig, sig *antv1.StrategySignal) {
	if s.runRepo == nil {
		return
	}
	var runIDPtr *uuid.UUID
	if cfg.RunID != uuid.Nil {
		id := cfg.RunID
		runIDPtr = &id
	}
	params := repository.InsertSignalParams{
		AccountID:  cfg.AccountID,
		Symbol:     cfg.Symbol,
		SignalType: sig.GetSignalType(),
		Volume:     sig.GetVolume(),
		Price:      sig.GetPrice(),
		StopLoss:   sig.GetStopLoss(),
		TakeProfit: sig.GetTakeProfit(),
		Reason:     sig.GetReason(),
		RunID:      runIDPtr,
	}
	if err := s.runRepo.InsertSignal(ctx, params); err != nil {
		s.log.Warn("LiveStrategyRunner: failed to persist signal", zap.Error(err))
	}
	if cfg.RunID != uuid.Nil {
		if err := s.runRepo.IncrementSignalCount(ctx, cfg.RunID); err != nil {
			s.log.Warn("LiveStrategyRunner: failed to increment signal count", zap.Error(err))
		}
	}
}

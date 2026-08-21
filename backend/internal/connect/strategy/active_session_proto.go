// active_session_proto.go — proto conversion helpers for active strategy sessions.
package strategy

import (
	"context"
	"time"

	"github.com/google/uuid"
	"google.golang.org/protobuf/types/known/timestamppb"

	antv1 "alphaforge/gen/proto/ant/v1"
	"alphaforge/internal/mthub"
)

// activeSessionToProto converts an ActiveSession to a proto ActiveStrategy.
// If tickFn is non-nil, it is called to populate bid/ask/lastTickAt for the session's symbol.
// posCache is injected by the server converter (not read from ActiveSession) to avoid
// the race between Register→notify-watcher and posCache field initialization.
func activeSessionToProto(
	sess *ActiveSession,
	tickFn func(accountID, symbol string) (bid, ask string, tickAt *time.Time),
	posCache *PositionCache,
) *antv1.ActiveStrategy {
	sess.pnlMu.RLock()
	pnl := sess.PnL
	lastTick := sess.LastTickAt
	sess.pnlMu.RUnlock()

	pb := &antv1.ActiveStrategy{
		RunId:       sess.RunID.String(),
		UserId:      sess.UserID.String(),
		AccountId:   sess.AccountID,
		Symbol:      sess.Symbol,
		Timeframe:   sess.Timeframe,
		Mode:        sess.Mode,
		StartedAt:   timestamppb.New(sess.StartedAt),
		SignalCount: int32(sess.SignalCount),
		ErrorCount:  int32(sess.ErrorCount),
		LastError:   sess.LastError,
		StderrTail:  sess.StderrTail,
		ScheduleId:  sess.ScheduleID.String(),
		Pnl:         pnl,
	}
	if !sess.LastSignalAt.IsZero() {
		pb.LastSignalAt = timestamppb.New(sess.LastSignalAt)
	}
	if !lastTick.IsZero() {
		pb.LastTickAt = timestamppb.New(lastTick)
	}
	if tickFn != nil {
		bid, ask, tickAt := tickFn(sess.AccountID, sess.Symbol)
		pb.Bid, pb.Ask = bid, ask
		if tickAt != nil {
			pb.LastTickAt = timestamppb.New(*tickAt)
		}
	}
	if sess.diag != nil {
		snap := sess.diag.SnapshotDiag()
		// L3: Enrich snapshot with order truth from PositionCache + barrier.
		// posCache is injected (not read from sess) to avoid race.
		enrichDiagSnapshot(&snap, sess, posCache)
		pb.Diagnostics = diagToProto(snap)
	}
	return pb
}

// enrichDiagSnapshot populates L3 diagnostic fields from PositionCache and
// TradeBarrier. This is the single server-side computation point — the
// frontend renders only, never infers authoritative state (rule 5).
//
// Lifecycle persistence: OrderLifecycle and LastBrokerTicket are persisted
// in sessionDiag (via RecordLifecycle at logOrderLifecycle). The barrier's
// transient state is only used for ExecutionState — the persisted lifecycle
// survives Release() which clears the barrier to idle/ticket=0.
func enrichDiagSnapshot(snap *DiagSnapshot, sess *ActiveSession, posCache *PositionCache) {
	now := time.Now()
	snap.ScheduleMagic = sess.MagicNumber
	snap.VmOrdersTotal = snap.OrdersTotalSeen

	// Execution state from TradeBarrier (transient — reflects current barrier)
	if sess.barrier != nil {
		snap.ExecutionState = sess.barrier.State().String()
	} else {
		snap.ExecutionState = "idle"
	}

	// OrderLifecycle + LastBrokerTicket are already in snap from SnapshotDiag
	// (persisted in sessionDiag via RecordLifecycle). They survive Release().

	// Order truth from PositionCache (injected, not from sess field)
	// DataAvailable is based on actual snapshot existence, not just posCache!=nil.
	// This distinguishes "cache exists but no data for this account" from "has data".
	if posCache != nil {
		rawSnap := posCache.GetSnapshot(sess.AccountID)
		if rawSnap != nil {
			snap.DataAvailable = true
			enrichFromPositionCache(snap, sess.MagicNumber, posCache, rawSnap, now)
		} else {
			// Cache exists but no snapshot for this account yet
			snap.DataAvailable = false
		}
	} else {
		// No broker data source (paper mode / not wired).
		// DataAvailable=false → frontend shows N/A, not Stale/Warning.
		snap.DataAvailable = false
	}
}

// enrichFromPositionCache computes broker/magic/pending order counts and
// freshness metadata from the PositionCache snapshot.
// rawSnap is pre-fetched by the caller to avoid double-fetch.
func enrichFromPositionCache(snap *DiagSnapshot, magic int32, pc *PositionCache, rawSnap *mthub.PositionSnapshot, now time.Time) {
	accountID := rawSnap.AccountID
	// Broker account-level: all positions + pending orders (no magic filter)
	snap.BrokerAccountOrders = len(rawSnap.Positions) + len(rawSnap.PendingOrders)
	snap.PendingBrokerOrders = len(rawSnap.PendingOrders)

	// Strategy magic orders: positions + pending matching schedule magic
	magicCount := 0
	for _, p := range rawSnap.Positions {
		if p.Magic == magic {
			magicCount++
		}
	}
	for _, p := range rawSnap.PendingOrders {
		if p.Magic == magic {
			magicCount++
		}
	}
	snap.StrategyMagicOrders = magicCount

	// Financial freshness
	if !rawSnap.CapturedAt.IsZero() {
		snap.FinancialSource = rawSnap.FinancialsSource
		snap.FinancialCapturedAt = rawSnap.CapturedAt.UnixMilli()
		ageMs := now.Sub(rawSnap.CapturedAt).Milliseconds()
		if ageMs >= 0 {
			snap.FinancialAgeMs = ageMs
		}
	}
	snap.FinancialFresh = isFinancialFresh(pc, accountID, now)

	// Positions freshness (independently tracked)
	if !rawSnap.PositionsCapturedAt.IsZero() {
		snap.PositionsSource = rawSnap.PositionsSource
		snap.PositionsCapturedAt = rawSnap.PositionsCapturedAt.UnixMilli()
		ageMs := now.Sub(rawSnap.PositionsCapturedAt).Milliseconds()
		if ageMs >= 0 {
			snap.PositionsAgeMs = ageMs
		}
	}
	snap.PositionsFresh = isPositionsFresh(pc, accountID, now)
}

// isFinancialFresh checks PositionCache financial freshness without returning
// the snapshot — used for the boolean fresh flag.
func isFinancialFresh(pc *PositionCache, accountID string, now time.Time) bool {
	_, ok := pc.GetFreshFinancialSnapshot(accountID, now)
	return ok
}

// isPositionsFresh checks PositionCache positions freshness.
func isPositionsFresh(pc *PositionCache, accountID string, now time.Time) bool {
	_, ok := pc.GetFreshPositionSnapshot(accountID, now)
	return ok
}

// tickPriceFn returns a closure that fetches the latest bid/ask/tick-time for a given
// account+symbol from the MtHub tick broker. Returns nil if mtHub is unavailable.
func (s *StrategyExecutionServer) tickPriceFn() func(accountID, symbol string) (bid, ask string, tickAt *time.Time) {
	if s.mtHub == nil {
		return nil
	}
	return func(accountID, symbol string) (string, string, *time.Time) {
		tick := s.mtHub.LatestTick(accountID, symbol)
		if tick == nil {
			return "", "", nil
		}
		return tick.Bid.String(), tick.Ask.String(), &tick.Time
	}
}

// enrichWithStrategyName fills StrategyName on each ActiveStrategy proto by
// looking up schedule_id → schedule name. nil lookup = no-op (name stays empty).
func (s *StrategyExecutionServer) enrichWithStrategyName(ctx context.Context, pbs []*antv1.ActiveStrategy) {
	if s.scheduleNameLookup == nil {
		return
	}
	for _, pb := range pbs {
		if pb.ScheduleId == "" {
			continue
		}
		scheduleID, err := uuid.Parse(pb.ScheduleId)
		if err != nil {
			continue
		}
		pb.StrategyName = s.scheduleNameLookup(ctx, scheduleID)
	}
}

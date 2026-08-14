// active_session_proto.go — proto conversion helpers for active strategy sessions.
package strategy

import (
	"context"
	"time"

	"github.com/google/uuid"
	"google.golang.org/protobuf/types/known/timestamppb"

	antv1 "alphaforge/gen/proto/ant/v1"
)

// activeSessionToProto converts an ActiveSession to a proto ActiveStrategy.
// If tickFn is non-nil, it is called to populate bid/ask/lastTickAt for the session's symbol.
func activeSessionToProto(sess *ActiveSession, tickFn func(accountID, symbol string) (bid, ask string, tickAt *time.Time)) *antv1.ActiveStrategy {
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
	return pb
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

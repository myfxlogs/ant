// strategy_active_watch.go — RPC handlers for active strategy monitoring (read + SSE).
package strategy

import (
	"context"
	"errors"
	"fmt"
	"time"

	"connectrpc.com/connect"
	"github.com/google/uuid"
	"google.golang.org/protobuf/types/known/timestamppb"

	antv1 "alphaforge/gen/proto/ant/v1"
	"alphaforge/internal/mthub"
	"alphaforge/internal/service"
)

// ListActiveStrategies returns currently running strategy sessions.
func (s *StrategyExecutionServer) ListActiveStrategies(ctx context.Context, req *connect.Request[antv1.ListActiveStrategiesRequest]) (*connect.Response[antv1.ListActiveStrategiesResponse], error) {
	uid, err := userIDRequire(ctx)
	if err != nil {
		return nil, err
	}
	if s.sessionRegistry == nil {
		return connect.NewResponse(&antv1.ListActiveStrategiesResponse{}), nil
	}

	var sessions []*ActiveSession
	if accountFilter := req.Msg.GetAccountId(); accountFilter != "" {
		accountUUID, parseErr := uuid.Parse(accountFilter)
		if parseErr != nil {
			return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("invalid account_id: %w", parseErr))
		}
		if err := s.checkBoundAccount(ctx, uid, accountUUID); err != nil {
			if errors.Is(err, service.ErrAccountLimitExceeded) {
				return nil, connect.NewError(connect.CodePermissionDenied, err)
			}
			if errors.Is(err, service.ErrAccountNotOwned) {
				return nil, connect.NewError(connect.CodeNotFound, err)
			}
			return nil, connect.NewError(connect.CodeInternal, err)
		}
		sessions = s.sessionRegistry.ListByAccount(accountFilter)
	} else {
		sessions = s.sessionRegistry.ListByUser(uid)
	}

	pbStrategies := make([]*antv1.ActiveStrategy, len(sessions))
	tickFn := s.tickPriceFn()
	for i, sess := range sessions {
		pbStrategies[i] = activeSessionToProto(sess, tickFn)
	}
	s.enrichWithStrategyName(ctx, pbStrategies)
	return connect.NewResponse(&antv1.ListActiveStrategiesResponse{Strategies: pbStrategies}), nil
}

// GetActiveStrategy returns a single active strategy session by run ID.
func (s *StrategyExecutionServer) GetActiveStrategy(ctx context.Context, req *connect.Request[antv1.GetActiveStrategyRequest]) (*connect.Response[antv1.GetActiveStrategyResponse], error) {
	uid, err := userIDRequire(ctx)
	if err != nil {
		return nil, err
	}
	if s.sessionRegistry == nil {
		return nil, connect.NewError(connect.CodeUnavailable, fmt.Errorf("session registry not configured"))
	}

	runID, err := uuid.Parse(req.Msg.GetRunId())
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("invalid run_id: %w", err))
	}

	sess, ok := s.sessionRegistry.Get(runID)
	if !ok {
		return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("active strategy %s not found", runID))
	}
	if sess.UserID != uid {
		return nil, connect.NewError(connect.CodePermissionDenied, fmt.Errorf("strategy %s not found", runID))
	}

	pb := activeSessionToProto(sess, s.tickPriceFn())
	s.enrichWithStrategyName(ctx, []*antv1.ActiveStrategy{pb})
	return connect.NewResponse(&antv1.GetActiveStrategyResponse{Strategy: pb}), nil
}

// WatchStrategySignals streams real-time signals from a running strategy via SSE.
func (s *StrategyExecutionServer) WatchStrategySignals(ctx context.Context, req *connect.Request[antv1.WatchStrategySignalsRequest], stream *connect.ServerStream[antv1.StrategySignalEvent]) error {
	uid, err := userIDRequire(ctx)
	if err != nil {
		return err
	}
	if s.sessionRegistry == nil {
		return connect.NewError(connect.CodeUnavailable, fmt.Errorf("session registry not configured"))
	}

	runID, err := uuid.Parse(req.Msg.GetRunId())
	if err != nil {
		return connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("invalid run_id: %w", err))
	}

	sess, ok := s.sessionRegistry.Get(runID)
	if !ok {
		return connect.NewError(connect.CodeNotFound, fmt.Errorf("active strategy %s not found", runID))
	}
	if sess.UserID != uid {
		return connect.NewError(connect.CodePermissionDenied, fmt.Errorf("strategy %s not found", runID))
	}

	sigCh := sess.SubscribeSignals()

	// Heartbeat: send empty event periodically to keep SSE alive when no
	// signals are dispatched. Prevents中间层 from closing idle streams.
	hbInterval := s.heartbeatInterval
	if hbInterval <= 0 {
		hbInterval = 20 * time.Second
	}
	heartbeat := time.NewTicker(hbInterval)
	defer heartbeat.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil
		case event, ok := <-sigCh:
			if !ok {
				return nil
			}
			if err := stream.Send(&antv1.StrategySignalEvent{
				RunId:      event.RunID.String(),
				AccountId:  event.AccountID,
				Symbol:     event.Symbol,
				SignalType: event.SignalType,
				Volume:     event.Volume,
				Price:      event.Price,
				StopLoss:   event.StopLoss,
				TakeProfit: event.TakeProfit,
				Reason:     event.Reason,
				Timestamp:  timestamppb.New(event.Timestamp),
			}); err != nil {
				return err
			}
		case <-heartbeat.C:
			if err := stream.Send(&antv1.StrategySignalEvent{}); err != nil {
				return err
			}
		}
	}
}

// WatchActiveStrategies streams the active strategy list via SSE.
// Sends the current list immediately, then pushes updates whenever
// sessions are registered, deregistered, or change state.
// Also pushes updates when tick prices change (throttled to 500ms).
func (s *StrategyExecutionServer) WatchActiveStrategies(
	ctx context.Context,
	req *connect.Request[antv1.WatchActiveStrategiesRequest],
	stream *connect.ServerStream[antv1.WatchActiveStrategiesEvent],
) error {
	uid, err := userIDRequire(ctx)
	if err != nil {
		return err
	}
	if s.sessionRegistry == nil {
		return stream.Send(&antv1.WatchActiveStrategiesEvent{})
	}

	accountFilter := req.Msg.GetAccountId()

	// F1: IDOR - check account ownership if accountFilter is provided.
	if accountFilter != "" {
		accountUUID, parseErr := uuid.Parse(accountFilter)
		if parseErr != nil {
			return connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("invalid account_id: %w", parseErr))
		}
		if err := s.checkBoundAccount(ctx, uid, accountUUID); err != nil {
			if errors.Is(err, service.ErrAccountLimitExceeded) {
				return connect.NewError(connect.CodePermissionDenied, err)
			}
			if errors.Is(err, service.ErrAccountNotOwned) {
				return connect.NewError(connect.CodeNotFound, err)
			}
			return connect.NewError(connect.CodeInternal, err)
		}
	}

	tickFn := s.tickPriceFn()

	sendList := func() error {
		var sessions []*ActiveSession
		if accountFilter != "" {
			sessions = s.sessionRegistry.ListByAccount(accountFilter)
		} else {
			sessions = s.sessionRegistry.ListByUser(uid)
		}
		pbStrats := make([]*antv1.ActiveStrategy, 0, len(sessions))
		for _, sess := range sessions {
			pbStrats = append(pbStrats, activeSessionToProto(sess, tickFn))
		}
		s.enrichWithStrategyName(ctx, pbStrats)
		return stream.Send(&antv1.WatchActiveStrategiesEvent{Strategies: pbStrats})
	}

	// Send current state immediately.
	if err := sendList(); err != nil {
		return err
	}

	// Subscribe to registry changes.
	notifCh, cancelWatch := s.sessionRegistry.Watch()
	defer cancelWatch()

	// Subscribe to tick updates for real-time prices.
	var tickCh <-chan *mthub.TickUpdate
	var cancelTickWatch func()
	if s.mtHub != nil {
		tickCh, cancelTickWatch = s.mtHub.WatchAllTicks()
		defer cancelTickWatch()
	}

	// Throttle tick-driven updates to 500ms to avoid flooding the SSE stream.
	tickTimer := time.NewTimer(0)
	if !tickTimer.Stop() {
		<-tickTimer.C
	}
	defer tickTimer.Stop()

	// Heartbeat: send empty event periodically to keep SSE alive when no
	// session/tick changes occur. Prevents中间层 from closing idle streams.
	hbInterval := s.heartbeatInterval
	if hbInterval <= 0 {
		hbInterval = 20 * time.Second
	}
	heartbeat := time.NewTicker(hbInterval)
	defer heartbeat.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil
		case <-notifCh:
			if err := sendList(); err != nil {
				return err
			}
		case <-tickCh:
			// Drain any buffered ticks, then wait 500ms before sending.
			for {
				select {
				case <-tickCh:
					continue
				default:
				}
				break
			}
			tickTimer.Reset(500 * time.Millisecond)
		case <-tickTimer.C:
			if err := sendList(); err != nil {
				return err
			}
		case <-heartbeat.C:
			if err := stream.Send(&antv1.WatchActiveStrategiesEvent{Heartbeat: true}); err != nil {
				return err
			}
		}
	}
}

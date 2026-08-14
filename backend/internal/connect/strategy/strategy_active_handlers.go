// strategy_active_handlers.go — RPC handlers for active strategy monitoring + control.
package strategy

import (
	"context"
	"errors"
	"fmt"
	"sync/atomic"
	"time"

	"connectrpc.com/connect"
	"github.com/google/uuid"
	"go.uber.org/zap"
	"google.golang.org/protobuf/types/known/timestamppb"

	antv1 "alphaforge/gen/proto/ant/v1"
	"alphaforge/internal/mthub"
	"alphaforge/internal/repository"
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

// StopStrategy cancels a running strategy session by run ID.
func (s *StrategyExecutionServer) StopStrategy(ctx context.Context, req *connect.Request[antv1.StopStrategyRequest]) (*connect.Response[antv1.StopStrategyResponse], error) {
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
		return connect.NewResponse(&antv1.StopStrategyResponse{
			Success: false,
			Error:   fmt.Sprintf("active strategy %s not found", runID),
		}), nil
	}
	if sess.UserID != uid {
		return nil, connect.NewError(connect.CodePermissionDenied, fmt.Errorf("strategy %s not found", runID))
	}

	if err := s.sessionRegistry.Stop(runID); err != nil {
		return connect.NewResponse(&antv1.StopStrategyResponse{
			Success: false,
			Error:   err.Error(),
		}), nil
	}

	s.log.Info("StopStrategy: session cancelled", zap.String("run_id", runID.String()))
	return connect.NewResponse(&antv1.StopStrategyResponse{Success: true}), nil
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

// StartStrategy launches a new live or paper strategy run.
// This is the unified entry point replacing StartPaperStrategy on PaperTradingService.
func (s *StrategyExecutionServer) StartStrategy(ctx context.Context, req *connect.Request[antv1.StartStrategyRequest]) (*connect.Response[antv1.StartStrategyResponse], error) {
	uid, err := userIDRequire(ctx)
	if err != nil {
		return nil, err
	}
	if s.barSource == nil {
		return nil, connect.NewError(connect.CodeUnavailable, fmt.Errorf("strategy runner not configured"))
	}
	if s.gate == nil {
		return nil, connect.NewError(connect.CodeUnavailable, fmt.Errorf("risk gate not configured"))
	}
	if _, ok := s.barSource.(LiveBarSubscriber); !ok {
		return nil, connect.NewError(connect.CodeUnavailable, fmt.Errorf("bar source does not support streaming"))
	}

	mode := req.Msg.GetMode()
	if mode == "" {
		mode = "paper"
	}

	if err := s.checkStrategyQuota(ctx, uid, mode); err != nil {
		return nil, err
	}

	cfg := LiveStrategyConfig{
		AccountID:    req.Msg.GetAccountId(),
		Symbol:       req.Msg.GetSymbol(),
		Timeframe:    req.Msg.GetTimeframe(),
		Code:         req.Msg.GetStrategyCode(),
		Mode:         mode,
		Params:       req.Msg.GetParams(),
		UserID:       uid.String(),
		ExtraSymbols: req.Msg.GetExtraSymbols(),
		StrategyID:   req.Msg.GetStrategyId(),
		TickSeq:      new(atomic.Int64),
	}

	if err := s.resolveModeAndAccount(ctx, uid, mode, &cfg); err != nil {
		return nil, err
	}

	// LEAKAGE-1: Pre-check bound account before launching (user-facing error).
	// RunLiveStrategy also checks (non-bypassable), but this gives the user
	// a proper PermissionDenied error instead of a silent goroutine failure.
	if mode == modeLive && cfg.AccountID != "" {
		if accountUUID, parseErr := uuid.Parse(cfg.AccountID); parseErr == nil && accountUUID != uuid.Nil {
			if err := s.checkBoundAccount(ctx, uid, accountUUID); err != nil {
				if errors.Is(err, service.ErrAccountLimitExceeded) {
					return nil, connect.NewError(connect.CodePermissionDenied, err)
				}
				if errors.Is(err, service.ErrAccountNotOwned) {
					return nil, connect.NewError(connect.CodeNotFound, err)
				}
				return nil, connect.NewError(connect.CodeInternal, err)
			}
		}
	}

	runID, err := s.createStrategyRun(ctx, uid, cfg)
	if err != nil {
		return connect.NewResponse(&antv1.StartStrategyResponse{
			Success: false,
			Error:   err.Error(),
		}), nil
	}
	cfg.RunID = runID

	runCtx, cancel := context.WithCancel(context.Background())
	if s.sessionRegistry != nil && runID != uuid.Nil {
		sess := s.sessionRegistry.Register(runID, uid, cfg.AccountID, cfg.Symbol, cfg.Timeframe, cfg.Mode, cfg.ScheduleID, cancel)
		if sess != nil {
			cfg.PreRegisteredSession = sess
		}
	}

	go func() {
		defer cancel()
		s.log.Info("StartStrategy: launching LiveStrategyRunner",
			zap.String("user", uid.String()),
			zap.String("account", cfg.AccountID),
			zap.String("symbol", cfg.Symbol),
			zap.String("timeframe", cfg.Timeframe),
			zap.String("mode", cfg.Mode),
			zap.String("run_id", runID.String()),
		)
		if err := s.RunLiveStrategy(runCtx, cfg); err != nil {
			s.log.Warn("StartStrategy: LiveStrategyRunner exited with error",
				zap.String("account", cfg.AccountID),
				zap.Error(err),
			)
		}
	}()

	return connect.NewResponse(&antv1.StartStrategyResponse{
		Success: true,
		RunId: func() string {
			if runID != uuid.Nil {
				return runID.String()
			}
			return ""
		}(),
	}), nil
}

func (s *StrategyExecutionServer) checkStrategyQuota(ctx context.Context, uid uuid.UUID, mode string) error {
	if s.quotaChecker == nil || s.runRepo == nil {
		return nil
	}
	activeCount, _ := s.runRepo.CountActiveByUser(ctx, uid)
	if !s.quotaChecker.CheckStrategyLimit(uid, activeCount) {
		return connect.NewError(connect.CodeResourceExhausted,
			fmt.Errorf("strategy limit reached for your plan (%d active)", activeCount))
	}
	if mode == modeLive {
		liveCount, _ := s.runRepo.CountActiveLiveByUser(ctx, uid)
		if !s.quotaChecker.CheckLiveStrategyLimit(uid, liveCount) {
			return connect.NewError(connect.CodeResourceExhausted,
				fmt.Errorf("live strategy limit reached for your plan"))
		}
	}
	return nil
}

func (s *StrategyExecutionServer) resolveModeAndAccount(ctx context.Context, uid uuid.UUID, mode string, cfg *LiveStrategyConfig) error {
	// If the caller already selected an account (panel/schedule), it is the
	// single source of truth — both trading and bar source follow it.
	if cfg.AccountID != "" {
		if accountUUID, parseErr := uuid.Parse(cfg.AccountID); parseErr == nil && accountUUID != uuid.Nil {
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
		cfg.DataSourceAccountID = cfg.AccountID
		return nil
	}
	// Fallback: auto-select a connected MT account via accountLookup.
	if s.accountLookup == nil {
		if mode == modeLive {
			return connect.NewError(connect.CodeUnavailable, fmt.Errorf("account lookup not configured"))
		}
		return nil
	}
	mt4ID := s.accountLookup(ctx, uid.String())
	if mt4ID == "" {
		if mode == modeLive {
			return connect.NewError(connect.CodeFailedPrecondition,
				fmt.Errorf("no connected MT account found — please bind an MT account before starting a live strategy"))
		}
		return nil
	}
	cfg.DataSourceAccountID = mt4ID
	cfg.AccountID = mt4ID
	return nil
}

func (s *StrategyExecutionServer) createStrategyRun(ctx context.Context, uid uuid.UUID, cfg LiveStrategyConfig) (uuid.UUID, error) {
	if s.runRepo == nil {
		return uuid.Nil, fmt.Errorf("run repository not configured")
	}
	run := &repository.StrategyRun{
		UserID:       uid,
		AccountID:    cfg.AccountID,
		Symbol:       cfg.Symbol,
		Timeframe:    cfg.Timeframe,
		Mode:         cfg.Mode,
		StrategyCode: cfg.Code,
		Status:       "running",
	}
	if err := s.runRepo.Create(ctx, run); err != nil {
		s.log.Error("StartStrategy: failed to create run record", zap.Error(err))
		return uuid.Nil, fmt.Errorf("failed to create run record: %w", err)
	}
	return run.ID, nil
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

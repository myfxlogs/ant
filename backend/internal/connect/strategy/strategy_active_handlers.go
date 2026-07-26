// strategy_active_handlers.go — RPC handlers for active strategy monitoring + control.
package strategy

import (
	"context"
	"fmt"

	"connectrpc.com/connect"
	"github.com/google/uuid"
	"go.uber.org/zap"
	"google.golang.org/protobuf/types/known/timestamppb"

	antv1 "alphaforge/gen/proto/ant/v1"
	"alphaforge/internal/repository"
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
	if req.Msg.GetAccountId() != "" {
		sessions = s.sessionRegistry.ListByAccount(req.Msg.GetAccountId())
	} else {
		sessions = s.sessionRegistry.ListByUser(uid)
	}

	pbStrategies := make([]*antv1.ActiveStrategy, len(sessions))
	for i, sess := range sessions {
		pbStrategies[i] = activeSessionToProto(sess)
	}
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

	return connect.NewResponse(&antv1.GetActiveStrategyResponse{Strategy: activeSessionToProto(sess)}), nil
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
		}
	}
}

// activeSessionToProto converts an ActiveSession to a proto ActiveStrategy.
func activeSessionToProto(sess *ActiveSession) *antv1.ActiveStrategy {
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
	}
	if !sess.LastSignalAt.IsZero() {
		pb.LastSignalAt = timestamppb.New(sess.LastSignalAt)
	}
	return pb
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

	// P3.2: Enforce subscription plan strategy limits.
	if s.quotaChecker != nil && s.runRepo != nil {
		activeCount, _ := s.runRepo.CountActiveByUser(ctx, uid)
		if !s.quotaChecker.CheckStrategyLimit(uid, activeCount) {
			return nil, connect.NewError(connect.CodeResourceExhausted,
				fmt.Errorf("strategy limit reached for your plan (%d active)", activeCount))
		}
		if mode == "live" {
			liveCount, _ := s.runRepo.CountActiveLiveByUser(ctx, uid)
			if !s.quotaChecker.CheckLiveStrategyLimit(uid, liveCount) {
				return nil, connect.NewError(connect.CodeResourceExhausted,
					fmt.Errorf("live strategy limit reached for your plan"))
			}
		}
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
	}

	// Live mode: use MT4 account ID for order routing.
	if mode == "live" {
		if s.accountLookup != nil {
			if mt4ID := s.accountLookup(ctx, uid.String()); mt4ID != "" {
				cfg.DataSourceAccountID = mt4ID
				cfg.AccountID = mt4ID // route orders to real MT4 account
			}
		}
	} else {
		// Paper mode: use linked MT4 account for bar data subscription only.
		if s.accountLookup != nil {
			if mt4ID := s.accountLookup(ctx, uid.String()); mt4ID != "" {
				cfg.DataSourceAccountID = mt4ID
			}
		}
	}

	// Pre-create the run record synchronously so we can return run_id immediately.
	runID := uuid.Nil
	if s.runRepo != nil {
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
			return connect.NewResponse(&antv1.StartStrategyResponse{
				Success: false,
				Error:   "failed to create run record: " + err.Error(),
			}), nil
		}
		runID = run.ID
		cfg.RunID = runID
	} else {
		return connect.NewResponse(&antv1.StartStrategyResponse{
			Success: false,
			Error:   "run repository not configured",
		}), nil
	}

	// Synchronously register session — atomic conflict detection.
	// If account already has a running session, Register returns nil.
	runCtx, cancel := context.WithCancel(context.Background())
	if s.sessionRegistry != nil && runID != uuid.Nil {
		sess := s.sessionRegistry.Register(runID, uid, cfg.AccountID, cfg.Symbol, cfg.Timeframe, cfg.Mode, cancel)
		if sess == nil {
			cancel()
			// Mark the pre-created run record as error.
			if s.runRepo != nil {
				_ = s.runRepo.UpdateStopped(context.Background(), runID, "error", "duplicate strategy for account")
			}
			return connect.NewResponse(&antv1.StartStrategyResponse{
				Success: false,
				Error:   fmt.Sprintf("strategy already running for account %s", cfg.AccountID),
			}), nil
		}
		cfg.PreRegisteredSession = sess
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

// WatchActiveStrategies streams the active strategy list via SSE.
// Sends the current list immediately, then pushes updates whenever
// sessions are registered, deregistered, or change state.
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

	sendList := func() error {
		var sessions []*ActiveSession
		if accountFilter != "" {
			sessions = s.sessionRegistry.ListByAccount(accountFilter)
		} else {
			sessions = s.sessionRegistry.ListByUser(uid)
		}
		pbStrats := make([]*antv1.ActiveStrategy, 0, len(sessions))
		for _, sess := range sessions {
			pbStrats = append(pbStrats, activeSessionToProto(sess))
		}
		return stream.Send(&antv1.WatchActiveStrategiesEvent{Strategies: pbStrats})
	}

	// Send current state immediately.
	if err := sendList(); err != nil {
		return err
	}

	// Subscribe to registry changes.
	notifCh, cancelWatch := s.sessionRegistry.Watch()
	defer cancelWatch()

	for {
		select {
		case <-ctx.Done():
			return nil
		case <-notifCh:
			if err := sendList(); err != nil {
				return err
			}
		}
	}
}

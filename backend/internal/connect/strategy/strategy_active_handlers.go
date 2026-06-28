// strategy_active_handlers.go — RPC handlers for active strategy monitoring + control.
package strategy

import (
	"context"
	"fmt"
	"time"

	"connectrpc.com/connect"
	"github.com/google/uuid"
	"go.uber.org/zap"
	"google.golang.org/protobuf/types/known/timestamppb"

	antv1 "anttrader/gen/proto/ant/v1"
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
	if _, err := userIDRequire(ctx); err != nil {
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

	return connect.NewResponse(&antv1.GetActiveStrategyResponse{Strategy: activeSessionToProto(sess)}), nil
}

// StopStrategy cancels a running strategy session by run ID.
func (s *StrategyExecutionServer) StopStrategy(ctx context.Context, req *connect.Request[antv1.StopStrategyRequest]) (*connect.Response[antv1.StopStrategyResponse], error) {
	if _, err := userIDRequire(ctx); err != nil {
		return nil, err
	}
	if s.sessionRegistry == nil {
		return nil, connect.NewError(connect.CodeUnavailable, fmt.Errorf("session registry not configured"))
	}

	runID, err := uuid.Parse(req.Msg.GetRunId())
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("invalid run_id: %w", err))
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
	if _, err := userIDRequire(ctx); err != nil {
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

	mode := req.Msg.GetMode()
	if mode == "" {
		mode = "paper"
	}

	cfg := LiveStrategyConfig{
		AccountID: req.Msg.GetAccountId(),
		Symbol:    req.Msg.GetSymbol(),
		Timeframe: req.Msg.GetTimeframe(),
		Code:      req.Msg.GetStrategyCode(),
		Mode:      mode,
		Params:    req.Msg.GetParams(),
		UserID:    uid.String(),
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

	runCtx, cancel := context.WithCancel(context.Background())

	go func() {
		defer cancel()
		s.log.Info("StartStrategy: launching LiveStrategyRunner",
			zap.String("user", uid.String()),
			zap.String("account", cfg.AccountID),
			zap.String("symbol", cfg.Symbol),
			zap.String("timeframe", cfg.Timeframe),
			zap.String("mode", cfg.Mode),
		)
		if err := s.RunLiveStrategy(runCtx, cfg); err != nil {
			s.log.Warn("StartStrategy: LiveStrategyRunner exited with error",
				zap.String("account", cfg.AccountID),
				zap.Error(err),
			)
		}
	}()

	// Wait briefly for the run record to be created so we can return the run_id.
	// RunLiveStrategy creates the run record synchronously before entering the event loop.
	runID := ""
	for i := 0; i < 10; i++ {
		if cfg.RunID != uuid.Nil {
			runID = cfg.RunID.String()
			break
		}
		select {
		case <-ctx.Done():
			break
		case <-time.After(50 * time.Millisecond):
		}
	}

	return connect.NewResponse(&antv1.StartStrategyResponse{
		Success: true,
		RunId:   runID,
	}), nil
}

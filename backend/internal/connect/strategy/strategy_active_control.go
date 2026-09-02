// strategy_active_control.go — RPC handlers for active strategy control (start / stop / resolve / quota).
package strategy

import (
	"context"
	"errors"
	"fmt"
	"sync/atomic"

	"connectrpc.com/connect"
	"github.com/google/uuid"
	"go.uber.org/zap"

	antv1 "alphaforge/gen/proto/ant/v1"
	"alphaforge/internal/repository"
	"alphaforge/internal/service"
)

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
		sess := s.sessionRegistry.Register(runID, uid, cfg.AccountID, cfg.Symbol, cfg.Timeframe, cfg.Mode, cfg.ScheduleID, cfg.StrategyID, cancel)
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

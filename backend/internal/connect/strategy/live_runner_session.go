// live_runner_session.go — active session lifecycle helpers for the live strategy runner.
package strategy

import (
	"context"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"go.uber.org/zap"

	"alphaforge/strategy/backtest"
)

func (s *StrategyExecutionServer) registerLiveSession(cfg *LiveStrategyConfig, runID uuid.UUID, runCancel func(), cleanupOrphan func(string)) *ActiveSession {
	activeSess := cfg.PreRegisteredSession
	uid, _ := uuid.Parse(cfg.UserID)
	if activeSess == nil && s.sessionRegistry != nil {
		activeSess = s.sessionRegistry.Register(runID, uid, cfg.AccountID, cfg.Symbol, cfg.Timeframe, cfg.Mode, cfg.ScheduleID, runCancel)
		if activeSess == nil {
			cleanupOrphan("session registration failed")
			return nil
		}
	}
	if activeSess != nil {
		s.log.Info("LiveStrategyRunner: session registered", zap.String("run_id", runID.String()))
		// OMS-EXIT-FIX Task 4: the live goroutine is the authority — if the DB
		// row was wrongly marked stopped (e.g. a transient second instance ran
		// CleanupStaleRuns), flip it back to running on register.
		if s.runRepo != nil {
			if err := s.runRepo.MarkRunning(context.Background(), runID); err != nil {
				s.log.Warn("LiveStrategyRunner: MarkRunning failed", zap.String("run_id", runID.String()), zap.Error(err))
			}
		}
		if s.sessionRegistry != nil {
			s.sessionRegistry.InsertScheduleRunLog(context.Background(), uid, cfg.ScheduleID,
				"start", "register", "success", "", "", decimal.Zero)
		}
	}
	return activeSess
}

func (s *StrategyExecutionServer) setupShadowVerifier(runCtx context.Context, cfg *LiveStrategyConfig) {
	if cfg.ShadowVerifier == nil && cfg.Code != "" {
		btCfg := backtest.Config{
			Symbol:         cfg.Symbol,
			Timeframe:      cfg.Timeframe,
			Params:         cfg.Params,
			InitialCapital: decimal.NewFromInt(10000),
		}
		cfg.ShadowVerifier = NewShadowVerifier(cfg.Code, btCfg, s.log)
	}
	if cfg.ShadowVerifier != nil {
		cfg.ShadowVerifier.Start(runCtx)
	}
}

func (s *StrategyExecutionServer) cleanupLiveSession(runID uuid.UUID, cfg LiveStrategyConfig, activeSess *ActiveSession) {
	if cfg.ShadowVerifier != nil {
		cfg.ShadowVerifier.Stop()
	}
	if s.sessionRegistry != nil && runID != uuid.Nil {
		s.sessionRegistry.Deregister(runID)
	}
	if s.runRepo != nil && runID != uuid.Nil {
		status := "stopped"
		if activeSess != nil && activeSess.ErrorCount > 0 {
			status = "error"
		}
		errMsg := ""
		if activeSess != nil {
			errMsg = activeSess.LastError
		}
		if err := s.runRepo.UpdateStopped(context.Background(), runID, status, errMsg); err != nil {
			s.log.Warn("LiveStrategyRunner: failed to update run record on stop", zap.Error(err))
		}
		uid, _ := uuid.Parse(cfg.UserID)
		if s.sessionRegistry != nil {
			s.sessionRegistry.InsertScheduleRunLog(context.Background(), uid, cfg.ScheduleID,
				"complete", "cleanup", status, errMsg, "", decimal.Zero)
		}
	}
}

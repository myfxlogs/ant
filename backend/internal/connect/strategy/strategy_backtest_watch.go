package strategy

import (
	"context"
	"fmt"
	"time"

	"connectrpc.com/connect"
	"github.com/google/uuid"
	"go.uber.org/zap"

	antv1 "alphaforge/gen/proto/ant/v1"
)

func (s *StrategyExecutionServer) WatchBacktestRun(ctx context.Context, req *connect.Request[antv1.WatchBacktestRunRequest], stream *connect.ServerStream[antv1.BacktestRunUpdate]) error {
	runID, err := uuid.Parse(req.Msg.RunId)
	if err != nil {
		return err
	}
	userID, err := userIDRequire(ctx)
	if err != nil {
		return err
	}

	// Send current state immediately so the client doesn't time out waiting
	// for the first ticker/notification (which can take up to 30s).
	run, err := s.backtestRepo.GetByID(ctx, userID, runID)
	if err != nil {
		return err
	}
	if run == nil {
		return connect.NewError(connect.CodeNotFound, fmt.Errorf("backtest run %s not found", runID))
	}
	bp := parseBacktestResult(run.ProtoResponse)
	if err := stream.Send(&antv1.BacktestRunUpdate{
		Run:                  toProtoBacktestRun(run),
		Metrics:              bp.Metrics,
		EquityCurve:          bp.EquityCurve,
		Risk:                 bp.Risk,
		ExecutionAssumptions: bp.ExecutionAssumptions,
		BlindSpots:           bp.BlindSpots,
	}); err != nil {
		return err
	}
	if isTerminalBacktestStatus(run.Status) {
		if run.Status == "SUCCEEDED" && run.AutoGate {
			restoreGateEvaluation(ctx, s, run, stream)
		}
		return nil
	}

	// Watch for status changes — LISTEN for push events, ticker as fallback.
	notifCh, listenCancel, _ := s.pgListen.Listen(ctx, "backtest_status")
	if listenCancel != nil {
		defer listenCancel()
	}
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()
	prevStatus := run.Status
	for {
		select {
		case <-ctx.Done():
			return nil
		case <-notifCh:
		case <-ticker.C:
		}
		run, err := s.backtestRepo.GetByID(ctx, userID, runID)
		if err != nil {
			s.log.Warn("WatchBacktestRun: transient DB error", zap.Error(err), zap.String("runID", runID.String()))
			continue
		}
		if run == nil {
			s.log.Warn("WatchBacktestRun: run not found", zap.String("runID", runID.String()))
			continue
		}
		if run.Status == prevStatus {
			continue
		}
		prevStatus = run.Status
		bp := parseBacktestResult(run.ProtoResponse)
		if err := stream.Send(&antv1.BacktestRunUpdate{
			Run:                  toProtoBacktestRun(run),
			Metrics:              bp.Metrics,
			EquityCurve:          bp.EquityCurve,
			Risk:                 bp.Risk,
			ExecutionAssumptions: bp.ExecutionAssumptions,
			BlindSpots:           bp.BlindSpots,
		}); err != nil {
			return err
		}
		if isTerminalBacktestStatus(run.Status) {
			if run.Status == "SUCCEEDED" && run.AutoGate {
				restoreGateEvaluation(ctx, s, run, stream)
			}
			return nil
		}
	}
}

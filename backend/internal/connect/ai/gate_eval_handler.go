package ai

import (
	"context"
	"fmt"

	"google.golang.org/protobuf/proto"

	"github.com/google/uuid"
	"go.uber.org/zap"

	antv1 "anttrader/gen/proto/ant/v1"
	antv1c "anttrader/gen/proto/ant/v1/antv1connect"

	aigates "anttrader/internal/ai"
	"anttrader/internal/interceptor"
	"anttrader/internal/notification"
	"anttrader/internal/repository"

	"connectrpc.com/connect"
)

// GateEvalServer handles ConnectRPC server-streaming gate evaluation.
type GateEvalServer struct {
	backtestRepo *repository.BacktestRunRepository
	log          *zap.Logger
	notifSender  *notification.Sender
}

var _ antv1c.GateServiceHandler = (*GateEvalServer)(nil)

// NewGateEvalServer creates a GateEvalServer.
func NewGateEvalServer(backtestRepo *repository.BacktestRunRepository, log *zap.Logger) *GateEvalServer {
	return &GateEvalServer{backtestRepo: backtestRepo, log: log}
}

// SetNotificationSender injects the notification sender for gate result events.
func (s *GateEvalServer) SetNotificationSender(ns *notification.Sender) { s.notifSender = ns }

// RunEvaluation fetches the backtest run, converts equity_curve to daily returns,
// runs the 6-gate pipeline, and streams each gate result via SSE.
func (s *GateEvalServer) RunEvaluation(
	ctx context.Context,
	req *connect.Request[antv1.RunGateEvaluationRequest],
	stream *connect.ServerStream[antv1.GateEvaluationUpdate],
) error {
	userID, err := uuid.Parse(interceptor.GetUserID(ctx))
	if err != nil {
		return fmt.Errorf("unauthorized: %w", err)
	}
	run, err := s.fetchRun(ctx, userID, req.Msg.BacktestRunId)
	if err != nil {
		return err
	}
	input, err := buildGateInput(run, req.Msg)
	if err != nil {
		return err
	}
	result, err := streamGateResults(ctx, input, stream)
	if err != nil {
		return err
	}

	// Emit notification for gate evaluation result.
	if s.notifSender != nil {
		if result.Passed {
			_, _ = s.notifSender.Send(ctx, userID, "gate_passed",
				fmt.Sprintf("Gate Evaluation Passed: %s", run.Symbol),
				fmt.Sprintf("Strategy for %s passed all %d gates", run.Symbol, len(result.Gates)),
				fmt.Sprintf(`{"run_id":"%s","symbol":"%s","gates":%d}`, run.ID, run.Symbol, len(result.Gates)))
		} else {
			firstFail := string(result.FirstFail)
			if firstFail == "" {
				firstFail = "unknown"
			}
			_, _ = s.notifSender.Send(ctx, userID, "gate_failed",
				fmt.Sprintf("Gate Evaluation Failed: %s", run.Symbol),
				fmt.Sprintf("Strategy for %s failed at gate: %s", run.Symbol, firstFail),
				fmt.Sprintf(`{"run_id":"%s","symbol":"%s","failed_at":"%s"}`, run.ID, run.Symbol, firstFail))
		}
	}
	return nil
}

// fetchRun fetches the backtest run and validates it is completed.
func (s *GateEvalServer) fetchRun(ctx context.Context, userID uuid.UUID, rawID string) (*repository.BacktestRun, error) {
	runID, err := uuid.Parse(rawID)
	if err != nil {
		return nil, fmt.Errorf("invalid backtest_run_id: %w", err)
	}
	run, err := s.backtestRepo.GetByID(ctx, userID, runID)
	if err != nil {
		return nil, fmt.Errorf("backtest run not found: %w", err)
	}
	if run.Status != "SUCCEEDED" {
		return nil, fmt.Errorf("backtest run not completed (status: %s)", run.Status)
	}
	return run, nil
}

// buildGateInput converts a backtest run into pipeline input.
func buildGateInput(run *repository.BacktestRun, req *antv1.RunGateEvaluationRequest) (aigates.PipelineInput, error) {
	dailyReturns := equityCurveToDailyReturns(run.ProtoResponse)
	if len(dailyReturns) < 10 {
		return aigates.PipelineInput{},
			fmt.Errorf("insufficient data: need 10+ daily returns, got %d", len(dailyReturns))
	}
	n := int(req.NumAttempts)
	if n <= 0 {
		n = 1
	}
	return aigates.PipelineInput{
		Expression:   req.Expression,
		DailyReturns: dailyReturns,
		NumAttempts:  n,
		PaperMetrics: aigates.PaperGateMetrics{
			PaperDays:         int(req.PaperDays),
			PaperNetPnL:       req.PaperNetPnl,
			PaperNetReturn:    req.PaperNetReturn,
			BacktestNetReturn: req.BacktestNetReturn,
			PaperTradeCount:   int(req.PaperTradeCount),
		},
	}, nil
}

// streamGateResults runs the pipeline and streams each gate result + summary.
// Returns the pipeline result for downstream notification emission.
func streamGateResults(
	ctx context.Context,
	input aigates.PipelineInput,
	stream *connect.ServerStream[antv1.GateEvaluationUpdate],
) (*aigates.PipelineResult, error) {
	result := aigates.Pipeline(input)
	for _, gate := range result.Gates {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}
		if err := stream.Send(&antv1.GateEvaluationUpdate{
			Gate: toProtoGateResult(gate),
		}); err != nil {
			return nil, err
		}
	}
	if err := stream.Send(&antv1.GateEvaluationUpdate{
		Completed: toProtoSummary(result),
	}); err != nil {
		return nil, err
	}
	return &result, nil
}

func toProtoGateResult(g aigates.GateStatus) *antv1.GateResult {
	return &antv1.GateResult{
		Gate: string(g.Gate), Passed: g.Passed,
		Skipped: g.Skipped, Reason: g.Reason,
		Score: g.Score, DurationMs: g.Duration,
	}
}

func toProtoSummary(r aigates.PipelineResult) *antv1.GatePipelineSummary {
	return &antv1.GatePipelineSummary{
		Passed: r.Passed, FirstFail: string(r.FirstFail),
		Summary: r.Summary, TotalDurationMs: r.TotalDuration,
	}
}

// equityCurveToDailyReturns extracts equity curve from proto binary ExecuteBacktestResponse.
func equityCurveToDailyReturns(protoResp []byte) []float64 {
	if len(protoResp) == 0 {
		return nil
	}
	var resp antv1.ExecuteBacktestResponse
	if err := proto.Unmarshal(protoResp, &resp); err != nil {
		return nil
	}
	equity := resp.GetEquityCurve()
	if len(equity) < 2 {
		return nil
	}
	rets := make([]float64, len(equity)-1)
	for i := 1; i < len(equity); i++ {
		rets[i-1] = equity[i] - equity[i-1]
	}
	return rets
}

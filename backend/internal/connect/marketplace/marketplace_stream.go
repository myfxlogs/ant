package marketplace

import (
	"context"
	"fmt"
	"strings"
	"time"

	"connectrpc.com/connect"
	"github.com/google/uuid"
	"go.uber.org/zap"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"

	antv1 "anttrader/gen/proto/ant/v1"
)

func statusToEnum(s string) antv1.BacktestRunStatus {
	switch s {
	case "PENDING":
		return antv1.BacktestRunStatus_BACKTEST_RUN_STATUS_PENDING
	case "RUNNING":
		return antv1.BacktestRunStatus_BACKTEST_RUN_STATUS_RUNNING
	case "SUCCEEDED":
		return antv1.BacktestRunStatus_BACKTEST_RUN_STATUS_SUCCEEDED
	case "FAILED":
		return antv1.BacktestRunStatus_BACKTEST_RUN_STATUS_FAILED
	case "CANCEL_REQUESTED":
		return antv1.BacktestRunStatus_BACKTEST_RUN_STATUS_CANCEL_REQUESTED
	case "CANCELED":
		return antv1.BacktestRunStatus_BACKTEST_RUN_STATUS_CANCELED
	default:
		return antv1.BacktestRunStatus_BACKTEST_RUN_STATUS_UNSPECIFIED
	}
}

const statusNotifyChannel = "backtest_status_change"

// streamBacktestProgress pushes BacktestRunUpdate messages via server stream.
// Uses PG LISTEN/NOTIFY for push-first updates, with a 30s polling fallback.
func (s *MarketplaceServer) streamBacktestProgress(ctx context.Context, runID uuid.UUID, stream *connect.ServerStream[antv1.BacktestRunUpdate]) error {
	// Push initial state immediately.
	if terminal, err := s.pollAndSend(ctx, stream, runID); err != nil || terminal {
		return err
	}

	// Try PG LISTEN for push-first updates.
	notifCh, listenCancel, _ := s.pgListen.Listen(ctx, statusNotifyChannel)
	if listenCancel != nil {
		defer listenCancel()

		// 30s fallback ticker in case notification is missed.
		fallback := time.NewTicker(30 * time.Second)
		defer fallback.Stop()

		for {
			select {
			case <-ctx.Done():
				return nil
			case raw, ok := <-notifCh:
				if !ok {
					return nil // connection closed
				}
				// Payload format: "runID,status"
				if !matchRun(raw, runID) {
					continue
				}
			case <-fallback.C:
				// Safety net.
			}

			terminal, err := s.pollAndSend(ctx, stream, runID)
			if err != nil {
				return err
			}
			if terminal {
				return nil
			}
		}
	}

	// Fallback: pure polling if pgListen is unavailable.
	s.log.Warn("marketplace stream: pgListen unavailable, falling back to polling",
		zap.String("runID", runID.String()))
	ticker := time.NewTicker(800 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			terminal, err := s.pollAndSend(ctx, stream, runID)
			if err != nil {
				return err
			}
			if terminal {
				return nil
			}
		}
	}
}

// matchRun checks whether a notification payload belongs to the given runID.
// Payload format: "runID,status"
func matchRun(payload string, target uuid.UUID) bool {
	parts := strings.SplitN(payload, ",", 2)
	if len(parts) == 0 {
		return false
	}
	rid, err := uuid.Parse(parts[0])
	if err != nil {
		return false
	}
	return rid == target
}

func (s *MarketplaceServer) pollAndSend(ctx context.Context, stream *connect.ServerStream[antv1.BacktestRunUpdate], runID uuid.UUID) (bool, error) {
	row, err := s.svc.QueryBacktestRun(ctx, runID)
	if err != nil {
		return false, fmt.Errorf("query backtest run: %w", err)
	}

	update := &antv1.BacktestRunUpdate{
		Run: &antv1.BacktestRun{
			Id:         runID.String(),
			Symbol:     row.Symbol,
			Timeframe:  row.Timeframe,
			Status:     statusToEnum(row.Status),
			Error:      row.Error,
			IsTerminal: terminalStatus(row.Status),
		},
	}
	if row.TemplateID != nil {
		update.Run.TemplateId = proto.String(row.TemplateID.String())
	}
	if row.StartedAt != nil {
		update.Run.StartedAt = timestamppb.New(*row.StartedAt)
	}
	if row.FinishedAt != nil {
		update.Run.FinishedAt = timestamppb.New(*row.FinishedAt)
	}

	// Parse metrics from stored proto_response on completion.
	if row.Status == "SUCCEEDED" && len(row.ProtoResponse) > 0 {
		var result antv1.ExecuteBacktestResponse
		if err := proto.Unmarshal(row.ProtoResponse, &result); err == nil {
			if m := result.GetMetrics(); m != nil {
				update.Metrics = &antv1.BacktestMetrics{
					TotalReturn:   m.GetTotalReturn(),
					AnnualReturn:  m.GetAnnualReturn(),
					MaxDrawdown:   m.GetMaxDrawdown(),
					SharpeRatio:   m.GetSharpeRatio(),
					WinRate:       m.GetWinRate(),
					ProfitFactor:  m.GetProfitFactor(),
					TotalTrades:   m.GetTotalTrades(),
					WinningTrades: m.GetWinningTrades(),
					LosingTrades:  m.GetLosingTrades(),
					AverageProfit: m.GetAverageProfit(),
					AverageLoss:   m.GetAverageLoss(),
				}
			}
			update.EquityCurve = result.GetEquityCurve()
			if r := result.GetRisk(); r != nil {
				update.Risk = &antv1.BacktestRisk{
					Score:      r.GetScore(),
					Level:      r.GetLevel(),
					Reasons:    r.GetReasons(),
					Warnings:   r.GetWarnings(),
					IsReliable: r.GetIsReliable(),
				}
			}
		}
	}

	if err := stream.Send(update); err != nil {
		return false, err
	}
	return terminalStatus(row.Status), nil
}

func terminalStatus(s string) bool {
	switch s {
	case "SUCCEEDED", "FAILED", "CANCELED":
		return true
	}
	return false
}

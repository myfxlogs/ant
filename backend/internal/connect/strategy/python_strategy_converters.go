package strategy

import (

	"github.com/google/uuid"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"

	antv1 "anttrader/gen/proto/ant/v1"
	"anttrader/internal/repository"
)

func toProtoBacktestRun(r *repository.BacktestRun) *antv1.BacktestRun {
	if r == nil {
		return nil
	}
	out := &antv1.BacktestRun{
		Id:        r.ID.String(),
		Symbol:    r.Symbol,
		Timeframe: r.Timeframe,
		Mode:      stringToBacktestMode(r.Mode),
		Status:    backtestStatusToProto(r.Status),
		Error:     r.Error,
	}
	if r.AccountID != uuid.Nil {
		out.AccountId = r.AccountID.String()
	}
	if r.StartedAt != nil {
		out.StartedAt = timestamppb.New(*r.StartedAt)
	}
	if r.FinishedAt != nil {
		out.FinishedAt = timestamppb.New(*r.FinishedAt)
	}
	out.CreatedAt = timestamppb.New(r.CreatedAt)
	return out
}

func backtestStatusToProto(s string) antv1.BacktestRunStatus {
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

func backtestModeToString(m antv1.BacktestRunMode) string {
	switch m {
	case antv1.BacktestRunMode_BACKTEST_RUN_MODE_DATASET:
		return "DATASET"
	default:
		return "KLINE_RANGE"
	}
}

func stringToBacktestMode(s string) antv1.BacktestRunMode {
	switch s {
	case "DATASET":
		return antv1.BacktestRunMode_BACKTEST_RUN_MODE_DATASET
	default:
		return antv1.BacktestRunMode_BACKTEST_RUN_MODE_KLINE_RANGE
	}
}

func parseProtoResponse(raw []byte) *antv1.ExecuteBacktestResponse {
	if len(raw) == 0 {
		return nil
	}
	var resp antv1.ExecuteBacktestResponse
	if err := proto.Unmarshal(raw, &resp); err != nil {
		return nil
	}
	return &resp
}

func parseMetrics(raw []byte) *antv1.BacktestMetrics {
	resp := parseProtoResponse(raw)
	if resp == nil || !resp.GetSuccess() {
		return nil
	}
	m := resp.GetMetrics()
	return &antv1.BacktestMetrics{
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

func parseRisk(raw []byte) *antv1.BacktestRisk {
	resp := parseProtoResponse(raw)
	if resp == nil || !resp.GetSuccess() {
		return nil
	}
	r := resp.GetRisk()
	return &antv1.BacktestRisk{
		Score:      int32(r.GetScore()),
		Level:      r.GetLevel(),
		Reasons:    r.GetReasons(),
		Warnings:   r.GetWarnings(),
		IsReliable: r.GetIsReliable(),
	}
}

func parseEquityCurve(raw []byte) []float64 {
	resp := parseProtoResponse(raw)
	if resp == nil {
		return nil
	}
	return resp.GetEquityCurve()
}

func strPtr(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

func f64Ptr(v float64) *float64 { return &v }

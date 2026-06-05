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
	if r.FromTs != nil {
		out.From = timestamppb.New(*r.FromTs)
	}
	if r.ToTs != nil {
		out.To = timestamppb.New(*r.ToTs)
	}
	if r.TemplateID != nil {
		out.TemplateId = proto.String(r.TemplateID.String())
	}
	if r.TemplateDraftID != nil {
		out.TemplateDraftId = proto.String(r.TemplateDraftID.String())
	}
	out.ExtraSymbols = r.ExtraSymbols
	if r.DatasetID != nil {
		out.DatasetId = proto.String(r.DatasetID.String())
	}
	out.IsTerminal = r.Status == "SUCCEEDED" || r.Status == "FAILED" || r.Status == "CANCELED"
	out.IsSucceeded = r.Status == "SUCCEEDED"
	// Deserialize config snapshot to proto.
	if len(r.ConfigSnapshot) > 0 {
		var ec antv1.BacktestExecutionConfig
		if err := proto.Unmarshal(r.ConfigSnapshot, &ec); err == nil {
			out.ExecutionConfig = &ec
		}
	}
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

func parseExecutionAssumptions(raw []byte) *antv1.ExecutionAssumptions {
	resp := parseProtoResponse(raw)
	if resp == nil {
		return nil
	}
	return resp.GetExecutionAssumptions()
}

func strPtr(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

func f64Ptr(v float64) *float64 { return &v }
func boolPtr(v bool) *bool { return &v }

// tradeDirectionToString converts proto TradeDirection enum to DB string.
func tradeDirectionToString(d antv1.TradeDirection) string {
	switch d {
	case antv1.TradeDirection_TRADE_DIRECTION_LONG:
		return "long"
	case antv1.TradeDirection_TRADE_DIRECTION_SHORT:
		return "short"
	case antv1.TradeDirection_TRADE_DIRECTION_BOTH:
		return "both"
	default:
		return "both"
	}
}

// stringToTradeDirection converts DB string to proto TradeDirection enum.
func stringToTradeDirection(s string) antv1.TradeDirection {
	switch s {
	case "long":
		return antv1.TradeDirection_TRADE_DIRECTION_LONG
	case "short":
		return antv1.TradeDirection_TRADE_DIRECTION_SHORT
	case "both":
		return antv1.TradeDirection_TRADE_DIRECTION_BOTH
	default:
		return antv1.TradeDirection_TRADE_DIRECTION_BOTH
	}
}

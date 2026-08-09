package strategy

import (
	"github.com/google/uuid"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"

	antv1 "alphaforge/gen/proto/ant/v1"
	"alphaforge/internal/repository"
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
	out.IsTerminal = isTerminalBacktestStatus(r.Status)
	out.IsSucceeded = r.Status == StatusSucceeded
	if r.StrategyID != nil {
		out.StrategyId = proto.String(r.StrategyID.String())
	}
	out.FixDepth = int32(r.FixDepth)
	if r.Name != nil {
		out.Name = proto.String(*r.Name)
	}
	// Deserialize config snapshot to proto.
	// DiscardUnknown prevents stale fields from older proto schemas
	// (e.g. commission/leverage as double) from being preserved as
	// unknown fields and re-serialized with wrong wire types, which
	// causes "premature EOF" in the JS protobuf parser.
	if len(r.ConfigSnapshot) > 0 {
		var ec antv1.BacktestExecutionConfig
		opts := proto.UnmarshalOptions{DiscardUnknown: true}
		if err := opts.Unmarshal(r.ConfigSnapshot, &ec); err == nil {
			out.ExecutionConfig = &ec
		}
	}
	return out
}

func backtestStatusToProto(s string) antv1.BacktestRunStatus {
	switch s {
	case StatusPending:
		return antv1.BacktestRunStatus_BACKTEST_RUN_STATUS_PENDING
	case StatusRunning:
		return antv1.BacktestRunStatus_BACKTEST_RUN_STATUS_RUNNING
	case StatusSucceeded:
		return antv1.BacktestRunStatus_BACKTEST_RUN_STATUS_SUCCEEDED
	case StatusFailed:
		return antv1.BacktestRunStatus_BACKTEST_RUN_STATUS_FAILED
	case StatusCancelRequested:
		return antv1.BacktestRunStatus_BACKTEST_RUN_STATUS_CANCEL_REQUESTED
	case StatusCanceled:
		return antv1.BacktestRunStatus_BACKTEST_RUN_STATUS_CANCELED
	case StatusDegraded:
		return antv1.BacktestRunStatus_BACKTEST_RUN_STATUS_DEGRADED
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

// backtestParsed holds the result of a single unmarshal of ExecuteBacktestResponse.
type backtestParsed struct {
	Metrics              *antv1.BacktestMetrics
	EquityCurve          []string
	Risk                 *antv1.BacktestRisk
	ExecutionAssumptions *antv1.ExecutionAssumptions
	BlindSpots           []*antv1.BacktestBlindSpot
}

// parseBacktestResult unmarshals the proto response once and extracts all fields.
// Replaces the previous pattern of calling parseMetrics + parseRisk + parseEquityCurve
// + parseExecutionAssumptions, each of which unmarshaled independently (4× overhead).
func parseBacktestResult(raw []byte) backtestParsed {
	resp := parseProtoResponse(raw)
	if resp == nil {
		return backtestParsed{}
	}
	p := backtestParsed{
		EquityCurve:          resp.GetEquityCurve(),
		ExecutionAssumptions: resp.GetExecutionAssumptions(),
	}
	if resp.GetSuccess() {
		m := resp.GetMetrics()
		p.Metrics = &antv1.BacktestMetrics{
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
		r := resp.GetRisk()
		p.Risk = &antv1.BacktestRisk{
			Score:      r.GetScore(),
			Level:      r.GetLevel(),
			Reasons:    r.GetReasons(),
			Warnings:   r.GetWarnings(),
			IsReliable: r.GetIsReliable(),
		}
	}
	// Extract blind spots (ADR-0028 Part B: diagnostic panel needs these in the watch stream).
	for _, bs := range resp.GetBlindSpots() {
		p.BlindSpots = append(p.BlindSpots, &antv1.BacktestBlindSpot{
			Id:          bs.GetId(),
			Description: bs.GetDescription(),
			Severity:    bs.GetSeverity(),
			Category:    bs.GetCategory(),
			Location:    bs.GetLocation(),
		})
	}
	return p
}

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

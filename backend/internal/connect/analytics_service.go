package connect

import (
	"context"
	"errors"
	"fmt"
	"time"

	"connectrpc.com/connect"
	"github.com/google/uuid"
	"google.golang.org/protobuf/types/known/timestamppb"

	v1 "anttrader/gen/proto"
	"anttrader/internal/model"
	"anttrader/internal/service"
)

type AnalyticsService struct {
	analyticsSvc *service.AnalyticsService
}

func NewAnalyticsService(analyticsSvc *service.AnalyticsService) *AnalyticsService {
	return &AnalyticsService{
		analyticsSvc: analyticsSvc,
	}
}

func (s *AnalyticsService) GetAccountAnalytics(ctx context.Context, req *connect.Request[v1.GetAccountAnalyticsRequest]) (*connect.Response[v1.AccountAnalytics], error) {
	accountID, err := uuid.Parse(req.Msg.AccountId)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}

	userID, err := getUserIDFromContext(ctx)
	if err != nil {
		return nil, err
	}

	now := time.Now()
	start := now.AddDate(-1, 0, 0)
	year := now.Year()

	analytics, err := s.analyticsSvc.GetAccountAnalytics(ctx, userID, accountID, start, now, year)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	return connect.NewResponse(convertAccountAnalyticsToProto(analytics)), nil
}

func (s *AnalyticsService) GetSummary(ctx context.Context, req *connect.Request[v1.GetSummaryRequest]) (*connect.Response[v1.Summary], error) {
	accountID, err := uuid.Parse(req.Msg.AccountId)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}

	userID, err := getUserIDFromContext(ctx)
	if err != nil {
		return nil, err
	}

	now := time.Now()
	start := now.AddDate(-1, 0, 0)

	tradeStats, err := s.analyticsSvc.GetTradeSummary(ctx, userID, accountID, start, now)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	riskMetrics, err := s.analyticsSvc.GetRiskMetrics(ctx, userID, accountID, start, now)
	if err != nil {
		riskMetrics = &model.RiskMetrics{}
	}

	summary := &v1.Summary{
		TotalTrades:  int32(tradeStats.TotalTrades),
		WinRate:      tradeStats.WinRate,
		ProfitFactor: tradeStats.ProfitFactor,
		SharpeRatio:  riskMetrics.SharpeRatio,
		MaxDrawdown:  riskMetrics.MaxDrawdown,
		TotalProfit:  tradeStats.NetProfit,
		TotalBalance: 0,
		TotalEquity:  0,
	}

	return connect.NewResponse(summary), nil
}

func (s *AnalyticsService) GetRiskMetrics(ctx context.Context, req *connect.Request[v1.GetRiskMetricsRequest]) (*connect.Response[v1.RiskMetrics], error) {
	accountID, err := uuid.Parse(req.Msg.AccountId)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}

	userID, err := getUserIDFromContext(ctx)
	if err != nil {
		return nil, err
	}

	now := time.Now()
	start := now.AddDate(-1, 0, 0)

	metrics, err := s.analyticsSvc.GetRiskMetrics(ctx, userID, accountID, start, now)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	return connect.NewResponse(convertRiskMetricsToProto(metrics)), nil
}

func (s *AnalyticsService) GetSymbolStats(ctx context.Context, req *connect.Request[v1.GetSymbolStatsRequest]) (*connect.Response[v1.SymbolStatsResponse], error) {
	accountID, err := uuid.Parse(req.Msg.AccountId)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}

	userID, err := getUserIDFromContext(ctx)
	if err != nil {
		return nil, err
	}

	now := time.Now()
	start := now.AddDate(-1, 0, 0)

	stats, err := s.analyticsSvc.GetSymbolStats(ctx, userID, accountID, start, now)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	response := &v1.SymbolStatsResponse{
		Stats: make([]*v1.SymbolStats, len(stats)),
	}
	totalTrades := 0
	for _, st := range stats {
		if st != nil {
			totalTrades += st.TotalTrades
		}
	}
	for i, st := range stats {
		response.Stats[i] = convertSymbolStatsToProtoWithShare(st, totalTrades)
	}

	return connect.NewResponse(response), nil
}

func (s *AnalyticsService) GetReport(ctx context.Context, req *connect.Request[v1.GetReportRequest]) (*connect.Response[v1.Report], error) {
	accountID, err := uuid.Parse(req.Msg.AccountId)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}

	userID, err := getUserIDFromContext(ctx)
	if err != nil {
		return nil, err
	}

	now := time.Now()
	var start time.Time
	switch req.Msg.Period {
	case "week":
		start = now.AddDate(0, 0, -7)
	case "month":
		start = now.AddDate(0, -1, 0)
	case "quarter":
		start = now.AddDate(0, -3, 0)
	case "year":
		start = now.AddDate(-1, 0, 0)
	default:
		start = now.AddDate(-1, 0, 0)
	}

	report, err := s.analyticsSvc.GetTradeReport(ctx, userID, accountID, start, now)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	return connect.NewResponse(convertTradeReportToProto(report, req.Msg.ReportType)), nil
}

func (s *AnalyticsService) GetRecentTrades(ctx context.Context, req *connect.Request[v1.GetRecentTradesRequest]) (*connect.Response[v1.RecentTradesResponse], error) {
	accountID, err := uuid.Parse(req.Msg.AccountId)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}

	userID, err := getUserIDFromContext(ctx)
	if err != nil {
		return nil, err
	}

	page := int(req.Msg.Page)
	if page <= 0 {
		page = 1
	}
	pageSize := int(req.Msg.PageSize)
	if pageSize <= 0 {
		pageSize = 10
	}

	trades, total, err := s.analyticsSvc.GetRecentTradesPaginated(ctx, userID, accountID, page, pageSize)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	response := &v1.RecentTradesResponse{
		Trades: make([]*v1.TradeRecord, len(trades)),
		Total:  int32(total),
	}
	for i, t := range trades {
		response.Trades[i] = convertTradeRecordToProto(t)
	}

	return connect.NewResponse(response), nil
}

func (s *AnalyticsService) GetMonthlyPnL(ctx context.Context, req *connect.Request[v1.GetMonthlyPnLRequest]) (*connect.Response[v1.MonthlyPnLResponse], error) {
	accountID, err := uuid.Parse(req.Msg.AccountId)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}

	userID, err := getUserIDFromContext(ctx)
	if err != nil {
		return nil, err
	}

	year := int(req.Msg.Year)
	if year <= 0 {
		year = time.Now().Year()
	}

	monthlyPnL, err := s.analyticsSvc.GetMonthlyPnL(ctx, userID, accountID, year)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	response := &v1.MonthlyPnLResponse{
		MonthlyPnl: make([]*v1.MonthlyPnL, len(monthlyPnL)),
	}
	for i, m := range monthlyPnL {
		response.MonthlyPnl[i] = convertMonthlyPnLToProto(m)
	}

	return connect.NewResponse(response), nil
}

func (s *AnalyticsService) GetMonthlyAnalysis(ctx context.Context, req *connect.Request[v1.GetMonthlyAnalysisRequest]) (*connect.Response[v1.MonthlyAnalysisResponse], error) {
	accountID, err := uuid.Parse(req.Msg.AccountId)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}

	userID, err := getUserIDFromContext(ctx)
	if err != nil {
		return nil, err
	}

	years, points, err := s.analyticsSvc.GetMonthlyAnalysis(ctx, userID, accountID)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	response := &v1.MonthlyAnalysisResponse{
		Years: make([]int32, 0, len(years)),
		Data:  make([]*v1.MonthlyAnalysisPoint, 0, len(points)),
	}

	for _, y := range years {
		response.Years = append(response.Years, int32(y))
	}

	for _, p := range points {
		response.Data = append(response.Data, &v1.MonthlyAnalysisPoint{
			Year:   int32(p.Year),
			Month:  int32(p.Month),
			Change: p.Change,
			Profit: p.Profit,
			Lots:   p.Lots,
			Pips:   p.Pips,
			Trades: int32(p.Trades),
		})
	}

	return connect.NewResponse(response), nil
}

func (s *AnalyticsService) GetMonthlyAnalysisBonus(ctx context.Context, req *connect.Request[v1.GetMonthlyAnalysisBonusRequest]) (*connect.Response[v1.MonthlyAnalysisBonusResponse], error) {
	accountID, err := uuid.Parse(req.Msg.AccountId)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}

	userID, err := getUserIDFromContext(ctx)
	if err != nil {
		return nil, err
	}

	year := int(req.Msg.Year)
	month := int(req.Msg.Month)
	if month < 1 || month > 12 {
		return nil, connect.NewError(connect.CodeInvalidArgument, errInvalidMonth)
	}
	if year < 1970 || year > 2100 {
		year = time.Now().Year()
	}

	bonus, err := s.analyticsSvc.GetMonthlyAnalysisBonus(ctx, userID, accountID, year, month)
	if err != nil {
		if errors.Is(err, service.ErrAccountNotFound) {
			return nil, connect.NewError(connect.CodeNotFound, err)
		}
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	resp := &v1.MonthlyAnalysisBonusResponse{
		RiskRatio:             bonus.RiskRatio,
		AverageHoldingSeconds: bonus.AvgHoldingSeconds,
		TotalTrades:           int32(bonus.TotalTrades),
		SymbolPopularity:      make([]*v1.MonthlySymbolPopularity, 0, len(bonus.Symbols)),
		SymbolRiskRatios:      make([]*v1.MonthlySymbolRiskRatio, 0, len(bonus.SymbolRisks)),
		SymbolHoldingSplit:    make([]*v1.MonthlySymbolHoldingSplit, 0, len(bonus.SymbolHoldings)),
	}
	for _, row := range bonus.Symbols {
		resp.SymbolPopularity = append(resp.SymbolPopularity, &v1.MonthlySymbolPopularity{
			Symbol:       row.Symbol,
			Trades:       int32(row.Trades),
			SharePercent: row.SharePercent,
		})
	}
	for _, row := range bonus.SymbolRisks {
		resp.SymbolRiskRatios = append(resp.SymbolRiskRatios, &v1.MonthlySymbolRiskRatio{
			Symbol:    row.Symbol,
			RiskRatio: row.RiskRatio,
		})
	}
	for _, row := range bonus.SymbolHoldings {
		resp.SymbolHoldingSplit = append(resp.SymbolHoldingSplit, &v1.MonthlySymbolHoldingSplit{
			Symbol:           row.Symbol,
			BullsSeconds:     row.BullsSeconds,
			ShortTermSeconds: row.ShortTermSeconds,
		})
	}

	return connect.NewResponse(resp), nil
}

var errInvalidMonth = fmt.Errorf("month must be between 1 and 12")

func convertAccountAnalyticsToProto(a *model.AccountAnalytics) *v1.AccountAnalytics {
	proto := &v1.AccountAnalytics{}

	if a.TradeStats != nil {
		proto.TotalTrades = int32(a.TradeStats.TotalTrades)
		proto.WinningTrades = int32(a.TradeStats.WinningTrades)
		proto.LosingTrades = int32(a.TradeStats.LosingTrades)
		proto.WinRate = a.TradeStats.WinRate
		proto.ProfitFactor = a.TradeStats.ProfitFactor
		proto.Profit = a.TradeStats.NetProfit

		proto.TradeStats = &v1.TradeStats{
			TotalTrades:          int32(a.TradeStats.TotalTrades),
			WinningTrades:        int32(a.TradeStats.WinningTrades),
			LosingTrades:         int32(a.TradeStats.LosingTrades),
			BuyTrades:            int32(a.TradeStats.BuyTrades),
			SellTrades:           int32(a.TradeStats.SellTrades),
			WinRate:              a.TradeStats.WinRate,
			TotalProfit:          a.TradeStats.TotalProfit,
			TotalLoss:            a.TradeStats.TotalLoss,
			NetProfit:            a.TradeStats.NetProfit,
			ProfitFactor:         a.TradeStats.ProfitFactor,
			AverageProfit:        a.TradeStats.AverageProfit,
			AverageLoss:          a.TradeStats.AverageLoss,
			AverageTrade:         a.TradeStats.AverageTrade,
			AverageVolume:        a.TradeStats.AverageVolume,
			LargestWin:           a.TradeStats.LargestWin,
			LargestLoss:          a.TradeStats.LargestLoss,
			MaxConsecutiveWins:   int32(a.TradeStats.MaxConsecutiveWins),
			MaxConsecutiveLosses: int32(a.TradeStats.MaxConsecutiveLosses),
			AverageHoldingTime:   a.TradeStats.AverageHoldingTime,
			TotalDeposit:         a.TradeStats.TotalDeposit,
			TotalWithdrawal:      a.TradeStats.TotalWithdrawal,
			NetDeposit:           a.TradeStats.NetDeposit,
		}
	}

	if a.RiskMetrics != nil {
		proto.MaxDrawdown = a.RiskMetrics.MaxDrawdown
		proto.SharpeRatio = a.RiskMetrics.SharpeRatio
		proto.RiskMetrics = convertRiskMetricsToProto(a.RiskMetrics)
	}

	if len(a.EquityCurve) > 0 {
		proto.EquityCurve = make([]*v1.EquityPoint, len(a.EquityCurve))
		for i, point := range a.EquityCurve {
			proto.EquityCurve[i] = &v1.EquityPoint{
				Date:    point.Date,
				Equity:  point.Equity,
				Balance: point.Balance,
				Profit:  point.Profit,
			}
		}
	}

	if len(a.SymbolStats) > 0 {
		totalTrades := 0
		for _, st := range a.SymbolStats {
			if st != nil {
				totalTrades += st.TotalTrades
			}
		}
		proto.SymbolStats = make([]*v1.SymbolStats, len(a.SymbolStats))
		for i, s := range a.SymbolStats {
			proto.SymbolStats[i] = convertSymbolStatsToProtoWithShare(s, totalTrades)
		}
	}

	if len(a.DailyPnL) > 0 {
		proto.DailyPnl = make([]*v1.DailyPnL, len(a.DailyPnL))
		for i, d := range a.DailyPnL {
			proto.DailyPnl[i] = &v1.DailyPnL{
				Date:                  d.Date,
				Profit:                d.PnL,
				Trades:                int32(d.Trades),
				Day:                   d.Day,
				Lots:                  d.Lots,
				Balance:               d.Balance,
				ProfitFactor:          d.ProfitFactor,
				MaxFloatingLossAmount: d.MaxFloatingLossAmount,
				MaxFloatingLossRatio:  d.MaxFloatingLossRatio,
				MaxFloatingProfitAmount: d.MaxFloatingProfitAmount,
				MaxFloatingProfitRatio: d.MaxFloatingProfitRatio,
			}
		}
	}

	if len(a.HourlyStats) > 0 {
		proto.HourlyStats = make([]*v1.HourlyStats, len(a.HourlyStats))
		for i, h := range a.HourlyStats {
			proto.HourlyStats[i] = &v1.HourlyStats{
				Hour:                   int32(h.HourStart),
				Profit:                 h.Profit,
				Trades:                 int32(h.Trades),
				WinRate:                h.WinRate,
				Lots:                   h.Lots,
				Balance:                h.Balance,
				ProfitFactor:           h.ProfitFactor,
				MaxFloatingLossAmount:  h.MaxFloatingLossAmount,
				MaxFloatingLossRatio:   h.MaxFloatingLossRatio,
				MaxFloatingProfitAmount: h.MaxFloatingProfitAmount,
				MaxFloatingProfitRatio: h.MaxFloatingProfitRatio,
			}
		}
	}

	if len(a.WeekdayPnL) > 0 {
		proto.WeekdayPnl = make([]*v1.WeekdayPnL, len(a.WeekdayPnL))
		for i, w := range a.WeekdayPnL {
			proto.WeekdayPnl[i] = &v1.WeekdayPnL{
				Weekday: int32(w.Weekday),
				Profit:  w.PnL,
				Trades:  int32(w.Trades),
			}
		}
	}

	if len(a.MonthlyPnL) > 0 {
		proto.MonthlyPnl = make([]*v1.MonthlyPnL, len(a.MonthlyPnL))
		for i, m := range a.MonthlyPnL {
			proto.MonthlyPnl[i] = convertMonthlyPnLToProto(m)
		}
	}

	if len(a.RecentTrades) > 0 {
		proto.RecentTrades = make([]*v1.TradeRecord, len(a.RecentTrades))
		for i, t := range a.RecentTrades {
			proto.RecentTrades[i] = convertTradeRecordToProto(t)
		}
	}

	return proto
}

func convertRiskMetricsToProto(m *model.RiskMetrics) *v1.RiskMetrics {
	return &v1.RiskMetrics{
		ValueAtRisk:        m.ValueAtRisk95,
		ExpectedShortfall:  m.ExpectedShortfall,
		Volatility:         m.Volatility,
		SortinoRatio:       m.SortinoRatio,
		CalmarRatio:        m.CalmarRatio,
		MaxDrawdown:        m.MaxDrawdown,
		MaxDrawdownPercent: m.MaxDrawdownPercent,
		SharpeRatio:        m.SharpeRatio,
		AverageDailyReturn: m.AverageDailyReturn,
		ReturnStdDev:       m.ReturnStdDev,
		Beta:               0,
		Alpha:              0,
	}
}

func convertSymbolStatsToProto(s *model.SymbolStats) *v1.SymbolStats {
	return &v1.SymbolStats{
		Symbol:    s.Symbol,
		Trades:    int32(s.TotalTrades),
		Profit:    s.NetProfit,
		WinRate:   s.WinRate,
		AvgProfit: s.AverageProfit,
		AvgLoss:   s.TotalLoss / float64(max(s.LosingTrades, 1)),
	}
}

func convertSymbolStatsToProtoWithShare(s *model.SymbolStats, totalTrades int) *v1.SymbolStats {
	proto := convertSymbolStatsToProto(s)
	if totalTrades > 0 {
		proto.TradeSharePercent = (float64(s.TotalTrades) / float64(totalTrades)) * 100
	}
	return proto
}

func convertTradeRecordToProto(t *model.TradeRecord) *v1.TradeRecord {
	return &v1.TradeRecord{
		Ticket:     t.Ticket,
		Symbol:     t.Symbol,
		Type:       t.OrderType,
		Volume:     t.Volume,
		OpenPrice:  t.OpenPrice,
		ClosePrice: t.ClosePrice,
		Profit:     t.Profit,
		OpenTime:   timestamppb.New(t.OpenTime),
		CloseTime:  timestamppb.New(t.CloseTime),
	}
}

func convertMonthlyPnLToProto(m *model.MonthlyPnL) *v1.MonthlyPnL {
	return &v1.MonthlyPnL{
		Month:  m.Month,
		Profit: m.Profit,
		Trades: int32(m.Trades),
	}
}

func convertTradeReportToProto(r *model.TradeReport, reportType string) *v1.Report {
	return &v1.Report{
		Id:              uuid.New().String(),
		AccountId:       r.AccountID,
		ReportType:      reportType,
		Title:           "analytics.reports.tradeReport.title",
		Content:         "analytics.reports.tradeReport.content",
		Recommendations: []string{},
		CreatedAt:       timestamppb.Now(),
	}
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

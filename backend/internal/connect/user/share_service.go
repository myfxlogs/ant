package user

import (
	"context"
	"fmt"
	"math"
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"

	"alphaforge/internal/model"
	"alphaforge/internal/mthub"
	"alphaforge/internal/repository"
)

// sharePerformance holds computed share performance data.
// Built by BuildSharePerformance from raw repository data.
type sharePerformance struct {
	UserName      string
	TotalTrades   int
	TotalReturn   string
	WinRate       string
	MaxDrawdown   string
	SharpeRatio   string
	EquityCurve   []string
	EquityTimesMs []int64
	Trades        []*model.TradeRecord
	TotalVolume   string
	ProfitFactor  string
	AvgHoldingMs  int64
	TradeStats    tradeStatsPayload
	SymbolStats   []symbolStatPayload
}

type tradeStatsPayload struct {
	WinningTrades int
	LosingTrades  int
	BestTrade     string
	WorstTrade    string
	AvgWin        string
	AvgLoss       string
}

type symbolStatPayload struct {
	Symbol string
	Count  int
	Net    string
}

// BuildSharePerformance fetches and computes all performance metrics for a share token.
// Used by both the ConnectRPC handler and the OG image HTTP handler.
func BuildSharePerformance(
	ctx context.Context,
	st *repository.ShareToken,
	userRepo *repository.UserRepository,
	eqRepo *repository.AnalyticsRepository,
	tradeRecords *repository.TradeRecordRepository,
	mthubSvc *mthub.MtHubService,
) (*sharePerformance, error) {
	// User name resolution.
	user, _ := userRepo.GetByID(ctx, st.UserID)
	userName := "匿名用户"
	if user != nil {
		if user.Email != "" {
			userName = user.Email
		}
		if user.Nickname != nil && *user.Nickname != "" {
			userName = *user.Nickname
		}
	}

	aid, _ := uuid.Parse(st.AccountID)
	start := time.Now().AddDate(-1, 0, 0)
	end := time.Now()

	// Equity curve.
	equityPoints, _ := eqRepo.GetEquityCurve(ctx, aid, start, end)
	equityVals := make([]string, 0, len(equityPoints))
	equityTimesMs := make([]int64, 0, len(equityPoints))
	for _, p := range equityPoints {
		equityVals = append(equityVals, p.Equity.String())
		if t, err := time.Parse("2006-01-02", p.Date); err == nil {
			equityTimesMs = append(equityTimesMs, t.UnixMilli())
		} else {
			equityTimesMs = append(equityTimesMs, 0)
		}
	}

	// Trades + stats.
	trades, _ := tradeRecords.GetByAccountID(ctx, st.UserID, aid, start, end, 50)
	stats := summarizeTrades(trades)

	// Max drawdown: real peak-to-trough from equity curve (decimal).
	// REUSE: algorithm pattern from analytics_rolling.go:103 (runningMax + dd%)
	// and live_performance.go:208-222 (decimal peak.Sub(equity).Div(peak)).
	maxDD := computeMaxDrawdownPct(equityPoints)

	// Sharpe ratio.
	sharpeStr := "0"
	if sharpeVal := computeSharpe(equityPoints); sharpeVal != 0 {
		sharpeStr = strconv.FormatFloat(sharpeVal, 'f', 4, 64)
	}

	return &sharePerformance{
		UserName:      userName,
		TotalTrades:   len(trades),
		TotalReturn:   stats.totalReturnStr(),
		WinRate:       stats.winRateStr(),
		MaxDrawdown:   maxDD.String(),
		SharpeRatio:   sharpeStr,
		EquityCurve:   equityVals,
		EquityTimesMs: equityTimesMs,
		Trades:        trades,
		TotalVolume:   stats.totalVolStr(),
		ProfitFactor:  stats.profitFactorStr(),
		AvgHoldingMs:  stats.avgHoldingMs(),
		TradeStats:    stats.toPayload(),
		SymbolStats:   aggregateSymbolStats(trades),
	}, nil
}

// FormatSharedTrades converts TradeRecord slice to proto SharedTrade slice.
func FormatSharedTrades(trades []*model.TradeRecord) []sharedTrade {
	out := make([]sharedTrade, 0, len(trades))
	for _, t := range trades {
		out = append(out, sharedTrade{
			Symbol:      t.Symbol,
			Side:        t.OrderType,
			Volume:      t.Volume.String(),
			Profit:      t.Profit.String(),
			CloseTimeMs: t.CloseTime.UnixMilli(),
		})
	}
	return out
}

type sharedTrade struct {
	Symbol      string
	Side        string
	Volume      string
	Profit      string
	CloseTimeMs int64
}

// FormatSharedPositions converts MT orders to proto SharedPosition slice.
// Returns nil if mthubSvc is nil or query fails.
func FormatSharedPositions(ctx context.Context, mthubSvc *mthub.MtHubService, accountID string) []sharedPosition {
	if mthubSvc == nil {
		return nil
	}
	orders, err := mthubSvc.OpenedOrders(ctx, accountID)
	if err != nil {
		return nil
	}
	out := make([]sharedPosition, 0, len(orders))
	for _, o := range orders {
		side := "BUY"
		if o.Side == -1 {
			side = "SELL"
		}
		out = append(out, sharedPosition{
			Symbol:    o.SymbolRaw,
			Type:      side,
			Volume:    o.Volume.String(),
			OpenPrice: o.OpenPrice.String(),
			Profit:    o.Profit.String(),
		})
	}
	return out
}

type sharedPosition struct {
	Symbol    string
	Type      string
	Volume    string
	OpenPrice string
	Profit    string
}

// tradeSummary holds computed metrics from a set of trades.
type tradeSummary struct {
	totalProfit, totalVolume, grossProfit, grossLoss decimal.Decimal
	bestTrade, worstTrade                            decimal.Decimal
	wins, losses                                     int
	openTimeSum                                      int64
}

func summarizeTrades(trades []*model.TradeRecord) tradeSummary {
	var s tradeSummary
	for _, t := range trades {
		s.totalProfit = s.totalProfit.Add(t.Profit)
		s.totalVolume = s.totalVolume.Add(t.Volume)
		if t.Profit.IsPositive() {
			s.wins++
			s.grossProfit = s.grossProfit.Add(t.Profit)
		} else {
			s.losses++
			s.grossLoss = s.grossLoss.Add(t.Profit.Abs())
		}
		if s.bestTrade.IsZero() || t.Profit.GreaterThan(s.bestTrade) {
			s.bestTrade = t.Profit
		}
		if s.worstTrade.IsZero() || t.Profit.LessThan(s.worstTrade) {
			s.worstTrade = t.Profit
		}
		s.openTimeSum += t.CloseTime.Sub(t.OpenTime).Milliseconds()
	}
	return s
}

func (s tradeSummary) totalReturnStr() string {
	return s.totalProfit.String()
}

func (s tradeSummary) winRateStr() string {
	if s.wins+s.losses == 0 {
		return "0"
	}
	return decimal.NewFromInt(int64(s.wins)).Div(decimal.NewFromInt(int64(s.wins + s.losses))).Mul(decimal.NewFromInt(100)).String()
}

func (s tradeSummary) profitFactorStr() string {
	if !s.grossLoss.IsPositive() {
		return "0"
	}
	return s.grossProfit.Div(s.grossLoss).String()
}

func (s tradeSummary) totalVolStr() string {
	return s.totalVolume.String()
}

func (s tradeSummary) avgHoldingMs() int64 {
	if n := s.wins + s.losses; n > 0 {
		return s.openTimeSum / int64(n)
	}
	return 0
}

func (s tradeSummary) toPayload() tradeStatsPayload {
	avgWin := decimal.Zero
	if s.wins > 0 {
		avgWin = s.grossProfit.Div(decimal.NewFromInt(int64(s.wins)))
	}
	avgLoss := decimal.Zero
	if s.losses > 0 {
		avgLoss = s.grossLoss.Div(decimal.NewFromInt(int64(s.losses)))
	}
	return tradeStatsPayload{
		WinningTrades: s.wins,
		LosingTrades:  s.losses,
		BestTrade:     s.bestTrade.String(),
		WorstTrade:    s.worstTrade.String(),
		AvgWin:        avgWin.String(),
		AvgLoss:       avgLoss.String(),
	}
}

// computeMaxDrawdownPct calculates the true peak-to-trough maximum drawdown
// percentage from equity curve points using decimal arithmetic.
// REUSE: algorithm pattern from analytics_rolling.go:103 (runningMax + dd%)
// and live_performance.go:208-222 (decimal peak.Sub(equity).Div(peak)).
func computeMaxDrawdownPct(equityPoints []*model.EquityPoint) decimal.Decimal {
	if len(equityPoints) < 2 {
		return decimal.Zero
	}
	var maxDD decimal.Decimal
	runningPeak := equityPoints[0].Equity
	for _, p := range equityPoints[1:] {
		if p.Equity.GreaterThan(runningPeak) {
			runningPeak = p.Equity
		}
		if runningPeak.GreaterThan(decimal.Zero) {
			dd := runningPeak.Sub(p.Equity).Div(runningPeak).Mul(decimal.NewFromInt(100))
			if dd.GreaterThan(maxDD) {
				maxDD = dd
			}
		}
	}
	return maxDD
}

// aggregateSymbolStats groups trades by symbol and computes count + net profit.
// REUSE: aggregation pattern from analytics_compute.go:228/258 (by-symbol grouping).
func aggregateSymbolStats(trades []*model.TradeRecord) []symbolStatPayload {
	if len(trades) == 0 {
		return nil
	}
	type acc struct {
		count int
		net   decimal.Decimal
	}
	m := make(map[string]*acc)
	order := make([]string, 0, len(trades))
	for _, t := range trades {
		sym := t.Symbol
		if sym == "" {
			sym = "-"
		}
		if _, ok := m[sym]; !ok {
			m[sym] = &acc{}
			order = append(order, sym)
		}
		m[sym].count++
		m[sym].net = m[sym].net.Add(t.Profit)
	}
	out := make([]symbolStatPayload, 0, len(order))
	for _, sym := range order {
		out = append(out, symbolStatPayload{
			Symbol: sym,
			Count:  m[sym].count,
			Net:    m[sym].net.String(),
		})
	}
	return out
}

// computeSharpe calculates annualized Sharpe ratio from equity curve points.
func computeSharpe(equityPoints []*model.EquityPoint) float64 {
	if len(equityPoints) < 2 {
		return 0
	}
	var sum, sumSq float64
	var returns []float64
	for i := 1; i < len(equityPoints); i++ {
		prev, ok1 := equityPoints[i-1].Equity.Float64()
		if !ok1 || prev == 0 {
			continue
		}
		curr, ok2 := equityPoints[i].Equity.Float64()
		if !ok2 {
			continue
		}
		r := (curr - prev) / prev
		returns = append(returns, r)
		sum += r
	}
	if len(returns) < 2 {
		return 0
	}
	n := float64(len(returns))
	mean := sum / n
	for _, r := range returns {
		diff := r - mean
		sumSq += diff * diff
	}
	variance := sumSq / (n - 1)
	if variance <= 0 {
		return 0
	}
	std := math.Sqrt(variance)
	if std == 0 {
		return 0
	}
	return mean / std * math.Sqrt(252)
}

// fmtShareURL returns the share URL path for a token.
func fmtShareURL(token string) string {
	return fmt.Sprintf("/share/%s", token)
}

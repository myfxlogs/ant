package system

import (
	"context"
	"fmt"
	"math"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"connectrpc.com/connect"

	antv1 "anttrader/gen/proto/ant/v1"
	"anttrader/internal/model"
	"anttrader/internal/repository"
)

// ── GetRollingMetrics ──

func (s *AnalyticsServer) GetRollingMetrics(ctx context.Context, req *connect.Request[antv1.GetRollingMetricsRequest]) (*connect.Response[antv1.GetRollingMetricsResponse], error) {
	accountID, err := uuid.Parse(req.Msg.AccountId)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("invalid account_id: %w", err))
	}
	if err := s.verifyAccountOwnership(ctx, req.Msg.AccountId); err != nil {
		return nil, err
	}

	// Check cache — return immediately on hit.
	if s.cache != nil {
		if cached, _ := s.cache.GetRolling(ctx, req.Msg.AccountId); cached != nil {
			return connect.NewResponse(cached), nil
		}
	}

	now := time.Now()
	start := now.AddDate(-1, 0, 0)

	// Equity curve — base for rolling metrics and drawdown.
	equityCurve, err := s.repo.GetEquityCurve(ctx, accountID, start, now)
	if err != nil {
		s.log.Warn("analytics: get equity curve for rolling metrics failed", zap.Error(err))
	}
	equityCurve = appendLiveEquity(ctx, s.repo, accountID, equityCurve)

	// Rolling Sharpe — 20-trade window from daily returns.
	dailyReturns := dailyReturnsToPercent(equityCurve)
	rollingSharpe := computeRollingSharpe(dailyReturns, equityCurve)

	// Drawdown events.
	ddEvents := computeDrawdownEvents(equityCurve)

	// Equity curve with drawdown overlay.
	eqProto, ddProto := buildEquityDrawdown(equityCurve)

	// Monthly win rates — from actual trade records.
	monthlyWinRates := computeMonthlyWinRates(ctx, s.repo, accountID, start, now)

	resp := &antv1.GetRollingMetricsResponse{
		RollingSharpe:   rollingSharpe,
		DrawdownEvents:  ddEvents,
		MonthlyWinRates: monthlyWinRates,
		EquityCurve:     eqProto,
		DrawdownCurve:   ddProto,
	}

	if s.cache != nil {
		if err := s.cache.SetRolling(ctx, req.Msg.AccountId, resp); err != nil {
			s.log.Warn("analytics cache: set rolling failed", zap.Error(err))
		}
	}

	return connect.NewResponse(resp), nil
}

func computeRollingSharpe(dailyReturns []float64, curve []*model.EquityPoint) []*antv1.RollingPoint {
	const window = 20
	if len(dailyReturns) < window || len(curve) < window {
		return nil
	}
	points := make([]*antv1.RollingPoint, 0, len(dailyReturns)-window+1)
	for i := window - 1; i < len(dailyReturns); i++ {
		slice := dailyReturns[i-window+1 : i+1]
		avg := mean(slice)
		std := stdev(slice, avg)
		sharpe := 0.0
		if std > 0 {
			sharpe = avg / std * math.Sqrt(252)
		}
		dateIdx := i
		if dateIdx >= len(curve) {
			dateIdx = len(curve) - 1
		}
		points = append(points, &antv1.RollingPoint{
			Date:  curve[dateIdx].Date,
			Value: math.Round(sharpe*100) / 100,
		})
	}
	return points
}

func computeDrawdownEvents(curve []*model.EquityPoint) []*antv1.DrawdownEvent {
	if len(curve) < 2 {
		return nil
	}
	var events []*antv1.DrawdownEvent
	var inDD bool
	var ddStart string
	var ddDepth float64
	runningMax := 0.0

	for _, p := range curve {
		eq := p.Equity.InexactFloat64()
		if eq > runningMax {
			runningMax = eq
			if inDD {
				events[len(events)-1].EndDate = p.Date
				events[len(events)-1].RecoveryDate = p.Date
				inDD = false
			}
		}
		if runningMax > 0 {
			dd := (runningMax - eq) / runningMax * 100
			if dd > 5 && !inDD {
				inDD = true
				ddStart = p.Date
				ddDepth = dd
				events = append(events, &antv1.DrawdownEvent{
					StartDate:    ddStart,
					DepthPercent: math.Round(dd*100) / 100,
				})
			}
			if inDD {
				if dd > ddDepth {
					ddDepth = dd
					if len(events) > 0 {
						events[len(events)-1].DepthPercent = math.Round(dd*100) / 100
					}
				}
			}
		}
	}
	return events
}

func computeMonthlyWinRates(ctx context.Context, repo *repository.AnalyticsRepository, accountID uuid.UUID, start, end time.Time) []*antv1.MonthlyWinRate {
	// Query all calendar years spanned by [start, end] (up to 2 for a 1-year window).
	var allMonthlyPnL []monthlyPnLWithYear
	for y := start.Year(); y <= end.Year(); y++ {
		monthlyPnL, err := repo.GetMonthlyPnL(ctx, accountID, y)
		if err != nil {
			continue
		}
		for _, m := range monthlyPnL {
			// Include only months within [start, end] and with actual trades.
			monthStart := time.Date(y, time.Month(m.MonthNum), 1, 0, 0, 0, 0, time.UTC)
			monthEnd := monthStart.AddDate(0, 1, -1)
			if monthEnd.Before(start) || monthStart.After(end) {
				continue
			}
			if m.Trades == 0 {
				continue
			}
			allMonthlyPnL = append(allMonthlyPnL, monthlyPnLWithYear{year: y, MonthlyPnL: m})
		}
	}
	if len(allMonthlyPnL) == 0 {
		return nil
	}
	out := make([]*antv1.MonthlyWinRate, 0, len(allMonthlyPnL))
	for _, m := range allMonthlyPnL {
		wr := float64(m.WinTrades) / float64(m.Trades) * 100
		// Use "YYYY-MM" format so months from different years are distinguishable.
		label := fmt.Sprintf("%d-%02d", m.year, m.MonthNum)
		out = append(out, &antv1.MonthlyWinRate{
			Month:       label,
			WinRate:     math.Round(wr*100) / 100,
			TotalTrades: int64(m.Trades),
		})
	}
	return out
}

type monthlyPnLWithYear struct {
	year int
	*model.MonthlyPnL
}

func mean(vals []float64) float64 {
	if len(vals) == 0 {
		return 0
	}
	sum := 0.0
	for _, v := range vals {
		sum += v
	}
	return sum / float64(len(vals))
}

func stdev(vals []float64, avg float64) float64 {
	if len(vals) < 2 {
		return 0
	}
	sumSq := 0.0
	for _, v := range vals {
		d := v - avg
		sumSq += d * d
	}
	return math.Sqrt(sumSq / float64(len(vals)-1))
}

func buildEquityDrawdown(curve []*model.EquityPoint) ([]*antv1.EquityPoint, []*antv1.DrawdownPoint) {
	if len(curve) == 0 {
		return nil, nil
	}
	eqProto := make([]*antv1.EquityPoint, len(curve))
	ddProto := make([]*antv1.DrawdownPoint, 0, len(curve))
	runningMax := 0.0
	for i, p := range curve {
		eq := p.Equity.InexactFloat64()
		eqProto[i] = &antv1.EquityPoint{
			Date:    p.Date,
			Equity:  p.Equity.String(),
			Balance: p.Balance.String(),
		}
		if eq > runningMax {
			runningMax = eq
		}
		dd := 0.0
		if runningMax > 0 {
			dd = (runningMax - eq) / runningMax * 100
		}
		ddProto = append(ddProto, &antv1.DrawdownPoint{
			Date:            p.Date,
			DrawdownPercent: math.Round(dd*100) / 100,
		})
	}
	return eqProto, ddProto
}

package service

import (
	"context"
	"errors"
	"math"
	"sort"
	"time"

	"github.com/google/uuid"

	"anttrader/internal/model"
)

type symbolMonthAgg struct {
	winProfit float64
	lossAbs   float64
	lots      float64
	trades    int
	bullSecs  float64
	shortSecs float64
}

func symbolRiskRatioFromAgg(a *symbolMonthAgg) float64 {
	if a.lossAbs > 0 {
		return a.winProfit / a.lossAbs
	}
	if a.winProfit > 0 {
		return 999.99
	}
	return 0
}

func buildMonthlyBonusSymbolViews(aggs map[string]*symbolMonthAgg, pieTopN int) ([]*model.MonthlyBonusSymbol, []*model.MonthlyBonusRiskRow, []*model.MonthlyBonusHoldingRow) {
	if pieTopN <= 0 {
		pieTopN = 8
	}
	type lotRow struct {
		sym  string
		lots float64
		tr   int
	}
	rows := make([]lotRow, 0, len(aggs))
	totalLots := 0.0
	for sym, a := range aggs {
		if a.trades == 0 {
			continue
		}
		rows = append(rows, lotRow{sym: sym, lots: a.lots, tr: a.trades})
		totalLots += a.lots
	}
	if len(rows) == 0 {
		return nil, nil, nil
	}
	sort.Slice(rows, func(i, j int) bool {
		if rows[i].lots != rows[j].lots {
			return rows[i].lots > rows[j].lots
		}
		return rows[i].sym < rows[j].sym
	})

	var pie []*model.MonthlyBonusSymbol
	if totalLots <= 0 {
		totalTr := 0
		for _, r := range rows {
			totalTr += r.tr
		}
		n := pieTopN
		if n > len(rows) {
			n = len(rows)
		}
		otherTr := 0
		for i := n; i < len(rows); i++ {
			otherTr += rows[i].tr
		}
		for i := 0; i < n; i++ {
			pie = append(pie, &model.MonthlyBonusSymbol{
				Symbol:       rows[i].sym,
				Trades:       rows[i].tr,
				SharePercent: float64(rows[i].tr) / float64(totalTr) * 100,
			})
		}
		if otherTr > 0 && len(rows) > n {
			pie = append(pie, &model.MonthlyBonusSymbol{
				Symbol:       "Other",
				Trades:       otherTr,
				SharePercent: float64(otherTr) / float64(totalTr) * 100,
			})
		}
	} else {
		n := pieTopN
		if n > len(rows) {
			n = len(rows)
		}
		topLots := 0.0
		for i := 0; i < n; i++ {
			topLots += rows[i].lots
		}
		otherLots := totalLots - topLots
		for i := 0; i < n; i++ {
			pie = append(pie, &model.MonthlyBonusSymbol{
				Symbol:       rows[i].sym,
				Trades:       rows[i].tr,
				SharePercent: rows[i].lots / totalLots * 100,
			})
		}
		if otherLots > 0 && len(rows) > n {
			otherTr := 0
			for i := n; i < len(rows); i++ {
				otherTr += rows[i].tr
			}
			pie = append(pie, &model.MonthlyBonusSymbol{
				Symbol:       "Other",
				Trades:       otherTr,
				SharePercent: otherLots / totalLots * 100,
			})
		}
	}

	const maxChartSymbols = 20
	type symAct struct {
		sym  string
		lots float64
	}
	act := make([]symAct, 0, len(aggs))
	for sym, a := range aggs {
		if a.trades == 0 {
			continue
		}
		act = append(act, symAct{sym: sym, lots: a.lots})
	}
	sort.Slice(act, func(i, j int) bool {
		if act[i].lots != act[j].lots {
			return act[i].lots > act[j].lots
		}
		return act[i].sym < act[j].sym
	})
	if len(act) > maxChartSymbols {
		act = act[:maxChartSymbols]
	}
	syms := make([]string, len(act))
	for i := range act {
		syms[i] = act[i].sym
	}
	sort.Strings(syms)

	riskOut := make([]*model.MonthlyBonusRiskRow, 0, len(syms))
	holdOut := make([]*model.MonthlyBonusHoldingRow, 0, len(syms))
	for _, sym := range syms {
		a := aggs[sym]
		riskOut = append(riskOut, &model.MonthlyBonusRiskRow{
			Symbol:    sym,
			RiskRatio: symbolRiskRatioFromAgg(a),
		})
		holdOut = append(holdOut, &model.MonthlyBonusHoldingRow{
			Symbol:           sym,
			BullsSeconds:     a.bullSecs,
			ShortTermSeconds: a.shortSecs,
		})
	}
	return pie, riskOut, holdOut
}

func aggregateMonthlyBonusBySymbol(records []*TradeRecord) map[string]*symbolMonthAgg {
	out := make(map[string]*symbolMonthAgg)
	for _, r := range records {
		if isBalanceTradeRecord(r) {
			continue
		}
		sym := bonusSymbolKey(r)
		a, ok := out[sym]
		if !ok {
			a = &symbolMonthAgg{}
			out[sym] = a
		}
		a.trades++
		a.lots += r.Volume
		if r.Profit > 0 {
			a.winProfit += r.Profit
		} else if r.Profit < 0 {
			a.lossAbs += math.Abs(r.Profit)
		}
		sec := tradeHoldSeconds(r)
		if isBuySideRecord(r) {
			a.bullSecs += sec
		} else if isSellSideRecord(r) {
			a.shortSecs += sec
		}
	}
	return out
}

func (s *AnalyticsService) GetMonthlyAnalysis(ctx context.Context, userID, accountID uuid.UUID) ([]int, []*model.MonthlyAnalysisPoint, error) {
	if _, err := s.verifyAccountAccess(ctx, userID, accountID); err != nil {
		return nil, nil, err
	}

	years, err := s.analyticsRepo.GetMonthlyAnalysisYears(ctx, accountID)
	if err != nil {
		return nil, nil, err
	}
	if len(years) == 0 {
		return []int{}, []*model.MonthlyAnalysisPoint{}, nil
	}

	rawPoints, err := s.analyticsRepo.GetMonthlyAnalysisRaw(ctx, accountID)
	if err != nil {
		return nil, nil, err
	}

	pointByMonth := make(map[int]map[int]*model.MonthlyAnalysisPoint, len(years))
	for _, year := range years {
		pointByMonth[year] = make(map[int]*model.MonthlyAnalysisPoint, 12)
	}
	for _, p := range rawPoints {
		if _, ok := pointByMonth[p.Year]; !ok {
			pointByMonth[p.Year] = make(map[int]*model.MonthlyAnalysisPoint, 12)
			years = append(years, p.Year)
		}
		pointByMonth[p.Year][p.Month] = &model.MonthlyAnalysisPoint{
			Year:   p.Year,
			Month:  p.Month,
			Profit: p.Profit,
			Lots:   p.Lots,
			Pips:   p.Pips,
			Trades: p.Trades,
		}
	}

	sort.Ints(years)

	initialBalance, err := s.analyticsRepo.GetAccountInitialBalance(ctx, accountID)
	if err != nil {
		initialBalance = 0
	}

	equityBase := initialBalance
	result := make([]*model.MonthlyAnalysisPoint, 0, len(years)*12)
	for _, year := range years {
		months := pointByMonth[year]
		for month := 1; month <= 12; month++ {
			point, ok := months[month]
			if !ok {
				point = &model.MonthlyAnalysisPoint{
					Year:   year,
					Month:  month,
					Profit: 0,
					Lots:   0,
					Pips:   0,
					Trades: 0,
				}
			}

			if equityBase != 0 {
				point.Change = (point.Profit / equityBase) * 100
			} else {
				point.Change = 0
			}
			equityBase += point.Profit
			result = append(result, point)
		}
	}

	return years, result, nil
}

func (s *AnalyticsService) GetMonthlyAnalysisBonus(ctx context.Context, userID, accountID uuid.UUID, year, month int) (*model.MonthlyAnalysisBonus, error) {
	if month < 1 || month > 12 {
		return nil, errors.New("invalid month")
	}

	if _, err := s.verifyAccountAccess(ctx, userID, accountID); err != nil {
		return nil, err
	}

	start := time.Date(year, time.Month(month), 1, 0, 0, 0, 0, time.UTC)
	end := start.AddDate(0, 1, 0).Add(-time.Nanosecond)

	records, err := s.analyticsRepo.GetTradeRecords(ctx, accountID, start, end)
	if err != nil {
		return nil, err
	}

	stats := s.calculateTradeStatsFromRecords(records)
	aggs := aggregateMonthlyBonusBySymbol(records)
	pie, risks, holdings := buildMonthlyBonusSymbolViews(aggs, 8)

	avgHolding, err := s.analyticsRepo.GetHoldingTimeStats(ctx, accountID, start, end)
	if err != nil {
		return nil, err
	}

	return &model.MonthlyAnalysisBonus{
		RiskRatio:         stats.ProfitFactor,
		Symbols:           pie,
		SymbolRisks:       risks,
		SymbolHoldings:    holdings,
		AvgHoldingSeconds: avgHolding,
		TotalTrades:       stats.TotalTrades,
	}, nil
}

package user

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// HandleGetSharedPerformanceJSON returns enhanced share data as plain JSON.
func (s *ShareServer) HandleGetSharedPerformanceJSON(w http.ResponseWriter, r *http.Request) {
	token := r.URL.Query().Get("token")
	if token == "" {
		http.Error(w, `{"error":"missing token"}`, http.StatusBadRequest)
		return
	}
	st, err := s.repo.GetByToken(r.Context(), token)
	if err != nil || time.Now().After(st.ExpiresAt) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]bool{"expired": true})
		return
	}
	s.repo.IncrementView(r.Context(), st.Token)

	user, _ := s.userRepo.GetByID(r.Context(), st.UserID)
	userName := "Anonymous"
	if user != nil {
		if user.Nickname != nil && *user.Nickname != "" {
			userName = *user.Nickname
		} else if user.Email != "" {
			userName = user.Email
		}
	}

	aid, _ := uuid.Parse(st.AccountID)
	start := time.Now().AddDate(-1, 0, 0)
	end := time.Now()

	equityPoints, _ := s.eqRepo.GetEquityCurve(r.Context(), aid, start, end)
	equityVals := make([]float64, 0, len(equityPoints))
	for _, p := range equityPoints {
		f, _ := p.Equity.Float64()
		equityVals = append(equityVals, f)
	}

	trades, _ := s.tradeRecords.GetByAccountID(r.Context(), st.UserID, aid, start, end, 100)
	var totalProfit, totalVolume decimal.Decimal
	wins, losses := 0, 0
	grossProfit, grossLoss := decimal.Zero, decimal.Zero
	var openTimeSum int64
	var maxDD decimal.Decimal

	type tradeJSON struct {
		Symbol      string  `json:"symbol"`
		Side        string  `json:"side"`
		Volume      float64 `json:"volume"`
		Profit      float64 `json:"profit"`
		CloseTimeMs int64   `json:"closeTimeMs"`
	}
	tradesOut := make([]tradeJSON, 0, len(trades))
	for _, t := range trades {
		totalProfit = totalProfit.Add(t.Profit)
		totalVolume = totalVolume.Add(t.Volume)
		if t.Profit.IsPositive() {
			wins++
			grossProfit = grossProfit.Add(t.Profit)
		} else {
			losses++
			grossLoss = grossLoss.Add(t.Profit.Abs())
		}
		if t.Profit.LessThan(maxDD) {
			maxDD = t.Profit
		}
		openTimeSum += t.CloseTime.Sub(t.OpenTime).Milliseconds()
		vol, _ := t.Volume.Float64()
		prof, _ := t.Profit.Float64()
		tradesOut = append(tradesOut, tradeJSON{
			Symbol: t.Symbol, Side: t.OrderType,
			Volume: vol, Profit: prof,
			CloseTimeMs: t.CloseTime.UnixMilli(),
		})
	}

	totalRet, _ := totalProfit.Float64()
	winRate := 0.0
	if wins+losses > 0 { winRate = float64(wins) / float64(wins+losses) * 100 }
	maxDDval, _ := maxDD.Float64()
	profitFactor := 0.0
	if grossLoss.IsPositive() {
		pf, _ := grossProfit.Div(grossLoss).Float64()
		profitFactor = pf
	}
	avgHoldingMs := int64(0)
	if len(trades) > 0 { avgHoldingMs = openTimeSum / int64(len(trades)) }
	totalVol, _ := totalVolume.Float64()
	sharpe := 0.0
	if len(equityVals) > 1 {
		var sum, sumSq float64
		dailyReturns := make([]float64, 0, len(equityVals)-1)
		for i := 1; i < len(equityVals); i++ {
			if equityVals[i-1] != 0 {
				r := (equityVals[i] - equityVals[i-1]) / equityVals[i-1]
				dailyReturns = append(dailyReturns, r)
				sum += r
			}
		}
		if len(dailyReturns) > 1 {
			mean := sum / float64(len(dailyReturns))
			for _, r := range dailyReturns {
				sumSq += (r - mean) * (r - mean)
			}
			std := 0.0
			if sumSq > 0 { std = sumSq / float64(len(dailyReturns)-1) }
			if std > 0 { sharpe = mean / std * 16 }
		}
	}

	// Positions — only if the share token allows it. Use cached snapshot,
	// fetching live from the MT broker only once to populate the cache.
	var positionsOut interface{}
	if st.ShowPositions {
		type posJSON struct {
			Symbol    string  `json:"symbol"`
			Type      string  `json:"type"`
			Volume    float64 `json:"volume"`
			OpenPrice float64 `json:"openPrice"`
			Profit    float64 `json:"profit"`
		}
		posList := make([]posJSON, 0)

		// Try cached snapshot first.
		if cached, err := s.repo.GetPositionsSnapshot(r.Context(), st.Token); err == nil && cached != nil {
			positionsOut = cached
		} else if s.mthub != nil {
			// Fetch live from broker and cache.
			if orders, err := s.mthub.OpenedOrders(r.Context(), st.AccountID); err == nil {
				for _, o := range orders {
					vol, _ := o.Volume.Float64()
					openP, _ := o.OpenPrice.Float64()
					prof, _ := o.Profit.Float64()
					side := "BUY"
					if o.Side == -1 { side = "SELL" }
					posList = append(posList, posJSON{
						Symbol: o.SymbolRaw, Type: side,
						Volume: vol, OpenPrice: openP, Profit: prof,
					})
				}
				// Cache snapshot for subsequent views.
				s.repo.SetPositionsSnapshot(r.Context(), st.Token, posList)
			}
			positionsOut = posList
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"userName":        userName,
		"totalReturn":     totalRet,
		"winRate":         winRate,
		"maxDrawdown":     maxDDval,
		"totalTrades":     len(trades),
		"totalVolume":     totalVol,
		"profitFactor":    profitFactor,
		"avgHoldingMs":    avgHoldingMs,
		"sharpeRatio":     sharpe,
		"equityCurve":     equityVals,
		"trades":          tradesOut,
		"showPositions":   st.ShowPositions,
		"positions":       positionsOut,
	})
}

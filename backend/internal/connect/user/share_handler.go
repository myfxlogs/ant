package user

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"connectrpc.com/connect"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/shopspring/decimal"
	"go.uber.org/zap"

	antv1 "anttrader/gen/proto/ant/v1"
	antv1c "anttrader/gen/proto/ant/v1/antv1connect"
	"anttrader/internal/interceptor"
	"anttrader/internal/mthub"
	"anttrader/internal/repository"
)

type ShareServer struct {
	repo         *repository.ShareRepository
	tradeRecords *repository.TradeRecordRepository
	eqRepo       *repository.AnalyticsRepository
	userRepo     *repository.UserRepository
	mthub        *mthub.MtHubService
	pg           *pgxpool.Pool
	jwtSecret    string
	log          *zap.Logger
}

var _ antv1c.ShareServiceHandler = (*ShareServer)(nil)

func NewShareServer(repo *repository.ShareRepository, tradeRecords *repository.TradeRecordRepository, eqRepo *repository.AnalyticsRepository, userRepo *repository.UserRepository, mthubSvc *mthub.MtHubService, pg *pgxpool.Pool, jwtSecret string, log *zap.Logger) *ShareServer {
	return &ShareServer{repo: repo, tradeRecords: tradeRecords, eqRepo: eqRepo, userRepo: userRepo, mthub: mthubSvc, pg: pg, jwtSecret: jwtSecret, log: log}
}

func (s *ShareServer) CreateShareToken(ctx context.Context, req *connect.Request[antv1.CreateShareTokenRequest]) (*connect.Response[antv1.CreateShareTokenResponse], error) {
	uid, _ := uuid.Parse(interceptor.GetUserID(ctx))
	if uid == uuid.Nil {
		return nil, connect.NewError(connect.CodeUnauthenticated, fmt.Errorf("login required"))
	}
	expireDays := req.Msg.ExpireDays
	if expireDays <= 0 { expireDays = 7 }
	b := make([]byte, 16)
	rand.Read(b)
	token := hex.EncodeToString(b)
	st := &repository.ShareToken{
		UserID: uid, AccountID: req.Msg.AccountId, Token: token,
		Description: req.Msg.Description,
		ExpiresAt:   time.Now().Add(time.Duration(expireDays) * 24 * time.Hour),
	}
	if err := s.repo.Create(ctx, st); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&antv1.CreateShareTokenResponse{
		Token: token, ShareUrl: fmt.Sprintf("/share/%s", token),
		ExpiresAt: st.ExpiresAt.Format(time.RFC3339),
	}), nil
}

func (s *ShareServer) GetSharedPerformance(ctx context.Context, req *connect.Request[antv1.GetSharedPerformanceRequest]) (*connect.Response[antv1.GetSharedPerformanceResponse], error) {
	st, err := s.repo.GetByToken(ctx, req.Msg.Token)
	if err != nil || time.Now().After(st.ExpiresAt) {
		return connect.NewResponse(&antv1.GetSharedPerformanceResponse{Expired: true}), nil
	}
	s.repo.IncrementView(ctx, st.Token)

	user, _ := s.userRepo.GetByID(ctx, st.UserID)
	userName := "匿名用户"
	if user != nil {
		if user.Email != "" { userName = user.Email }
		if user.Nickname != nil && *user.Nickname != "" { userName = *user.Nickname }
	}

	aid, _ := uuid.Parse(st.AccountID)
	// Equity curve
	start := time.Now().AddDate(-1, 0, 0)
	end := time.Now()
	equityPoints, _ := s.eqRepo.GetEquityCurve(ctx, aid, start, end)
	equityVals := make([]float64, 0, len(equityPoints))
	for _, p := range equityPoints {
		f, _ := p.Equity.Float64()
		equityVals = append(equityVals, f)
	}

	// Recent trades + basic stats from trade_records (not trade_logs).
	trades, _ := s.tradeRecords.GetByAccountID(ctx, st.UserID, aid, start, end, 50)
	var totalProfit decimal.Decimal
	wins, losses := 0, 0
	var maxDD decimal.Decimal
	pbTrades := make([]*antv1.SharedTrade, 0, len(trades))
	for _, t := range trades {
		totalProfit = totalProfit.Add(t.Profit)
		if t.Profit.IsPositive() { wins++ } else { losses++ }
		if t.Profit.LessThan(maxDD) { maxDD = t.Profit }
		vol, _ := t.Volume.Float64()
		prof, _ := t.Profit.Float64()
		pbTrades = append(pbTrades, &antv1.SharedTrade{
			Symbol: t.Symbol, Side: t.OrderType, Volume: vol, Profit: prof,
			CloseTimeMs: t.CloseTime.UnixMilli(),
		})
	}
	totalRet, _ := totalProfit.Float64()
	winRate := 0.0
	if wins+losses > 0 { winRate = float64(wins) / float64(wins+losses) * 100 }
	maxDDval, _ := maxDD.Float64()

	// Additional metrics
	var totalVolume decimal.Decimal
	var profitFactor float64
	grossProfit, grossLoss := decimal.Zero, decimal.Zero
	var openTimeSum int64
	for _, t := range trades {
		totalVolume = totalVolume.Add(t.Volume)
		if t.Profit.IsPositive() {
			grossProfit = grossProfit.Add(t.Profit)
		} else {
			grossLoss = grossLoss.Add(t.Profit.Abs())
		}
		openTimeSum += t.CloseTime.Sub(t.OpenTime).Milliseconds()
	}
	if grossLoss.IsPositive() {
		pf, _ := grossProfit.Div(grossLoss).Float64()
		profitFactor = pf
	}
	avgHoldingMs := int64(0)
	if len(trades) > 0 {
		avgHoldingMs = openTimeSum / int64(len(trades))
	}
	totalVol, _ := totalVolume.Float64()

	resp := connect.NewResponse(&antv1.GetSharedPerformanceResponse{
		UserName: userName, TotalTrades: int32(len(trades)),
		TotalReturn: totalRet, WinRate: winRate, MaxDrawdown: maxDDval,
		EquityCurve: equityVals, Trades: pbTrades,
	})
	resp.Header().Set("X-Total-Volume", fmt.Sprintf("%.2f", totalVol))
	resp.Header().Set("X-Profit-Factor", fmt.Sprintf("%.2f", profitFactor))
	resp.Header().Set("X-Avg-Holding-Ms", fmt.Sprintf("%d", avgHoldingMs))
	return resp, nil
}

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

// HandleCreateShareTokenREST creates a share token via plain JSON (supports show_positions).
func (s *ShareServer) HandleCreateShareTokenREST(w http.ResponseWriter, r *http.Request) {
	tokenStr := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
	if tokenStr == "" || tokenStr == r.Header.Get("Authorization") {
		http.Error(w, `{"error":"login required"}`, http.StatusUnauthorized)
		return
	}
	claims, err := interceptor.ValidateToken(tokenStr, s.jwtSecret)
	if err != nil {
		http.Error(w, `{"error":"invalid token"}`, http.StatusUnauthorized)
		return
	}
	uid, err := uuid.Parse(claims.UserID)
	if err != nil || uid == uuid.Nil {
		http.Error(w, `{"error":"login required"}`, http.StatusUnauthorized)
		return
	}
	var req struct {
		AccountID     string `json:"account_id"`
		Description   string `json:"description"`
		ShowPositions bool   `json:"show_positions"`
		ExpireDays    int    `json:"expire_days"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid request"}`, http.StatusBadRequest)
		return
	}
	if req.ExpireDays <= 0 {
		req.ExpireDays = 7
	}
	b := make([]byte, 16)
	rand.Read(b)
	token := hex.EncodeToString(b)
	st := &repository.ShareToken{
		UserID: uid, AccountID: req.AccountID, Token: token,
		Description: req.Description, ShowPositions: req.ShowPositions,
		ExpiresAt: time.Now().Add(time.Duration(req.ExpireDays) * 24 * time.Hour),
	}
	if err := s.repo.Create(r.Context(), st); err != nil {
		s.log.Error("HandleCreateShareTokenREST: create failed", zap.Error(err))
		http.Error(w, `{"error":"create failed"}`, http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"token": token, "shareUrl": "/share/" + token,
		"expiresAt": st.ExpiresAt.Format(time.RFC3339),
	})
}

// HandleUpdateShareToken updates show_positions flag.
func (s *ShareServer) HandleUpdateShareToken(w http.ResponseWriter, r *http.Request) {
	tokenStr := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
	if tokenStr == "" {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}
	claims, err := interceptor.ValidateToken(tokenStr, s.jwtSecret)
	if err != nil {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}
	uid, _ := uuid.Parse(claims.UserID)
	var req struct {
		Token         string `json:"token"`
		ShowPositions *bool  `json:"show_positions"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid request"}`, http.StatusBadRequest)
		return
	}
	if req.ShowPositions != nil {
		s.repo.UpdateShowPositions(r.Context(), uid, req.Token, *req.ShowPositions)
	}
	w.Write([]byte(`{"ok":true}`))
}

// HandleListShareTokens returns the current user's share tokens with view counts.
func (s *ShareServer) HandleListShareTokens(w http.ResponseWriter, r *http.Request) {
	tokenStr := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
	if tokenStr == "" || tokenStr == r.Header.Get("Authorization") {
		http.Error(w, `{"error":"missing authorization header"}`, http.StatusUnauthorized)
		return
	}
	claims, err := interceptor.ValidateToken(tokenStr, s.jwtSecret)
	if err != nil {
		http.Error(w, `{"error":"invalid token"}`, http.StatusUnauthorized)
		return
	}
	uid, err := uuid.Parse(claims.UserID)
	if err != nil || uid == uuid.Nil {
		http.Error(w, `{"error":"invalid user"}`, http.StatusUnauthorized)
		return
	}
	tokens, err := s.repo.ListByUser(r.Context(), uid)
	if err != nil {
		s.log.Error("ListShareTokens: db", zap.Error(err))
		http.Error(w, `{"error":"internal"}`, http.StatusInternalServerError)
		return
	}
	type item struct {
		Token         string `json:"token"`
		ShareURL      string `json:"shareUrl"`
		Description   string `json:"description"`
		ShowPositions bool   `json:"showPositions"`
		ViewCount     int    `json:"viewCount"`
		ExpiresAt     string `json:"expiresAt"`
		CreatedAt     string `json:"createdAt"`
	}
	items := make([]item, 0, len(tokens))
	for _, t := range tokens {
		items = append(items, item{
			Token: t.Token, ShareURL: fmt.Sprintf("/share/%s", t.Token),
			Description: t.Description, ShowPositions: t.ShowPositions,
			ViewCount: t.ViewCount,
			ExpiresAt: t.ExpiresAt.Format(time.RFC3339),
			CreatedAt: t.CreatedAt.Format(time.RFC3339),
		})
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(items)
}

// HandleListAllShareTokens returns all share tokens (admin only).
func (s *ShareServer) HandleListAllShareTokens(w http.ResponseWriter, r *http.Request) {
	page := 1
	pageSize := 20
	if v := r.URL.Query().Get("page"); v != "" { fmt.Sscanf(v, "%d", &page) }
	if v := r.URL.Query().Get("pageSize"); v != "" { fmt.Sscanf(v, "%d", &pageSize) }
	limit, offset := pageSize, (page-1)*pageSize
	if limit > 100 { limit = 100 }

	tokens, total, err := s.repo.ListAll(r.Context(), limit, offset)
	if err != nil {
		s.log.Error("ListAllShareTokens: db", zap.Error(err))
		http.Error(w, `{"error":"internal"}`, http.StatusInternalServerError)
		return
	}
	type item struct {
		Token         string `json:"token"`
		ShareURL      string `json:"shareUrl"`
		UserID        string `json:"userId"`
		Description   string `json:"description"`
		ShowPositions bool   `json:"showPositions"`
		ViewCount     int    `json:"viewCount"`
		ExpiresAt     string `json:"expiresAt"`
		CreatedAt     string `json:"createdAt"`
	}
	items := make([]item, 0, len(tokens))
	for _, t := range tokens {
		items = append(items, item{
			Token: t.Token, ShareURL: fmt.Sprintf("/share/%s", t.Token),
			UserID: t.UserID.String(), Description: t.Description,
			ShowPositions: t.ShowPositions, ViewCount: t.ViewCount,
			ExpiresAt: t.ExpiresAt.Format(time.RFC3339),
			CreatedAt: t.CreatedAt.Format(time.RFC3339),
		})
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"items": items, "total": total, "page": page, "pageSize": pageSize,
	})
}

// HandleDeleteShareToken deletes a share token owned by the current user.
func (s *ShareServer) HandleDeleteShareToken(w http.ResponseWriter, r *http.Request) {
	tokenStr := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
	if tokenStr == "" || tokenStr == r.Header.Get("Authorization") {
		http.Error(w, `{"error":"missing authorization header"}`, http.StatusUnauthorized)
		return
	}
	claims, err := interceptor.ValidateToken(tokenStr, s.jwtSecret)
	if err != nil {
		http.Error(w, `{"error":"invalid token"}`, http.StatusUnauthorized)
		return
	}
	uid, err := uuid.Parse(claims.UserID)
	if err != nil || uid == uuid.Nil {
		http.Error(w, `{"error":"invalid user"}`, http.StatusUnauthorized)
		return
	}
	var req struct{ Token string `json:"token"` }
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Token == "" {
		http.Error(w, `{"error":"invalid request"}`, http.StatusBadRequest)
		return
	}
	if err := s.repo.DeleteByUser(r.Context(), uid, req.Token); err != nil {
		s.log.Error("DeleteShareToken: db", zap.Error(err))
		http.Error(w, `{"error":"delete failed"}`, http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"ok":true}`))
}

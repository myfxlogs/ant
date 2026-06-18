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
	"github.com/shopspring/decimal"
	"go.uber.org/zap"

	antv1 "anttrader/gen/proto/ant/v1"
	antv1c "anttrader/gen/proto/ant/v1/antv1connect"
	"anttrader/internal/interceptor"
	"anttrader/internal/repository"
)

type ShareServer struct {
	repo         *repository.ShareRepository
	tradeRecords *repository.TradeRecordRepository
	eqRepo       *repository.AnalyticsRepository
	userRepo     *repository.UserRepository
	jwtSecret    string
	log          *zap.Logger
}

var _ antv1c.ShareServiceHandler = (*ShareServer)(nil)

func NewShareServer(repo *repository.ShareRepository, tradeRecords *repository.TradeRecordRepository, eqRepo *repository.AnalyticsRepository, userRepo *repository.UserRepository, jwtSecret string, log *zap.Logger) *ShareServer {
	return &ShareServer{repo: repo, tradeRecords: tradeRecords, eqRepo: eqRepo, userRepo: userRepo, jwtSecret: jwtSecret, log: log}
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

	return connect.NewResponse(&antv1.GetSharedPerformanceResponse{
		UserName: userName, TotalTrades: int32(len(trades)),
		TotalReturn: totalRet, WinRate: winRate, MaxDrawdown: maxDDval,
		EquityCurve: equityVals, Trades: pbTrades,
	}), nil
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
		Token       string `json:"token"`
		ShareURL    string `json:"shareUrl"`
		Description string `json:"description"`
		ViewCount   int    `json:"viewCount"`
		ExpiresAt   string `json:"expiresAt"`
		CreatedAt   string `json:"createdAt"`
	}
	items := make([]item, 0, len(tokens))
	for _, t := range tokens {
		items = append(items, item{
			Token: t.Token, ShareURL: fmt.Sprintf("/share/%s", t.Token),
			Description: t.Description, ViewCount: t.ViewCount,
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
		Token       string `json:"token"`
		ShareURL    string `json:"shareUrl"`
		UserID      string `json:"userId"`
		Description string `json:"description"`
		ViewCount   int    `json:"viewCount"`
		ExpiresAt   string `json:"expiresAt"`
		CreatedAt   string `json:"createdAt"`
	}
	items := make([]item, 0, len(tokens))
	for _, t := range tokens {
		items = append(items, item{
			Token: t.Token, ShareURL: fmt.Sprintf("/share/%s", t.Token),
			UserID: t.UserID.String(), Description: t.Description,
			ViewCount: t.ViewCount,
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

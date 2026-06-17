package user

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
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
	repo     *repository.ShareRepository
	tradeLog *repository.TradeLogRepository
	eqRepo   *repository.AnalyticsRepository
	userRepo *repository.UserRepository
	log      *zap.Logger
}

var _ antv1c.ShareServiceHandler = (*ShareServer)(nil)

func NewShareServer(repo *repository.ShareRepository, tradeLog *repository.TradeLogRepository, eqRepo *repository.AnalyticsRepository, userRepo *repository.UserRepository, log *zap.Logger) *ShareServer {
	return &ShareServer{repo: repo, tradeLog: tradeLog, eqRepo: eqRepo, userRepo: userRepo, log: log}
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

	// Recent trades + basic stats
	trades, _ := s.tradeLog.ListByAccountID(ctx, aid, 0, 50)
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
			Symbol: t.Symbol, Side: t.Action, Volume: vol, Profit: prof,
			CloseTimeMs: t.CreatedAt.UnixMilli(),
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

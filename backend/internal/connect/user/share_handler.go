package user

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"strconv"
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
	equityVals := make([]string, 0, len(equityPoints))
	for _, p := range equityPoints {
		equityVals = append(equityVals, p.Equity.String())
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
			Symbol: t.Symbol, Side: t.OrderType, Volume: strconv.FormatFloat(vol, 'f', -1, 64), Profit: strconv.FormatFloat(prof, 'f', -1, 64),
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

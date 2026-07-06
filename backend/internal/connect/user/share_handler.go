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
	"anttrader/internal/model"
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

	// Recent trades + stats from trade_records.
	trades, _ := s.tradeRecords.GetByAccountID(ctx, st.UserID, aid, start, end, 50)
	stats := summarizeTrades(trades)
	pbTrades := make([]*antv1.SharedTrade, 0, len(trades))
	for _, t := range trades {
		vol, _ := t.Volume.Float64()
		prof, _ := t.Profit.Float64()
		pbTrades = append(pbTrades, &antv1.SharedTrade{
			Symbol: t.Symbol, Side: t.OrderType, Volume: strconv.FormatFloat(vol, 'f', -1, 64), Profit: strconv.FormatFloat(prof, 'f', -1, 64),
			CloseTimeMs: t.CloseTime.UnixMilli(),
		})
	}

	resp := connect.NewResponse(&antv1.GetSharedPerformanceResponse{
		UserName: userName, TotalTrades: int32(len(trades)),
		TotalReturn: stats.totalReturn(), WinRate: stats.winRate(), MaxDrawdown: stats.maxDrawdown(),
		EquityCurve: equityVals, Trades: pbTrades,
	})
	resp.Header().Set("X-Total-Volume", fmt.Sprintf("%.2f", stats.totalVol()))
	resp.Header().Set("X-Profit-Factor", fmt.Sprintf("%.2f", stats.profitFactor()))
	resp.Header().Set("X-Avg-Holding-Ms", fmt.Sprintf("%d", stats.avgHoldingMs()))
	return resp, nil
}

// tradeSummary holds computed metrics from a set of trades.
// Used by both the ConnectRPC and HTTP share handlers.
type tradeSummary struct {
	totalProfit, totalVolume, grossProfit, grossLoss, maxDD decimal.Decimal
	wins, losses                                            int
	openTimeSum                                             int64
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
		if t.Profit.LessThan(s.maxDD) {
			s.maxDD = t.Profit
		}
		s.openTimeSum += t.CloseTime.Sub(t.OpenTime).Milliseconds()
	}
	return s
}

func (s tradeSummary) totalReturn() float64 {
	v, _ := s.totalProfit.Float64()
	return v
}

func (s tradeSummary) winRate() float64 {
	if s.wins+s.losses == 0 {
		return 0
	}
	return float64(s.wins) / float64(s.wins+s.losses) * 100
}

func (s tradeSummary) maxDrawdown() float64 {
	v, _ := s.maxDD.Float64()
	return v
}

func (s tradeSummary) profitFactor() float64 {
	if !s.grossLoss.IsPositive() {
		return 0
	}
	pf, _ := s.grossProfit.Div(s.grossLoss).Float64()
	return pf
}

func (s tradeSummary) avgHoldingMs() int64 {
	if len := s.wins + s.losses; len > 0 {
		return s.openTimeSum / int64(len)
	}
	return 0
}

func (s tradeSummary) totalVol() float64 {
	v, _ := s.totalVolume.Float64()
	return v
}

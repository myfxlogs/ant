package user

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"math"
	"strconv"
	"time"

	"connectrpc.com/connect"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/shopspring/decimal"
	"go.uber.org/zap"

	antv1 "alphaforge/gen/proto/ant/v1"
	antv1c "alphaforge/gen/proto/ant/v1/antv1connect"
	"alphaforge/internal/interceptor"
	"alphaforge/internal/model"
	"alphaforge/internal/mthub"
	"alphaforge/internal/repository"
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
	if expireDays <= 0 {
		expireDays = 7
	}
	b := make([]byte, 16)
	rand.Read(b)
	token := hex.EncodeToString(b)
	st := &repository.ShareToken{
		UserID: uid, AccountID: req.Msg.AccountId, Token: token,
		Description: req.Msg.Description, ShowPositions: req.Msg.ShowPositions,
		ExpiresAt: time.Now().Add(time.Duration(expireDays) * 24 * time.Hour),
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
	if err != nil || st == nil || time.Now().After(st.ExpiresAt) {
		return connect.NewResponse(&antv1.GetSharedPerformanceResponse{Expired: true}), nil
	}
	s.repo.IncrementView(ctx, st.Token)

	user, _ := s.userRepo.GetByID(ctx, st.UserID)
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
	// Equity curve
	start := time.Now().AddDate(-1, 0, 0)
	end := time.Now()
	equityPoints, _ := s.eqRepo.GetEquityCurve(ctx, aid, start, end)
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

	// Recent trades + stats from trade_records.
	trades, _ := s.tradeRecords.GetByAccountID(ctx, st.UserID, aid, start, end, 50)
	stats := summarizeTrades(trades)
	pbTrades := make([]*antv1.SharedTrade, 0, len(trades))
	for _, t := range trades {
		pbTrades = append(pbTrades, &antv1.SharedTrade{
			Symbol: t.Symbol, Side: t.OrderType, Volume: t.Volume.String(), Profit: t.Profit.String(),
			CloseTimeMs: t.CloseTime.UnixMilli(),
		})
	}

	// Sharpe ratio from equity curve.
	sharpeStr := "0"
	if len(equityVals) > 1 {
		var sum, sumSq decimal.Decimal
		dailyReturns := make([]decimal.Decimal, 0, len(equityVals)-1)
		for i := 1; i < len(equityVals); i++ {
			prev, err := decimal.NewFromString(equityVals[i-1])
			if err != nil || prev.IsZero() {
				continue
			}
			curr, err := decimal.NewFromString(equityVals[i])
			if err != nil {
				continue
			}
			r := curr.Sub(prev).Div(prev)
			dailyReturns = append(dailyReturns, r)
			sum = sum.Add(r)
		}
		if len(dailyReturns) > 1 {
			n := decimal.NewFromInt(int64(len(dailyReturns)))
			mean := sum.Div(n)
			for _, r := range dailyReturns {
				diff := r.Sub(mean)
				sumSq = sumSq.Add(diff.Mul(diff))
			}
			variance := sumSq.Div(n.Sub(decimal.NewFromInt(1)))
			if variance.IsPositive() {
				vFloat, _ := variance.Float64()
				std := math.Sqrt(vFloat)
				if std > 0 {
					meanFloat, _ := mean.Float64()
					sharpe := meanFloat / std * math.Sqrt(252)
					sharpeStr = strconv.FormatFloat(sharpe, 'f', 4, 64)
				}
			}
		}
	}

	// Positions — only if the share token allows it.
	var pbPositions []*antv1.SharedPosition
	if st.ShowPositions {
		if cached, _ := s.repo.GetPositionsSnapshot(ctx, st.Token); cached != nil {
			pbPositions = cached
		} else if s.mthub != nil {
			if orders, err := s.mthub.OpenedOrders(ctx, st.AccountID); err == nil {
				pbPositions = make([]*antv1.SharedPosition, 0, len(orders))
				for _, o := range orders {
					side := "BUY"
					if o.Side == -1 {
						side = "SELL"
					}
					pbPositions = append(pbPositions, &antv1.SharedPosition{
						Symbol: o.SymbolRaw, Type: side,
						Volume: o.Volume.String(), OpenPrice: o.OpenPrice.String(), Profit: o.Profit.String(),
					})
				}
				_ = s.repo.SetPositionsSnapshot(ctx, st.Token, pbPositions)
			}
		}
	}

	resp := connect.NewResponse(&antv1.GetSharedPerformanceResponse{
		UserName: userName, TotalTrades: int32(len(trades)),
		TotalReturn: stats.totalReturnStr(), WinRate: stats.winRateStr(), MaxDrawdown: stats.maxDrawdownStr(),
		SharpeRatio: sharpeStr,
		EquityCurve: equityVals, EquityTimesMs: equityTimesMs, Trades: pbTrades,
		TotalVolume: stats.totalVolStr(), ProfitFactor: stats.profitFactorStr(),
		AvgHoldingMs: stats.avgHoldingMs(), ShowPositions: st.ShowPositions,
		Positions: pbPositions,
	})
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

func (s tradeSummary) totalReturnStr() string {
	return s.totalProfit.String()
}

func (s tradeSummary) winRateStr() string {
	if s.wins+s.losses == 0 {
		return "0"
	}
	return decimal.NewFromInt(int64(s.wins)).Div(decimal.NewFromInt(int64(s.wins + s.losses))).Mul(decimal.NewFromInt(100)).String()
}

func (s tradeSummary) maxDrawdownStr() string {
	return s.maxDD.String()
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

func (s *ShareServer) UpdateShareToken(ctx context.Context, req *connect.Request[antv1.UpdateShareTokenRequest]) (*connect.Response[antv1.UpdateShareTokenResponse], error) {
	uid, _ := uuid.Parse(interceptor.GetUserID(ctx))
	if uid == uuid.Nil {
		return nil, connect.NewError(connect.CodeUnauthenticated, fmt.Errorf("login required"))
	}
	if err := s.repo.UpdateShowPositions(ctx, uid, req.Msg.Token, req.Msg.ShowPositions); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&antv1.UpdateShareTokenResponse{}), nil
}

func (s *ShareServer) ListShareTokens(ctx context.Context, req *connect.Request[antv1.ListShareTokensRequest]) (*connect.Response[antv1.ListShareTokensResponse], error) {
	uid, _ := uuid.Parse(interceptor.GetUserID(ctx))
	if uid == uuid.Nil {
		return nil, connect.NewError(connect.CodeUnauthenticated, fmt.Errorf("login required"))
	}
	tokens, err := s.repo.ListByUser(ctx, uid)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	items := make([]*antv1.ShareTokenItem, 0, len(tokens))
	for _, t := range tokens {
		items = append(items, &antv1.ShareTokenItem{
			Token: t.Token, ShareUrl: fmt.Sprintf("/share/%s", t.Token),
			Description: t.Description, ShowPositions: t.ShowPositions,
			ViewCount: int32(t.ViewCount),
			ExpiresAt: t.ExpiresAt.Format(time.RFC3339),
			CreatedAt: t.CreatedAt.Format(time.RFC3339),
		})
	}
	return connect.NewResponse(&antv1.ListShareTokensResponse{Items: items}), nil
}

func (s *ShareServer) DeleteShareToken(ctx context.Context, req *connect.Request[antv1.DeleteShareTokenRequest]) (*connect.Response[antv1.DeleteShareTokenResponse], error) {
	uid, _ := uuid.Parse(interceptor.GetUserID(ctx))
	if uid == uuid.Nil {
		return nil, connect.NewError(connect.CodeUnauthenticated, fmt.Errorf("login required"))
	}
	if err := s.repo.DeleteByUser(ctx, uid, req.Msg.Token); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&antv1.DeleteShareTokenResponse{}), nil
}

func (s *ShareServer) ListAllShareTokens(ctx context.Context, req *connect.Request[antv1.ListAllShareTokensRequest]) (*connect.Response[antv1.ListAllShareTokensResponse], error) {
	uid, _ := uuid.Parse(interceptor.GetUserID(ctx))
	if uid == uuid.Nil {
		return nil, connect.NewError(connect.CodeUnauthenticated, fmt.Errorf("login required"))
	}
	page := int(req.Msg.Page)
	pageSize := int(req.Msg.PageSize)
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 20
	}
	if pageSize > 100 {
		pageSize = 100
	}
	limit, offset := pageSize, (page-1)*pageSize
	tokens, total, err := s.repo.ListAll(ctx, limit, offset)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	items := make([]*antv1.AdminShareTokenItem, 0, len(tokens))
	for _, t := range tokens {
		items = append(items, &antv1.AdminShareTokenItem{
			Token: t.Token, ShareUrl: fmt.Sprintf("/share/%s", t.Token),
			UserId: t.UserID.String(), Description: t.Description,
			ShowPositions: t.ShowPositions, ViewCount: int32(t.ViewCount),
			ExpiresAt: t.ExpiresAt.Format(time.RFC3339),
			CreatedAt: t.CreatedAt.Format(time.RFC3339),
		})
	}
	return connect.NewResponse(&antv1.ListAllShareTokensResponse{
		Items: items, Total: int32(total), Page: int32(page), PageSize: int32(pageSize),
	}), nil
}

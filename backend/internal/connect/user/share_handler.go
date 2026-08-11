package user

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"connectrpc.com/connect"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"

	antv1 "alphaforge/gen/proto/ant/v1"
	antv1c "alphaforge/gen/proto/ant/v1/antv1connect"
	"alphaforge/internal/interceptor"
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
		Token: token, ShareUrl: fmtShareURL(token),
		ExpiresAt: st.ExpiresAt.Format(time.RFC3339),
	}), nil
}

func (s *ShareServer) GetSharedPerformance(ctx context.Context, req *connect.Request[antv1.GetSharedPerformanceRequest]) (*connect.Response[antv1.GetSharedPerformanceResponse], error) {
	st, err := s.repo.GetByToken(ctx, req.Msg.Token)
	if err != nil || st == nil || time.Now().After(st.ExpiresAt) {
		return connect.NewResponse(&antv1.GetSharedPerformanceResponse{Expired: true}), nil
	}
	_ = s.repo.IncrementView(ctx, st.Token)

	perf, err := BuildSharePerformance(ctx, st, s.userRepo, s.eqRepo, s.tradeRecords, s.mthub)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	// Query decay_status from marketplace_strategies via linked_account_id (real-time value).
	var decayStatus string
	err = s.pg.QueryRow(ctx, buildShareDecayStatusQuery(), st.AccountID).Scan(&decayStatus)
	decayStatus = resolveDecayStatus(err, s.log, st.AccountID, decayStatus)

	// Format trades for proto.
	sharedTrades := FormatSharedTrades(perf.Trades)
	pbTrades := make([]*antv1.SharedTrade, 0, len(sharedTrades))
	for _, t := range sharedTrades {
		pbTrades = append(pbTrades, &antv1.SharedTrade{
			Symbol: t.Symbol, Side: t.Side, Volume: t.Volume, Profit: t.Profit,
			CloseTimeMs: t.CloseTimeMs,
		})
	}

	// Positions — only if the share token allows it.
	var pbPositions []*antv1.SharedPosition
	if st.ShowPositions {
		if cached, _ := s.repo.GetPositionsSnapshot(ctx, st.Token); cached != nil {
			pbPositions = cached
		} else {
			positions := FormatSharedPositions(ctx, s.mthub, st.AccountID)
			if len(positions) > 0 {
				pbPositions = make([]*antv1.SharedPosition, 0, len(positions))
				for _, p := range positions {
					pbPositions = append(pbPositions, &antv1.SharedPosition{
						Symbol: p.Symbol, Type: p.Type,
						Volume: p.Volume, OpenPrice: p.OpenPrice, Profit: p.Profit,
					})
				}
				_ = s.repo.SetPositionsSnapshot(ctx, st.Token, pbPositions)
			}
		}
	}

	// Trade stats + symbol stats from backend (zero-trust: no frontend computation).
	var pbTradeStats *antv1.ShareTradeStats
	if perf.TradeStats.BestTrade != "" || perf.TradeStats.WorstTrade != "" {
		pbTradeStats = &antv1.ShareTradeStats{
			WinningTrades: int32(perf.TradeStats.WinningTrades),
			LosingTrades:  int32(perf.TradeStats.LosingTrades),
			BestTrade:     perf.TradeStats.BestTrade,
			WorstTrade:    perf.TradeStats.WorstTrade,
			AvgWin:        perf.TradeStats.AvgWin,
			AvgLoss:       perf.TradeStats.AvgLoss,
		}
	}
	pbSymbolStats := make([]*antv1.ShareSymbolStat, 0, len(perf.SymbolStats))
	for _, s := range perf.SymbolStats {
		pbSymbolStats = append(pbSymbolStats, &antv1.ShareSymbolStat{
			Symbol: s.Symbol, Count: int32(s.Count), Net: s.Net,
		})
	}

	return connect.NewResponse(&antv1.GetSharedPerformanceResponse{
		UserName: perf.UserName, TotalTrades: int32(perf.TotalTrades),
		TotalReturn: perf.TotalReturn, WinRate: perf.WinRate, MaxDrawdown: perf.MaxDrawdown,
		SharpeRatio: perf.SharpeRatio,
		EquityCurve: perf.EquityCurve, EquityTimesMs: perf.EquityTimesMs, Trades: pbTrades,
		TotalVolume: perf.TotalVolume, ProfitFactor: perf.ProfitFactor,
		AvgHoldingMs: perf.AvgHoldingMs, ShowPositions: st.ShowPositions,
		Positions:   pbPositions,
		TradeStats:  pbTradeStats,
		SymbolStats: pbSymbolStats,
		DecayStatus: decayStatus,
	}), nil
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
			Token: t.Token, ShareUrl: fmtShareURL(t.Token),
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
			Token: t.Token, ShareUrl: fmtShareURL(t.Token),
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

// buildShareDecayStatusQuery returns the SQL string for querying decay_status
// from marketplace_strategies. ORDER BY updated_at DESC LIMIT 1 ensures
// deterministic row selection when multiple published rows exist.
func buildShareDecayStatusQuery() string {
	return `SELECT COALESCE(decay_status, 'none') FROM marketplace_strategies WHERE linked_account_id = $1 AND status = 'published' ORDER BY updated_at DESC LIMIT 1`
}

// resolveDecayStatus handles the error from the decay_status query:
// ErrNoRows → "none" (no published strategy for this account)
// other errors → log + "none" (never swallow silently)
// nil error → return the scanned value as-is
func resolveDecayStatus(err error, log *zap.Logger, accountID string, scanned string) string {
	if err == nil {
		return scanned
	}
	if errors.Is(err, pgx.ErrNoRows) {
		return "none"
	}
	log.Warn("share: failed to query decay_status", zap.String("account_id", accountID), zap.Error(err))
	return "none"
}

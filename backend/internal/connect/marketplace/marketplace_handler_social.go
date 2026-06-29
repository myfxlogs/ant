package marketplace

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"go.uber.org/zap"

	"connectrpc.com/connect"
	"google.golang.org/protobuf/types/known/timestamppb"

	antv1 "anttrader/gen/proto/ant/v1"
	"anttrader/internal/interceptor"
	"anttrader/internal/marketplace"
)

// --- Rating ---------------------------------------------------------------

func (s *MarketplaceServer) RateStrategy(ctx context.Context, req *connect.Request[antv1.RateStrategyRequest]) (*connect.Response[antv1.RateStrategyResponse], error) {
	userID := interceptor.GetUserID(ctx)
	if userID == "" {
		return nil, connect.NewError(connect.CodeUnauthenticated, fmt.Errorf("authentication required"))
	}
	avg, count, err := s.svc.Rate(ctx, userID, req.Msg.StrategyId, req.Msg.Rating)
	if err != nil {
		s.log.Error("RateStrategy", zap.Error(err))
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&antv1.RateStrategyResponse{AvgRating: avg, RatingCount: count}), nil
}

func (s *MarketplaceServer) ListRatings(ctx context.Context, req *connect.Request[antv1.ListRatingsRequest]) (*connect.Response[antv1.ListRatingsResponse], error) {
	items, avg, count, err := s.svc.ListRatings(ctx, req.Msg.StrategyId)
	if err != nil {
		s.log.Error("ListRatings", zap.Error(err))
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	resp := &antv1.ListRatingsResponse{AvgRating: avg, RatingCount: count}
	for _, r := range items {
		resp.Ratings = append(resp.Ratings, &antv1.RatingItem{
			Id: r.ID, UserId: r.UserID, Rating: r.Rating,
			CreatedAt: timestamppb.New(r.CreatedAt),
		})
	}
	return connect.NewResponse(resp), nil
}

// --- Comment ---------------------------------------------------------------

func (s *MarketplaceServer) CommentOnStrategy(ctx context.Context, req *connect.Request[antv1.CommentOnStrategyRequest]) (*connect.Response[antv1.CommentOnStrategyResponse], error) {
	userID := interceptor.GetUserID(ctx)
	if userID == "" {
		return nil, connect.NewError(connect.CodeUnauthenticated, fmt.Errorf("authentication required"))
	}
	id, err := s.svc.Comment(ctx, userID, req.Msg.StrategyId, req.Msg.Content)
	if err != nil {
		s.log.Error("CommentOnStrategy", zap.Error(err))
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&antv1.CommentOnStrategyResponse{Id: id}), nil
}

func (s *MarketplaceServer) ListComments(ctx context.Context, req *connect.Request[antv1.ListCommentsRequest]) (*connect.Response[antv1.ListCommentsResponse], error) {
	items, total, err := s.svc.ListComments(ctx, req.Msg.StrategyId, req.Msg.Limit, req.Msg.Offset)
	if err != nil {
		s.log.Error("ListComments", zap.Error(err))
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	resp := &antv1.ListCommentsResponse{Total: total}
	for _, c := range items {
		resp.Comments = append(resp.Comments, &antv1.CommentItem{
			Id: c.ID, UserId: c.UserID, UserName: c.UserName,
			Content: c.Content, CreatedAt: timestamppb.New(c.CreatedAt),
		})
	}
	return connect.NewResponse(resp), nil
}

// --- Admin Pricing ---------------------------------------------------------

func (s *MarketplaceServer) SetStrategyPricing(ctx context.Context, req *connect.Request[antv1.SetStrategyPricingRequest]) (*connect.Response[antv1.SetStrategyPricingResponse], error) {
	uid, err := uuid.Parse(interceptor.GetUserID(ctx))
	if err != nil {
		return nil, connect.NewError(connect.CodeUnauthenticated, err)
	}
	ok, err := s.admin.IsAdmin(ctx, uid)
	if err != nil || !ok {
		return nil, connect.NewError(connect.CodePermissionDenied, fmt.Errorf("admin required"))
	}
	m := req.Msg
	if err := s.svc.SetPricing(ctx, m.StrategyId, m.PriceModel, parseFloat64(m.PriceAmount), m.PlatformFeeRate); err != nil {
		s.log.Error("SetStrategyPricing", zap.Error(err))
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&antv1.SetStrategyPricingResponse{
		StrategyId: m.StrategyId, PriceModel: m.PriceModel, PriceAmount: m.PriceAmount, PlatformFeeRate: m.PlatformFeeRate,
	}), nil
}

// --- Unpublish Strategy ---

func (s *MarketplaceServer) UnpublishStrategy(ctx context.Context, req *connect.Request[antv1.UnpublishMarketStrategyRequest]) (*connect.Response[antv1.UnpublishMarketStrategyResponse], error) {
	userID := interceptor.GetUserID(ctx)
	if userID == "" {
		return nil, connect.NewError(connect.CodeUnauthenticated, fmt.Errorf("authentication required"))
	}
	uid, _ := uuid.Parse(userID)
	isAdmin, _ := s.admin.IsAdmin(ctx, uid)

	m := req.Msg
	if err := s.svc.Unpublish(ctx, m.StrategyId, userID, isAdmin); err != nil {
		s.log.Error("UnpublishStrategy", zap.Error(err))
		msg := err.Error()
		if strings.Contains(msg, "only the publisher") {
			return nil, connect.NewError(connect.CodePermissionDenied, err)
		}
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&antv1.UnpublishMarketStrategyResponse{
		StrategyId: m.StrategyId, Status: "hidden",
	}), nil
}

// --- Publisher Stats ---

func (s *MarketplaceServer) GetPublisherStats(ctx context.Context, req *connect.Request[antv1.GetPublisherStatsRequest]) (*connect.Response[antv1.GetPublisherStatsResponse], error) {
	userID := interceptor.GetUserID(ctx)
	if userID == "" {
		return nil, connect.NewError(connect.CodeUnauthenticated, fmt.Errorf("authentication required"))
	}
	stats, err := s.svc.GetPublisherStats(ctx, userID)
	if err != nil {
		s.log.Error("GetPublisherStats", zap.Error(err))
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&antv1.GetPublisherStatsResponse{
		TotalPublished:   stats.TotalPublished,
		TotalSubscribers: stats.TotalSubscribers,
		TotalRevenue:     stats.TotalRevenue,
		MonthlyRevenue:   stats.MonthlyRevenue,
		AvgRating:        stats.AvgRating,
		TopStrategyId:    stats.TopStrategyID,
		TopStrategyTitle: stats.TopStrategyTitle,
		TopStrategySubs:  stats.TopStrategySubs,
	}), nil
}

// --- Marketplace Backtest ---

func (s *MarketplaceServer) RunMarketBacktest(ctx context.Context, req *connect.Request[antv1.RunMarketBacktestRequest], stream *connect.ServerStream[antv1.BacktestRunUpdate]) error {
	m := req.Msg
	userID := interceptor.GetUserID(ctx)
	if userID == "" {
		return connect.NewError(connect.CodeUnauthenticated, fmt.Errorf("authentication required"))
	}

	dir := "both"
	if m.ExecutionConfig != nil {
		switch m.ExecutionConfig.TradeDirection {
		case antv1.TradeDirection_TRADE_DIRECTION_LONG:
			dir = "long"
		case antv1.TradeDirection_TRADE_DIRECTION_SHORT:
			dir = "short"
		}
	}
	params := marketplace.StartBacktestParams{
		UserID:         userID,
		StrategyID:     m.StrategyId,
		Symbol:         m.Symbol,
		Timeframe:      m.Timeframe,
		StartDateMs:    m.StartDateMs,
		EndDateMs:      m.EndDateMs,
		InitialCapital: parseDecimal(m.InitialCapital),
		TradeDirection: dir,
	}
	if m.ExecutionConfig != nil {
		params.Commission = parseDecimal(m.ExecutionConfig.Commission)
		params.Slippage = parseDecimal(m.ExecutionConfig.Slippage)
		params.Leverage = parseDecimal(m.ExecutionConfig.Leverage)
	}
	runID, err := s.svc.StartMarketBacktest(ctx, params)
	if err != nil {
		s.log.Error("RunMarketBacktest", zap.Error(err))
		msg := err.Error()
		if strings.Contains(msg, "access denied") {
			return connect.NewError(connect.CodePermissionDenied, err)
		}
		if strings.Contains(msg, "not found") {
			return connect.NewError(connect.CodeNotFound, err)
		}
		return connect.NewError(connect.CodeInternal, err)
	}

	runUUID, _ := uuid.Parse(runID)
	return s.streamBacktestProgress(ctx, runUUID, stream)
}

func parseFloat64(s string) float64 {
	f, _ := strconv.ParseFloat(s, 64)
	return f
}

func parseDecimal(s string) decimal.Decimal {
	d, err := decimal.NewFromString(s)
	if err != nil {
		return decimal.Zero
	}
	return d
}

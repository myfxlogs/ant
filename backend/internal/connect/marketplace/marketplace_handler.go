package marketplace

import (
	"context"

	"go.uber.org/zap"

	"connectrpc.com/connect"
	"google.golang.org/protobuf/types/known/timestamppb"

	antv1 "anttrader/gen/proto/ant/v1"
	antv1c "anttrader/gen/proto/ant/v1/antv1connect"
	"anttrader/internal/marketplace"
)

// MarketplaceServer implements ant.v1.MarketplaceServiceHandler.
type MarketplaceServer struct {
	svc *marketplace.Service
	log *zap.Logger
}

var _ antv1c.MarketplaceServiceHandler = (*MarketplaceServer)(nil)

func NewMarketplaceServer(svc *marketplace.Service, log *zap.Logger) *MarketplaceServer {
	return &MarketplaceServer{svc: svc, log: log}
}

func (s *MarketplaceServer) PublishStrategy(ctx context.Context, req *connect.Request[antv1.PublishStrategyRequest]) (*connect.Response[antv1.PublishStrategyResponse], error) {
	m := req.Msg
	id, err := s.svc.Publish(ctx, marketplace.PublishParams{
		UserID:      m.UserId,
		StrategyID:  m.StrategyId,
		Title:       m.Title,
		Description: m.Description,
		PriceModel:  m.PriceModel,
		PriceAmount: m.PriceAmount,
		AssetClass:  m.AssetClass,
		Symbols:     m.Symbols,
		Timeframe:   m.Timeframe,
		RiskLevel:   m.RiskLevel,
		Tags:        m.Tags,
	})
	if err != nil {
		s.log.Error("PublishStrategy", zap.Error(err))
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&antv1.PublishStrategyResponse{PublishId: id}), nil
}

func (s *MarketplaceServer) Subscribe(ctx context.Context, req *connect.Request[antv1.SubscribeRequest]) (*connect.Response[antv1.SubscribeResponse], error) {
	m := req.Msg
	id, err := s.svc.Subscribe(ctx, m.UserId, m.PublisherUserId, m.StrategyId, m.Kind)
	if err != nil {
		s.log.Error("Subscribe", zap.Error(err))
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&antv1.SubscribeResponse{SubscriptionId: id}), nil
}

func (s *MarketplaceServer) Unsubscribe(ctx context.Context, req *connect.Request[antv1.UnsubscribeRequest]) (*connect.Response[antv1.UnsubscribeResponse], error) {
	m := req.Msg
	if err := s.svc.Unsubscribe(ctx, m.UserId, m.SubscriptionId); err != nil {
		s.log.Error("Unsubscribe", zap.Error(err))
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&antv1.UnsubscribeResponse{}), nil
}

func (s *MarketplaceServer) ListPublished(ctx context.Context, req *connect.Request[antv1.ListPublishedRequest]) (*connect.Response[antv1.ListPublishedResponse], error) {
	m := req.Msg
	list, err := s.svc.ListPublished(ctx, m.UserId, int(m.Limit), m.AssetClass, m.Keyword, m.SortBy)
	if err != nil {
		s.log.Error("ListPublished", zap.Error(err))
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	resp := &antv1.ListPublishedResponse{}
	for _, p := range list {
		item := &antv1.PublishedStrategy{
			PublishId:        p.PublishID,
			StrategyId:       p.StrategyID,
			StrategyName:     p.StrategyName,
			PublisherUserId:  p.PublisherUserID,
			PublishedAt:      timestamppb.New(p.PublishedAt),
			Title:            p.Title,
			Description:      p.Description,
			PriceModel:       p.PriceModel,
			AssetClass:       p.AssetClass,
			Symbols:          p.Symbols,
			RiskLevel:        p.RiskLevel,
			Tags:             p.Tags,
			TotalSubscribers: int32(p.TotalSubscribers),
		}
		if p.PriceAmount != nil {
			item.PriceAmount = *p.PriceAmount
		}
		if p.Timeframe != nil {
			item.Timeframe = *p.Timeframe
		}
		if p.WinRate != nil {
			item.WinRate = *p.WinRate
		}
		if p.TotalPnL != nil {
			item.TotalPnl = *p.TotalPnL
		}
			item.AvgRating = p.AvgRating
			item.RatingCount = p.RatingCount
		resp.Strategies = append(resp.Strategies, item)
	}
	return connect.NewResponse(resp), nil
}

func (s *MarketplaceServer) ListSubscriptions(ctx context.Context, req *connect.Request[antv1.ListSubscriptionsRequest]) (*connect.Response[antv1.ListSubscriptionsResponse], error) {
	list, err := s.svc.ListSubscriptions(ctx, req.Msg.UserId)
	if err != nil {
		s.log.Error("ListSubscriptions", zap.Error(err))
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	resp := &antv1.ListSubscriptionsResponse{}
	for _, sub := range list {
		resp.Subscriptions = append(resp.Subscriptions, &antv1.SubscriptionItem{
			SubscriptionId: sub.SubscriptionID, TargetUserId: sub.TargetUserID,
			StrategyId: sub.StrategyID, Kind: sub.Kind,
			Active: sub.Active, CreatedAt: timestamppb.New(sub.CreatedAt),
		})
	}
	return connect.NewResponse(resp), nil
}

// --- Rating ---------------------------------------------------------------

func (s *MarketplaceServer) RateStrategy(ctx context.Context, req *connect.Request[antv1.RateStrategyRequest]) (*connect.Response[antv1.RateStrategyResponse], error) {
	m := req.Msg
	avg, count, err := s.svc.Rate(ctx, m.UserId, m.StrategyId, m.Rating)
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
	m := req.Msg
	id, err := s.svc.Comment(ctx, m.UserId, m.StrategyId, m.Content)
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
	m := req.Msg
	if err := s.svc.SetPricing(ctx, m.StrategyId, m.PriceModel, m.PriceAmount); err != nil {
		s.log.Error("SetStrategyPricing", zap.Error(err))
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&antv1.SetStrategyPricingResponse{
		StrategyId: m.StrategyId, PriceModel: m.PriceModel, PriceAmount: m.PriceAmount,
	}), nil
}

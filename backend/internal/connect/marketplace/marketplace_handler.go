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
	id, err := s.svc.Publish(ctx, m.UserId, m.StrategyId)
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
	list, err := s.svc.ListPublished(ctx, m.UserId, int(m.Limit))
	if err != nil {
		s.log.Error("ListPublished", zap.Error(err))
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	resp := &antv1.ListPublishedResponse{}
	for _, p := range list {
		resp.Strategies = append(resp.Strategies, &antv1.PublishedStrategy{
			PublishId: p.PublishID, StrategyId: p.StrategyID,
			StrategyName: p.StrategyName, PublisherUserId: p.PublisherUserID,
			PublishedAt: timestamppb.New(p.PublishedAt),
		})
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

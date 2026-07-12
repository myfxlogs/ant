package marketplace

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"go.uber.org/zap"

	"connectrpc.com/connect"
	"google.golang.org/protobuf/types/known/timestamppb"

	antv1 "alphaforge/gen/proto/ant/v1"
	"alphaforge/internal/interceptor"
	"alphaforge/internal/marketplace"
)

func (s *MarketplaceServer) PublishStrategy(ctx context.Context, req *connect.Request[antv1.PublishStrategyRequest]) (*connect.Response[antv1.PublishStrategyResponse], error) {
	m := req.Msg
	var snapshotJSON *string
	if m.BacktestSnapshot != nil {
		if b, err := json.Marshal(m.BacktestSnapshot); err == nil {
			s := string(b)
			snapshotJSON = &s
		}
	}
	userID := interceptor.GetUserID(ctx)
	if userID == "" {
		return nil, connect.NewError(connect.CodeUnauthenticated, fmt.Errorf("authentication required"))
	}
	id, err := s.svc.Publish(ctx, marketplace.PublishParams{
		UserID:               userID,
		StrategyID:           m.StrategyId,
		Title:                m.Title,
		Description:          m.Description,
		PriceModel:           m.PriceModel,
		PriceAmount:          m.PriceAmount,
		AssetClass:           m.AssetClass,
		Symbols:              m.Symbols,
		Timeframe:            m.Timeframe,
		RiskLevel:            m.RiskLevel,
		Tags:                 m.Tags,
		CodeSnippet:          m.CodeSnippet,
		BacktestSnapshotJSON: snapshotJSON,
		PlatformFeeRate:      s.svc.GetPlatformFeeRate(ctx),
	})
	if err != nil {
		s.log.Error("PublishStrategy", zap.Error(err))
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&antv1.PublishStrategyResponse{PublishId: id}), nil
}

func (s *MarketplaceServer) Subscribe(ctx context.Context, req *connect.Request[antv1.SubscribeRequest]) (*connect.Response[antv1.SubscribeResponse], error) {
	m := req.Msg
	userID := interceptor.GetUserID(ctx)
	if userID == "" {
		return nil, connect.NewError(connect.CodeUnauthenticated, fmt.Errorf("authentication required"))
	}
	id, err := s.svc.Subscribe(ctx, userID, m.PublisherUserId, m.StrategyId, m.Kind)
	if err != nil {
		s.log.Error("Subscribe", zap.Error(err))
		msg := err.Error()
		if strings.Contains(msg, "cannot subscribe to your own") {
			return nil, connect.NewError(connect.CodeInvalidArgument, err)
		}
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&antv1.SubscribeResponse{SubscriptionId: id}), nil
}

func (s *MarketplaceServer) Unsubscribe(ctx context.Context, req *connect.Request[antv1.UnsubscribeRequest]) (*connect.Response[antv1.UnsubscribeResponse], error) {
	userID := interceptor.GetUserID(ctx)
	if userID == "" {
		return nil, connect.NewError(connect.CodeUnauthenticated, fmt.Errorf("authentication required"))
	}
	if err := s.svc.Unsubscribe(ctx, userID, req.Msg.SubscriptionId); err != nil {
		s.log.Error("Unsubscribe", zap.Error(err))
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&antv1.UnsubscribeResponse{}), nil
}

func (s *MarketplaceServer) PurchaseStrategy(ctx context.Context, req *connect.Request[antv1.PurchaseStrategyRequest]) (*connect.Response[antv1.PurchaseStrategyResponse], error) {
	m := req.Msg
	userID := interceptor.GetUserID(ctx)
	if userID == "" {
		return nil, connect.NewError(connect.CodeUnauthenticated, fmt.Errorf("authentication required"))
	}
	result, err := s.svc.PurchaseStrategy(ctx, userID, m.StrategyId, m.IdempotencyKey)
	if err != nil {
		s.log.Error("PurchaseStrategy", zap.Error(err))
		msg := err.Error()
		if strings.Contains(msg, "insufficient balance") {
			return nil, connect.NewError(connect.CodeFailedPrecondition, err)
		}
		if strings.Contains(msg, "already subscribed") {
			return nil, connect.NewError(connect.CodeAlreadyExists, err)
		}
		if strings.Contains(msg, "cannot purchase your own") {
			return nil, connect.NewError(connect.CodeInvalidArgument, err)
		}
		if strings.Contains(msg, "not purchasable") || strings.Contains(msg, "not published") {
			return nil, connect.NewError(connect.CodeInvalidArgument, err)
		}
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&antv1.PurchaseStrategyResponse{
		SubscriptionId: result.SubscriptionID,
		TransactionId:  result.TransactionID,
		AmountCharged:  result.AmountCharged,
		BalanceAfter:   result.BalanceAfter,
	}), nil
}

func (s *MarketplaceServer) ListPublished(ctx context.Context, req *connect.Request[antv1.ListPublishedRequest]) (*connect.Response[antv1.ListPublishedResponse], error) {
	m := req.Msg
	list, err := s.svc.ListPublished(ctx, m.UserId, int(m.Limit), int(m.Offset), m.AssetClass, m.Keyword, m.SortBy)
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
			item.TotalPnl = strconv.FormatFloat(*p.TotalPnL, 'f', -1, 64)
		}
		if p.CodeSnippet != "" {
			item.CodeSnippet = p.CodeSnippet
		}
		if p.BacktestSnapshot != nil {
			item.BacktestSnapshot = &antv1.BacktestSnapshot{
				TotalReturn:  p.BacktestSnapshot.TotalReturn,
				AnnualReturn: p.BacktestSnapshot.AnnualReturn,
				MaxDrawdown:  p.BacktestSnapshot.MaxDrawdown,
				SharpeRatio:  p.BacktestSnapshot.SharpeRatio,
				WinRate:      p.BacktestSnapshot.WinRate,
				TotalTrades:  p.BacktestSnapshot.TotalTrades,
				Symbol:       p.BacktestSnapshot.Symbol,
				Timeframe:    p.BacktestSnapshot.Timeframe,
			}
		}
		item.AvgRating = p.AvgRating
		item.RatingCount = p.RatingCount
		resp.Strategies = append(resp.Strategies, item)
	}
	return connect.NewResponse(resp), nil
}

func (s *MarketplaceServer) ListSubscriptions(ctx context.Context, req *connect.Request[antv1.ListSubscriptionsRequest]) (*connect.Response[antv1.ListSubscriptionsResponse], error) {
	userID := interceptor.GetUserID(ctx)
	if userID == "" {
		return nil, connect.NewError(connect.CodeUnauthenticated, fmt.Errorf("authentication required"))
	}
	list, err := s.svc.ListSubscriptions(ctx, userID)
	if err != nil {
		s.log.Error("ListSubscriptions", zap.Error(err))
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	resp := &antv1.ListSubscriptionsResponse{}
	for _, sub := range list {
		item := &antv1.SubscriptionItem{
			SubscriptionId: sub.SubscriptionID, TargetUserId: sub.TargetUserID,
			StrategyId: sub.StrategyID, Kind: sub.Kind,
			Active: sub.Active, CreatedAt: timestamppb.New(sub.CreatedAt),
		}
		if sub.ExpiresAt != nil {
			item.ExpiresAt = timestamppb.New(*sub.ExpiresAt)
		}
		resp.Subscriptions = append(resp.Subscriptions, item)
	}
	return connect.NewResponse(resp), nil
}

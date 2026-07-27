package marketplace

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"connectrpc.com/connect"
	"google.golang.org/protobuf/types/known/timestamppb"

	antv1 "alphaforge/gen/proto/ant/v1"
	"alphaforge/internal/interceptor"
	"alphaforge/internal/marketplace"
)

func (s *MarketplaceServer) PublishStrategy(ctx context.Context, req *connect.Request[antv1.PublishStrategyRequest]) (*connect.Response[antv1.PublishStrategyResponse], error) {
	m := req.Msg
	userID := interceptor.GetUserID(ctx)
	if userID == "" {
		return nil, connect.NewError(connect.CodeUnauthenticated, fmt.Errorf("authentication required"))
	}

	// Read tamper-proof snapshot from DB by backtest_run_id.
	// Falls back to latest successful run for the strategy if not specified.
	var snapshotProto []byte
	if m.BacktestRunId != "" {
		runID, err := uuid.Parse(m.BacktestRunId)
		if err != nil {
			return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("invalid backtest_run_id: %w", err))
		}
		uid, err := uuid.Parse(userID)
		if err != nil {
			return nil, connect.NewError(connect.CodeUnauthenticated, fmt.Errorf("invalid user id"))
		}
		snapshotProto, err = s.fetchSnapshotByRunID(ctx, uid, runID)
		if err != nil {
			return nil, connect.NewError(connect.CodeFailedPrecondition, fmt.Errorf("marketplace: backtest snapshot: %w", err))
		}
	} else {
		// Fallback: find latest successful backtest run for this strategy template.
		strategyID, err := uuid.Parse(m.StrategyId)
		if err != nil {
			return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("invalid strategy_id"))
		}
		uid, _ := uuid.Parse(userID)
		snapshotProto, err = s.fetchLatestSnapshotForStrategy(ctx, uid, strategyID)
		if err != nil {
			return nil, connect.NewError(connect.CodeFailedPrecondition, fmt.Errorf("marketplace: no backtest snapshot: %w", err))
		}
	}

	id, err := s.svc.Publish(ctx, marketplace.PublishParams{
		UserID:                userID,
		StrategyID:            m.StrategyId,
		Title:                 m.Title,
		Description:           m.Description,
		PriceModel:            m.PriceModel,
		PriceAmount:           m.PriceAmount,
		AssetClass:            m.AssetClass,
		Symbols:               m.Symbols,
		Timeframe:             m.Timeframe,
		RiskLevel:             m.RiskLevel,
		Tags:                  m.Tags,
		CodeSnippet:           m.CodeSnippet,
		BacktestSnapshotProto: snapshotProto,
		PlatformFeeRate:       s.svc.GetPlatformFeeRate(ctx),
		Disclaimer:            m.Disclaimer,
		TrialDays:             int(m.TrialDays),
		RefundWindowDays:      int(m.RefundWindowDays),
	})
	if err != nil {
		s.log.Error("PublishStrategy", zap.Error(err))
		msg := err.Error()
		if strings.Contains(msg, "quality gate") {
			return nil, connect.NewError(connect.CodeFailedPrecondition, err)
		}
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&antv1.PublishStrategyResponse{PublishId: id}), nil
}

// fetchSnapshotByRunID reads the server-generated backtest_snapshot from a specific run.
// Returns error if the run doesn't exist, doesn't belong to the user, or has no snapshot.
func (s *MarketplaceServer) fetchSnapshotByRunID(ctx context.Context, userID, runID uuid.UUID) ([]byte, error) {
	var snapshot []byte
	var status string
	err := s.pgPool.QueryRow(ctx,
		`SELECT backtest_snapshot, status FROM backtest_runs WHERE id = $1 AND user_id = $2`,
		runID, userID,
	).Scan(&snapshot, &status)
	if err != nil {
		return nil, fmt.Errorf("backtest run not found: %w", err)
	}
	if status != "SUCCEEDED" {
		return nil, fmt.Errorf("backtest run status is %s, must be SUCCEEDED", status)
	}
	if len(snapshot) == 0 {
		return nil, fmt.Errorf("backtest run has no snapshot")
	}
	return snapshot, nil
}

// fetchLatestSnapshotForStrategy finds the most recent successful backtest run
// for a given strategy template and returns its server-generated snapshot.
// It matches by strategy_code_hash to ensure the snapshot corresponds to the
// current version of the strategy code, preventing stale snapshots from
// code changes.
func (s *MarketplaceServer) fetchLatestSnapshotForStrategy(ctx context.Context, userID, templateID uuid.UUID) ([]byte, error) {
	// Get the current code_hash from the strategy template.
	var codeHash *string
	if err := s.pgPool.QueryRow(ctx,
		`SELECT code_hash FROM strategy_templates WHERE id = $1`,
		templateID,
	).Scan(&codeHash); err != nil {
		return nil, fmt.Errorf("strategy template not found: %w", err)
	}

	var snapshot []byte
	if codeHash != nil && *codeHash != "" {
		// Match by code_hash for precise version alignment.
		err := s.pgPool.QueryRow(ctx,
			`SELECT backtest_snapshot FROM backtest_runs
			 WHERE user_id = $1 AND template_id = $2 AND status = 'SUCCEEDED'
			   AND strategy_code_hash = $3
			   AND backtest_snapshot IS NOT NULL
			 ORDER BY created_at DESC LIMIT 1`,
			userID, templateID, *codeHash,
		).Scan(&snapshot)
		if err == nil && len(snapshot) > 0 {
			return snapshot, nil
		}
	}

	// Fallback: any successful run for this template (less precise but usable).
	err := s.pgPool.QueryRow(ctx,
		`SELECT backtest_snapshot FROM backtest_runs
		 WHERE user_id = $1 AND template_id = $2 AND status = 'SUCCEEDED'
		   AND backtest_snapshot IS NOT NULL
		 ORDER BY created_at DESC LIMIT 1`,
		userID, templateID,
	).Scan(&snapshot)
	if err != nil {
		return nil, fmt.Errorf("no successful backtest with snapshot found for this strategy")
	}
	return snapshot, nil
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
	result, err := s.svc.PurchaseStrategy(ctx, userID, m.StrategyId, m.CouponCode, m.IdempotencyKey)
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
			item.WinRate = p.WinRate.InexactFloat64()
		}
		if p.TotalPnL != nil {
			item.TotalPnl = p.TotalPnL.StringFixed(2)
		}
		if p.CodeSnippet != "" {
			item.CodeSnippet = p.CodeSnippet
		}
		if p.BacktestSnapshotProto != nil {
			item.BacktestSnapshot = p.BacktestSnapshotProto
		}
		item.AvgRating = p.AvgRating
		item.RatingCount = p.RatingCount
		item.ProviderVerified = p.ProviderVerified
		item.ProviderType = p.ProviderType
		item.Disclaimer = p.Disclaimer
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

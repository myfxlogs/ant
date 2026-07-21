package notification

import (
	"context"
	"fmt"

	"connectrpc.com/connect"
	"github.com/google/uuid"
	"go.uber.org/zap"

	antv1 "alphaforge/gen/proto/ant/v1"
)

// GetNotificationPrefs returns the user's notification preferences (Phase 3.4).
func (s *NotificationServer) GetNotificationPrefs(
	ctx context.Context,
	_ *connect.Request[antv1.GetNotificationPrefsRequest],
) (*connect.Response[antv1.GetNotificationPrefsResponse], error) {
	uid := s.userID(ctx)
	if uid == uuid.Nil {
		return nil, connect.NewError(connect.CodeUnauthenticated, fmt.Errorf("authentication required"))
	}

	resp := &antv1.GetNotificationPrefsResponse{
		NewStrategyEnabled:    true,
		PriceChangeEnabled:    true,
		SubExpiringEnabled:    true,
		PerformanceAlertEnabled: true,
		NewRatingEnabled:      true,
	}

	if s.pg != nil {
		_ = s.pg.QueryRow(ctx,
			`SELECT new_strategy_enabled, price_change_enabled, sub_expiring_enabled,
			        performance_alert_enabled, new_rating_enabled
			 FROM marketplace_notification_prefs WHERE user_id = $1`,
			uid,
		).Scan(&resp.NewStrategyEnabled, &resp.PriceChangeEnabled,
			&resp.SubExpiringEnabled, &resp.PerformanceAlertEnabled,
			&resp.NewRatingEnabled)
	}

	return connect.NewResponse(resp), nil
}

// SetNotificationPrefs updates the user's notification preferences (Phase 3.4).
func (s *NotificationServer) SetNotificationPrefs(
	ctx context.Context,
	req *connect.Request[antv1.SetNotificationPrefsRequest],
) (*connect.Response[antv1.SetNotificationPrefsResponse], error) {
	uid := s.userID(ctx)
	if uid == uuid.Nil {
		return nil, connect.NewError(connect.CodeUnauthenticated, fmt.Errorf("authentication required"))
	}

	if s.pg == nil {
		return nil, connect.NewError(connect.CodeUnavailable, fmt.Errorf("database not configured"))
	}

	_, err := s.pg.Exec(ctx,
		`INSERT INTO marketplace_notification_prefs
		   (user_id, new_strategy_enabled, price_change_enabled, sub_expiring_enabled,
		    performance_alert_enabled, new_rating_enabled, updated_at)
		 VALUES ($1, $2, $3, $4, $5, $6, now())
		 ON CONFLICT (user_id) DO UPDATE SET
		   new_strategy_enabled = EXCLUDED.new_strategy_enabled,
		   price_change_enabled = EXCLUDED.price_change_enabled,
		   sub_expiring_enabled = EXCLUDED.sub_expiring_enabled,
		   performance_alert_enabled = EXCLUDED.performance_alert_enabled,
		   new_rating_enabled = EXCLUDED.new_rating_enabled,
		   updated_at = now()`,
		uid, req.Msg.NewStrategyEnabled, req.Msg.PriceChangeEnabled,
		req.Msg.SubExpiringEnabled, req.Msg.PerformanceAlertEnabled,
		req.Msg.NewRatingEnabled,
	)
	if err != nil {
		s.log.Error("set notification prefs failed", zap.Error(err))
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("update preferences: %w", err))
	}

	return connect.NewResponse(&antv1.SetNotificationPrefsResponse{}), nil
}

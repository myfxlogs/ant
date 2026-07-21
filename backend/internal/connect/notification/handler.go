package notification

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"
	"connectrpc.com/connect"

	antv1 "alphaforge/gen/proto/ant/v1"
	antv1c "alphaforge/gen/proto/ant/v1/antv1connect"
	"alphaforge/internal/interceptor"
	"alphaforge/internal/notification"
	"alphaforge/internal/repository"
)

// NotificationServer implements the NotificationService ConnectRPC handler.
type NotificationServer struct {
	repo *repository.NotificationRepository
	sub  *notification.Subscriber
	pg   *pgxpool.Pool
	log  *zap.Logger
}

var _ antv1c.NotificationServiceHandler = (*NotificationServer)(nil)

// NewNotificationServer creates a NotificationService handler.
func NewNotificationServer(repo *repository.NotificationRepository, sub *notification.Subscriber, pg *pgxpool.Pool, log *zap.Logger) *NotificationServer {
	return &NotificationServer{repo: repo, sub: sub, pg: pg, log: log}
}

func (s *NotificationServer) userID(ctx context.Context) uuid.UUID {
	raw := interceptor.GetUserID(ctx)
	if raw == "" {
		return uuid.Nil
	}
	id, err := uuid.Parse(raw)
	if err != nil {
		s.log.Warn("notification: userID parse failed", zap.String("raw", raw), zap.Error(err))
		return uuid.Nil
	}
	return id
}

func (s *NotificationServer) ListNotifications(
	ctx context.Context,
	req *connect.Request[antv1.ListNotificationsRequest],
) (*connect.Response[antv1.ListNotificationsResponse], error) {
	uid := s.userID(ctx)
	if uid == uuid.Nil {
		return nil, connect.NewError(connect.CodeUnauthenticated,
			fmt.Errorf("authentication required"))
	}
	limit := req.Msg.Limit
	if limit <= 0 {
		limit = 50
	}
	rows, totalUnread, err := s.repo.ListByUser(ctx, uid, limit, req.Msg.Offset, req.Msg.UnreadOnly)
	if err != nil {
		s.log.Error("list notifications failed", zap.Error(err))
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	notifs := make([]*antv1.Notification, len(rows))
	for i, r := range rows {
		notifs[i] = &antv1.Notification{
			Id:        r.ID.String(),
			UserId:    r.UserID.String(),
			Type:      r.Type,
			Title:     r.Title,
			Message:   r.Message,
			Data:      r.Data,
			IsRead:    r.IsRead,
			CreatedAt: r.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		}
	}
	return connect.NewResponse(&antv1.ListNotificationsResponse{
		Notifications: notifs,
		TotalUnread:   totalUnread,
	}), nil
}

func (s *NotificationServer) MarkRead(
	ctx context.Context,
	req *connect.Request[antv1.MarkReadRequest],
) (*connect.Response[antv1.MarkReadResponse], error) {
	id, err := uuid.Parse(req.Msg.Id)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	if err := s.repo.MarkRead(ctx, id); err != nil {
		s.log.Error("mark read failed", zap.Error(err))
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&antv1.MarkReadResponse{}), nil
}

func (s *NotificationServer) MarkAllRead(
	ctx context.Context,
	_ *connect.Request[antv1.MarkAllReadRequest],
) (*connect.Response[antv1.MarkAllReadResponse], error) {
	uid := s.userID(ctx)
	if uid == uuid.Nil {
		return nil, connect.NewError(connect.CodeUnauthenticated,
			fmt.Errorf("authentication required"))
	}
	if err := s.repo.MarkAllRead(ctx, uid); err != nil {
		s.log.Error("mark all read failed", zap.Error(err))
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&antv1.MarkAllReadResponse{}), nil
}

// StreamNotifications implements server-stream (SSE) for real-time notification delivery.
// The stream sends a keepalive comment every 15s and pushes new notifications as they
// are inserted by SendNotification.
func (s *NotificationServer) StreamNotifications(
	ctx context.Context,
	req *connect.Request[antv1.StreamNotificationsRequest],
	stream *connect.ServerStream[antv1.Notification],
) error {
	uid := s.userID(ctx)
	if uid == uuid.Nil {
		return connect.NewError(connect.CodeUnauthenticated,
			fmt.Errorf("authentication required"))
	}

	ch := s.sub.Subscribe(uid)
	defer s.sub.Unsubscribe(uid, ch)

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case n, ok := <-ch:
			if !ok {
				return nil
			}
			if req.Msg.UnreadOnly && n.IsRead {
				continue
			}
			if err := stream.Send(&antv1.Notification{
				Id:        n.ID.String(),
				UserId:    n.UserID.String(),
				Type:      n.Type,
				Title:     n.Title,
				Message:   n.Message,
				Data:      n.Data,
				IsRead:    n.IsRead,
				CreatedAt: n.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
			}); err != nil {
				return err
			}
		}
	}
}

// SendNotification creates a notification for a user and broadcasts it via SSE.
// Called internally by other services (backtest, tuning, gate) when events complete.
func (s *NotificationServer) SendNotification(
	ctx context.Context,
	req *connect.Request[antv1.SendNotificationRequest],
) (*connect.Response[antv1.SendNotificationResponse], error) {
	uid, err := uuid.Parse(req.Msg.UserId)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	row, err := s.repo.Insert(ctx, uid,
		req.Msg.Type, req.Msg.Title, req.Msg.Message, req.Msg.Data)
	if err != nil {
		s.log.Error("send notification insert failed", zap.Error(err))
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	// Broadcast to active SSE subscribers for this user
	s.sub.Publish(uid, row)
	return connect.NewResponse(&antv1.SendNotificationResponse{
		Id: row.ID.String(),
	}), nil
}

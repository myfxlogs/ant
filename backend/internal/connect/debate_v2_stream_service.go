package connect

import (
	"context"
	"time"

	"connectrpc.com/connect"

	v1 "anttrader/gen/proto"
	"anttrader/internal/service"
)

type DebateV2StreamService struct {
	svc *service.DebateV2Service
}

func NewDebateV2StreamService(svc *service.DebateV2Service) *DebateV2StreamService {
	return &DebateV2StreamService{svc: svc}
}

func (s *DebateV2StreamService) SubscribeDebateV2Session(ctx context.Context, req *connect.Request[v1.SubscribeDebateV2SessionRequest], stream *connect.ServerStream[v1.DebateV2SessionEvent]) error {
	userID, sid, err := debateV2UserAndSession(ctx, req.Msg.SessionId)
	if err != nil {
		return err
	}
	updates, err := s.svc.Subscribe(ctx, userID, sid, req.Msg.Locale)
	if err != nil {
		return connect.NewError(connect.CodeInvalidArgument, err)
	}
	heartbeat := time.NewTicker(25 * time.Second)
	defer heartbeat.Stop()
	for {
		select {
		case <-ctx.Done():
			return nil
		case <-heartbeat.C:
			if err := stream.Send(&v1.DebateV2SessionEvent{Type: "heartbeat"}); err != nil {
				return err
			}
		case dto, ok := <-updates:
			if !ok {
				return nil
			}
			if dto == nil {
				continue
			}
			if err := stream.Send(&v1.DebateV2SessionEvent{Type: "session", Session: debateV2Session(dto)}); err != nil {
				return err
			}
		}
	}
}

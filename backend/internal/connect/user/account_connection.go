package user

import (
	"context"
	"fmt"
	"time"

	"connectrpc.com/connect"
	"go.uber.org/zap"
	"google.golang.org/protobuf/types/known/emptypb"

	antv1 "anttrader/gen/proto/ant/v1"
)

// ConnectAccount connects an account to the broker and publishes a NATS event
// so the mdgateway runner reloads and connects the account.
// When a SessionReadyWaiter is configured, blocks on a channel (not polling)
// until the runner calls Hub.Register — eliminating "session not found" races.
func (s *AccountServer) ConnectAccount(ctx context.Context, req *connect.Request[antv1.ConnectAccountRequest]) (*connect.Response[antv1.ConnectAccountResponse], error) {
	userID, err := parseUserID(ctx)
	if err != nil {
		return nil, err
	}
	if err := s.svc.ConnectAccount(ctx, userID, req.Msg.Id); err != nil {
		s.log.Error("ConnectAccount", zap.Error(err))
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	// Publish NATS event to trigger mdgateway runner to connect this account.
	if s.publisher != nil {
		s.publisher.PublishConnect(ctx, req.Msg.Id, userID.String())
	}

	// Event-driven wait: block on a channel that Register() closes — zero CPU.
	msg := "connected"
	if s.sessionWaiter != nil {
		select {
		case <-s.sessionWaiter.WaitSession(req.Msg.Id):
			s.log.Info("ConnectAccount: session ready", zap.String("accountId", req.Msg.Id))
		case <-ctx.Done():
			s.log.Warn("ConnectAccount: context cancelled while waiting for session",
				zap.String("accountId", req.Msg.Id), zap.Error(ctx.Err()))
			msg = "connection initiated — session may take a moment to establish"
		case <-time.After(5 * time.Second):
			s.log.Warn("ConnectAccount: timed out waiting for session",
				zap.String("accountId", req.Msg.Id))
			msg = "connection initiated — session may take a moment to establish"
		}
	}
	return connect.NewResponse(&antv1.ConnectAccountResponse{Success: true, Message: msg}), nil
}

// DisconnectAccount marks the account as disconnected and publishes a NATS event.
func (s *AccountServer) DisconnectAccount(ctx context.Context, req *connect.Request[antv1.DisconnectAccountRequest]) (*connect.Response[emptypb.Empty], error) {
	userID, err := parseUserID(ctx)
	if err != nil {
		return nil, err
	}
	if err := s.svc.DisconnectAccount(ctx, userID, req.Msg.Id); err != nil {
		s.log.Error("DisconnectAccount", zap.Error(err))
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	if s.publisher != nil {
		s.publisher.PublishDisconnect(ctx, req.Msg.Id, userID.String())
	}
	return connect.NewResponse(&emptypb.Empty{}), nil
}

// ReconnectAccount marks the account for re-connection and publishes a NATS event.
func (s *AccountServer) ReconnectAccount(ctx context.Context, req *connect.Request[antv1.ReconnectAccountRequest]) (*connect.Response[emptypb.Empty], error) {
	userID, err := parseUserID(ctx)
	if err != nil {
		return nil, err
	}
	if err := s.svc.ReconnectAccount(ctx, userID, req.Msg.Id); err != nil {
		s.log.Error("ReconnectAccount", zap.Error(err))
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	if s.publisher != nil {
		s.publisher.PublishReconnect(ctx, req.Msg.Id, userID.String())
	}
	return connect.NewResponse(&emptypb.Empty{}), nil
}

// SearchBroker calls mtapi Search RPC for real broker discovery.
func (s *AccountServer) SearchBroker(ctx context.Context, req *connect.Request[antv1.SearchBrokerRequest]) (*connect.Response[antv1.SearchBrokerResponse], error) {
	if s.searcher == nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("broker search is not available"))
	}
	companies, err := s.searcher.Search(ctx, req.Msg.Company, req.Msg.MtType)
	if err != nil {
		s.log.Error("broker search failed", zap.Error(err))
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("broker search failed: %w", err))
	}
	return connect.NewResponse(&antv1.SearchBrokerResponse{Companies: companies}), nil
}

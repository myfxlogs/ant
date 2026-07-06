package user

import (
	"context"
	"fmt"
	"time"

	"connectrpc.com/connect"
	"go.uber.org/zap"
	"google.golang.org/protobuf/types/known/emptypb"

	"github.com/google/uuid"

	antv1 "anttrader/gen/proto/ant/v1"
	"anttrader/internal/service"
)

// ConnectAccount connects an account to the broker and publishes a NATS event
// so the mdgateway runner reloads and connects the account.
func (s *AccountServer) ConnectAccount(ctx context.Context, req *connect.Request[antv1.ConnectAccountRequest]) (*connect.Response[antv1.ConnectAccountResponse], error) {
	userID, err := parseUserID(ctx)
	if err != nil {
		return nil, err
	}
	if err := s.updateAccountStatus(ctx, userID, req.Msg.Id, service.StatusConnecting, func() {
		s.publisher.PublishConnect(ctx, req.Msg.Id, userID.String())
	}); err != nil {
		return nil, err
	}

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
	if err := s.updateAccountStatus(ctx, userID, req.Msg.Id, service.StatusDisconnected, func() {
		s.publisher.PublishDisconnect(ctx, req.Msg.Id, userID.String())
	}); err != nil {
		return nil, err
	}
	return connect.NewResponse(&emptypb.Empty{}), nil
}

// ReconnectAccount marks the account for re-connection and publishes a NATS event.
func (s *AccountServer) ReconnectAccount(ctx context.Context, req *connect.Request[antv1.ReconnectAccountRequest]) (*connect.Response[emptypb.Empty], error) {
	userID, err := parseUserID(ctx)
	if err != nil {
		return nil, err
	}
	if err := s.updateAccountStatus(ctx, userID, req.Msg.Id, service.StatusConnecting, func() {
		s.publisher.PublishReconnect(ctx, req.Msg.Id, userID.String())
	}); err != nil {
		return nil, err
	}
	return connect.NewResponse(&emptypb.Empty{}), nil
}

// updateAccountStatus sets the account status via SetStatus and invokes publishFn.
func (s *AccountServer) updateAccountStatus(ctx context.Context, userID uuid.UUID, id string, status service.AccountStatus, publishFn func()) error {
	if err := s.svc.SetStatus(ctx, userID, id, status); err != nil {
		s.log.Error("updateAccountStatus", zap.String("accountId", id), zap.String("status", string(status)), zap.Error(err))
		return connect.NewError(connect.CodeInternal, err)
	}
	if s.publisher != nil && publishFn != nil {
		publishFn()
	}
	return nil
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

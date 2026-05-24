package connect

import (
	"context"

	"connectrpc.com/connect"
	"google.golang.org/protobuf/types/known/emptypb"

	antv1 "anttrader/gen/proto/ant/v1"
	antv1c "anttrader/gen/proto/ant/v1/antv1connect"
	"anttrader/internal/service"
)

// AccountServer implements ant.v1.AccountServiceHandler.
type AccountServer struct {
	svc *service.PlatformService
}

var _ antv1c.AccountServiceHandler = (*AccountServer)(nil)

func NewAccountServer(svc *service.PlatformService) *AccountServer {
	return &AccountServer{svc: svc}
}

func (s *AccountServer) ListAccounts(ctx context.Context, req *connect.Request[antv1.ListAccountsRequest]) (*connect.Response[antv1.ListAccountsResponse], error) {
	accounts, err := s.svc.ListAccounts(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	out := &antv1.ListAccountsResponse{}
	for _, a := range accounts {
		out.Accounts = append(out.Accounts, &antv1.Account{
			Id: a.ID, UserId: a.UserID, Login: a.Login,
			MtType: a.Platform, BrokerCompany: a.Broker, BrokerServer: a.Server,
			IsDisabled: a.IsDisabled,
		})
	}
	return connect.NewResponse(out), nil
}

func (s *AccountServer) GetAccount(ctx context.Context, req *connect.Request[antv1.GetAccountRequest]) (*connect.Response[antv1.Account], error) {
	a, err := s.svc.GetAccount(ctx, req.Msg.Id)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&antv1.Account{
		Id: a.ID, UserId: a.UserID, Login: a.Login,
		MtType: a.Platform, BrokerCompany: a.Broker, BrokerServer: a.Server,
		IsDisabled: a.IsDisabled,
	}), nil
}

// Stub methods below — return unimplemented until backend logic is built.
func (s *AccountServer) CreateAccount(ctx context.Context, req *connect.Request[antv1.CreateAccountRequest]) (*connect.Response[antv1.Account], error) {
	return nil, connect.NewError(connect.CodeUnimplemented, nil)
}
func (s *AccountServer) UpdateAccount(ctx context.Context, req *connect.Request[antv1.UpdateAccountRequest]) (*connect.Response[antv1.Account], error) {
	return nil, connect.NewError(connect.CodeUnimplemented, nil)
}
func (s *AccountServer) DeleteAccount(ctx context.Context, req *connect.Request[antv1.DeleteAccountRequest]) (*connect.Response[emptypb.Empty], error) {
	return nil, connect.NewError(connect.CodeUnimplemented, nil)
}
func (s *AccountServer) ConnectAccount(ctx context.Context, req *connect.Request[antv1.ConnectAccountRequest]) (*connect.Response[antv1.ConnectAccountResponse], error) {
	return nil, connect.NewError(connect.CodeUnimplemented, nil)
}
func (s *AccountServer) DisconnectAccount(ctx context.Context, req *connect.Request[antv1.DisconnectAccountRequest]) (*connect.Response[emptypb.Empty], error) {
	return nil, connect.NewError(connect.CodeUnimplemented, nil)
}
func (s *AccountServer) ReconnectAccount(ctx context.Context, req *connect.Request[antv1.ReconnectAccountRequest]) (*connect.Response[emptypb.Empty], error) {
	return nil, connect.NewError(connect.CodeUnimplemented, nil)
}
func (s *AccountServer) SearchBroker(ctx context.Context, req *connect.Request[antv1.SearchBrokerRequest]) (*connect.Response[antv1.SearchBrokerResponse], error) {
	return nil, connect.NewError(connect.CodeUnimplemented, nil)
}
func (s *AccountServer) VerifyTradePermission(ctx context.Context, req *connect.Request[antv1.VerifyTradePermissionRequest]) (*connect.Response[antv1.VerifyTradePermissionResponse], error) {
	return nil, connect.NewError(connect.CodeUnimplemented, nil)
}
func (s *AccountServer) UpdateTradingPassword(ctx context.Context, req *connect.Request[antv1.UpdateTradingPasswordRequest]) (*connect.Response[antv1.UpdateTradingPasswordResponse], error) {
	return nil, connect.NewError(connect.CodeUnimplemented, nil)
}

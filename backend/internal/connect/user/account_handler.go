package user

import (
	"context"

	"errors"

	"go.uber.org/zap"

	"connectrpc.com/connect"
	"github.com/google/uuid"
	"google.golang.org/protobuf/types/known/emptypb"

	antv1 "anttrader/gen/proto/ant/v1"
	antv1c "anttrader/gen/proto/ant/v1/antv1connect"
	"anttrader/internal/interceptor"
	"anttrader/internal/mdgateway/adapter/brokersearch"
	"anttrader/internal/service"
)

// AccountServer implements ant.v1.AccountServiceHandler.
type AccountServer struct {
	svc      *service.PlatformService
	searcher *brokersearch.Searcher
	log      *zap.Logger
}

var _ antv1c.AccountServiceHandler = (*AccountServer)(nil)

func NewAccountServer(svc *service.PlatformService, log *zap.Logger) *AccountServer {
	return &AccountServer{svc: svc, log: log}
}

// SetSearcher injects the mtapi broker searcher (S2.2).
func (s *AccountServer) SetSearcher(searcher *brokersearch.Searcher) { s.searcher = searcher }

func (s *AccountServer) ListAccounts(ctx context.Context, req *connect.Request[antv1.ListAccountsRequest]) (*connect.Response[antv1.ListAccountsResponse], error) {
	userID, _ := uuid.Parse(interceptor.GetUserID(ctx))
	accounts, err := s.svc.ListAccounts(ctx, userID)
	if err != nil {
		s.log.Error("ListAccounts", zap.Error(err))
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	out := &antv1.ListAccountsResponse{}
	for _, a := range accounts {
		out.Accounts = append(out.Accounts, &antv1.Account{
			Id: a.ID, UserId: a.UserID, Login: a.Login,
			MtType: a.Platform, BrokerCompany: a.Broker, BrokerServer: a.Server,
			IsDisabled: a.IsDisabled, Status: a.Status,
			Balance: a.Balance, Equity: a.Equity, Margin: a.Margin,
			FreeMargin: a.FreeMargin, MarginLevel: a.MarginLevel,
			Leverage: a.Leverage, Currency: a.Currency,
			LastError: a.LastError,
		})
	}
	return connect.NewResponse(out), nil
}

func (s *AccountServer) GetAccount(ctx context.Context, req *connect.Request[antv1.GetAccountRequest]) (*connect.Response[antv1.Account], error) {
	userID, _ := uuid.Parse(interceptor.GetUserID(ctx))
	a, err := s.svc.GetAccount(ctx, userID, req.Msg.Id)
	if err != nil {
		s.log.Error("GetAccount", zap.Error(err))
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&antv1.Account{
		Id: a.ID, UserId: a.UserID, Login: a.Login,
		MtType: a.Platform, BrokerCompany: a.Broker, BrokerServer: a.Server,
		IsDisabled: a.IsDisabled, Status: a.Status,
		Balance: a.Balance, Equity: a.Equity, Margin: a.Margin,
		FreeMargin: a.FreeMargin, MarginLevel: a.MarginLevel,
		Leverage: a.Leverage, Currency: a.Currency,
		LastError: a.LastError,
	}), nil
}

// CreateAccount inserts a new MT account and returns it.
// Full connection test and AccountSummary fill require mthub (Phase 2).
func (s *AccountServer) CreateAccount(ctx context.Context, req *connect.Request[antv1.CreateAccountRequest]) (*connect.Response[antv1.Account], error) {
	r := req.Msg
	userID, _ := uuid.Parse(interceptor.GetUserID(ctx))
	id, err := s.svc.CreateAccount(ctx, userID, r.Login, r.Password, r.MtType, r.BrokerCompany, r.BrokerServer, r.BrokerHost)
	if err != nil {
		if errors.Is(err, service.ErrAccountAlreadyBound) {
			return nil, connect.NewError(connect.CodeAlreadyExists, err)
		}
		s.log.Error("CreateAccount", zap.Error(err))
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&antv1.Account{
		Id:            id,
		Login:         r.Login,
		MtType:        r.MtType,
		BrokerCompany: r.BrokerCompany,
		BrokerServer:  r.BrokerServer,
	}), nil
}

// UpdateAccount updates broker fields and disabled status.
func (s *AccountServer) UpdateAccount(ctx context.Context, req *connect.Request[antv1.UpdateAccountRequest]) (*connect.Response[antv1.Account], error) {
	r := req.Msg
	brokerCompany := ""
	if r.BrokerCompany != nil {
		brokerCompany = *r.BrokerCompany
	}
	brokerServer := ""
	if r.BrokerServer != nil {
		brokerServer = *r.BrokerServer
	}
	brokerHost := ""
	if r.BrokerHost != nil {
		brokerHost = *r.BrokerHost
	}
	userID, _ := uuid.Parse(interceptor.GetUserID(ctx))
	if err := s.svc.UpdateAccount(ctx, userID, r.Id, brokerCompany, brokerServer, brokerHost, r.IsDisabled); err != nil {
		s.log.Error("UpdateAccount", zap.Error(err))
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	// Return updated account
	a, err := s.svc.GetAccount(ctx, userID, r.Id)
	if err != nil {
		s.log.Error("UpdateAccount", zap.Error(err))
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&antv1.Account{
		Id: a.ID, UserId: a.UserID, Login: a.Login,
		MtType: a.Platform, BrokerCompany: a.Broker, BrokerServer: a.Server,
		IsDisabled: a.IsDisabled, Status: a.Status,
		Balance: a.Balance, Equity: a.Equity, Margin: a.Margin,
		FreeMargin: a.FreeMargin, MarginLevel: a.MarginLevel,
		Leverage: a.Leverage, Currency: a.Currency,
		LastError: a.LastError,
	}), nil
}

// DeleteAccount removes an MT account.
func (s *AccountServer) DeleteAccount(ctx context.Context, req *connect.Request[antv1.DeleteAccountRequest]) (*connect.Response[emptypb.Empty], error) {
	userID, _ := uuid.Parse(interceptor.GetUserID(ctx))
	if err := s.svc.DeleteAccount(ctx, userID, req.Msg.Id); err != nil {
		s.log.Error("DeleteAccount", zap.Error(err))
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&emptypb.Empty{}), nil
}

// ConnectAccount connects an account to the broker via mthub session management.
func (s *AccountServer) ConnectAccount(ctx context.Context, req *connect.Request[antv1.ConnectAccountRequest]) (*connect.Response[antv1.ConnectAccountResponse], error) {
	userID, _ := uuid.Parse(interceptor.GetUserID(ctx))
	if err := s.svc.ConnectAccount(ctx, userID, req.Msg.Id); err != nil {
		s.log.Error("ConnectAccount", zap.Error(err))
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&antv1.ConnectAccountResponse{Success: true, Message: "connected"}), nil
}

// DisconnectAccount marks the account as disconnected.
func (s *AccountServer) DisconnectAccount(ctx context.Context, req *connect.Request[antv1.DisconnectAccountRequest]) (*connect.Response[emptypb.Empty], error) {
	userID, _ := uuid.Parse(interceptor.GetUserID(ctx))
	if err := s.svc.DisconnectAccount(ctx, userID, req.Msg.Id); err != nil {
		s.log.Error("DisconnectAccount", zap.Error(err))
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&emptypb.Empty{}), nil
}

// ReconnectAccount marks the account for re-connection.
func (s *AccountServer) ReconnectAccount(ctx context.Context, req *connect.Request[antv1.ReconnectAccountRequest]) (*connect.Response[emptypb.Empty], error) {
	userID, _ := uuid.Parse(interceptor.GetUserID(ctx))
	if err := s.svc.ReconnectAccount(ctx, userID, req.Msg.Id); err != nil {
		s.log.Error("ReconnectAccount", zap.Error(err))
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&emptypb.Empty{}), nil
}

// SearchBroker calls mtapi Search RPC for real broker discovery.
// Falls back to a static Exness entry when the searcher is unavailable or the call fails.
func (s *AccountServer) SearchBroker(ctx context.Context, req *connect.Request[antv1.SearchBrokerRequest]) (*connect.Response[antv1.SearchBrokerResponse], error) {
	if s.searcher != nil {
		companies, err := s.searcher.Search(ctx, req.Msg.Company, req.Msg.MtType)
		if err != nil {
			s.log.Warn("broker search failed, falling back to mock", zap.Error(err))
		} else if len(companies) > 0 {
			return connect.NewResponse(&antv1.SearchBrokerResponse{Companies: companies}), nil
		}
	}
	return connect.NewResponse(&antv1.SearchBrokerResponse{
		Companies: []*antv1.BrokerCompany{
			{CompanyName: "Exness", Servers: []*antv1.BrokerServer{
				{Name: "Exness-Real", Access: []string{"mt4", "mt5"}},
			}},
		},
	}), nil
}

// VerifyTradePermission checks whether the account can trade based on PG state.
func (s *AccountServer) VerifyTradePermission(ctx context.Context, req *connect.Request[antv1.VerifyTradePermissionRequest]) (*connect.Response[antv1.VerifyTradePermissionResponse], error) {
	userID, _ := uuid.Parse(interceptor.GetUserID(ctx))
	acct, err := s.svc.GetAccount(ctx, userID, req.Msg.Id)
	if err != nil {
		s.log.Error("VerifyTradePermission", zap.Error(err))
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	// Check real account state: disabled or frozen accounts cannot trade.
	if acct.IsDisabled {
		return connect.NewResponse(&antv1.VerifyTradePermissionResponse{
			HasTradePermission: false,
			Verified:           true,
			Message:            "account is disabled",
		}), nil
	}
	if acct.Status == "frozen" || acct.Status == "needs_rebind" {
		return connect.NewResponse(&antv1.VerifyTradePermissionResponse{
			HasTradePermission: false,
			Verified:           true,
			Message:            "account status is " + acct.Status,
		}), nil
	}
	return connect.NewResponse(&antv1.VerifyTradePermissionResponse{
		HasTradePermission: true,
		Verified:           true,
		Message:            "account has trade permission",
	}), nil
}

// UpdateTradingPassword updates the trading password for an account.
func (s *AccountServer) UpdateTradingPassword(ctx context.Context, req *connect.Request[antv1.UpdateTradingPasswordRequest]) (*connect.Response[antv1.UpdateTradingPasswordResponse], error) {
	userID, _ := uuid.Parse(interceptor.GetUserID(ctx))
	if err := s.svc.UpdateTradingPassword(ctx, userID, req.Msg.Id, req.Msg.NewPassword); err != nil {
		s.log.Error("UpdateTradingPassword", zap.Error(err))
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&antv1.UpdateTradingPasswordResponse{Success: true}), nil
}

package user

import (
	"context"

	"errors"
	"fmt"

	"go.uber.org/zap"

	"connectrpc.com/connect"
	"github.com/jackc/pgx/v5"
	"github.com/google/uuid"
	"google.golang.org/protobuf/types/known/emptypb"

	antv1 "anttrader/gen/proto/ant/v1"
	antv1c "anttrader/gen/proto/ant/v1/antv1connect"
	"anttrader/internal/interceptor"
	"anttrader/internal/mdgateway"
	"anttrader/internal/mdgateway/adapter/brokersearch"
	"anttrader/internal/mdgateway/adapter/mdtick"
	"anttrader/internal/service"
)

// MTConnectionTester validates MT account credentials by connecting to the broker.
type MTConnectionTester interface {
	Test(ctx context.Context, platform, brokerHost, login, password string) (*mdtick.MTAccountInfo, error)
	// VerifyPassword only connects to the broker to verify credentials, without fetching account info.
	VerifyPassword(ctx context.Context, platform, brokerHost, login, password string) error
}

// AccountServer implements ant.v1.AccountServiceHandler.
type AccountServer struct {
	svc       *service.AccountService
	searcher  *brokersearch.Searcher
	publisher *mdgateway.AccountEventPublisher
	mtTester  MTConnectionTester
	log       *zap.Logger
}

var _ antv1c.AccountServiceHandler = (*AccountServer)(nil)

func NewAccountServer(svc *service.AccountService, searcher *brokersearch.Searcher, publisher *mdgateway.AccountEventPublisher, tester MTConnectionTester, log *zap.Logger) *AccountServer {
	return &AccountServer{svc: svc, searcher: searcher, publisher: publisher, mtTester: tester, log: log}
}

// parseUserID extracts and validates the user ID from the request context (#1).
// Returns a connect.Error(CodeUnauthenticated) on failure, matching AI's userIDFromCtx pattern.
func parseUserID(ctx context.Context) (uuid.UUID, error) {
	id, err := uuid.Parse(interceptor.GetUserID(ctx))
	if err != nil {
		return uuid.Nil, connect.NewError(connect.CodeUnauthenticated, fmt.Errorf("invalid user id"))
	}
	return id, nil
}

// accountToProto converts a service.AccountDTO to a protobuf Account message.
func accountToProto(a *service.AccountDTO) *antv1.Account {
	return &antv1.Account{
		Id: a.ID, UserId: a.UserID, Login: a.Login,
		MtType: a.Platform, BrokerCompany: a.Broker, BrokerServer: a.Server,
		IsDisabled: a.IsDisabled, Status: a.Status,
		Balance: a.Balance, Credit: a.Credit, Equity: a.Equity, Margin: a.Margin,
		FreeMargin: a.FreeMargin, MarginLevel: a.MarginLevel,
		Leverage: a.Leverage, Currency: a.Currency,
		LastError: a.LastError,
	}
}

func (s *AccountServer) ListAccounts(ctx context.Context, req *connect.Request[antv1.ListAccountsRequest]) (*connect.Response[antv1.ListAccountsResponse], error) {
	userID, err := parseUserID(ctx)
	if err != nil {
		return nil, err
	}
	accounts, err := s.svc.ListAccounts(ctx, userID)
	if err != nil {
		s.log.Error("ListAccounts", zap.Error(err))
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	out := &antv1.ListAccountsResponse{}
	for _, a := range accounts {
		out.Accounts = append(out.Accounts, accountToProto(&a))
	}
	return connect.NewResponse(out), nil
}

func (s *AccountServer) GetAccount(ctx context.Context, req *connect.Request[antv1.GetAccountRequest]) (*connect.Response[antv1.Account], error) {
	userID, err := parseUserID(ctx)
	if err != nil {
		return nil, err
	}
	a, err := s.svc.GetAccount(ctx, userID, req.Msg.Id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("account not found"))
		}
		s.log.Error("GetAccount", zap.Error(err))
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(accountToProto(a)), nil
}

// CreateAccount inserts a new MT account, verifies credentials by connecting to MT,
// and returns the account with balance/equity/currency filled from AccountSummary (#2: transactional).
func (s *AccountServer) CreateAccount(ctx context.Context, req *connect.Request[antv1.CreateAccountRequest]) (*connect.Response[antv1.Account], error) {
	r := req.Msg
	userID, err := parseUserID(ctx)
	if err != nil {
		return nil, err
	}

	// Begin transaction for the create + verify + update flow (#2).
	tx, err := s.svc.BeginTx(ctx)
	if err != nil {
		s.log.Error("CreateAccount: begin tx failed", zap.Error(err))
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	defer tx.Rollback(ctx) // no-op after successful Commit

	// 1. Insert into DB within the transaction.
	id, err := s.svc.CreateAccountTx(ctx, tx, userID, r.Login, r.Password, r.MtType, r.BrokerCompany, r.BrokerServer, r.BrokerHost)
	if err != nil {
		if errors.Is(err, service.ErrAccountAlreadyBound) {
			return nil, connect.NewError(connect.CodeAlreadyExists, err)
		}
		s.log.Error("CreateAccount: db insert failed", zap.Error(err))
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	// 2. Test MT connection to validate credentials
	if s.mtTester != nil {
		info, err := s.mtTester.Test(ctx, r.MtType, r.BrokerHost, r.Login, r.Password)
		if err != nil {
			// Commit the row so the account exists before we mark it (#H1).
			if commitErr := tx.Commit(ctx); commitErr != nil {
				s.log.Error("CreateAccount: tx commit failed", zap.String("id", id), zap.Error(commitErr))
				return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("account verification failed: %w", err))
			}
			// Mark the account as needs_rebind since credentials failed.
			if markErr := s.svc.MarkAccountNeedsRebind(ctx, userID, id); markErr != nil {
				s.log.Error("CreateAccount: mark needs_rebind failed", zap.String("id", id), zap.Error(markErr))
			}
			s.log.Warn("CreateAccount: MT connection failed", zap.String("accountId", id), zap.Error(err))
			return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("account verification failed: %w", err))
		}

		// Update account with MT info within the transaction (#3: propagate error).
		if err := s.svc.UpdateAccountInfoTx(ctx, tx, userID, id, info.Balance, info.Equity, info.Credit, info.Margin, info.FreeMargin, int64(info.Leverage), info.Currency); err != nil {
			s.log.Error("CreateAccount: UpdateAccountInfo failed", zap.Error(err))
			return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("update account info: %w", err))
		}
		s.log.Info("CreateAccount: verified and created",
			zap.String("id", id),
			zap.Float64("balance", info.Balance),
		)
	} else {
		s.log.Warn("CreateAccount: MT connection tester not available, skipping verification",
			zap.String("id", id))
	}

	// Commit the transaction.
	if err := tx.Commit(ctx); err != nil {
		s.log.Error("CreateAccount: tx commit failed", zap.Error(err))
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("commit transaction: %w", err))
	}

	// Fetch the full account back so the response includes balance/equity/margin/currency.
	a, err := s.svc.GetAccount(ctx, userID, id)
	if err != nil {
		s.log.Error("CreateAccount: get account after create", zap.Error(err))
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(accountToProto(a)), nil
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
	userID, err := parseUserID(ctx)
	if err != nil {
		return nil, err
	}
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
	return connect.NewResponse(accountToProto(a)), nil
}

// DeleteAccount removes an MT account. When a password is provided, it first
// verifies the credentials against the broker before performing the hard delete.
func (s *AccountServer) DeleteAccount(ctx context.Context, req *connect.Request[antv1.DeleteAccountRequest]) (*connect.Response[emptypb.Empty], error) {
	userID, err := parseUserID(ctx)
	if err != nil {
		return nil, err
	}

	// If the user provides a password, verify it against the MT broker first.
	if req.Msg.Password != "" {
		creds, err := s.svc.GetAccountCredentials(ctx, userID, req.Msg.Id)
		if err != nil {
			s.log.Error("DeleteAccount: get credentials", zap.Error(err))
			return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to get account info: %w", err))
		}
		// #7: Skip password verification when BrokerHost is empty.
		if creds.BrokerHost == "" {
			s.log.Warn("DeleteAccount: skipping password verification (empty broker host)", zap.String("accountId", req.Msg.Id))
		} else {
			if s.mtTester == nil {
				return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("MT connection tester not available"))
			}
			if err := s.mtTester.VerifyPassword(ctx, creds.Platform, creds.BrokerHost, creds.Login, req.Msg.Password); err != nil {
				s.log.Warn("DeleteAccount: password verification failed", zap.String("accountId", req.Msg.Id), zap.Error(err))
				return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("password verification failed: %w", err))
			}
		}
	}

	if err := s.svc.DeleteAccount(ctx, userID, req.Msg.Id); err != nil {
		s.log.Error("DeleteAccount", zap.Error(err))
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&emptypb.Empty{}), nil
}

// ConnectAccount connects an account to the broker and publishes a NATS event
// so the mdgateway runner reloads and connects the account.
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
	return connect.NewResponse(&antv1.ConnectAccountResponse{Success: true, Message: "connected"}), nil
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

// SearchBroker calls mtapi Search RPC for real broker discovery (#6: return error on failure).
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

// VerifyTradePermission checks whether the account can trade based on PG state.
func (s *AccountServer) VerifyTradePermission(ctx context.Context, req *connect.Request[antv1.VerifyTradePermissionRequest]) (*connect.Response[antv1.VerifyTradePermissionResponse], error) {
	userID, err := parseUserID(ctx)
	if err != nil {
		return nil, err
	}
	acct, err := s.svc.GetAccount(ctx, userID, req.Msg.Id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("account not found"))
		}
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

// UpdateTradingPassword updates the trading password for an account (#38: old password verification).
func (s *AccountServer) UpdateTradingPassword(ctx context.Context, req *connect.Request[antv1.UpdateTradingPasswordRequest]) (*connect.Response[antv1.UpdateTradingPasswordResponse], error) {
	userID, err := parseUserID(ctx)
	if err != nil {
		return nil, err
	}
	if err := s.svc.UpdateTradingPassword(ctx, userID, req.Msg.Id, req.Msg.OldPassword, req.Msg.NewPassword); err != nil {
		if errors.Is(err, service.ErrAccountPasswordMismatch) {
			return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("old password does not match"))
		}
		s.log.Error("UpdateTradingPassword", zap.Error(err))
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&antv1.UpdateTradingPasswordResponse{Success: true}), nil
}

// VerifyAccount connects to the MT broker with the given credentials and returns
// account summary info (balance, equity, etc.) WITHOUT saving anything to the database.
func (s *AccountServer) VerifyAccount(ctx context.Context, req *connect.Request[antv1.VerifyAccountRequest]) (*connect.Response[antv1.VerifyAccountResponse], error) {
	r := req.Msg
	if s.mtTester == nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("MT connection tester not available"))
	}
	info, err := s.mtTester.Test(ctx, r.MtType, r.BrokerHost, r.Login, r.Password)
	if err != nil {
		// #9: Mask login value in warning logs.
		s.log.Warn("VerifyAccount: connection failed", zap.String("accountLogin", maskLogin(r.Login)), zap.Error(err))
		return connect.NewResponse(&antv1.VerifyAccountResponse{
			Verified: false,
			Message:  err.Error(),
		}), nil
	}
	return connect.NewResponse(&antv1.VerifyAccountResponse{
		Verified:   true,
		Message:    "account verified",
		Balance:    info.Balance,
		Equity:     info.Equity,
		Margin:     info.Margin,
		FreeMargin: info.FreeMargin,
		Leverage:   info.Leverage,
		Currency:   info.Currency,
	}), nil
}

// maskLogin masks a login value for safe logging (#9).
func maskLogin(login string) string {
	if len(login) <= 3 {
		return "***"
	}
	return login[:3] + "***"
}

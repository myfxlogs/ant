package user

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"connectrpc.com/connect"
	"go.uber.org/zap"
	"google.golang.org/protobuf/types/known/emptypb"

	antv1 "anttrader/gen/proto/ant/v1"
	"anttrader/internal/service"
)

// CreateAccount inserts a new MT account, verifies credentials by connecting to MT,
// and returns the account with balance/equity/currency filled from AccountSummary.
func (s *AccountServer) CreateAccount(ctx context.Context, req *connect.Request[antv1.CreateAccountRequest]) (*connect.Response[antv1.Account], error) {
	r := req.Msg
	userID, err := parseUserID(ctx)
	if err != nil {
		return nil, err
	}

	// Begin transaction for the create + verify + update flow.
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

	// 2. Test MT connection and update account info within transaction.
	if err := s.verifyAndUpdateAccount(ctx, tx, userID, id, r); err != nil {
		return nil, err
	}
	// Commit and fetch full account for response.
	if err := tx.Commit(ctx); err != nil {
		s.log.Error("CreateAccount: tx commit failed", zap.Error(err))
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("commit transaction: %w", err))
	}

	// Trigger gateway connection after successful account creation.
	// Set status to 'connecting' and publish NATS event so mdgateway runner
	// picks it up and starts a persistent broker connection.
	if err := s.svc.ConnectAccount(ctx, userID, id); err != nil {
		s.log.Warn("CreateAccount: ConnectAccount after create failed", zap.String("id", id), zap.Error(err))
	} else if s.publisher != nil {
		s.publisher.PublishConnect(ctx, id, userID.String())
	}

	a, err := s.svc.GetAccount(ctx, userID, id)
	if err != nil {
		s.log.Error("CreateAccount: get account after create", zap.Error(err))
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(accountToProto(a)), nil
}

func (s *AccountServer) verifyAndUpdateAccount(ctx context.Context, tx pgx.Tx, userID uuid.UUID, accountID string, r *antv1.CreateAccountRequest) error {
	if s.mtTester == nil {
		s.log.Warn("CreateAccount: MT connection tester not available, skipping verification",
			zap.String("id", accountID))
		return nil
	}
	info, err := s.mtTester.Test(ctx, r.MtType, r.BrokerHost, r.Login, r.Password)
	if err != nil {
		s.log.Warn("CreateAccount: MT verification failed, rolling back",
			zap.String("accountId", accountID), zap.Error(err))
		return connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("account verification failed: %w", err))
	}
	if err := s.svc.UpdateAccountInfoTx(ctx, tx, userID, accountID, info.Balance, info.Equity, info.Credit, info.Margin, info.FreeMargin, int64(info.Leverage), info.Currency, info.IsInvestor); err != nil {
		s.log.Error("CreateAccount: UpdateAccountInfo failed", zap.Error(err))
		return connect.NewError(connect.CodeInternal, fmt.Errorf("update account info: %w", err))
	}
	s.log.Info("CreateAccount: verified and created",
		zap.String("id", accountID), zap.Float64("balance", info.Balance))
	return nil
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

	// DeleteAccount requires password verification for safety.
	creds, err := s.svc.GetAccountCredentials(ctx, userID, req.Msg.Id)
	if err != nil {
		s.log.Error("DeleteAccount: get credentials", zap.Error(err))
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to get account info: %w", err))
	}
	if creds.BrokerHost != "" {
		if req.Msg.Password == "" {
			return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("password is required to delete a verified account"))
		}
		if s.mtTester == nil {
			return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("MT connection tester not available"))
		}
		if err := s.mtTester.VerifyPassword(ctx, creds.Platform, creds.BrokerHost, creds.Login, req.Msg.Password); err != nil {
			s.log.Warn("DeleteAccount: password verification failed", zap.String("accountId", req.Msg.Id), zap.Error(err))
			return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("password verification failed: %w", err))
		}
	} else {
		s.log.Warn("DeleteAccount: skipping password verification (needs_rebind, no broker host)", zap.String("accountId", req.Msg.Id))
	}

	if err := s.svc.DeleteAccount(ctx, userID, req.Msg.Id); err != nil {
		s.log.Error("DeleteAccount", zap.Error(err))
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&emptypb.Empty{}), nil
}

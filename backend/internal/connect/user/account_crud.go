package user

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"connectrpc.com/connect"
	"go.uber.org/zap"
	"google.golang.org/protobuf/types/known/emptypb"

	antv1 "alphaforge/gen/proto/ant/v1"
	"alphaforge/internal/mdgateway/adapter/mdtick"
	"alphaforge/internal/service"
)

// CreateAccount inserts a new MT account, verifies credentials by connecting to MT,
// and returns the account with balance/equity/currency filled from AccountSummary.
func (s *AccountServer) CreateAccount(ctx context.Context, req *connect.Request[antv1.CreateAccountRequest]) (*connect.Response[antv1.Account], error) {
	r := req.Msg
	userID, err := parseUserID(ctx)
	if err != nil {
		return nil, err
	}

	// 1. Verify MT credentials BEFORE opening a PG transaction — holding a TX
	//    open across a network round-trip exhausts the connection pool under load.
	var info *mdtick.MTAccountInfo
	if s.mtTester != nil {
		result, err := s.verifyMTCredentials(ctx, r)
		if err != nil {
			return nil, err
		}
		info = result
	} else {
		s.log.Warn("CreateAccount: MT connection tester not available, skipping verification")
	}

	// 2. Begin transaction and insert.
	tx, err := s.svc.BeginTx(ctx)
	if err != nil {
		s.log.Error("CreateAccount: begin tx failed", zap.Error(err))
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	defer tx.Rollback(context.Background()) // use Background — user ctx may be cancelled

	firstHost := r.BrokerHost
	if idx := strings.IndexByte(firstHost, ','); idx > 0 {
		firstHost = firstHost[:idx]
	}
	// If verification found a working host, use it instead.
	if info != nil && info.BrokerHost != "" {
		firstHost = info.BrokerHost
	}
	id, err := s.svc.CreateAccountTx(ctx, tx, userID, r.Login, r.Password, r.MtType, r.BrokerCompany, r.BrokerServer, firstHost)
	if err != nil {
		if errors.Is(err, service.ErrAccountAlreadyBound) {
			return nil, connect.NewError(connect.CodeAlreadyExists, err)
		}
		s.log.Error("CreateAccount: db insert failed", zap.Error(err))
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	// 3. Update with verified account info (within transaction).
	if info != nil {
		if err := s.svc.UpdateAccountInfoTx(ctx, tx, userID, id, info.Balance, info.Equity, info.Credit, info.Margin, info.FreeMargin, int64(info.Leverage), info.Currency, info.IsInvestor); err != nil {
			s.log.Error("CreateAccount: UpdateAccountInfo failed", zap.Error(err))
			return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("update account info: %w", err))
		}
	}

	// 4. Commit.
	if err := tx.Commit(ctx); err != nil {
		s.log.Error("CreateAccount: tx commit failed", zap.Error(err))
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("commit transaction: %w", err))
	}

	// 5. Publish NATS event so mdgateway runner picks up the new account.
	if s.publisher != nil {
		s.publisher.PublishConnect(ctx, id, userID.String())
	}

	a, err := s.svc.GetAccount(ctx, userID, id)
	if err != nil {
		s.log.Error("CreateAccount: get account after create", zap.Error(err))
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	s.svc.LogAudit(ctx, uuid.MustParse(id), userID, "create",
		fmt.Sprintf("bound %s account %s on %s", r.MtType, r.Login, r.BrokerCompany))
	return connect.NewResponse(accountToProto(a)), nil
}

// verifyMTCredentials tests the MT connection and returns account info.
// Called BEFORE opening a PG transaction — network calls must not hold TX open.
func (s *AccountServer) verifyMTCredentials(ctx context.Context, r *antv1.CreateAccountRequest) (*mdtick.MTAccountInfo, error) {
	info, err := s.mtTester.Test(ctx, r.MtType, r.BrokerHost, r.Login, r.Password)
	if err != nil {
		s.log.Warn("CreateAccount: MT verification failed", zap.Error(err))
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("account verification failed: %w", err))
	}
	s.log.Info("CreateAccount: verified credentials",
		zap.String("host", info.BrokerHost), zap.String("balance", info.Balance.String()))
	return info, nil
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
	if err := s.svc.UpdateAccount(ctx, userID, r.Id, brokerCompany, brokerServer, brokerHost); err != nil {
		s.log.Error("UpdateAccount", zap.Error(err))
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	// Handle is_disabled: translate to connection state changes.
	// Migration 187 replaced the is_disabled column with the account_status state machine.
	if r.IsDisabled != nil {
		if *r.IsDisabled {
			err := s.updateAccountStatus(ctx, userID, r.Id, service.StatusDisconnected, func() {
				if s.publisher != nil {
					s.publisher.PublishDisconnect(ctx, r.Id, userID.String())
				}
			})
			if err != nil {
				s.log.Error("UpdateAccount: disconnect failed", zap.String("id", r.Id), zap.Error(err))
				return nil, connect.NewError(connect.CodeInternal, err)
			}
		} else {
			err := s.updateAccountStatus(ctx, userID, r.Id, service.StatusConnecting, func() {
				if s.publisher != nil {
					s.publisher.PublishConnect(ctx, r.Id, userID.String())
				}
			})
			if err != nil {
				s.log.Error("UpdateAccount: connect failed", zap.String("id", r.Id), zap.Error(err))
				return nil, connect.NewError(connect.CodeInternal, err)
			}
		}
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

	// Verify MT broker password — the broker is the only source of truth.
	// A locally stored password may be stale if the user changed it on the MT side.
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
		s.log.Warn("DeleteAccount: skipping password verification (no broker host)", zap.String("accountId", req.Msg.Id))
	}

	// Disconnect MT gateway before deleting the account.
	if s.publisher != nil {
		s.publisher.PublishDisconnect(ctx, req.Msg.Id, userID.String())
	}
	// Synchronously remove the gateway to prevent zombie gateways after delete.
	if s.stopGateway != nil {
		if err := s.stopGateway(ctx, req.Msg.Id); err != nil {
			s.log.Warn("DeleteAccount: gateway removal failed", zap.String("accountId", req.Msg.Id), zap.Error(err))
		}
	}

	if err := s.svc.DeleteAccount(ctx, userID, req.Msg.Id); err != nil {
		s.log.Error("DeleteAccount", zap.Error(err))
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	s.svc.LogAudit(ctx, uuid.MustParse(req.Msg.Id), userID, "delete",
		fmt.Sprintf("deleted account %s", creds.Login))
	return connect.NewResponse(&emptypb.Empty{}), nil
}

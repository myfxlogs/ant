package user

import (
	"context"
	"errors"
	"fmt"
	"strconv"

	"connectrpc.com/connect"
	"github.com/jackc/pgx/v5"
	"go.uber.org/zap"

	antv1 "anttrader/gen/proto/ant/v1"
	"anttrader/internal/service"
)

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

// UpdateTradingPassword updates the trading password for an account.
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
// account summary info WITHOUT saving anything to the database.
func (s *AccountServer) VerifyAccount(ctx context.Context, req *connect.Request[antv1.VerifyAccountRequest]) (*connect.Response[antv1.VerifyAccountResponse], error) {
	r := req.Msg
	if s.mtTester == nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("MT connection tester not available"))
	}
	info, err := s.mtTester.Test(ctx, r.MtType, r.BrokerHost, r.Login, r.Password)
	if err != nil {
		s.log.Warn("VerifyAccount: connection failed", zap.String("accountLogin", maskLogin(r.Login)), zap.Error(err))
		return connect.NewResponse(&antv1.VerifyAccountResponse{
			Verified: false,
			Message:  "Account verification failed — please check your credentials and broker server.",
		}), nil
	}
	return connect.NewResponse(&antv1.VerifyAccountResponse{
		Verified:   true,
		Message:    "account verified",
		Balance:    strconv.FormatFloat(info.Balance, 'f', -1, 64),
		Equity:     strconv.FormatFloat(info.Equity, 'f', -1, 64),
		Margin:     strconv.FormatFloat(info.Margin, 'f', -1, 64),
		FreeMargin: strconv.FormatFloat(info.FreeMargin, 'f', -1, 64),
		Leverage:   info.Leverage,
		Currency:   info.Currency,
	}), nil
}

// maskLogin masks a login value for safe logging.
func maskLogin(login string) string {
	if len(login) <= 3 {
		return "***"
	}
	return login[:3] + "***"
}

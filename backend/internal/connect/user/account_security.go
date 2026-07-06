package user

import (
	"context"
	"errors"
	"fmt"

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
	// Check real account state: frozen accounts cannot trade.
	if acct.Status == string(service.StatusFrozen) {
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

// VerifyAccount is deprecated. Use CreateAccount which verifies MT credentials inline.
func (s *AccountServer) VerifyAccount(ctx context.Context, req *connect.Request[antv1.VerifyAccountRequest]) (*connect.Response[antv1.VerifyAccountResponse], error) {
	return nil, connect.NewError(connect.CodeUnimplemented, fmt.Errorf("VerifyAccount is deprecated, use CreateAccount instead"))
}

package connect

import (
	"context"
	"errors"
	"strings"

	"connectrpc.com/connect"
	"github.com/google/uuid"

	v1 "anttrader/gen/proto"
	"anttrader/internal/interceptor"
)

// VerifyTradePermission 对应 proto RPC，轻量探测账户是否具备交易权限。
func (s *AccountService) VerifyTradePermission(
	ctx context.Context,
	req *connect.Request[v1.VerifyTradePermissionRequest],
) (*connect.Response[v1.VerifyTradePermissionResponse], error) {
	if s.accountSvc == nil {
		return nil, connect.NewError(connect.CodeFailedPrecondition, errors.New("account service not available"))
	}
	userIDStr := interceptor.GetUserID(ctx)
	if userIDStr == "" {
		return nil, connect.NewError(connect.CodeUnauthenticated, errors.New("errors.not_authenticated"))
	}
	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		return nil, connect.NewError(connect.CodeUnauthenticated, err)
	}
	accountID, err := uuid.Parse(strings.TrimSpace(req.Msg.Id))
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	res, err := s.accountSvc.VerifyTradePermission(ctx, userID, accountID)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&v1.VerifyTradePermissionResponse{
		HasTradePermission: res.HasTradePermission,
		IsInvestor:         res.IsInvestor,
		Verified:           res.Verified,
		Message:            res.Message,
	}), nil
}

// UpdateTradingPassword 对应 proto RPC，用新密码做一次 Connect 测试后覆盖已存密码。
func (s *AccountService) UpdateTradingPassword(
	ctx context.Context,
	req *connect.Request[v1.UpdateTradingPasswordRequest],
) (*connect.Response[v1.UpdateTradingPasswordResponse], error) {
	if s.accountSvc == nil {
		return nil, connect.NewError(connect.CodeFailedPrecondition, errors.New("account service not available"))
	}
	userIDStr := interceptor.GetUserID(ctx)
	if userIDStr == "" {
		return nil, connect.NewError(connect.CodeUnauthenticated, errors.New("errors.not_authenticated"))
	}
	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		return nil, connect.NewError(connect.CodeUnauthenticated, err)
	}
	accountID, err := uuid.Parse(strings.TrimSpace(req.Msg.Id))
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	newPwd := strings.TrimSpace(req.Msg.NewPassword)
	if newPwd == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("new_password cannot be empty"))
	}
	res, err := s.accountSvc.UpdateTradingPassword(ctx, userID, accountID, newPwd)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&v1.UpdateTradingPasswordResponse{
		Success:            res.Verified,
		HasTradePermission: res.HasTradePermission,
		IsInvestor:         res.IsInvestor,
		Message:            res.Message,
	}), nil
}

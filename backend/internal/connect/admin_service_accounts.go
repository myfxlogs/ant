package connect

import (
	"context"

	"connectrpc.com/connect"
	"github.com/google/uuid"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/timestamppb"

	v1 "anttrader/gen/proto"
	"anttrader/internal/model"
	"anttrader/internal/repository"
)

func (s *AdminService) ListAccountsAdmin(ctx context.Context, req *connect.Request[v1.ListAccountsAdminRequest]) (*connect.Response[v1.ListAccountsAdminResponse], error) {
	_, err := getUserIDFromContext(ctx)
	if err != nil {
		return nil, err
	}

	params := &model.AccountListParams{
		Page:     int(req.Msg.Page),
		PageSize: int(req.Msg.PageSize),
		Search:   req.Msg.Search,
		Status:   req.Msg.Status,
		MTType:   req.Msg.MtType,
		UserID:   req.Msg.UserId,
	}
	if params.Page <= 0 {
		params.Page = 1
	}
	if params.PageSize <= 0 {
		params.PageSize = 10
	}

	result, err := s.adminSvc.ListAccounts(ctx, params)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	accounts := make([]*v1.AccountWithUser, len(result.Data.([]*repository.AccountWithUser)))
	for i, a := range result.Data.([]*repository.AccountWithUser) {
		accounts[i] = convertAccountWithUserToProto(a)
	}

	return connect.NewResponse(&v1.ListAccountsAdminResponse{
		Accounts: accounts,
		Total:    int32(result.Total),
	}), nil
}

func (s *AdminService) GetAccountAdmin(ctx context.Context, req *connect.Request[v1.GetAccountAdminRequest]) (*connect.Response[v1.AccountWithUser], error) {
	_, err := getUserIDFromContext(ctx)
	if err != nil {
		return nil, err
	}

	uid, err := uuid.Parse(req.Msg.Id)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}

	account, err := s.adminSvc.GetAccount(ctx, uid)
	if err != nil {
		return nil, connect.NewError(connect.CodeNotFound, err)
	}

	return connect.NewResponse(convertAccountWithUserToProto(account)), nil
}

func (s *AdminService) FreezeAccount(ctx context.Context, req *connect.Request[v1.FreezeAccountRequest]) (*connect.Response[emptypb.Empty], error) {
	operatorID, err := getUserIDFromContext(ctx)
	if err != nil {
		return nil, err
	}

	uid, err := uuid.Parse(req.Msg.Id)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}

	if err := s.adminSvc.FreezeAccount(ctx, uid, operatorID); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	return connect.NewResponse(&emptypb.Empty{}), nil
}

func (s *AdminService) UnfreezeAccount(ctx context.Context, req *connect.Request[v1.UnfreezeAccountRequest]) (*connect.Response[emptypb.Empty], error) {
	operatorID, err := getUserIDFromContext(ctx)
	if err != nil {
		return nil, err
	}

	uid, err := uuid.Parse(req.Msg.Id)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}

	if err := s.adminSvc.UnfreezeAccount(ctx, uid, operatorID); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	return connect.NewResponse(&emptypb.Empty{}), nil
}

func derefString(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

func convertAccountWithUserToProto(a *repository.AccountWithUser) *v1.AccountWithUser {
	if a == nil {
		return &v1.AccountWithUser{}
	}
	return &v1.AccountWithUser{
		Id:              a.ID.String(),
		UserId:          a.UserID.String(),
		Login:           a.Login,
		MtType:          a.MTType,
		BrokerCompany:   a.BrokerCompany,
		BrokerServer:    a.BrokerServer,
		Status:          a.AccountStatus,
		AccountStatus:   a.AccountStatus,
		Balance:         a.Balance,
		Credit:          a.Credit,
		Equity:          a.Equity,
		Margin:          a.Margin,
		FreeMargin:      a.FreeMargin,
		MarginLevel:     a.MarginLevel,
		Leverage:        int32(a.Leverage),
		Currency:        a.Currency,
		UserEmail:       a.UserEmail,
		UserNickname:    derefString(a.UserNickname),
		LastConnectedAt: convertTimeToTimestamp(a.LastConnectedAt),
		CreatedAt:       timestamppb.New(a.CreatedAt),
	}
}

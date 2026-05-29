package admin

import (
	"context"

	"go.uber.org/zap"
	"google.golang.org/protobuf/types/known/timestamppb"

	"connectrpc.com/connect"
	"github.com/google/uuid"

	antv1 "anttrader/gen/proto/ant/v1"
	antv1c "anttrader/gen/proto/ant/v1/antv1connect"
	"anttrader/internal/model"
	"anttrader/internal/repository"
)

type AdminAccountServer struct {
	repo *repository.AdminRepository
	log  *zap.Logger
}

var _ antv1c.AdminAccountServiceHandler = (*AdminAccountServer)(nil)

func NewAdminAccountServer(repo *repository.AdminRepository, log *zap.Logger) *AdminAccountServer {
	return &AdminAccountServer{repo: repo, log: log}
}

func accountWithUserToProto(a *repository.AccountWithUser) *antv1.AccountWithUser {
	p := &antv1.AccountWithUser{
		Id:            a.ID.String(),
		UserId:        a.UserID.String(),
		UserEmail:     a.UserEmail,
		MtType:        a.MTType,
		BrokerCompany: a.BrokerCompany,
		BrokerServer:  a.BrokerServer,
		Login:         a.Login,
		Alias:         a.Alias,
		IsDisabled:    a.IsDisabled,
		Balance:       a.Balance,
		Credit:        a.Credit,
		Equity:        a.Equity,
		Margin:        a.Margin,
		FreeMargin:    a.FreeMargin,
		MarginLevel:   a.MarginLevel,
		Leverage:      int32(a.Leverage),
		Currency:      a.Currency,
		IsInvestor:    a.IsInvestor,
		AccountStatus: a.AccountStatus,
		StreamStatus:  a.StreamStatus,
		LastError:     a.LastError,
		AccountType:   a.AccountType,
		CreatedAt:     timestamppb.New(a.CreatedAt),
		UpdatedAt:     timestamppb.New(a.UpdatedAt),
	}
	if a.UserNickname != nil {
		p.UserNickname = *a.UserNickname
	}
	if a.LastConnectedAt != nil {
		p.LastConnectedAt = timestamppb.New(*a.LastConnectedAt)
	}
	if a.LastCheckedAt != nil {
		p.LastCheckedAt = timestamppb.New(*a.LastCheckedAt)
	}
	return p
}

func (s *AdminAccountServer) ListAccountsAdmin(ctx context.Context, req *connect.Request[antv1.ListAccountsAdminRequest]) (*connect.Response[antv1.ListAccountsAdminResponse], error) {
	params := &model.AccountListParams{
		Page:     int(req.Msg.Page),
		PageSize: int(req.Msg.PageSize),
		Search:   req.Msg.Search,
		Status:   req.Msg.Status,
		MTType:   req.Msg.MtType,
		UserID:   req.Msg.UserId,
	}
	accounts, total, err := s.repo.ListAccounts(ctx, params)
	if err != nil {
		s.log.Error("ListAccountsAdmin failed", zap.Error(err))
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	items := make([]*antv1.AccountWithUser, len(accounts))
	for i, a := range accounts {
		items[i] = accountWithUserToProto(a)
	}
	return connect.NewResponse(&antv1.ListAccountsAdminResponse{Accounts: items, Total: total}), nil
}

func (s *AdminAccountServer) FreezeAccount(ctx context.Context, req *connect.Request[antv1.FreezeAccountRequest]) (*connect.Response[antv1.FreezeAccountResponse], error) {
	id, err := uuid.Parse(req.Msg.Id)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	if err := s.repo.SetAccountStatus(ctx, id, "frozen"); err != nil {
		return nil, err
	}
	return connect.NewResponse(&antv1.FreezeAccountResponse{}), nil
}

func (s *AdminAccountServer) UnfreezeAccount(ctx context.Context, req *connect.Request[antv1.UnfreezeAccountRequest]) (*connect.Response[antv1.UnfreezeAccountResponse], error) {
	id, err := uuid.Parse(req.Msg.Id)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	if err := s.repo.SetAccountStatus(ctx, id, "active"); err != nil {
		return nil, err
	}
	return connect.NewResponse(&antv1.UnfreezeAccountResponse{}), nil
}

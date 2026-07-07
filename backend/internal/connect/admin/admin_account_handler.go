package admin

import (
	"context"

	"go.uber.org/zap"
	"google.golang.org/protobuf/types/known/timestamppb"

	"connectrpc.com/connect"
	"github.com/google/uuid"

	antv1 "anttrader/gen/proto/ant/v1"
	antv1c "anttrader/gen/proto/ant/v1/antv1connect"
	"anttrader/internal/mdgateway"
	"anttrader/internal/model"
	"anttrader/internal/repository"
)

type AdminAccountServer struct {
	repo      *repository.AdminRepository
	log       *zap.Logger
	publisher *mdgateway.AccountEventPublisher // nil in tests
}

var _ antv1c.AdminAccountServiceHandler = (*AdminAccountServer)(nil)

func NewAdminAccountServer(repo *repository.AdminRepository, log *zap.Logger) *AdminAccountServer {
	return &AdminAccountServer{repo: repo, log: log}
}

// WithPublisher injects a NATS publisher so freeze/unfreeze emit lifecycle events.
func (s *AdminAccountServer) WithPublisher(p *mdgateway.AccountEventPublisher) *AdminAccountServer {
	s.publisher = p
	return s
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
		// Financial values: use string-encoded decimal to avoid float64 precision loss.
		// When proto fields are migrated to string, this will become direct string assignment.
		Balance:       a.Balance.String(),
		Credit:        a.Credit.String(),
		Equity:        a.Equity.String(),
		Margin:         a.Margin.String(),
		FreeMargin:    a.FreeMargin.String(),
		MarginLevel:   a.MarginLevel.String(),
		Leverage:      int32(a.Leverage),
		Currency:      a.Currency,
		IsInvestor:    a.IsInvestor,
		AccountStatus: a.AccountStatus,
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
	// Publish NATS BEFORE DB — if NATS is down, we don't freeze in DB either.
	// If DB write fails after NATS succeeds, the gateway disconnects briefly but
	// the DB status stays unfrozen (no permanent inconsistency).
	if s.publisher != nil {
		if acct, err := s.repo.GetAccountByID(ctx, id); err == nil {
			s.publisher.PublishDisconnect(ctx, id.String(), acct.UserID.String())
		}
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
	// Publish NATS BEFORE DB — same fail-safe ordering as FreezeAccount.
	if s.publisher != nil {
		if acct, err := s.repo.GetAccountByID(ctx, id); err == nil {
			s.publisher.PublishReconnect(ctx, id.String(), acct.UserID.String())
		}
	}
	if err := s.repo.SetAccountStatus(ctx, id, "disconnected"); err != nil {
		return nil, err
	}
	return connect.NewResponse(&antv1.UnfreezeAccountResponse{}), nil
}

// GetAccountAuditLogs returns audit log entries for an account via ConnectRPC.
// Auth is handled by the interceptor chain (authInterceptor + adminInterceptor).
func (s *AdminAccountServer) GetAccountAuditLogs(ctx context.Context, req *connect.Request[antv1.GetAccountAuditLogsRequest]) (*connect.Response[antv1.GetAccountAuditLogsResponse], error) {
	limit := int(req.Msg.Limit)
	if limit <= 0 || limit > 200 {
		limit = 100
	}
	entries, err := s.repo.GetAuditLogs(ctx, req.Msg.AccountId, limit)
	if err != nil {
		s.log.Error("GetAccountAuditLogs", zap.Error(err))
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	pb := make([]*antv1.AccountAuditLogEntry, len(entries))
	for i, e := range entries {
		pb[i] = &antv1.AccountAuditLogEntry{
			Id:        e.ID,
			AccountId: e.AccountID,
			UserId:    e.UserID,
			Action:    e.Action,
			Detail:    e.Detail,
			CreatedAt: timestamppb.New(e.CreatedAt),
		}
	}
	return connect.NewResponse(&antv1.GetAccountAuditLogsResponse{Entries: pb}), nil
}

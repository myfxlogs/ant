package admin

import (
	"context"

	"go.uber.org/zap"
	"google.golang.org/protobuf/types/known/timestamppb"

	"connectrpc.com/connect"
	"github.com/google/uuid"

	antv1 "alphaforge/gen/proto/ant/v1"
	antv1c "alphaforge/gen/proto/ant/v1/antv1connect"
	antinterceptor "alphaforge/internal/interceptor"
	"alphaforge/internal/model"
	"alphaforge/internal/repository"
	"alphaforge/internal/service"
	usersvc "alphaforge/internal/service/user"
)

type AdminUserServer struct {
	repo       *repository.AdminRepository
	resetRepo  *repository.PasswordResetRepo
	walletSvc  *service.WalletService
	acctSvc    *usersvc.AccountNumberService
	deletionSvc *service.UserDeletionService
	log        *zap.Logger
}

var _ antv1c.AdminUserServiceHandler = (*AdminUserServer)(nil)

func NewAdminUserServer(repo *repository.AdminRepository, resetRepo *repository.PasswordResetRepo, walletSvc *service.WalletService, acctSvc *usersvc.AccountNumberService, deletionSvc *service.UserDeletionService, log *zap.Logger) *AdminUserServer {
	return &AdminUserServer{repo: repo, resetRepo: resetRepo, walletSvc: walletSvc, acctSvc: acctSvc, deletionSvc: deletionSvc, log: log}
}

func userWithAccountsToProto(u *repository.UserWithAccounts) *antv1.UserWithAccounts {
	p := &antv1.UserWithAccounts{
		Id:        u.ID.String(),
		Email:     u.Email,
		Role:      u.Role,
		Status:    u.Status,
		Username:  u.Email, // email serves as username
		CreatedAt: timestamppb.New(u.CreatedAt),
		UpdatedAt: timestamppb.New(u.UpdatedAt),
	}
	if u.Nickname != nil {
		p.Nickname = *u.Nickname
	}
	if u.LastLoginAt != nil {
		p.LastLoginAt = timestamppb.New(*u.LastLoginAt)
	}
	if u.AccountNumber != nil {
		p.AccountNumber = *u.AccountNumber
	}
	if u.DeletedAt != nil {
		p.DeletedAt = timestamppb.New(*u.DeletedAt)
	}
	return p
}

func (s *AdminUserServer) GetDashboard(ctx context.Context, _ *connect.Request[antv1.GetDashboardRequest]) (*connect.Response[antv1.GetDashboardResponse], error) {
	stats, err := s.repo.GetDashboardStats(ctx)
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(&antv1.GetDashboardResponse{
		Stats: &antv1.DashboardStats{
			TotalUsers:     stats.TotalUsers,
			ActiveUsers:    stats.ActiveUsers,
			TotalAccounts:  stats.TotalAccounts,
			OnlineAccounts: stats.OnlineAccounts,
			TodayTrades:    stats.TodayTrades,
			TodayVolume:    stats.TodayVolume.String(),
			TodayProfit:    stats.TodayProfit.String(),
			SystemLoad:     stats.SystemLoad.InexactFloat64(),
			ActiveSubscriptions:   stats.ActiveSubscriptions,
			MonthlyRevenue:        stats.MonthlyRevenue.String(),
			TotalRevenue:          stats.TotalRevenue.String(),
			MarketplaceStrategies: stats.MarketplaceStrategies,
			MarketplaceSales:      stats.MarketplaceSales,
			MarketplaceRevenue:    stats.MarketplaceRevenue.String(),
			VerifiedUsers:         stats.VerifiedUsers,
		},
	}), nil
}

func (s *AdminUserServer) ListUsers(ctx context.Context, req *connect.Request[antv1.ListUsersRequest]) (*connect.Response[antv1.ListUsersResponse], error) {
	params := &model.UserListParams{
		Page:          int(req.Msg.Page),
		PageSize:      int(req.Msg.PageSize),
		Search:        req.Msg.Search,
		Status:        req.Msg.Status,
		Role:          req.Msg.Role,
		DeletedFilter: req.Msg.DeletedFilter,
	}
	users, total, err := s.repo.ListUsers(ctx, params)
	if err != nil {
		return nil, err
	}
	items := make([]*antv1.UserWithAccounts, len(users))
	for i, u := range users {
		items[i] = userWithAccountsToProto(u)
	}
	return connect.NewResponse(&antv1.ListUsersResponse{Users: items, Total: total}), nil
}


// validRole returns true if the role string is a recognized user role.
func validRole(role string) bool {
	switch role {
	case "user", "admin", "super_admin", "operation", "audit", "customer_service":
		return true
	}
	return false
}

// getActorID extracts the authenticated user ID from the request context.
func getActorID(ctx context.Context) uuid.UUID {
	raw := antinterceptor.GetUserID(ctx)
	if raw == "" {
		return uuid.Nil
	}
	id, err := uuid.Parse(raw)
	if err != nil {
		return uuid.Nil
	}
	return id
}

package admin

import (
	"context"
	"fmt"

	"go.uber.org/zap"
	"golang.org/x/crypto/bcrypt"
	"google.golang.org/protobuf/types/known/timestamppb"

	"connectrpc.com/connect"
	"github.com/google/uuid"

	antv1 "anttrader/gen/proto/ant/v1"
	antv1c "anttrader/gen/proto/ant/v1/antv1connect"
	"anttrader/internal/model"
	"anttrader/internal/repository"
)

type AdminUserServer struct {
	repo       *repository.AdminRepository
	resetRepo  *repository.PasswordResetRepo
	log        *zap.Logger
}

var _ antv1c.AdminUserServiceHandler = (*AdminUserServer)(nil)

func NewAdminUserServer(repo *repository.AdminRepository, resetRepo *repository.PasswordResetRepo, log *zap.Logger) *AdminUserServer {
	return &AdminUserServer{repo: repo, resetRepo: resetRepo, log: log}
}

func userWithAccountsToProto(u *repository.UserWithAccounts) *antv1.UserWithAccounts {
	p := &antv1.UserWithAccounts{
		Id:       u.ID.String(),
		Email:    u.Email,
		Role:     u.Role,
		Status:   u.Status,
		Username: u.Email, // email serves as username
		CreatedAt: timestamppb.New(u.CreatedAt),
		UpdatedAt: timestamppb.New(u.UpdatedAt),
	}
	if u.Nickname != nil {
		p.Nickname = *u.Nickname
	}
	if u.LastLoginAt != nil {
		p.LastLoginAt = timestamppb.New(*u.LastLoginAt)
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
			TodayVolume:    stats.TodayVolume,
			TodayProfit:    stats.TodayProfit,
			SystemLoad:     stats.SystemLoad,
		},
	}), nil
}

func (s *AdminUserServer) ListUsers(ctx context.Context, req *connect.Request[antv1.ListUsersRequest]) (*connect.Response[antv1.ListUsersResponse], error) {
	params := &model.UserListParams{
		Page:     int(req.Msg.Page),
		PageSize: int(req.Msg.PageSize),
		Search:   req.Msg.Search,
		Status:   req.Msg.Status,
		Role:     req.Msg.Role,
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

func (s *AdminUserServer) CreateUser(ctx context.Context, req *connect.Request[antv1.CreateUserRequest]) (*connect.Response[antv1.CreateUserResponse], error) {
	user := &model.User{
		Email: req.Msg.Username, // username maps to email in this system
		Role:  req.Msg.Role,
	}
	if user.Role == "" {
		user.Role = "user"
	}
	hashed, err := bcrypt.GenerateFromPassword([]byte(req.Msg.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("hash password: %w", err))
	}
	user.PasswordHash = string(hashed)
	if err := s.repo.CreateUser(ctx, user); err != nil {
		return nil, err
	}
	return connect.NewResponse(&antv1.CreateUserResponse{Id: user.ID.String()}), nil
}

func (s *AdminUserServer) UpdateUser(ctx context.Context, req *connect.Request[antv1.UpdateUserRequest]) (*connect.Response[antv1.UpdateUserResponse], error) {
	id, err := uuid.Parse(req.Msg.Id)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	existing, err := s.repo.GetUserByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if req.Msg.Email != "" {
		existing.Email = req.Msg.Email
	}
	if req.Msg.Role != "" {
		existing.Role = req.Msg.Role
	}
	if req.Msg.Status != "" {
		existing.Status = req.Msg.Status
	}
	if req.Msg.Nickname != "" {
		existing.Nickname = &req.Msg.Nickname
	}
	if err := s.repo.UpdateUser(ctx, existing); err != nil {
		return nil, err
	}
	return connect.NewResponse(&antv1.UpdateUserResponse{}), nil
}

func (s *AdminUserServer) DeleteUser(ctx context.Context, req *connect.Request[antv1.DeleteUserRequest]) (*connect.Response[antv1.DeleteUserResponse], error) {
	id, err := uuid.Parse(req.Msg.Id)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	if err := s.repo.DeleteUser(ctx, id); err != nil {
		return nil, err
	}
	return connect.NewResponse(&antv1.DeleteUserResponse{}), nil
}

func (s *AdminUserServer) DisableUser(ctx context.Context, req *connect.Request[antv1.DisableUserRequest]) (*connect.Response[antv1.DisableUserResponse], error) {
	id, err := uuid.Parse(req.Msg.Id)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	if err := s.repo.SetUserStatus(ctx, id, "disabled"); err != nil {
		return nil, err
	}
	return connect.NewResponse(&antv1.DisableUserResponse{}), nil
}

func (s *AdminUserServer) EnableUser(ctx context.Context, req *connect.Request[antv1.EnableUserRequest]) (*connect.Response[antv1.EnableUserResponse], error) {
	id, err := uuid.Parse(req.Msg.Id)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	if err := s.repo.SetUserStatus(ctx, id, "active"); err != nil {
		return nil, err
	}
	return connect.NewResponse(&antv1.EnableUserResponse{}), nil
}

func (s *AdminUserServer) ResetUserPassword(ctx context.Context, req *connect.Request[antv1.ResetUserPasswordRequest]) (*connect.Response[antv1.ResetUserPasswordResponse], error) {
	id, err := uuid.Parse(req.Msg.Id)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}

	// Hash a random temporary password and store in DB.
	tempPass, err := repository.GenerateToken()
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	hash, err := repository.HashPassword(tempPass)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	if err := s.repo.ResetUserPassword(ctx, id, hash); err != nil {
		return nil, err
	}

	// Create a one-time reset token so the user can set their own password.
	if _, err := s.resetRepo.CreateResetToken(ctx, id); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	// Response does not include the plaintext password.
	return connect.NewResponse(&antv1.ResetUserPasswordResponse{}), nil
}

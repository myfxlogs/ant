package connect

import (
	"context"
	"time"

	"connectrpc.com/connect"
	"github.com/google/uuid"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/timestamppb"

	v1 "anttrader/gen/proto"
	"anttrader/internal/model"
	"anttrader/internal/repository"
	"anttrader/internal/service"
)

func (s *AdminService) ListUsers(ctx context.Context, req *connect.Request[v1.ListUsersRequest]) (*connect.Response[v1.ListUsersResponse], error) {
	_, err := getUserIDFromContext(ctx)
	if err != nil {
		return nil, err
	}

	params := &model.UserListParams{
		Page:     int(req.Msg.Page),
		PageSize: int(req.Msg.PageSize),
		Search:   req.Msg.Search,
		Status:   req.Msg.Status,
		Role:     req.Msg.Role,
	}
	if params.Page <= 0 {
		params.Page = 1
	}
	if params.PageSize <= 0 {
		params.PageSize = 10
	}

	result, err := s.adminSvc.ListUsers(ctx, params)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	users := make([]*v1.UserWithAccounts, len(result.Data.([]*repository.UserWithAccounts)))
	for i, u := range result.Data.([]*repository.UserWithAccounts) {
		var nickname string
		if u.Nickname != nil {
			nickname = *u.Nickname
		}
		users[i] = &v1.UserWithAccounts{
			Id:             u.ID.String(),
			Email:          u.Email,
			Username:       nickname,
			Nickname:       nickname,
			Role:           u.Role,
			Status:         u.Status,
			MtAccountCount: int32(u.MTAccountCount),
			LastLoginAt:    convertTimeToTimestamp(u.LastLoginAt),
			CreatedAt:      convertTimeToTimestamp(&u.CreatedAt),
		}
	}

	return connect.NewResponse(&v1.ListUsersResponse{
		Users: users,
		Total: int32(result.Total),
	}), nil
}

func (s *AdminService) GetUser(ctx context.Context, req *connect.Request[v1.GetUserRequest]) (*connect.Response[v1.UserWithAccounts], error) {
	_, err := getUserIDFromContext(ctx)
	if err != nil {
		return nil, err
	}

	uid, err := uuid.Parse(req.Msg.Id)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}

	user, err := s.adminSvc.GetUser(ctx, uid)
	if err != nil {
		return nil, connect.NewError(connect.CodeNotFound, err)
	}

	return connect.NewResponse(convertUserToProto(user)), nil
}

func (s *AdminService) CreateUser(ctx context.Context, req *connect.Request[v1.CreateUserRequest]) (*connect.Response[v1.UserWithAccounts], error) {
	operatorID, err := getUserIDFromContext(ctx)
	if err != nil {
		return nil, err
	}

	createReq := &service.CreateUserRequest{
		Email:    req.Msg.Email,
		Password: req.Msg.Password,
		Nickname: req.Msg.Username,
		Role:     req.Msg.Role,
	}

	user, err := s.adminSvc.CreateUser(ctx, createReq, operatorID)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	return connect.NewResponse(convertUserToProto(user)), nil
}

func (s *AdminService) UpdateUser(ctx context.Context, req *connect.Request[v1.UpdateUserRequest]) (*connect.Response[v1.UserWithAccounts], error) {
	operatorID, err := getUserIDFromContext(ctx)
	if err != nil {
		return nil, err
	}

	uid, err := uuid.Parse(req.Msg.Id)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}

	updateReq := &service.UpdateUserRequest{}
	if req.Msg.Username != nil {
		updateReq.Nickname = *req.Msg.Username
	}
	if req.Msg.Role != nil {
		updateReq.Role = *req.Msg.Role
	}
	if req.Msg.Status != nil {
		updateReq.Status = *req.Msg.Status
	}

	user, err := s.adminSvc.UpdateUser(ctx, uid, updateReq, operatorID)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	return connect.NewResponse(convertUserToProto(user)), nil
}

func (s *AdminService) DeleteUser(ctx context.Context, req *connect.Request[v1.DeleteUserRequest]) (*connect.Response[emptypb.Empty], error) {
	operatorID, err := getUserIDFromContext(ctx)
	if err != nil {
		return nil, err
	}

	uid, err := uuid.Parse(req.Msg.Id)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}

	if err := s.adminSvc.DeleteUser(ctx, uid, operatorID); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	return connect.NewResponse(&emptypb.Empty{}), nil
}

func (s *AdminService) DisableUser(ctx context.Context, req *connect.Request[v1.DisableUserRequest]) (*connect.Response[emptypb.Empty], error) {
	operatorID, err := getUserIDFromContext(ctx)
	if err != nil {
		return nil, err
	}

	uid, err := uuid.Parse(req.Msg.Id)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}

	if err := s.adminSvc.DisableUser(ctx, uid, operatorID); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	return connect.NewResponse(&emptypb.Empty{}), nil
}

func (s *AdminService) EnableUser(ctx context.Context, req *connect.Request[v1.EnableUserRequest]) (*connect.Response[emptypb.Empty], error) {
	operatorID, err := getUserIDFromContext(ctx)
	if err != nil {
		return nil, err
	}

	uid, err := uuid.Parse(req.Msg.Id)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}

	if err := s.adminSvc.EnableUser(ctx, uid, operatorID); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	return connect.NewResponse(&emptypb.Empty{}), nil
}

func (s *AdminService) ResetUserPassword(ctx context.Context, req *connect.Request[v1.ResetUserPasswordRequest]) (*connect.Response[v1.ResetUserPasswordResponse], error) {
	operatorID, err := getUserIDFromContext(ctx)
	if err != nil {
		return nil, err
	}

	uid, err := uuid.Parse(req.Msg.Id)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}

	newPassword := generateRandomPassword()
	resetReq := &service.ResetPasswordRequest{
		NewPassword: newPassword,
	}

	if err := s.adminSvc.ResetUserPassword(ctx, uid, resetReq, operatorID); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	return connect.NewResponse(&v1.ResetUserPasswordResponse{
		NewPassword: newPassword,
	}), nil
}

func convertUserToProto(u *model.User) *v1.UserWithAccounts {
	var nickname string
	if u.Nickname != nil {
		nickname = *u.Nickname
	}
	var lastLoginAt *timestamppb.Timestamp
	if u.LastLoginAt != nil {
		lastLoginAt = timestamppb.New(*u.LastLoginAt)
	}
	var createdAt *timestamppb.Timestamp
	if !u.CreatedAt.IsZero() {
		createdAt = timestamppb.New(u.CreatedAt)
	}
	return &v1.UserWithAccounts{
		Id:          u.ID.String(),
		Email:       u.Email,
		Username:    nickname,
		Nickname:    nickname,
		Role:        u.Role,
		Status:      u.Status,
		LastLoginAt: lastLoginAt,
		CreatedAt:   createdAt,
	}
}

func generateRandomPassword() string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, 12)
	for i := range b {
		b[i] = charset[i%len(charset)]
	}
	return string(b)
}

func convertTimeToTimestamp(t *time.Time) *timestamppb.Timestamp {
	if t == nil {
		return nil
	}
	return timestamppb.New(*t)
}

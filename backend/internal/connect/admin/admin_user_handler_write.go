package admin

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"go.uber.org/zap"

	"connectrpc.com/connect"
	"github.com/google/uuid"

	antv1 "alphaforge/gen/proto/ant/v1"
	"alphaforge/internal/model"
	"alphaforge/internal/pkg/hash"
	"alphaforge/internal/repository"
	"alphaforge/internal/service"
	usersvc "alphaforge/internal/service/user"
)

func (s *AdminUserServer) CreateUser(ctx context.Context, req *connect.Request[antv1.CreateUserRequest]) (*connect.Response[antv1.CreateUserResponse], error) {
	email := strings.TrimSpace(req.Msg.Username)
	if email == "" || !strings.Contains(email, "@") {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("invalid email address"))
	}
	password := req.Msg.Password
	if len(password) < 8 {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("password must be at least 8 characters"))
	}
	role := req.Msg.Role
	if role == "" {
		role = "user"
	}
	if !validRole(role) {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("invalid role: %s", role))
	}

	user := &model.User{
		Email: email,
		Role:  role,
	}
	if err := s.resolveAccountNumber(ctx, req.Msg.AccountNumber, user); err != nil {
		return nil, err
	}
	hashed, err := hash.HashPassword(password)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("hash password: %w", err))
	}
	user.PasswordHash = hashed

	if err := s.createUserWithRetry(ctx, user); err != nil {
		return nil, err
	}
	if s.walletSvc != nil {
		if _, err := s.walletSvc.CreateWallet(ctx, user.ID); err != nil {
			s.log.Warn("admin: create wallet for new user failed",
				zap.String("userID", user.ID.String()), zap.Error(err))
		}
	}
	s.log.Info("admin: user created",
		zap.String("actor", getActorID(ctx).String()),
		zap.String("target", user.ID.String()),
		zap.String("email", email),
		zap.String("role", role))
	return connect.NewResponse(&antv1.CreateUserResponse{Id: user.ID.String()}), nil
}

func (s *AdminUserServer) resolveAccountNumber(ctx context.Context, acctNum string, user *model.User) error {
	if acctNum != "" {
		if err := usersvc.ValidateAccountNumber(acctNum); err != nil {
			return connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("invalid account_number: %w", err))
		}
		if s.acctSvc != nil {
			avail, err := s.acctSvc.IsAccountNumberAvailable(ctx, acctNum)
			if err != nil {
				return connect.NewError(connect.CodeInternal, fmt.Errorf("check account_number availability: %w", err))
			}
			if !avail {
				return connect.NewError(connect.CodeAlreadyExists, fmt.Errorf("account number %s is already taken", acctNum))
			}
		}
		user.AccountNumber = &acctNum
		return nil
	}
	if s.acctSvc != nil {
		num, err := s.acctSvc.GenerateAccountNumber(ctx)
		if err != nil {
			s.log.Warn("admin: auto-generate account number failed, continuing without one", zap.Error(err))
			return nil
		}
		user.AccountNumber = &num
	}
	return nil
}

func (s *AdminUserServer) createUserWithRetry(ctx context.Context, user *model.User) error {
	const maxRetries = 3
	for attempt := 0; ; attempt++ {
		err := s.repo.CreateUser(ctx, user)
		if err == nil {
			return nil
		}
		if usersvc.IsAccountNumberViolation(err) {
			if attempt >= maxRetries {
				return connect.NewError(connect.CodeAlreadyExists, fmt.Errorf("account number collision after %d retries", maxRetries))
			}
			if s.acctSvc != nil {
				num, genErr := s.acctSvc.GenerateAccountNumber(ctx)
				if genErr != nil {
					return connect.NewError(connect.CodeInternal, fmt.Errorf("regenerate account number: %w", genErr))
				}
				user.AccountNumber = &num
			}
			continue
		}
		if usersvc.IsUniqueViolation(err) {
			return connect.NewError(connect.CodeAlreadyExists, fmt.Errorf("email already registered"))
		}
		return err
	}
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
	prevStatus := existing.Status
	prevRole := existing.Role
	if req.Msg.Email != "" {
		existing.Email = req.Msg.Email
	}
	if req.Msg.Role != "" {
		if !validRole(req.Msg.Role) {
			return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("invalid role: %s", req.Msg.Role))
		}
		existing.Role = req.Msg.Role
	}
	if req.Msg.Status != "" {
		if !validStatus(req.Msg.Status) {
			return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("invalid status: %s", req.Msg.Status))
		}
		existing.Status = req.Msg.Status
	}
	if req.Msg.Nickname != "" {
		existing.Nickname = &req.Msg.Nickname
	}
	if req.Msg.AccountNumber != "" {
		if err := usersvc.ValidateAccountNumber(req.Msg.AccountNumber); err != nil {
			return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("invalid account_number: %w", err))
		}
		if existing.AccountNumber == nil || *existing.AccountNumber != req.Msg.AccountNumber {
			if s.acctSvc != nil {
				avail, err := s.acctSvc.IsAccountNumberAvailableExcluding(ctx, req.Msg.AccountNumber, id.String())
				if err != nil {
					return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("check account_number: %w", err))
				}
				if !avail {
					return nil, connect.NewError(connect.CodeAlreadyExists, fmt.Errorf("account number %s is already taken", req.Msg.AccountNumber))
				}
			}
			existing.AccountNumber = &req.Msg.AccountNumber
		}
	}
	if err := s.repo.UpdateUser(ctx, existing); err != nil {
		return nil, err
	}
	if prevStatus != existing.Status && existing.Status != "active" {
		if err := s.repo.IncrementTokenVersion(ctx, id); err != nil {
			s.log.Warn("admin: update user — increment token version failed",
				zap.String("target", id.String()), zap.Error(err))
		}
	}
	if prevRole != existing.Role {
		if err := s.repo.IncrementTokenVersion(ctx, id); err != nil {
			s.log.Warn("admin: update user role — increment token version failed",
				zap.String("target", id.String()), zap.Error(err))
		}
	}
	s.log.Info("admin: user updated",
		zap.String("actor", getActorID(ctx).String()),
		zap.String("target", id.String()),
		zap.String("role", existing.Role),
		zap.String("status", existing.Status))
	return connect.NewResponse(&antv1.UpdateUserResponse{}), nil
}

func (s *AdminUserServer) DeleteUser(ctx context.Context, req *connect.Request[antv1.DeleteUserRequest]) (*connect.Response[antv1.DeleteUserResponse], error) {
	id, err := uuid.Parse(req.Msg.Id)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	actorID := getActorID(ctx)
	switch err := s.deletionSvc.SoftDeleteUser(ctx, actorID, id); {
	case err == nil:
		_ = s.repo.IncrementTokenVersion(ctx, id)
	case errors.Is(err, service.ErrCannotDeleteSelf):
		return nil, connect.NewError(connect.CodePermissionDenied, err)
	case errors.Is(err, service.ErrCannotDeleteLastAdmin):
		return nil, connect.NewError(connect.CodeFailedPrecondition, err)
	default:
		s.log.Error("admin: delete user failed", zap.Error(err), zap.String("target", id.String()))
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&antv1.DeleteUserResponse{}), nil
}

func (s *AdminUserServer) DeleteUsers(ctx context.Context, req *connect.Request[antv1.DeleteUsersRequest]) (*connect.Response[antv1.DeleteUsersResponse], error) {
	actorID := getActorID(ctx)
	if len(req.Msg.Ids) == 0 {
		return connect.NewResponse(&antv1.DeleteUsersResponse{}), nil
	}
	deleted, failed, errors := s.deletionSvc.SoftDeleteUsers(ctx, actorID, req.Msg.Ids)
	for _, rawID := range req.Msg.Ids {
		if uid, err := uuid.Parse(rawID); err == nil {
			_ = s.repo.IncrementTokenVersion(ctx, uid)
		}
	}
	return connect.NewResponse(&antv1.DeleteUsersResponse{
		DeletedCount: int32(deleted),
		FailedCount:  int32(failed),
		Errors:       errors,
	}), nil
}

func (s *AdminUserServer) RestoreUser(ctx context.Context, req *connect.Request[antv1.RestoreUserRequest]) (*connect.Response[antv1.RestoreUserResponse], error) {
	id, err := uuid.Parse(req.Msg.Id)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	actorID := getActorID(ctx)
	switch err := s.deletionSvc.RestoreUser(ctx, actorID, id); {
	case err == nil:
	case errors.Is(err, service.ErrUserNotDeleted):
		return nil, connect.NewError(connect.CodeNotFound, err)
	default:
		s.log.Error("admin: restore user failed", zap.Error(err), zap.String("target", id.String()))
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&antv1.RestoreUserResponse{}), nil
}

func (s *AdminUserServer) DisableUser(ctx context.Context, req *connect.Request[antv1.DisableUserRequest]) (*connect.Response[antv1.DisableUserResponse], error) {
	id, err := uuid.Parse(req.Msg.Id)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	if err := s.repo.SetUserStatus(ctx, id, "disabled"); err != nil {
		return nil, err
	}
	if err := s.repo.IncrementTokenVersion(ctx, id); err != nil {
		s.log.Warn("admin: disable user — increment token version failed",
			zap.String("target", id.String()), zap.Error(err))
	}
	s.log.Info("admin: user disabled",
		zap.String("actor", getActorID(ctx).String()),
		zap.String("target", id.String()))
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
	s.log.Info("admin: user enabled",
		zap.String("actor", getActorID(ctx).String()),
		zap.String("target", id.String()))
	return connect.NewResponse(&antv1.EnableUserResponse{}), nil
}

func (s *AdminUserServer) ResetUserPassword(ctx context.Context, req *connect.Request[antv1.ResetUserPasswordRequest]) (*connect.Response[antv1.ResetUserPasswordResponse], error) {
	id, err := uuid.Parse(req.Msg.Id)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}

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

	if _, err := s.resetRepo.CreateResetToken(ctx, id); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	return connect.NewResponse(&antv1.ResetUserPasswordResponse{}), nil
}

package admin

import (
	"context"
	"fmt"
	"strings"

	"go.uber.org/zap"
	"google.golang.org/protobuf/types/known/timestamppb"

	"connectrpc.com/connect"
	"github.com/google/uuid"

	antv1 "anttrader/gen/proto/ant/v1"
	antv1c "anttrader/gen/proto/ant/v1/antv1connect"
	antinterceptor "anttrader/internal/interceptor"
	"anttrader/internal/model"
	"anttrader/internal/pkg/hash"
	"anttrader/internal/repository"
	"anttrader/internal/service"
)

type AdminUserServer struct {
	repo      *repository.AdminRepository
	resetRepo *repository.PasswordResetRepo
	walletSvc *service.WalletService
	acctSvc   *service.AccountNumberService
	log       *zap.Logger
}

var _ antv1c.AdminUserServiceHandler = (*AdminUserServer)(nil)

func NewAdminUserServer(repo *repository.AdminRepository, resetRepo *repository.PasswordResetRepo, walletSvc *service.WalletService, acctSvc *service.AccountNumberService, log *zap.Logger) *AdminUserServer {
	return &AdminUserServer{repo: repo, resetRepo: resetRepo, walletSvc: walletSvc, acctSvc: acctSvc, log: log}
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
			TodayVolume:    stats.TodayVolume.InexactFloat64(),
			TodayProfit:    stats.TodayProfit.InexactFloat64(),
			SystemLoad:     stats.SystemLoad.InexactFloat64(),
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
	// Account number: admin-specified or auto-generated.
	if acctNum := req.Msg.AccountNumber; acctNum != "" {
		if err := service.ValidateAccountNumber(acctNum); err != nil {
			return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("invalid account_number: %w", err))
		}
		if s.acctSvc != nil {
			avail, err := s.acctSvc.IsAccountNumberAvailable(ctx, acctNum)
			if err != nil {
				return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("check account_number availability: %w", err))
			}
			if !avail {
				return nil, connect.NewError(connect.CodeAlreadyExists, fmt.Errorf("account number %s is already taken", acctNum))
			}
		}
		user.AccountNumber = &acctNum
	} else if s.acctSvc != nil {
		// Auto-generate when admin doesn't specify one.
		num, err := s.acctSvc.GenerateAccountNumber(ctx)
		if err != nil {
			s.log.Warn("admin: auto-generate account number failed, continuing without one", zap.Error(err))
		} else {
			user.AccountNumber = &num
		}
	}
	hashed, err := hash.HashPassword(password)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("hash password: %w", err))
	}
	user.PasswordHash = hashed

	// Try INSERT with retry on account_number unique violation (race window between
	// GenerateAccountNumber SELECT and this INSERT — the UNIQUE constraint is the ultimate guard).
	const maxRetries = 3
	for attempt := 0; ; attempt++ {
		err := s.repo.CreateUser(ctx, user)
		if err == nil {
			break
		}
		if service.IsAccountNumberViolation(err) {
			if attempt >= maxRetries {
				return nil, connect.NewError(connect.CodeAlreadyExists, fmt.Errorf("account number collision after %d retries", maxRetries))
			}
			// Regenerate account_number and retry.
			if s.acctSvc != nil {
				num, genErr := s.acctSvc.GenerateAccountNumber(ctx)
				if genErr != nil {
					return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("regenerate account number: %w", genErr))
				}
				user.AccountNumber = &num
			}
			continue
		}
		if service.IsUniqueViolation(err) {
			// Non-account_number unique violation → duplicate email.
			return nil, connect.NewError(connect.CodeAlreadyExists, fmt.Errorf("email already registered"))
		}
		return nil, err
	}
	// Auto-create wallet for admin-created users (best-effort).
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
		// Validate role against allowlist.
		if !validRole(req.Msg.Role) {
			return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("invalid role: %s", req.Msg.Role))
		}
		existing.Role = req.Msg.Role
	}
	if req.Msg.Status != "" {
		existing.Status = req.Msg.Status
	}
	if req.Msg.Nickname != "" {
		existing.Nickname = &req.Msg.Nickname
	}
	// Account number: validate, check uniqueness (excluding self), then update.
	if req.Msg.AccountNumber != "" {
		if err := service.ValidateAccountNumber(req.Msg.AccountNumber); err != nil {
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
	// Prevent self-deletion.
	actorID := getActorID(ctx)
	if actorID != uuid.Nil && actorID == id {
		return nil, connect.NewError(connect.CodePermissionDenied, fmt.Errorf("cannot delete yourself"))
	}
	// Prevent deleting the last admin.
	adminCount, err := s.repo.CountAdmins(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	target, err := s.repo.GetUserByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if target != nil && target.Role == "admin" && adminCount <= 1 {
		return nil, connect.NewError(connect.CodeFailedPrecondition, fmt.Errorf("cannot delete the last admin"))
	}

	// Record audit snapshot before soft-delete.
	affected, _ := s.repo.GetAffectedTableCounts(ctx, id)
	recordAuditLog(ctx, s.repo, actorID, "delete_user", id.String(), target.Email, affected)

	if err := s.repo.DeleteUser(ctx, id); err != nil {
		s.log.Error("admin: delete user failed", zap.Error(err), zap.String("target", id.String()))
		return nil, err
	}
	s.log.Info("admin: user deleted",
		zap.String("actor", getActorID(ctx).String()),
		zap.String("target", id.String()))
	return connect.NewResponse(&antv1.DeleteUserResponse{}), nil
}

// DeleteUsers implements the batch-delete RPC for admin user management.
func (s *AdminUserServer) DeleteUsers(ctx context.Context, req *connect.Request[antv1.DeleteUsersRequest]) (*connect.Response[antv1.DeleteUsersResponse], error) {
	actorID := getActorID(ctx)
	ids := req.Msg.Ids
	if len(ids) == 0 {
		return connect.NewResponse(&antv1.DeleteUsersResponse{}), nil
	}
	adminCount, err := s.repo.CountAdmins(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	uuids := make([]uuid.UUID, 0, len(ids))
	type targetInfo struct{ id uuid.UUID; email string }
	var targets []targetInfo
	var errors []string
	for _, id := range ids {
		uid, err := uuid.Parse(id)
		if err != nil {
			errors = append(errors, fmt.Sprintf("%s: invalid id", id))
			continue
		}
		// Prevent self-deletion.
		if actorID != uuid.Nil && actorID == uid {
			errors = append(errors, fmt.Sprintf("%s: cannot delete yourself", id))
			continue
		}
		// Prevent deleting the last admin.
		target, err := s.repo.GetUserByID(ctx, uid)
		if err != nil {
			errors = append(errors, fmt.Sprintf("%s: %v", id, err))
			continue
		}
		if target != nil && target.Role == "admin" && adminCount <= 1 {
			errors = append(errors, fmt.Sprintf("%s: cannot delete the last admin", id))
			continue
		}
		uuids = append(uuids, uid)
		targets = append(targets, targetInfo{id: uid, email: target.Email})
	}

	// Record audit logs before soft-delete.
	for _, t := range targets {
		affected, _ := s.repo.GetAffectedTableCounts(ctx, t.id)
		recordAuditLog(ctx, s.repo, actorID, "batch_delete", t.id.String(), t.email, affected)
	}

	var deleted int64
	if len(uuids) > 0 {
		var err error
		deleted, err = s.repo.DeleteUsers(ctx, uuids)
		if err != nil {
			s.log.Error("admin: batch delete users failed", zap.Error(err), zap.Int("count", len(uuids)))
			return nil, connect.NewError(connect.CodeInternal, err)
		}
	}
	s.log.Info("admin: batch delete users",
		zap.String("actor", actorID.String()),
		zap.Int64("deleted", deleted),
		zap.Int("failed", len(errors)))
	return connect.NewResponse(&antv1.DeleteUsersResponse{
		DeletedCount: int32(deleted),
		FailedCount:  int32(len(errors)),
		Errors:       errors,
	}), nil
}

// RestoreUser clears the soft-delete marker on a previously deleted user.
func (s *AdminUserServer) RestoreUser(ctx context.Context, req *connect.Request[antv1.RestoreUserRequest]) (*connect.Response[antv1.RestoreUserResponse], error) {
	id, err := uuid.Parse(req.Msg.Id)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	if err := s.repo.RestoreUser(ctx, id); err != nil {
		if err == repository.ErrUserNotFound {
			return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("user not found or not deleted"))
		}
		s.log.Error("admin: restore user failed", zap.Error(err), zap.String("target", id.String()))
		return nil, err
	}
	actorID := getActorID(ctx)
	s.log.Info("admin: user restored",
		zap.String("actor", actorID.String()),
		zap.String("target", id.String()))
	// Best-effort audit log.
	recordAuditLog(ctx, s.repo, actorID, "restore_user", id.String(), id.String(), nil)
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

// validRole returns true if the role string is a recognized user role.
func validRole(role string) bool {
	switch role {
	case "user", "admin", "super_admin", "operation", "audit", "customer_service":
		return true
	}
	return false
}

// recordAuditLog writes an admin action to the audit log. Best-effort:
// log errors are emitted but not returned — audit failure must not block the
// primary operation (delete / restore).
func recordAuditLog(ctx context.Context, repo *repository.AdminRepository, actorID uuid.UUID, action, targetID, targetEmail string, affected map[string]int64) {
	if err := repo.InsertAuditLog(ctx, actorID, action, targetID, targetEmail, affected); err != nil {
		// audit failure must not block the operation; the zap logger is
		// the fallback trace.
	}
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

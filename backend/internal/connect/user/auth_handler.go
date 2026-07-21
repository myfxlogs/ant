package user

import (
	"context"
	"fmt"

	"go.uber.org/zap"

	"connectrpc.com/connect"
	"github.com/google/uuid"
	"google.golang.org/protobuf/types/known/emptypb"

	antv1 "alphaforge/gen/proto/ant/v1"
	antv1c "alphaforge/gen/proto/ant/v1/antv1connect"
	"alphaforge/internal/interceptor"
	"alphaforge/internal/notifier"
	"alphaforge/internal/repository"
	"alphaforge/internal/service"
)

// AuthServer implements ant.v1.AuthServiceHandler.
type AuthServer struct {
	users             *repository.UserRepository
	registrationSvc   *service.RegistrationService
	emailVerifSvc     *service.EmailVerificationService
	passwordResetRepo *repository.PasswordResetRepo
	emailNotifier     *notifier.EmailNotifier
	appURL            string // base URL for reset links, e.g. "https://alfq.org"
	requireEmailVerif bool
	jwtSecret         string
	log               *zap.Logger
	insecure          bool
}

var _ antv1c.AuthServiceHandler = (*AuthServer)(nil)

func NewAuthServer(users *repository.UserRepository, jwtSecret string, log *zap.Logger) *AuthServer {
	return &AuthServer{users: users, jwtSecret: jwtSecret, log: log}
}

// WithRegistration wires the RegistrationService for user registration.
func (s *AuthServer) WithRegistration(regSvc *service.RegistrationService) *AuthServer {
	s.registrationSvc = regSvc
	return s
}

// WithEmailVerification wires the EmailVerificationService for email verification.
func (s *AuthServer) WithEmailVerification(evSvc *service.EmailVerificationService) *AuthServer {
	s.emailVerifSvc = evSvc
	return s
}

// SetRequireEmailVerification blocks login for unverified users.
func (s *AuthServer) SetRequireEmailVerification(v bool) { s.requireEmailVerif = v }

// SetInsecureCookies disables the Secure flag on refresh_token cookies for
// local/dev deployments without TLS.
func (s *AuthServer) SetInsecureCookies(v bool) { s.insecure = v }

// WithPasswordReset wires the PasswordResetRepo and EmailNotifier for self-service password reset.
func (s *AuthServer) WithPasswordReset(repo *repository.PasswordResetRepo, email *notifier.EmailNotifier, appURL string) *AuthServer {
	s.passwordResetRepo = repo
	s.emailNotifier = email
	s.appURL = appURL
	return s
}

// ForgotPassword sends a password reset email. Always returns success to prevent user enumeration.
func (s *AuthServer) ForgotPassword(ctx context.Context, req *connect.Request[antv1.ForgotPasswordRequest]) (*connect.Response[antv1.ForgotPasswordResponse], error) {
	email := req.Msg.Email
	if email == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("email required"))
	}
	if s.passwordResetRepo == nil || s.emailNotifier == nil {
		return connect.NewResponse(&antv1.ForgotPasswordResponse{Success: false, Message: "Password reset is not configured"}), nil
	}

	user, err := s.users.GetByEmail(ctx, email)
	if err != nil || user == nil {
		// User not found — return success anyway to prevent enumeration.
		return connect.NewResponse(&antv1.ForgotPasswordResponse{Success: true, Message: "If the email exists, a reset link has been sent"}), nil
	}

	token, err := s.passwordResetRepo.CreateResetToken(ctx, user.ID)
	if err != nil {
		s.log.Error("ForgotPassword: create token", zap.Error(err))
		return connect.NewResponse(&antv1.ForgotPasswordResponse{Success: true, Message: "If the email exists, a reset link has been sent"}), nil
	}

	resetURL := fmt.Sprintf("%s/reset-password?token=%s", s.appURL, token)
	body := fmt.Sprintf("You requested a password reset for your AlphaForge account.\n\nClick the link below to reset your password:\n%s\n\nThis link expires in 24 hours.\n\nIf you did not request this, ignore this email.", resetURL)
	if err := s.emailNotifier.SendTo(email, "AlphaForge Password Reset", body); err != nil {
		s.log.Error("ForgotPassword: send email", zap.Error(err))
	}

	return connect.NewResponse(&antv1.ForgotPasswordResponse{Success: true, Message: "If the email exists, a reset link has been sent"}), nil
}

// ResetPassword validates a reset token and sets a new password.
func (s *AuthServer) ResetPassword(ctx context.Context, req *connect.Request[antv1.ResetPasswordRequest]) (*connect.Response[antv1.ResetPasswordResponse], error) {
	token := req.Msg.Token
	newPassword := req.Msg.NewPassword
	if token == "" || newPassword == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("token and new_password required"))
	}
	if len(newPassword) < 8 {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("password must be at least 8 characters"))
	}
	if s.passwordResetRepo == nil {
		return nil, connect.NewError(connect.CodeUnavailable, fmt.Errorf("password reset is not configured"))
	}

	userID, err := s.passwordResetRepo.ValidateResetToken(ctx, token)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("invalid or expired reset token"))
	}

	passwordHash, err := repository.HashPassword(newPassword)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to hash password"))
	}
	if err := s.users.UpdatePassword(ctx, userID, passwordHash); err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to update password"))
	}
	// Invalidate all existing sessions for this user.
	if err := s.users.IncrementTokenVersion(ctx, userID); err != nil {
		s.log.Warn("ResetPassword: increment token version failed", zap.Error(err))
	}
	if err := s.passwordResetRepo.ConsumeResetToken(ctx, token); err != nil {
		s.log.Warn("ResetPassword: consume token failed", zap.Error(err))
	}

	return connect.NewResponse(&antv1.ResetPasswordResponse{Success: true, Message: "Password has been reset"}), nil
}

// Login implements the ConnectRPC handler for user authentication.
// Accepts login (preferred) or email (backward compat) fields.
// Supports email OR account_number as login identifier in a single query.
func (s *AuthServer) Login(ctx context.Context, req *connect.Request[antv1.LoginRequest]) (*connect.Response[antv1.LoginResponse], error) {
	m := req.Msg
	login := m.Login
	if login == "" {
		login = m.Email // backward compat
	}
	if login == "" {
		return nil, connect.NewError(connect.CodeUnauthenticated, fmt.Errorf("login or email is required"))
	}
	user, err := s.users.GetByLogin(ctx, login)
	if err != nil {
		return nil, connect.NewError(connect.CodeUnauthenticated, fmt.Errorf("invalid credentials"))
	}
	if user.Status != "active" {
		return nil, connect.NewError(connect.CodePermissionDenied, fmt.Errorf("account is disabled"))
	}
	if s.requireEmailVerif && user.EmailVerifiedAt == nil {
		return nil, connect.NewError(connect.CodePermissionDenied, fmt.Errorf("email not verified — please check your inbox for the verification link"))
	}
	if !service.VerifyPassword(user.PasswordHash, m.Password) {
		return nil, connect.NewError(connect.CodeUnauthenticated, fmt.Errorf("invalid credentials"))
	}
	// Update last login timestamp (best-effort).
	if err := s.users.UpdateLastLogin(ctx, user.ID); err != nil {
		s.log.Warn("Login: update last login failed", zap.String("userID", user.ID.String()), zap.Error(err))
	}
	tokenEmail := user.Email
	accessToken, err := s.issueAccessToken(user.ID.String(), tokenEmail)
	if err != nil {
		s.log.Error("Login: issue access token", zap.Error(err))
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	refreshToken, err := s.issueRefreshToken(user.ID.String(), tokenEmail, user.TokenVersion)
	if err != nil {
		s.log.Error("Login: issue refresh token", zap.Error(err))
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	nickname := user.Email
	if user.Nickname != nil && *user.Nickname != "" {
		nickname = *user.Nickname
	}
	capTier, perms, _ := s.users.GetCapabilities(ctx, user.ID, user.Role)
	var acctNum string
	if user.AccountNumber != nil {
		acctNum = *user.AccountNumber
	}
	resp := connect.NewResponse(&antv1.LoginResponse{
		AccessToken: accessToken,
		User: &antv1.User{
			Id: user.ID.String(), Email: user.Email, Username: nickname, Role: user.Role,
			Permissions: perms, CapabilityTier: int32(capTier), AccountNumber: acctNum,
			EmailVerified: user.EmailVerifiedAt != nil,
		},
	})
	resp.Header().Set("Set-Cookie", s.makeRefreshCookie(refreshToken))
	return resp, nil
}

// Logout implements the ConnectRPC handler for user logout.
func (s *AuthServer) Logout(ctx context.Context, req *connect.Request[emptypb.Empty]) (*connect.Response[emptypb.Empty], error) {
	userID := interceptor.GetUserID(ctx)
	if userID != "" {
		if uid, err := uuid.Parse(userID); err == nil {
			if err := s.users.IncrementTokenVersion(ctx, uid); err != nil {
				s.log.Warn("Logout: increment token version failed", zap.Error(err))
			}
		}
	}
	resp := connect.NewResponse(&emptypb.Empty{})
	resp.Header().Set("Set-Cookie", s.clearRefreshCookie())
	return resp, nil
}

// GetMe implements the ConnectRPC handler for fetching the current user.
func (s *AuthServer) GetMe(ctx context.Context, req *connect.Request[emptypb.Empty]) (*connect.Response[antv1.GetMeResponse], error) {
	userID := interceptor.GetUserID(ctx)
	if userID == "" {
		return nil, connect.NewError(connect.CodeUnauthenticated, fmt.Errorf("not authenticated"))
	}
	uid, err := uuid.Parse(userID)
	if err != nil {
		return nil, connect.NewError(connect.CodeUnauthenticated, fmt.Errorf("invalid user id"))
	}
	user, err := s.users.GetByID(ctx, uid)
	if err != nil {
		return nil, connect.NewError(connect.CodeUnauthenticated, fmt.Errorf("user not found"))
	}
	nickname := user.Email
	if user.Nickname != nil && *user.Nickname != "" {
		nickname = *user.Nickname
	}
	capTier, perms, _ := s.users.GetCapabilities(ctx, uid, user.Role)
	var acctNum string
	if user.AccountNumber != nil {
		acctNum = *user.AccountNumber
	}
	return connect.NewResponse(&antv1.GetMeResponse{
		User: &antv1.User{
			Id: userID, Email: user.Email, Username: nickname, Role: user.Role,
			Permissions: perms, CapabilityTier: int32(capTier), AccountNumber: acctNum,
			EmailVerified: user.EmailVerifiedAt != nil,
		},
	}), nil
}

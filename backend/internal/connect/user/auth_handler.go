package user

import (
	"context"
	"fmt"

	"go.uber.org/zap"

	"connectrpc.com/connect"
	"github.com/google/uuid"
	"google.golang.org/protobuf/types/known/emptypb"

	antv1 "anttrader/gen/proto/ant/v1"
	antv1c "anttrader/gen/proto/ant/v1/antv1connect"
	"anttrader/internal/interceptor"
	"anttrader/internal/repository"
	"anttrader/internal/service"
)

// AuthServer implements ant.v1.AuthServiceHandler.
// Handles authentication (login, logout, refresh). Registration is delegated to RegistrationService.
type AuthServer struct {
	users           *repository.UserRepository
	registrationSvc *service.RegistrationService // nil if registration is not wired
	jwtSecret       string
	log             *zap.Logger
	insecure        bool // disables Secure cookie flag for non-TLS dev deployments
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

// SetInsecureCookies disables the Secure flag on refresh_token cookies for
// local/dev deployments without TLS.
func (s *AuthServer) SetInsecureCookies(v bool) { s.insecure = v }

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
	refreshToken, err := s.issueRefreshToken(user.ID.String(), tokenEmail)
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
		},
	})
	resp.Header().Set("Set-Cookie", s.makeRefreshCookie(refreshToken))
	return resp, nil
}

// Logout implements the ConnectRPC handler for user logout.
func (s *AuthServer) Logout(ctx context.Context, req *connect.Request[emptypb.Empty]) (*connect.Response[emptypb.Empty], error) {
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
		},
	}), nil
}

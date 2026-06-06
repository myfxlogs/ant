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
)

// AuthServer implements ant.v1.AuthServiceHandler.
type AuthServer struct {
	users     *repository.UserRepository
	jwtSecret string
	log       *zap.Logger
	insecure  bool // disables Secure cookie flag for non-TLS dev deployments
}

var _ antv1c.AuthServiceHandler = (*AuthServer)(nil)

func NewAuthServer(users *repository.UserRepository, jwtSecret string, log *zap.Logger) *AuthServer {
	return &AuthServer{users: users, jwtSecret: jwtSecret, log: log}
}

// SetInsecureCookies disables the Secure flag on refresh_token cookies for
// local/dev deployments without TLS.
func (s *AuthServer) SetInsecureCookies(v bool) { s.insecure = v }

// Login implements the ConnectRPC handler for user authentication.
func (s *AuthServer) Login(ctx context.Context, req *connect.Request[antv1.LoginRequest]) (*connect.Response[antv1.LoginResponse], error) {
	m := req.Msg
	user, err := s.users.GetByEmail(ctx, m.Email)
	if err != nil {
		return nil, connect.NewError(connect.CodeUnauthenticated, fmt.Errorf("invalid credentials"))
	}
	if !verifyArgon2id(user.PasswordHash, m.Password) {
		return nil, connect.NewError(connect.CodeUnauthenticated, fmt.Errorf("invalid credentials"))
	}
	accessToken, err := s.issueAccessToken(user.ID.String(), m.Email)
	if err != nil {
		s.log.Error("Login: issue access token", zap.Error(err))
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	refreshToken, err := s.issueRefreshToken(user.ID.String(), m.Email)
	if err != nil {
		s.log.Error("Login: issue refresh token", zap.Error(err))
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	nickname := m.Email
	if user.Nickname != nil && *user.Nickname != "" {
		nickname = *user.Nickname
	}
	capTier, perms, _ := s.users.GetCapabilities(ctx, user.ID, user.Role)
	resp := connect.NewResponse(&antv1.LoginResponse{
		AccessToken: accessToken,
		User: &antv1.User{
			Id: user.ID.String(), Email: user.Email, Username: nickname, Role: user.Role,
			Permissions: perms, CapabilityTier: int32(capTier),
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
	return connect.NewResponse(&antv1.GetMeResponse{
		User: &antv1.User{
			Id: userID, Email: user.Email, Username: nickname, Role: user.Role,
			Permissions: perms, CapabilityTier: int32(capTier),
		},
	}), nil
}

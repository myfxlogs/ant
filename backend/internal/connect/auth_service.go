package connect

import (
	"context"
	"errors"

	"connectrpc.com/connect"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/timestamppb"

	v1 "anttrader/gen/proto"
	"anttrader/internal/interceptor"
	"anttrader/internal/model"
	"anttrader/internal/pkg/hash"
	"anttrader/internal/repository"
)

type AuthService struct {
	userRepo  *repository.UserRepository
	jwtSecret string
}

func NewAuthService(userRepo *repository.UserRepository, jwtSecret string) *AuthService {
	return &AuthService{
		userRepo:  userRepo,
		jwtSecret: jwtSecret,
	}
}

func (s *AuthService) Login(ctx context.Context, req *connect.Request[v1.LoginRequest]) (*connect.Response[v1.LoginResponse], error) {
	user, err := s.userRepo.GetByEmail(ctx, req.Msg.Email)
	if err != nil {
		return nil, connect.NewError(connect.CodeUnauthenticated, errors.New("errors.invalid_credentials"))
	}

	valid, err := hash.VerifyPassword(req.Msg.Password, user.PasswordHash)
	if err != nil {
		return nil, connect.NewError(connect.CodeUnauthenticated, errors.New("errors.invalid_credentials"))
	}
	if !valid {
		return nil, connect.NewError(connect.CodeUnauthenticated, errors.New("errors.invalid_credentials"))
	}

	token, err := s.generateToken(user.ID)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	return connect.NewResponse(&v1.LoginResponse{
		AccessToken: token,
		User: &v1.User{
			Id:       user.ID.String(),
			Username: user.Email,
			Email:    user.Email,
			Role:     user.Role,
		},
	}), nil
}

func (s *AuthService) Logout(ctx context.Context, req *connect.Request[emptypb.Empty]) (*connect.Response[emptypb.Empty], error) {
	return connect.NewResponse(&emptypb.Empty{}), nil
}

func (s *AuthService) RefreshToken(ctx context.Context, req *connect.Request[v1.RefreshTokenRequest]) (*connect.Response[v1.RefreshTokenResponse], error) {
	claims, err := interceptor.ValidateToken(req.Msg.RefreshToken, s.jwtSecret)
	if err != nil {
		return nil, connect.NewError(connect.CodeUnauthenticated, err)
	}

	userID, err := uuid.Parse(claims.UserID)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	token, err := s.generateToken(userID)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	return connect.NewResponse(&v1.RefreshTokenResponse{
		AccessToken: token,
	}), nil
}

func (s *AuthService) GetMe(ctx context.Context, req *connect.Request[emptypb.Empty]) (*connect.Response[v1.GetMeResponse], error) {
	userID := interceptor.GetUserID(ctx)
	if userID == "" {
		return nil, connect.NewError(connect.CodeUnauthenticated, errors.New("errors.not_authenticated"))
	}

	uid, err := uuid.Parse(userID)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	user, err := s.userRepo.GetByID(ctx, uid)
	if err != nil {
		return nil, connect.NewError(connect.CodeNotFound, errors.New("errors.user_not_found"))
	}

	return connect.NewResponse(&v1.GetMeResponse{
		User: &v1.User{
			Id:        user.ID.String(),
			Username:  user.Email,
			Email:     user.Email,
			Role:      user.Role,
			CreatedAt: timestamppb.New(user.CreatedAt),
			UpdatedAt: timestamppb.New(user.UpdatedAt),
		},
	}), nil
}

func (s *AuthService) Register(ctx context.Context, req *connect.Request[v1.RegisterRequest]) (*connect.Response[v1.RegisterResponse], error) {
	exists, err := s.userRepo.ExistsByEmail(ctx, req.Msg.Email)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	if exists {
		return nil, connect.NewError(connect.CodeAlreadyExists, errors.New("errors.email_already_registered"))
	}

	hashedPassword, err := hash.HashPassword(req.Msg.Password)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	user := &model.User{
		Email:        req.Msg.Email,
		PasswordHash: hashedPassword,
		Role:         "user",
		Status:       "active",
	}

	if err := s.userRepo.Create(ctx, user); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	return connect.NewResponse(&v1.RegisterResponse{
		User: &v1.User{
			Id:       user.ID.String(),
			Username: user.Email,
			Email:    user.Email,
			Role:     user.Role,
		},
	}), nil
}

func (s *AuthService) generateToken(userID uuid.UUID) (string, error) {
	claims := &interceptor.JWTClaims{
		UserID: userID.String(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(s.jwtSecret))
}

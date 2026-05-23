package service

import (
	"context"
	"errors"

	"github.com/google/uuid"

	"anttrader/internal/model"
	apperrors "anttrader/internal/pkg/errors"
	"anttrader/internal/pkg/hash"
	"anttrader/internal/pkg/jwt"
	"anttrader/internal/repository"
	"anttrader/pkg/logger"

	"go.uber.org/zap"
)

type AuthService struct {
	userRepo *repository.UserRepository
}

func NewAuthService(userRepo *repository.UserRepository) *AuthService {
	return &AuthService{userRepo: userRepo}
}

type RegisterRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required,min=8"`
	Nickname string `json:"nickname"`
}

type LoginRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

type LoginResponse struct {
	User         *model.User `json:"user"`
	AccessToken  string      `json:"access_token"`
	RefreshToken string      `json:"refresh_token"`
	ExpiresIn    int64       `json:"expires_in"`
}

type ChangePasswordRequest struct {
	OldPassword string `json:"old_password" binding:"required"`
	NewPassword string `json:"new_password" binding:"required,min=8"`
}

func (s *AuthService) Register(ctx context.Context, req *RegisterRequest) (*model.User, error) {
	exists, err := s.userRepo.ExistsByEmail(ctx, req.Email)
	if err != nil {
		logger.Error("Failed to check if user exists", zap.String("email", req.Email), zap.Error(err))
		return nil, apperrors.New(apperrors.InternalError, err)
	}
	if exists {
		logger.Warn("User already exists", zap.String("email", req.Email))
		return nil, apperrors.New(apperrors.UserAlreadyExists, nil)
	}

	passwordHash, err := hash.HashPassword(req.Password)
	if err != nil {
		logger.Error("Failed to hash password", zap.String("email", req.Email), zap.Error(err))
		return nil, apperrors.New(apperrors.InternalError, err)
	}

	nickname := req.Nickname
	if nickname == "" {
		nickname = req.Email
	}

	user := &model.User{
		Email:        req.Email,
		PasswordHash: passwordHash,
		Nickname:     &nickname,
		Role:         "user",
		Status:       "active",
	}

	if err := s.userRepo.Create(ctx, user); err != nil {
		logger.Error("Failed to create user", zap.String("email", req.Email), zap.Error(err))
		return nil, apperrors.New(apperrors.InternalError, err)
	}

	// Clear password hash before returning
	user.PasswordHash = ""
	return user, nil
}

func (s *AuthService) Login(ctx context.Context, req *LoginRequest) (*LoginResponse, error) {
	user, err := s.userRepo.GetByEmail(ctx, req.Email)
	if err != nil {
		logger.Warn("User not found", zap.String("email", req.Email))
		return nil, apperrors.New(apperrors.UserNotFound, err)
	}

	if user.Status != "active" {
		logger.Warn("User account disabled", zap.String("email", req.Email), zap.String("status", user.Status))
		return nil, apperrors.New(apperrors.UserDisabled, nil)
	}

	valid, err := hash.VerifyPassword(req.Password, user.PasswordHash)
	if err != nil {
		logger.Error("Failed to verify password", zap.String("email", req.Email), zap.Error(err))
		return nil, apperrors.New(apperrors.InternalError, err)
	}
	if !valid {
		logger.Warn("Invalid password attempt", zap.String("email", req.Email))
		return nil, apperrors.New(apperrors.InvalidPassword, nil)
	}

	tokenPair, err := jwt.GenerateTokenPair(user.ID.String(), user.Email, user.Role)
	if err != nil {
		logger.Error("Failed to generate tokens", zap.String("email", req.Email), zap.Error(err))
		return nil, apperrors.New(apperrors.InternalError, err)
	}

	// Update last login time
	if err := s.userRepo.UpdateLastLogin(ctx, user.ID); err != nil {
		logger.Warn("Failed to update last login time", zap.String("user_id", user.ID.String()), zap.Error(err))
	}

	// Clear password hash before returning
	user.PasswordHash = ""
	
	return &LoginResponse{
		User:         user,
		AccessToken:  tokenPair.AccessToken,
		RefreshToken: tokenPair.RefreshToken,
		ExpiresIn:    tokenPair.ExpiresIn,
	}, nil
}

func (s *AuthService) GetUserByID(ctx context.Context, id uuid.UUID) (*model.User, error) {
	user, err := s.userRepo.GetByID(ctx, id)
	if err != nil {
		return nil, apperrors.New(apperrors.UserNotFound, err)
	}
	return user, nil
}

func (s *AuthService) RefreshToken(ctx context.Context, refreshToken string) (*jwt.TokenPair, error) {
	claims, err := jwt.ValidateToken(refreshToken)
	if err != nil {
		return nil, apperrors.New(apperrors.TokenInvalid, err)
	}

	userID, err := uuid.Parse(claims.UserID)
	if err != nil {
		return nil, apperrors.New(apperrors.TokenInvalid, err)
	}

	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return nil, apperrors.New(apperrors.UserNotFound, err)
	}

	if user.Status != "active" {
		return nil, apperrors.New(apperrors.UserDisabled, nil)
	}

	return jwt.GenerateTokenPair(user.ID.String(), user.Email, user.Role)
}

func (s *AuthService) ChangePassword(ctx context.Context, userID uuid.UUID, req *ChangePasswordRequest) error {
	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return apperrors.New(apperrors.UserNotFound, err)
	}

	valid, err := hash.VerifyPassword(req.OldPassword, user.PasswordHash)
	if err != nil {
		return apperrors.New(apperrors.InternalError, err)
	}
	if !valid {
		return apperrors.New(apperrors.OldPasswordIncorrect, nil)
	}

	passwordHash, err := hash.HashPassword(req.NewPassword)
	if err != nil {
		return apperrors.New(apperrors.InternalError, err)
	}

	return s.userRepo.UpdatePassword(ctx, userID, passwordHash)
}

func (s *AuthService) ValidateToken(tokenString string) (*jwt.Claims, error) {
	claims, err := jwt.ValidateToken(tokenString)
	if err != nil {
		if errors.Is(err, jwt.ErrTokenExpired) {
			return nil, apperrors.New(apperrors.TokenExpired, err)
		}
		return nil, apperrors.New(apperrors.TokenInvalid, err)
	}
	return claims, nil
}

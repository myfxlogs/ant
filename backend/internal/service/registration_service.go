package service

import (
	"context"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"anttrader/internal/model"
	"anttrader/internal/pkg/hash"
)

// RegistrationService orchestrates the user registration flow:
// 1. Create user record
// 2. Auto-assign account number
// 3. Create wallet
//
// Keeps AuthServer focused on authentication and avoids mixing concerns.
type RegistrationService struct {
	users           UserCreator
	accountNumber   AccountNumberAssigner
	wallet          WalletCreator
	log             *zap.Logger
}

// UserCreator is the subset of UserRepository needed during registration.
type UserCreator interface {
	Create(ctx context.Context, user *model.User) error
	ExistsByEmail(ctx context.Context, email string) (bool, error)
}

// AccountNumberAssigner is the subset of AccountNumberService needed.
type AccountNumberAssigner interface {
	GenerateAccountNumber(ctx context.Context) (string, error)
	SetAccountNumber(ctx context.Context, userID, num string) error
}

// WalletCreator is the subset of WalletService needed.
type WalletCreator interface {
	CreateWallet(ctx context.Context, userID uuid.UUID) (*model.Wallet, error)
}

func NewRegistrationService(users UserCreator, acctSvc AccountNumberAssigner, wallet WalletCreator, log *zap.Logger) *RegistrationService {
	return &RegistrationService{users: users, accountNumber: acctSvc, wallet: wallet, log: log}
}

// RegisterUser creates a user, assigns an account number, and creates a wallet.
// Returns the created user and the assigned account number (empty if assignment failed).
func (s *RegistrationService) RegisterUser(ctx context.Context, email, password, nickname string) (*model.User, string, error) {
	exists, err := s.users.ExistsByEmail(ctx, email)
	if err != nil {
		return nil, "", err
	}
	if exists {
		return nil, "", ErrEmailAlreadyRegistered
	}

	passwordHash, err := hash.HashPassword(password)
	if err != nil {
		return nil, "", err
	}

	if nickname == "" {
		nickname = email
	}
	user := &model.User{
		Email:        email,
		PasswordHash: passwordHash,
		Nickname:     &nickname,
		Role:         "user",
		Status:       "active",
	}
	if err := s.users.Create(ctx, user); err != nil {
		return nil, "", err
	}

	// Auto-assign account number (best-effort; registration succeeds even if pool is exhausted).
	var acctNum string
	if s.accountNumber != nil {
		num, err := s.accountNumber.GenerateAccountNumber(ctx)
		if err != nil {
			s.log.Warn("RegisterUser: auto-assign account number failed (pool exhausted?)",
				zap.String("userID", user.ID.String()), zap.Error(err))
		} else {
			if err := s.accountNumber.SetAccountNumber(ctx, user.ID.String(), num); err != nil {
				s.log.Warn("RegisterUser: set account number failed",
					zap.String("userID", user.ID.String()), zap.Error(err))
			} else {
				acctNum = num
			}
		}
	}

	// Create wallet (best-effort).
	if s.wallet != nil {
		if _, err := s.wallet.CreateWallet(ctx, user.ID); err != nil {
			s.log.Warn("RegisterUser: create wallet failed",
				zap.String("userID", user.ID.String()), zap.Error(err))
		}
	}

	return user, acctNum, nil
}

var ErrEmailAlreadyRegistered = &errEmailAlreadyRegistered{}

type errEmailAlreadyRegistered struct{}

func (e *errEmailAlreadyRegistered) Error() string { return "email already registered" }

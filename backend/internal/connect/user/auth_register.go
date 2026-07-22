package user

import (
	"context"
	"errors"
	"fmt"

	"connectrpc.com/connect"
	"go.uber.org/zap"

	antv1 "alphaforge/gen/proto/ant/v1"
	"alphaforge/internal/service"
)

// Register implements the ConnectRPC handler for user registration.
// Delegates to RegistrationService for the full flow: create user → assign account number → create wallet.
func (s *AuthServer) Register(ctx context.Context, req *connect.Request[antv1.RegisterRequest]) (*connect.Response[antv1.RegisterResponse], error) {
	m := req.Msg
	username := m.Username
	if username == "" {
		username = m.Email
	}
	if s.registrationSvc == nil {
		s.log.Error("Register: RegistrationService not wired")
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("registration not available"))
	}
	user, acctNum, err := s.registrationSvc.RegisterUser(ctx, m.Email, m.Password, username)
	if err != nil {
		if errors.Is(err, service.ErrEmailAlreadyRegistered) {
			return nil, connect.NewError(connect.CodeAlreadyExists, fmt.Errorf("email already registered"))
		}
		s.log.Error("Register: registration failed", zap.Error(err))
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&antv1.RegisterResponse{
		User:                  &antv1.User{Id: user.ID.String(), Email: m.Email, AccountNumber: acctNum},
		EmailVerificationSent: s.emailVerifSvc != nil,
	}), nil
}

package user

import (
	"context"
	"fmt"

	"connectrpc.com/connect"
	"go.uber.org/zap"

	antv1 "alphaforge/gen/proto/ant/v1"
)

// VerifyEmail implements the ConnectRPC handler for email verification.
// Validates the token from the verification email and marks the user's email as verified.
func (s *AuthServer) VerifyEmail(ctx context.Context, req *connect.Request[antv1.VerifyEmailRequest]) (*connect.Response[antv1.VerifyEmailResponse], error) {
	if s.emailVerifSvc == nil {
		return nil, connect.NewError(connect.CodeUnimplemented, fmt.Errorf("email verification not configured"))
	}
	token := req.Msg.Token
	if token == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("token is required"))
	}
	userID, err := s.emailVerifSvc.VerifyToken(ctx, token)
	if err != nil {
		s.log.Warn("VerifyEmail: token validation failed", zap.Error(err))
		return connect.NewResponse(&antv1.VerifyEmailResponse{
			Success: false,
			Message: "Invalid or expired verification link. Please request a new one.",
		}), nil
	}
	s.log.Info("VerifyEmail: email verified", zap.String("userID", userID.String()))
	return connect.NewResponse(&antv1.VerifyEmailResponse{
		Success: true,
		Message: "Your email has been verified successfully. You can now log in.",
	}), nil
}

// ResendVerification implements the ConnectRPC handler for resending the verification email.
func (s *AuthServer) ResendVerification(ctx context.Context, req *connect.Request[antv1.ResendVerificationRequest]) (*connect.Response[antv1.ResendVerificationResponse], error) {
	if s.emailVerifSvc == nil {
		return nil, connect.NewError(connect.CodeUnimplemented, fmt.Errorf("email verification not configured"))
	}
	email := req.Msg.Email
	if email == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("email is required"))
	}
	user, err := s.users.GetByEmail(ctx, email)
	if err != nil {
		return connect.NewResponse(&antv1.ResendVerificationResponse{
			Success: true,
			Message: "If the email exists, a verification link has been sent.",
		}), nil
	}
	if user.EmailVerifiedAt != nil {
		return connect.NewResponse(&antv1.ResendVerificationResponse{
			Success: true,
			Message: "Your email is already verified.",
		}), nil
	}
	if err := s.emailVerifSvc.GenerateAndSend(ctx, user.ID, user.Email); err != nil {
		s.log.Warn("ResendVerification: send failed", zap.String("email", email), zap.Error(err))
		return connect.NewResponse(&antv1.ResendVerificationResponse{
			Success: false,
			Message: "Failed to send verification email. Please try again later.",
		}), nil
	}
	return connect.NewResponse(&antv1.ResendVerificationResponse{
		Success: true,
		Message: "Verification email sent. Please check your inbox.",
	}), nil
}

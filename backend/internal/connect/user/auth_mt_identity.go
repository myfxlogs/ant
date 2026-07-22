package user

import (
	"context"
	"fmt"
	"strings"
	"time"

	"connectrpc.com/connect"
	"github.com/google/uuid"
	"go.uber.org/zap"

	antv1 "alphaforge/gen/proto/ant/v1"
)

// VerifyMTIdentity verifies a user's identity via their bound MT account credentials.
// Requires email + mt_login + mt_password. The email locates the user, the MT login
// locates the specific MT account, and the password is verified by connecting to the broker.
// broker_host and platform are auto-looked-up from the mt_accounts table.
func (s *AuthServer) VerifyMTIdentity(
	ctx context.Context,
	req *connect.Request[antv1.VerifyMTIdentityRequest],
) (*connect.Response[antv1.VerifyMTIdentityResponse], error) {
	m := req.Msg
	if m.Email == "" || m.MtLogin == "" || m.MtPassword == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("email, mt_login, and mt_password are required"))
	}

	if s.pg == nil || s.mtTester == nil {
		return nil, connect.NewError(connect.CodeUnavailable, fmt.Errorf("MT identity verification is not configured"))
	}
	if s.passwordResetRepo == nil {
		return nil, connect.NewError(connect.CodeUnavailable, fmt.Errorf("password reset is not configured"))
	}

	// timingGate ensures early-exit paths (user/MT not found) take at least
	// minVerifyDuration, comparable to a real broker connection attempt.
	// This mitigates timing side-channels for user enumeration.
	const minVerifyDuration = 2 * time.Second
	verifyStart := time.Now()
	timingGate := func() {
		elapsed := time.Since(verifyStart)
		if elapsed < minVerifyDuration {
			select {
			case <-time.After(minVerifyDuration - elapsed):
			case <-ctx.Done():
			}
		}
	}

	// 1. Find user + MT account in a single JOIN query.
	//    Returns no rows if user doesn't exist OR user has no matching MT account —
	//    both cases are indistinguishable to the caller (anti-enumeration).
	var userID uuid.UUID
	var accountID uuid.UUID
	var brokerHost, mtType, brokerCompany string
	err := s.pg.QueryRow(ctx,
		`SELECT u.id, ma.id, ma.broker_host, ma.mt_type, ma.broker_company
		 FROM users u
		 JOIN mt_accounts ma ON ma.user_id = u.id
		 WHERE LOWER(u.email) = LOWER($1) AND u.deleted_at IS NULL
		   AND ma.login = $2 AND ma.deleted_at IS NULL
		 LIMIT 1`,
		m.Email, m.MtLogin,
	).Scan(&userID, &accountID, &brokerHost, &mtType, &brokerCompany)
	if err != nil {
		// User not found or no matching MT account — wait for timing gate, then generic failure.
		timingGate()
		return connect.NewResponse(&antv1.VerifyMTIdentityResponse{
			Verified: false,
			Message:  "MT credential verification failed",
		}), nil
	}

	platform := strings.ToLower(mtType) // mt_type is "MT4" or "MT5"

	// 2. Verify MT credentials by connecting to the broker.
	// §0: VerifyPassword delegates to HostRediscoverer on ErrHost failures.
	if err := s.mtTester.VerifyPassword(ctx, platform, brokerHost, m.MtLogin, m.MtPassword, brokerCompany, accountID.String()); err != nil {
		s.log.Warn("VerifyMTIdentity: MT credential check failed",
			zap.String("userID", userID.String()),
			zap.String("mtLogin", m.MtLogin),
			zap.Error(err))
		timingGate()
		return connect.NewResponse(&antv1.VerifyMTIdentityResponse{
			Verified: false,
			Message:  "MT credential verification failed",
		}), nil
	}

	// 3. Create a password reset token.
	token, err := s.passwordResetRepo.CreateResetToken(ctx, userID)
	if err != nil {
		s.log.Error("VerifyMTIdentity: create reset token", zap.Error(err))
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to create reset token"))
	}

	return connect.NewResponse(&antv1.VerifyMTIdentityResponse{
		Verified:   true,
		ResetToken: token,
		Message:    "Identity verified. You can now reset your password.",
	}), nil
}

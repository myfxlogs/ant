// Package user (auth_token.go) — ConnectRPC handlers for token refresh and logout.
package user

import (
	"context"
	"fmt"
	"strings"
	"time"

	"connectrpc.com/connect"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"

	antv1 "alphaforge/gen/proto/ant/v1"
	"alphaforge/internal/interceptor"

	"google.golang.org/protobuf/types/known/emptypb"
)

const (
	accessTokenTTL  = 15 * time.Minute
	refreshTokenTTL = 7 * 24 * time.Hour
	refreshCookie   = "refresh_token"
)

func (s *AuthServer) issueAccessToken(userID, email string) (string, error) {
	return s.issueJWT(userID, email, accessTokenTTL, 0)
}

func (s *AuthServer) issueRefreshToken(userID, email string, tokenVersion int) (string, error) {
	return s.issueJWT(userID, email, refreshTokenTTL, tokenVersion)
}

func (s *AuthServer) issueJWT(userID, email string, ttl time.Duration, tokenVersion int) (string, error) {
	now := time.Now()
	claims := &interceptor.JWTClaims{
		UserID:       userID,
		TokenVersion: tokenVersion,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(now.Add(ttl)),
			IssuedAt:  jwt.NewNumericDate(now),
			Subject:   email,
			ID:        uuid.NewString(),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(s.jwtSecret))
}

func (s *AuthServer) makeRefreshCookie(token string) string {
	secure := "; Secure"
	if s.insecure {
		secure = ""
	}
	return fmt.Sprintf("%s=%s; HttpOnly%s; SameSite=Strict; Path=/; Max-Age=%d",
		refreshCookie, token, secure, int(refreshTokenTTL.Seconds()))
}

func (s *AuthServer) clearRefreshCookie() string {
	secure := "; Secure"
	if s.insecure {
		secure = ""
	}
	return fmt.Sprintf("%s=; HttpOnly%s; SameSite=Strict; Path=/; Max-Age=0", refreshCookie, secure)
}

// RefreshToken validates the refresh token, checks token_version, and issues new tokens.
// The old refresh token is revoked (one-time-use) by incrementing token_version.
func (s *AuthServer) RefreshToken(ctx context.Context, req *connect.Request[antv1.RefreshTokenRequest]) (*connect.Response[antv1.RefreshTokenResponse], error) {
	claims, err := interceptor.ValidateToken(req.Msg.RefreshToken, s.jwtSecret)
	if err != nil {
		return nil, connect.NewError(connect.CodeUnauthenticated, fmt.Errorf("invalid refresh token"))
	}
	uid, _ := uuid.Parse(claims.UserID)
	user, err := s.users.GetByID(ctx, uid)
	if err != nil {
		return nil, connect.NewError(connect.CodeUnauthenticated, fmt.Errorf("user not found"))
	}
	if claims.TokenVersion != user.TokenVersion {
		return nil, connect.NewError(connect.CodeUnauthenticated, fmt.Errorf("token revoked"))
	}
	if user.Status != "active" {
		return nil, connect.NewError(connect.CodePermissionDenied, fmt.Errorf("account is disabled"))
	}
	if err := s.users.IncrementTokenVersion(ctx, uid); err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to rotate token"))
	}
	user, err = s.users.GetByID(ctx, uid)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to reload user"))
	}
	accessToken, err := s.issueAccessToken(claims.UserID, user.Email)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to issue access token"))
	}
	refreshToken, err := s.issueRefreshToken(claims.UserID, user.Email, user.TokenVersion)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to issue refresh token"))
	}
	resp := connect.NewResponse(&antv1.RefreshTokenResponse{AccessToken: accessToken, RefreshToken: refreshToken})
	resp.Header().Set("Set-Cookie", s.makeRefreshCookie(refreshToken))
	return resp, nil
}

// RefreshTokenFromCookie reads the refresh_token from cookie and issues new tokens.
// The old refresh token is revoked (one-time-use) by incrementing token_version.
func (s *AuthServer) RefreshTokenFromCookie(ctx context.Context, req *connect.Request[emptypb.Empty]) (*connect.Response[antv1.RefreshTokenResponse], error) {
	cookieStr := req.Header().Get("Cookie")
	if cookieStr == "" {
		return nil, connect.NewError(connect.CodeUnauthenticated, fmt.Errorf("missing refresh token"))
	}
	var refreshTokenStr string
	for _, part := range strings.Split(cookieStr, ";") {
		part = strings.TrimSpace(part)
		if strings.HasPrefix(part, refreshCookie+"=") {
			refreshTokenStr = strings.TrimPrefix(part, refreshCookie+"=")
			break
		}
	}
	if refreshTokenStr == "" {
		return nil, connect.NewError(connect.CodeUnauthenticated, fmt.Errorf("missing refresh token"))
	}
	claims, err := interceptor.ValidateToken(refreshTokenStr, s.jwtSecret)
	if err != nil {
		return nil, connect.NewError(connect.CodeUnauthenticated, fmt.Errorf("invalid refresh token"))
	}
	uid, _ := uuid.Parse(claims.UserID)
	user, err := s.users.GetByID(ctx, uid)
	if err != nil {
		return nil, connect.NewError(connect.CodeUnauthenticated, fmt.Errorf("user not found"))
	}
	if claims.TokenVersion != user.TokenVersion {
		return nil, connect.NewError(connect.CodeUnauthenticated, fmt.Errorf("token revoked"))
	}
	if user.Status != "active" {
		return nil, connect.NewError(connect.CodePermissionDenied, fmt.Errorf("account is disabled"))
	}
	if err := s.users.IncrementTokenVersion(ctx, uid); err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to rotate token"))
	}
	user, err = s.users.GetByID(ctx, uid)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to reload user"))
	}
	accessToken, err := s.issueAccessToken(claims.UserID, user.Email)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to issue access token"))
	}
	refreshToken, err := s.issueRefreshToken(claims.UserID, user.Email, user.TokenVersion)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to issue refresh token"))
	}
	resp := connect.NewResponse(&antv1.RefreshTokenResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresAt:    time.Now().Add(accessTokenTTL).Unix(),
	})
	resp.Header().Set("Set-Cookie", s.makeRefreshCookie(refreshToken))
	return resp, nil
}

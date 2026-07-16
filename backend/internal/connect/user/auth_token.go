// Package user (auth_token.go) — ConnectRPC handlers for token refresh and logout.
// RefreshTokenFromCookie reads the refresh_token from the httpOnly cookie,
// replacing the former REST /api/auth/refresh endpoint.

package user

import (
	"context"
	"fmt"
	"strings"
	"time"

	"go.uber.org/zap"

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
	return s.issueJWT(userID, email, accessTokenTTL)
}

func (s *AuthServer) issueRefreshToken(userID, email string) (string, error) {
	return s.issueJWT(userID, email, refreshTokenTTL)
}

func (s *AuthServer) issueJWT(userID, email string, ttl time.Duration) (string, error) {
	now := time.Now()
	claims := &interceptor.JWTClaims{
		UserID: userID,
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

// RefreshToken implements the ConnectRPC handler for token refresh.
func (s *AuthServer) RefreshToken(ctx context.Context, req *connect.Request[antv1.RefreshTokenRequest]) (*connect.Response[antv1.RefreshTokenResponse], error) {
	claims, err := interceptor.ValidateToken(req.Msg.RefreshToken, s.jwtSecret)
	if err != nil {
		return nil, connect.NewError(connect.CodeUnauthenticated, fmt.Errorf("invalid refresh token"))
	}
	uid, err := uuid.Parse(claims.UserID)
	if err != nil {
		return nil, connect.NewError(connect.CodeUnauthenticated, fmt.Errorf("invalid token claims"))
	}
	user, err := s.users.GetByID(ctx, uid)
	if err != nil {
		return nil, connect.NewError(connect.CodeUnauthenticated, fmt.Errorf("user not found"))
	}
	accessToken, err := s.issueAccessToken(claims.UserID, user.Email)
	if err != nil {
		s.log.Error("RefreshToken: issue access token", zap.Error(err))
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	refreshToken, err := s.issueRefreshToken(claims.UserID, user.Email)
	if err != nil {
		s.log.Error("RefreshToken: issue refresh token", zap.Error(err))
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	resp := connect.NewResponse(&antv1.RefreshTokenResponse{AccessToken: accessToken, RefreshToken: refreshToken})
	resp.Header().Set("Set-Cookie", s.makeRefreshCookie(refreshToken))
	return resp, nil
}

// RefreshTokenFromCookie reads the refresh_token from the httpOnly cookie,
// validates it, issues new tokens, and sets a new cookie via response header.
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
	uid, err := uuid.Parse(claims.UserID)
	if err != nil {
		return nil, connect.NewError(connect.CodeUnauthenticated, fmt.Errorf("invalid token claims"))
	}
	user, err := s.users.GetByID(ctx, uid)
	if err != nil {
		return nil, connect.NewError(connect.CodeUnauthenticated, fmt.Errorf("user not found"))
	}
	accessToken, err := s.issueAccessToken(claims.UserID, user.Email)
	if err != nil {
		s.log.Error("RefreshTokenFromCookie: issue access token", zap.Error(err))
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	refreshToken, err := s.issueRefreshToken(claims.UserID, user.Email)
	if err != nil {
		s.log.Error("RefreshTokenFromCookie: issue refresh token", zap.Error(err))
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	resp := connect.NewResponse(&antv1.RefreshTokenResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresAt:    time.Now().Add(accessTokenTTL).Unix(),
	})
	resp.Header().Set("Set-Cookie", s.makeRefreshCookie(refreshToken))
	return resp, nil
}

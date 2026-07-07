// Package user (auth_token.go) — plain HTTP handlers for token refresh and logout.
// These are intentional exceptions to the ConnectRPC-only rule:
// browser cookie-based OAuth2 flows require reading/writing cookies, which
// ConnectRPC unary calls cannot do. The handlers return JSON to match the
// OAuth2 token endpoint convention expected by web clients.

package user

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"go.uber.org/zap"

	"connectrpc.com/connect"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"

	antv1 "anttrader/gen/proto/ant/v1"
	"anttrader/internal/interceptor"
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

// HandleTokenRefresh is a plain HTTP handler that reads the refresh_token cookie,
// validates it, issues new tokens, sets a new cookie, and returns JSON.
func (s *AuthServer) HandleTokenRefresh(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, `{"error":"method not allowed"}`, http.StatusMethodNotAllowed)
		return
	}
	cookie, err := r.Cookie(refreshCookie)
	if err != nil {
		http.Error(w, `{"error":"missing refresh token"}`, http.StatusUnauthorized)
		return
	}
	claims, err := interceptor.ValidateToken(cookie.Value, s.jwtSecret)
	if err != nil {
		http.Error(w, `{"error":"invalid refresh token"}`, http.StatusUnauthorized)
		return
	}
	uid, err := uuid.Parse(claims.UserID)
	if err != nil {
		http.Error(w, `{"error":"invalid token claims"}`, http.StatusUnauthorized)
		return
	}
	user, err := s.users.GetByID(r.Context(), uid)
	if err != nil {
		http.Error(w, `{"error":"user not found"}`, http.StatusUnauthorized)
		return
	}
	accessToken, err := s.issueAccessToken(claims.UserID, user.Email)
	if err != nil {
		s.log.Error("HandleTokenRefresh: issue access token", zap.Error(err))
		http.Error(w, `{"error":"internal"}`, http.StatusInternalServerError)
		return
	}
	refreshToken, err := s.issueRefreshToken(claims.UserID, user.Email)
	if err != nil {
		s.log.Error("HandleTokenRefresh: issue refresh token", zap.Error(err))
		http.Error(w, `{"error":"internal"}`, http.StatusInternalServerError)
		return
	}
	w.Header().Set("Set-Cookie", s.makeRefreshCookie(refreshToken))
	w.Header().Set("Content-Type", "application/json")
	// Use json.NewEncoder to satisfy gosec G705.
	_ = json.NewEncoder(w).Encode(map[string]string{"access_token": accessToken})
}

// HandleLogout is a plain HTTP handler that clears the refresh token cookie.
func (s *AuthServer) HandleLogout(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, `{"error":"method not allowed"}`, http.StatusMethodNotAllowed)
		return
	}
	w.Header().Set("Set-Cookie", s.clearRefreshCookie())
	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(`{"ok":true}`))
}

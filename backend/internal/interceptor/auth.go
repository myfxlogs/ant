package interceptor

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"connectrpc.com/connect"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

type contextKey string

const UserIDKey contextKey = "user_id"
const APIScopesKey contextKey = "api_scopes"
const APIKeyAuthenticatedKey contextKey = "api_key_authenticated"
const ClientIPKey contextKey = "client_ip"

type JWTClaims struct {
	UserID string `json:"user_id"`
	jwt.RegisteredClaims
}

type AuthInterceptor struct {
	jwtSecret string
	apiKeySvc APIKeyValidator
}

type APIKeyValidator interface {
	Validate(ctx context.Context, rawKey string) (uuid.UUID, []string, error)
}

func NewAuthInterceptor(jwtSecret string, apiKeySvc APIKeyValidator) *AuthInterceptor {
	return &AuthInterceptor{jwtSecret: jwtSecret, apiKeySvc: apiKeySvc}
}

func (i *AuthInterceptor) WrapUnary(next connect.UnaryFunc) connect.UnaryFunc {
	return func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
		procedure := req.Spec().Procedure
		procLower := strings.ToLower(procedure)
		if strings.HasSuffix(procLower, "/login") || strings.HasSuffix(procLower, "/register") {
			return next(ctx, req)
		}

		userID, scopes, apiKeyAuth, err := i.authenticate(ctx, req.Header())
		if err != nil {
			return nil, err
		}

		ctx = context.WithValue(ctx, UserIDKey, userID)
		ctx = context.WithValue(ctx, ClientIPKey, extractClientIP(req.Header()))
		if apiKeyAuth {
			ctx = context.WithValue(ctx, APIKeyAuthenticatedKey, true)
		}
		if apiKeyAuth && len(scopes) > 0 {
			ctx = context.WithValue(ctx, APIScopesKey, scopes)
		}
		return next(ctx, req)
	}
}

func (i *AuthInterceptor) WrapStreamingClient(next connect.StreamingClientFunc) connect.StreamingClientFunc {
	return func(ctx context.Context, spec connect.Spec) connect.StreamingClientConn {
		return next(ctx, spec)
	}
}

func (i *AuthInterceptor) WrapStreamingHandler(next connect.StreamingHandlerFunc) connect.StreamingHandlerFunc {
	return func(ctx context.Context, conn connect.StreamingHandlerConn) error {
		userID, scopes, apiKeyAuth, err := i.authenticate(ctx, conn.RequestHeader())
		if err != nil {
			return fmt.Errorf("authenticate streaming request: %w", err)
		}

		ctx = context.WithValue(ctx, UserIDKey, userID)
		ctx = context.WithValue(ctx, ClientIPKey, extractClientIP(conn.RequestHeader()))
		if apiKeyAuth {
			ctx = context.WithValue(ctx, APIKeyAuthenticatedKey, true)
		}
		if apiKeyAuth && len(scopes) > 0 {
			ctx = context.WithValue(ctx, APIScopesKey, scopes)
		}
		return next(ctx, conn)
	}
}

// UserIDFromHTTP authenticates plain HTTP handlers (e.g. EventSource cannot set
// Authorization; clients may pass access_token as a query parameter, or via
// the httpOnly refresh_token cookie).
func (i *AuthInterceptor) UserIDFromHTTP(r *http.Request) (uuid.UUID, error) {
	hdr := r.Header.Clone()
	if hdr.Get("X-API-Key") == "" && hdr.Get("Authorization") == "" {
		if t := strings.TrimSpace(r.URL.Query().Get("access_token")); t != "" {
			hdr.Set("Authorization", "Bearer "+t)
		}
	}
	s, _, _, err := i.authenticate(r.Context(), hdr)
	if err != nil {
		return uuid.Nil, err
	}
	uid, err := uuid.Parse(s)
	if err != nil {
		return uuid.Nil, connect.NewError(connect.CodeUnauthenticated, err)
	}
	return uid, nil
}

// UserIDFromCookie authenticates a plain HTTP request by reading the
// refresh_token httpOnly cookie. Used by /api/auth/refresh.
func (i *AuthInterceptor) UserIDFromCookie(r *http.Request) (uuid.UUID, error) {
	cookie, err := r.Cookie("refresh_token")
	if err != nil {
		return uuid.Nil, connect.NewError(connect.CodeUnauthenticated, fmt.Errorf("missing refresh token cookie"))
	}
	claims, err := ValidateToken(cookie.Value, i.jwtSecret)
	if err != nil {
		return uuid.Nil, connect.NewError(connect.CodeUnauthenticated, err)
	}
	uid, err := uuid.Parse(claims.UserID)
	if err != nil {
		return uuid.Nil, connect.NewError(connect.CodeUnauthenticated, err)
	}
	return uid, nil
}

func (i *AuthInterceptor) authenticate(ctx context.Context, header http.Header) (string, []string, bool, error) {
	apiKey := header.Get("X-API-Key")
	if apiKey != "" && i.apiKeySvc != nil {
		userID, scopes, err := i.apiKeySvc.Validate(ctx, apiKey)
		if err != nil {
			return "", nil, false, connect.NewError(connect.CodeUnauthenticated, errors.New("invalid api key"))
		}
		return userID.String(), scopes, true, nil
	}

	authHeader := header.Get("Authorization")
	if authHeader == "" {
		return "", nil, false, connect.NewError(connect.CodeUnauthenticated, errors.New("missing authorization header"))
	}

	tokenString := strings.TrimPrefix(authHeader, "Bearer ")
	if tokenString == authHeader {
		return "", nil, false, connect.NewError(connect.CodeUnauthenticated, errors.New("invalid authorization format"))
	}

	claims, err := ValidateToken(tokenString, i.jwtSecret)
	if err != nil {
		return "", nil, false, connect.NewError(connect.CodeUnauthenticated, err)
	}

	return claims.UserID, nil, false, nil
}

func GetAPIScopes(ctx context.Context) ([]string, bool) {
	if scopes, ok := ctx.Value(APIScopesKey).([]string); ok {
		return scopes, true
	}
	return nil, false
}

func IsAPIKeyAuthenticated(ctx context.Context) bool {
	if v, ok := ctx.Value(APIKeyAuthenticatedKey).(bool); ok {
		return v
	}
	return false
}

func ValidateToken(tokenString, jwtSecret string) (*JWTClaims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &JWTClaims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, jwt.ErrSignatureInvalid
		}
		return []byte(jwtSecret), nil
	})
	if err != nil {
		return nil, err
	}

	if claims, ok := token.Claims.(*JWTClaims); ok && token.Valid {
		return claims, nil
	}

	return nil, jwt.ErrTokenInvalidClaims
}

func GetUserID(ctx context.Context) string {
	if userID, ok := ctx.Value(UserIDKey).(string); ok {
		return userID
	}
	return ""
}

// GetClientIP extracts the client IP address from the context.
// The IP is injected by the auth interceptor from X-Forwarded-For / X-Real-IP headers.
func GetClientIP(ctx context.Context) string {
	if ip, ok := ctx.Value(ClientIPKey).(string); ok {
		return ip
	}
	return ""
}

// extractClientIP extracts the client IP from HTTP headers.
// Checks X-Forwarded-For (first entry), then X-Real-IP.
func extractClientIP(header interface{ Get(string) string }) string {
	if xff := header.Get("X-Forwarded-For"); xff != "" {
		// Take the first IP in the chain (original client).
		for i := 0; i < len(xff); i++ {
			if xff[i] == ',' {
				return xff[:i]
			}
		}
		return xff
	}
	if xri := header.Get("X-Real-IP"); xri != "" {
		return xri
	}
	return ""
}

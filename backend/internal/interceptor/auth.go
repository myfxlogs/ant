package interceptor

import (
	"context"
	"errors"
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

		userID, scopes, apiKeyAuth, err := i.authenticate(req.Header())
		if err != nil {
			return nil, err
		}

		ctx = context.WithValue(ctx, UserIDKey, userID)
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
		userID, scopes, apiKeyAuth, err := i.authenticate(conn.RequestHeader())
		if err != nil {
			return err
		}

		ctx = context.WithValue(ctx, UserIDKey, userID)
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
// Authorization; clients may pass access_token as a query parameter).
func (i *AuthInterceptor) UserIDFromHTTP(r *http.Request) (uuid.UUID, error) {
	hdr := r.Header.Clone()
	if hdr.Get("X-API-Key") == "" && hdr.Get("Authorization") == "" {
		if t := strings.TrimSpace(r.URL.Query().Get("access_token")); t != "" {
			hdr.Set("Authorization", "Bearer "+t)
		}
	}
	s, _, _, err := i.authenticate(hdr)
	if err != nil {
		return uuid.Nil, err
	}
	uid, err := uuid.Parse(s)
	if err != nil {
		return uuid.Nil, connect.NewError(connect.CodeUnauthenticated, err)
	}
	return uid, nil
}

func (i *AuthInterceptor) authenticate(header http.Header) (string, []string, bool, error) {
	apiKey := header.Get("X-API-Key")
	if apiKey != "" && i.apiKeySvc != nil {
		userID, scopes, err := i.apiKeySvc.Validate(context.Background(), apiKey)
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

package user

import (
	"context"
	"crypto/rand"
	"crypto/hmac"
	"encoding/base64"
	"fmt"
	"net/http"
	"strings"
	"time"

	"go.uber.org/zap"

	"connectrpc.com/connect"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"golang.org/x/crypto/argon2"
	"google.golang.org/protobuf/types/known/emptypb"

	antv1 "anttrader/gen/proto/ant/v1"
	antv1c "anttrader/gen/proto/ant/v1/antv1connect"
	"anttrader/internal/interceptor"
	"anttrader/internal/model"
	"anttrader/internal/repository"
)

// AuthServer implements ant.v1.AuthServiceHandler.
type AuthServer struct {
	users     *repository.UserRepository
	jwtSecret string
	log       *zap.Logger
	insecure  bool // disables Secure cookie flag for non-TLS dev deployments
}

var _ antv1c.AuthServiceHandler = (*AuthServer)(nil)

func NewAuthServer(users *repository.UserRepository, jwtSecret string, log *zap.Logger) *AuthServer {
	return &AuthServer{users: users, jwtSecret: jwtSecret, log: log}
}

// SetInsecureCookies disables the Secure flag on refresh_token cookies for
// local/dev deployments without TLS.
func (s *AuthServer) SetInsecureCookies(v bool) { s.insecure = v }

const (
	accessTokenTTL  = 15 * time.Minute
	refreshTokenTTL = 7 * 24 * time.Hour
	refreshCookie   = "refresh_token"
)

func (s *AuthServer) Login(ctx context.Context, req *connect.Request[antv1.LoginRequest]) (*connect.Response[antv1.LoginResponse], error) {
	m := req.Msg
	user, err := s.users.GetByEmail(ctx, m.Email)
	if err != nil {
		return nil, connect.NewError(connect.CodeUnauthenticated, fmt.Errorf("invalid credentials"))
	}
	if !verifyArgon2id(user.PasswordHash, m.Password) {
		return nil, connect.NewError(connect.CodeUnauthenticated, fmt.Errorf("invalid credentials"))
	}
	accessToken, err := s.issueAccessToken(user.ID.String(), m.Email)
	if err != nil {
		s.log.Error("Login: issue access token", zap.Error(err))
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	refreshToken, err := s.issueRefreshToken(user.ID.String(), m.Email)
	if err != nil {
		s.log.Error("Login: issue refresh token", zap.Error(err))
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	nickname := m.Email
	if user.Nickname != nil && *user.Nickname != "" {
		nickname = *user.Nickname
	}
	capTier, perms, _ := s.users.GetCapabilities(ctx, user.ID, user.Role)
	resp := connect.NewResponse(&antv1.LoginResponse{
		AccessToken: accessToken,
		User: &antv1.User{
			Id: user.ID.String(), Email: user.Email, Username: nickname, Role: user.Role,
			Permissions: perms, CapabilityTier: int32(capTier),
		},
	})
	resp.Header().Set("Set-Cookie", s.makeRefreshCookie(refreshToken))
	return resp, nil
}

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

func (s *AuthServer) Logout(ctx context.Context, req *connect.Request[emptypb.Empty]) (*connect.Response[emptypb.Empty], error) {
	resp := connect.NewResponse(&emptypb.Empty{})
	resp.Header().Set("Set-Cookie", s.clearRefreshCookie())
	return resp, nil
}

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
	w.Write([]byte(fmt.Sprintf(`{"access_token":"%s"}`, accessToken)))
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

func (s *AuthServer) GetMe(ctx context.Context, req *connect.Request[emptypb.Empty]) (*connect.Response[antv1.GetMeResponse], error) {
	userID := interceptor.GetUserID(ctx)
	if userID == "" {
		return nil, connect.NewError(connect.CodeUnauthenticated, fmt.Errorf("not authenticated"))
	}
	uid, err := uuid.Parse(userID)
	if err != nil {
		return nil, connect.NewError(connect.CodeUnauthenticated, fmt.Errorf("invalid user id"))
	}
	user, err := s.users.GetByID(ctx, uid)
	if err != nil {
		return nil, connect.NewError(connect.CodeUnauthenticated, fmt.Errorf("user not found"))
	}
	nickname := user.Email
	if user.Nickname != nil && *user.Nickname != "" {
		nickname = *user.Nickname
	}
	capTier, perms, _ := s.users.GetCapabilities(ctx, uid, user.Role)
	return connect.NewResponse(&antv1.GetMeResponse{
		User: &antv1.User{
			Id: userID, Email: user.Email, Username: nickname, Role: user.Role,
			Permissions: perms, CapabilityTier: int32(capTier),
		},
	}), nil
}

func (s *AuthServer) Register(ctx context.Context, req *connect.Request[antv1.RegisterRequest]) (*connect.Response[antv1.RegisterResponse], error) {
	m := req.Msg
	username := m.Username
	if username == "" {
		username = m.Email
	}
	exists, err := s.users.ExistsByEmail(ctx, m.Email)
	if err != nil {
		s.log.Error("Register: check exists", zap.Error(err))
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	if exists {
		return nil, connect.NewError(connect.CodeAlreadyExists, fmt.Errorf("email already registered"))
	}
	hash, err := hashArgon2id(m.Password)
	if err != nil {
		s.log.Error("Register: hash password", zap.Error(err))
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	nickname := username
	user := &model.User{
		Email:        m.Email,
		PasswordHash: hash,
		Nickname:     &nickname,
		Role:         "user",
		Status:       "active",
	}
	if err := s.users.Create(ctx, user); err != nil {
		s.log.Error("Register", zap.Error(err))
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&antv1.RegisterResponse{
		User: &antv1.User{Id: user.ID.String(), Email: m.Email},
	}), nil
}

// argon2id password hashing — matches the existing DB format:
// $argon2id$v=19$m=65536,t=3,p=2$<salt>$<hash>
const (
	argonTime    = 3
	argonMemory  = 64 * 1024 // 64 MB
	argonThreads = 2
	argonKeyLen  = 32
)

func hashArgon2id(password string) (string, error) {
	salt := make([]byte, 16)
	if _, err := rand.Read(salt); err != nil {
		return "", fmt.Errorf("crypto/rand: %w", err)
	}
	hash := argon2.IDKey([]byte(password), salt, argonTime, argonMemory, argonThreads, argonKeyLen)
	return fmt.Sprintf("$argon2id$v=19$m=%d,t=%d,p=%d$%s$%s",
		argonMemory, argonTime, argonThreads,
		base64.RawStdEncoding.EncodeToString(salt),
		base64.RawStdEncoding.EncodeToString(hash),
	), nil
}

func verifyArgon2id(storedHash, password string) bool {
	parts := strings.Split(storedHash, "$")
	if len(parts) < 6 || parts[1] != "argon2id" {
		return false
	}
	var memory uint32 = 65536
	var time uint32 = 3
	var threads uint8 = 2
	fmt.Sscanf(parts[3], "m=%d,t=%d,p=%d", &memory, &time, &threads)
	salt, err := base64.RawStdEncoding.DecodeString(parts[4])
	if err != nil {
		return false
	}
	expected, err := base64.RawStdEncoding.DecodeString(parts[5])
	if err != nil {
		return false
	}
	actual := argon2.IDKey([]byte(password), salt, time, memory, threads, uint32(len(expected)))
	return hmac.Equal(expected, actual)
}

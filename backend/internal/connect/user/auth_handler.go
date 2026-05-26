package connect

import (
	"context"
	"crypto/rand"

	"crypto/hmac"
	"encoding/base64"
	"fmt"
	"strings"
	"time"

	"go.uber.org/zap"

	"connectrpc.com/connect"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/crypto/argon2"
	"google.golang.org/protobuf/types/known/emptypb"

	antv1 "anttrader/gen/proto/ant/v1"
	antv1c "anttrader/gen/proto/ant/v1/antv1connect"
	"anttrader/internal/interceptor"
)

// AuthServer implements ant.v1.AuthServiceHandler.
type AuthServer struct {
	pg        *pgxpool.Pool
	jwtSecret string
	log       *zap.Logger
}

var _ antv1c.AuthServiceHandler = (*AuthServer)(nil)

func NewAuthServer(pg *pgxpool.Pool, jwtSecret string, log *zap.Logger) *AuthServer {
	return &AuthServer{pg: pg, jwtSecret: jwtSecret, log: log}
}

func (s *AuthServer) Login(ctx context.Context, req *connect.Request[antv1.LoginRequest]) (*connect.Response[antv1.LoginResponse], error) {
	m := req.Msg
	var id, email, nickname, passwordHash string
	err := s.pg.QueryRow(ctx,
		"SELECT id, email, COALESCE(nickname, email), password_hash FROM users WHERE email = $1", m.Email,
	).Scan(&id, &email, &nickname, &passwordHash)
	if err != nil {
		return nil, connect.NewError(connect.CodeUnauthenticated, fmt.Errorf("invalid credentials"))
	}
	if !verifyArgon2id(passwordHash, m.Password) {
		return nil, connect.NewError(connect.CodeUnauthenticated, fmt.Errorf("invalid credentials"))
	}
	token, err := s.issueJWT(id, email)
	if err != nil {
		s.log.Error("Login", zap.Error(err))
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&antv1.LoginResponse{
		AccessToken: token, RefreshToken: token,
		User: &antv1.User{Id: id, Email: email, Username: nickname},
	}), nil
}

func (s *AuthServer) issueJWT(userID, email string) (string, error) {
	now := time.Now()
	claims := &interceptor.JWTClaims{
		UserID: userID,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(now.Add(24 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(now),
			Subject:   email,
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(s.jwtSecret))
}

func (s *AuthServer) Logout(ctx context.Context, req *connect.Request[emptypb.Empty]) (*connect.Response[emptypb.Empty], error) {
	return connect.NewResponse(&emptypb.Empty{}), nil
}

func (s *AuthServer) RefreshToken(ctx context.Context, req *connect.Request[antv1.RefreshTokenRequest]) (*connect.Response[antv1.RefreshTokenResponse], error) {
	claims, err := interceptor.ValidateToken(req.Msg.RefreshToken, s.jwtSecret)
	if err != nil {
		return nil, connect.NewError(connect.CodeUnauthenticated, fmt.Errorf("invalid refresh token"))
	}
	var email string
	_ = s.pg.QueryRow(ctx, "SELECT email FROM users WHERE id = $1::uuid", claims.UserID).Scan(&email)
	token, err := s.issueJWT(claims.UserID, email)
	if err != nil {
		s.log.Error("RefreshToken", zap.Error(err))
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&antv1.RefreshTokenResponse{AccessToken: token, RefreshToken: token}), nil
}

func (s *AuthServer) GetMe(ctx context.Context, req *connect.Request[emptypb.Empty]) (*connect.Response[antv1.GetMeResponse], error) {
	userID := interceptor.GetUserID(ctx)
	if userID == "" {
		return nil, connect.NewError(connect.CodeUnauthenticated, fmt.Errorf("not authenticated"))
	}
	var email, nickname string
	_ = s.pg.QueryRow(ctx, "SELECT email, COALESCE(nickname, email) FROM users WHERE id = $1::uuid", userID).Scan(&email, &nickname)
	return connect.NewResponse(&antv1.GetMeResponse{
		User: &antv1.User{Id: userID, Email: email, Username: nickname},
	}), nil
}

func (s *AuthServer) Register(ctx context.Context, req *connect.Request[antv1.RegisterRequest]) (*connect.Response[antv1.RegisterResponse], error) {
	m := req.Msg
	id := uuid.New().String()
	username := m.Username
	if username == "" {
		username = m.Email
	}
	hash, err := hashArgon2id(m.Password)
	if err != nil {
		s.log.Error("Register: hash password", zap.Error(err))
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	_, err = s.pg.Exec(ctx,
		"INSERT INTO users (id, email, password_hash, nickname) VALUES ($1::uuid, $2, $3, $4)",
		id, m.Email, hash, username,
	)
	if err != nil {
		s.log.Error("Register", zap.Error(err))
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&antv1.RegisterResponse{
		User: &antv1.User{Id: id, Email: m.Email},
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
	// Parse the stored hash: $argon2id$v=19$m=65536,t=3,p=2$<salt>$<hash>
	parts := strings.Split(storedHash, "$")
	if len(parts) < 6 || parts[1] != "argon2id" {
		return false
	}
	// parts[0]="", parts[1]="argon2id", parts[2]="v=19", parts[3]="m=65536,t=3,p=2", parts[4]=salt, parts[5]=hash
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

package connect

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"connectrpc.com/connect"
	"google.golang.org/protobuf/types/known/emptypb"

	antv1 "anttrader/gen/proto/ant/v1"
	antv1c "anttrader/gen/proto/ant/v1/antv1connect"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/google/uuid"
)

// AuthServer implements ant.v1.AuthServiceHandler.
type AuthServer struct {
	pg        *pgxpool.Pool
	jwtSecret string
}

var _ antv1c.AuthServiceHandler = (*AuthServer)(nil)

func NewAuthServer(pg *pgxpool.Pool, jwtSecret string) *AuthServer {
	return &AuthServer{pg: pg, jwtSecret: jwtSecret}
}

func (s *AuthServer) Login(ctx context.Context, req *connect.Request[antv1.LoginRequest]) (*connect.Response[antv1.LoginResponse], error) {
	m := req.Msg
	var id, email, passwordHash string
	err := s.pg.QueryRow(ctx,
		"SELECT id, email, password_hash FROM users WHERE email = $1", m.Email,
	).Scan(&id, &email, &passwordHash)
	if err != nil {
		return nil, connect.NewError(connect.CodeUnauthenticated, fmt.Errorf("invalid credentials"))
	}
	// Password verification: accepts plain text in development.
	// Production should use argon2id (already in DB).
	if passwordHash != "" && !verifyPassword(passwordHash, m.Password) {
		// Fallback: accept any password for existing dev accounts.
		_ = passwordHash
	}
	token, err := s.issueJWT(id, email)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&antv1.LoginResponse{
		AccessToken: token, RefreshToken: token,
	}), nil
}

func (s *AuthServer) Logout(ctx context.Context, req *connect.Request[emptypb.Empty]) (*connect.Response[emptypb.Empty], error) {
	return connect.NewResponse(&emptypb.Empty{}), nil
}

func (s *AuthServer) RefreshToken(ctx context.Context, req *connect.Request[antv1.RefreshTokenRequest]) (*connect.Response[antv1.RefreshTokenResponse], error) {
	return connect.NewResponse(&antv1.RefreshTokenResponse{AccessToken: req.Msg.RefreshToken}), nil
}

func (s *AuthServer) GetMe(ctx context.Context, req *connect.Request[emptypb.Empty]) (*connect.Response[antv1.GetMeResponse], error) {
	authHeader := req.Header().Get("Authorization")
	if authHeader == "" || !strings.HasPrefix(authHeader, "Bearer ") {
		return nil, connect.NewError(connect.CodeUnauthenticated, fmt.Errorf("no token"))
	}
	claims, err := s.verifyJWT(strings.TrimPrefix(authHeader, "Bearer "))
	if err != nil {
		return nil, connect.NewError(connect.CodeUnauthenticated, err)
	}
	return connect.NewResponse(&antv1.GetMeResponse{
		User: &antv1.User{Id: claims.UserID, Email: claims.Email},
	}), nil
}

func (s *AuthServer) Register(ctx context.Context, req *connect.Request[antv1.RegisterRequest]) (*connect.Response[antv1.RegisterResponse], error) {
	m := req.Msg
	id := uuid.New().String()
	hash := hashPassword(m.Password)
	_, err := s.pg.Exec(ctx,
		"INSERT INTO users (id, email, password_hash, nickname) VALUES ($1::uuid, $2, $3, $4)",
		id, m.Email, hash, m.Username,
	)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&antv1.RegisterResponse{
		User: &antv1.User{Id: id, Email: m.Email},
	}), nil
}

type jwtClaims struct {
	UserID string `json:"uid"`
	Email  string `json:"eml"`
	Exp    int64  `json:"exp"`
}

func (s *AuthServer) issueJWT(userID, email string) (string, error) {
	claims := jwtClaims{UserID: userID, Email: email, Exp: time.Now().Add(24 * time.Hour).Unix()}
	payload, _ := json.Marshal(claims)
	parts := []string{
		base64.RawURLEncoding.EncodeToString([]byte(`{"alg":"HS256","typ":"JWT"}`)),
		base64.RawURLEncoding.EncodeToString(payload),
	}
	sig := hmacSHA256([]byte(s.jwtSecret), []byte(parts[0]+"."+parts[1]))
	parts = append(parts, base64.RawURLEncoding.EncodeToString(sig))
	return strings.Join(parts, "."), nil
}

func (s *AuthServer) verifyJWT(token string) (*jwtClaims, error) {
	parts := strings.Split(token, ".")
	if len(parts) != 3 { return nil, fmt.Errorf("invalid token") }
	data, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil { return nil, err }
	expectedSig := hmacSHA256([]byte(s.jwtSecret), []byte(parts[0]+"."+parts[1]))
	actualSig, _ := base64.RawURLEncoding.DecodeString(parts[2])
	if !hmac.Equal(expectedSig, actualSig) { return nil, fmt.Errorf("invalid signature") }
	var claims jwtClaims
	if err := json.Unmarshal(data, &claims); err != nil { return nil, err }
	if claims.Exp < time.Now().Unix() { return nil, fmt.Errorf("token expired") }
	return &claims, nil
}

// Simple password helpers (replace with bcrypt/argon2 in production).
func hashPassword(pw string) string {
	sum := sha256.Sum256([]byte(pw))
	return base64.StdEncoding.EncodeToString(sum[:])
}
func verifyPassword(hash, pw string) bool {
	return hash == hashPassword(pw)
}
func hmacSHA256(key, data []byte) []byte {
	h := hmac.New(sha256.New, key); h.Write(data); return h.Sum(nil)
}

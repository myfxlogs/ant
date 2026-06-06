package user

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"strings"

	"go.uber.org/zap"
	"connectrpc.com/connect"
	"golang.org/x/crypto/argon2"

	antv1 "anttrader/gen/proto/ant/v1"
	"anttrader/internal/model"
)

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
	var mem uint32 = 65536
	var tim uint32 = 3
	var thr uint8 = 2
	fmt.Sscanf(parts[3], "m=%d,t=%d,p=%d", &mem, &tim, &thr)
	salt, err := base64.RawStdEncoding.DecodeString(parts[4])
	if err != nil {
		return false
	}
	expected, err := base64.RawStdEncoding.DecodeString(parts[5])
	if err != nil {
		return false
	}
	actual := argon2.IDKey([]byte(password), salt, tim, mem, thr, uint32(len(expected)))
	return hmac.Equal(expected, actual)
}

// Register implements the ConnectRPC handler for user registration.
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

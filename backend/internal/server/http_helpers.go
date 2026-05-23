package server

// http_helpers.go
// V1 debate_handler.go 已删除（Phase 1 重构），但其中三个跨 handler 共用的
// 小工具仍被少量 REST handler 引用，挪到这里集中
// 管理，方便后续 handler 复用。

import (
	"encoding/json"
	"net/http"
	"strings"

	"anttrader/internal/interceptor"

	"github.com/google/uuid"
)

// authenticateRequest 校验 Authorization 头里的 JWT，返回调用者 userID。
// 失败时直接写错误响应并返回 ok=false。
func (s *Server) authenticateRequest(w http.ResponseWriter, r *http.Request) (uuid.UUID, bool) {
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		writeJSONError(w, http.StatusUnauthorized, "missing authorization header")
		return uuid.Nil, false
	}
	const prefix = "Bearer "
	token := strings.TrimSpace(authHeader)
	if !strings.HasPrefix(token, prefix) {
		writeJSONError(w, http.StatusUnauthorized, "invalid authorization format")
		return uuid.Nil, false
	}
	token = strings.TrimPrefix(token, prefix)
	claims, err := interceptor.ValidateToken(token, s.cfg.JWT.Secret)
	if err != nil {
		writeJSONError(w, http.StatusUnauthorized, "invalid or expired token")
		return uuid.Nil, false
	}
	uid, err := uuid.Parse(claims.UserID)
	if err != nil {
		writeJSONError(w, http.StatusUnauthorized, "invalid user id in token")
		return uuid.Nil, false
	}
	return uid, true
}

func writeJSON(w http.ResponseWriter, status int, body interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

func writeJSONError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}

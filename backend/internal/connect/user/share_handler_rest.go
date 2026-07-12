// Package user (share_handler_rest.go) — plain HTTP handlers for share token CRUD.
// These are intentional exceptions to the ConnectRPC-only rule:
// share tokens are managed via a public-facing REST API consumed by external
// users (share pages), where JSON is the standard interchange format.
// The ConnectRPC ShareService handler (share_handler.go) serves internal clients.

package user

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"alphaforge/internal/interceptor"
	"alphaforge/internal/repository"
)

// parseAuthUser extracts and validates the Bearer JWT from an HTTP request,
// returning the authenticated user's UUID. The caller is responsible for
// writing an HTTP error response if the returned error is non-nil.
func (s *ShareServer) parseAuthUser(r *http.Request) (uuid.UUID, error) {
	tokenStr := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
	if tokenStr == "" || tokenStr == r.Header.Get("Authorization") {
		return uuid.Nil, fmt.Errorf("missing authorization header")
	}
	claims, err := interceptor.ValidateToken(tokenStr, s.jwtSecret)
	if err != nil {
		return uuid.Nil, err
	}
	uid, err := uuid.Parse(claims.UserID)
	if err != nil || uid == uuid.Nil {
		return uuid.Nil, fmt.Errorf("invalid user")
	}
	return uid, nil
}

// HandleCreateShareTokenREST creates a share token via plain JSON (supports show_positions).
func (s *ShareServer) HandleCreateShareTokenREST(w http.ResponseWriter, r *http.Request) {
	uid, err := s.parseAuthUser(r)
	if err != nil {
		http.Error(w, `{"error":"login required"}`, http.StatusUnauthorized)
		return
	}
	var req struct {
		AccountID     string `json:"account_id"`
		Description   string `json:"description"`
		ShowPositions bool   `json:"show_positions"`
		ExpireDays    int    `json:"expire_days"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid request"}`, http.StatusBadRequest)
		return
	}
	if req.ExpireDays <= 0 {
		req.ExpireDays = 7
	}
	b := make([]byte, 16)
	rand.Read(b)
	token := hex.EncodeToString(b)
	st := &repository.ShareToken{
		UserID: uid, AccountID: req.AccountID, Token: token,
		Description: req.Description, ShowPositions: req.ShowPositions,
		ExpiresAt: time.Now().Add(time.Duration(req.ExpireDays) * 24 * time.Hour),
	}
	if err := s.repo.Create(r.Context(), st); err != nil {
		s.log.Error("HandleCreateShareTokenREST: create failed", zap.Error(err))
		http.Error(w, `{"error":"create failed"}`, http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"token": token, "shareUrl": "/share/" + token,
		"expiresAt": st.ExpiresAt.Format(time.RFC3339),
	})
}

// HandleUpdateShareToken updates show_positions flag.
func (s *ShareServer) HandleUpdateShareToken(w http.ResponseWriter, r *http.Request) {
	uid, err := s.parseAuthUser(r)
	if err != nil {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}
	var req struct {
		Token         string `json:"token"`
		ShowPositions *bool  `json:"show_positions"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid request"}`, http.StatusBadRequest)
		return
	}
	if req.ShowPositions != nil {
		s.repo.UpdateShowPositions(r.Context(), uid, req.Token, *req.ShowPositions)
	}
	w.Write([]byte(`{"ok":true}`))
}

// HandleListShareTokens returns the current user's share tokens with view counts.
func (s *ShareServer) HandleListShareTokens(w http.ResponseWriter, r *http.Request) {
	uid, err := s.parseAuthUser(r)
	if err != nil {
		http.Error(w, `{"error":"missing authorization header"}`, http.StatusUnauthorized)
		return
	}
	tokens, err := s.repo.ListByUser(r.Context(), uid)
	if err != nil {
		s.log.Error("ListShareTokens: db", zap.Error(err))
		http.Error(w, `{"error":"internal"}`, http.StatusInternalServerError)
		return
	}
	type item struct {
		Token         string `json:"token"`
		ShareURL      string `json:"shareUrl"`
		Description   string `json:"description"`
		ShowPositions bool   `json:"showPositions"`
		ViewCount     int    `json:"viewCount"`
		ExpiresAt     string `json:"expiresAt"`
		CreatedAt     string `json:"createdAt"`
	}
	items := make([]item, 0, len(tokens))
	for _, t := range tokens {
		items = append(items, item{
			Token: t.Token, ShareURL: fmt.Sprintf("/share/%s", t.Token),
			Description: t.Description, ShowPositions: t.ShowPositions,
			ViewCount: t.ViewCount,
			ExpiresAt: t.ExpiresAt.Format(time.RFC3339),
			CreatedAt: t.CreatedAt.Format(time.RFC3339),
		})
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(items)
}

// HandleListAllShareTokens returns all share tokens (admin only).
func (s *ShareServer) HandleListAllShareTokens(w http.ResponseWriter, r *http.Request) {
	page := 1
	pageSize := 20
	if v := r.URL.Query().Get("page"); v != "" { fmt.Sscanf(v, "%d", &page) }
	if v := r.URL.Query().Get("pageSize"); v != "" { fmt.Sscanf(v, "%d", &pageSize) }
	limit, offset := pageSize, (page-1)*pageSize
	if limit > 100 { limit = 100 }

	tokens, total, err := s.repo.ListAll(r.Context(), limit, offset)
	if err != nil {
		s.log.Error("ListAllShareTokens: db", zap.Error(err))
		http.Error(w, `{"error":"internal"}`, http.StatusInternalServerError)
		return
	}
	type item struct {
		Token         string `json:"token"`
		ShareURL      string `json:"shareUrl"`
		UserID        string `json:"userId"`
		Description   string `json:"description"`
		ShowPositions bool   `json:"showPositions"`
		ViewCount     int    `json:"viewCount"`
		ExpiresAt     string `json:"expiresAt"`
		CreatedAt     string `json:"createdAt"`
	}
	items := make([]item, 0, len(tokens))
	for _, t := range tokens {
		items = append(items, item{
			Token: t.Token, ShareURL: fmt.Sprintf("/share/%s", t.Token),
			UserID: t.UserID.String(), Description: t.Description,
			ShowPositions: t.ShowPositions, ViewCount: t.ViewCount,
			ExpiresAt: t.ExpiresAt.Format(time.RFC3339),
			CreatedAt: t.CreatedAt.Format(time.RFC3339),
		})
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"items": items, "total": total, "page": page, "pageSize": pageSize,
	})
}

// HandleDeleteShareToken deletes a share token owned by the current user.
func (s *ShareServer) HandleDeleteShareToken(w http.ResponseWriter, r *http.Request) {
	uid, err := s.parseAuthUser(r)
	if err != nil {
		http.Error(w, `{"error":"missing authorization header"}`, http.StatusUnauthorized)
		return
	}
	var req struct{ Token string `json:"token"` }
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Token == "" {
		http.Error(w, `{"error":"invalid request"}`, http.StatusBadRequest)
		return
	}
	if err := s.repo.DeleteByUser(r.Context(), uid, req.Token); err != nil {
		s.log.Error("DeleteShareToken: db", zap.Error(err))
		http.Error(w, `{"error":"delete failed"}`, http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"ok":true}`))
}

package admin

import (
	"context"
	"encoding/json"
	"net/http"

	"go.uber.org/zap"

	"anttrader/internal/controlplane"
	"anttrader/internal/interceptor"
)

// SREHandler exposes the control-plane over plain HTTP for the admin UI.
type SREHandler struct {
	killSwitch *controlplane.KillSwitch
	breakers   *controlplane.BreakerRegistry
	canary     *controlplane.CanaryManager
	checker    interceptor.AdminChecker
	log        *zap.Logger
}

// NewSREHandler creates an SREHandler wired to the given control-plane components.
func NewSREHandler(
	ks *controlplane.KillSwitch,
	br *controlplane.BreakerRegistry,
	cm *controlplane.CanaryManager,
	checker interceptor.AdminChecker,
	log *zap.Logger,
) *SREHandler {
	return &SREHandler{killSwitch: ks, breakers: br, canary: cm, checker: checker, log: log}
}

func (h *SREHandler) requireAdmin(w http.ResponseWriter, r *http.Request, ai *interceptor.AuthInterceptor) bool {
	uid, err := ai.UserIDFromHTTP(r)
	if err != nil {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return false
	}
	ok, err := h.checker.IsAdmin(context.Background(), uid)
	if err != nil || !ok {
		http.Error(w, `{"error":"forbidden"}`, http.StatusForbidden)
		return false
	}
	return true
}

func writeJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

// --- Kill Switch ---

// HandleKillSwitchStatus returns the current kill-switch state.
func (h *SREHandler) HandleKillSwitchStatus(w http.ResponseWriter, r *http.Request, ai *interceptor.AuthInterceptor) {
	if !h.requireAdmin(w, r, ai) {
		return
	}
	if r.Method != http.MethodGet {
		http.Error(w, `{"error":"method not allowed"}`, http.StatusMethodNotAllowed)
		return
	}
	writeJSON(w, 200, h.killSwitch.Status())
}

// HandleKillSwitchEngage arms the kill switch.
func (h *SREHandler) HandleKillSwitchEngage(w http.ResponseWriter, r *http.Request, ai *interceptor.AuthInterceptor) {
	if !h.requireAdmin(w, r, ai) {
		return
	}
	if r.Method != http.MethodPost {
		http.Error(w, `{"error":"method not allowed"}`, http.StatusMethodNotAllowed)
		return
	}
	var body struct {
		Reason   string `json:"reason"`
		Operator string `json:"operator"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, `{"error":"invalid body"}`, http.StatusBadRequest)
		return
	}
	if body.Reason == "" {
		http.Error(w, `{"error":"reason is required"}`, http.StatusBadRequest)
		return
	}
	h.killSwitch.Engage(body.Reason, body.Operator)
	h.log.Warn("kill switch engaged", zap.String("reason", body.Reason), zap.String("operator", body.Operator))
	writeJSON(w, 200, h.killSwitch.Status())
}

// HandleKillSwitchDisengage disarms the kill switch.
func (h *SREHandler) HandleKillSwitchDisengage(w http.ResponseWriter, r *http.Request, ai *interceptor.AuthInterceptor) {
	if !h.requireAdmin(w, r, ai) {
		return
	}
	if r.Method != http.MethodPost {
		http.Error(w, `{"error":"method not allowed"}`, http.StatusMethodNotAllowed)
		return
	}
	h.killSwitch.Disengage()
	h.log.Warn("kill switch disengaged")
	writeJSON(w, 200, h.killSwitch.Status())
}

// --- Strategy Breakers ---

// HandleBreakersList returns all breaker statuses.
func (h *SREHandler) HandleBreakersList(w http.ResponseWriter, r *http.Request, ai *interceptor.AuthInterceptor) {
	if !h.requireAdmin(w, r, ai) {
		return
	}
	writeJSON(w, 200, h.breakers.List())
}

// HandleBreakerReset resets a specific breaker.
func (h *SREHandler) HandleBreakerReset(w http.ResponseWriter, r *http.Request, ai *interceptor.AuthInterceptor) {
	if !h.requireAdmin(w, r, ai) {
		return
	}
	if r.Method != http.MethodPost {
		http.Error(w, `{"error":"method not allowed"}`, http.StatusMethodNotAllowed)
		return
	}
	strategyID := r.URL.Query().Get("strategy_id")
	if strategyID == "" {
		http.Error(w, `{"error":"strategy_id is required"}`, http.StatusBadRequest)
		return
	}
	h.breakers.Reset(strategyID)
	h.log.Info("breaker reset", zap.String("strategy_id", strategyID))
	writeJSON(w, 200, map[string]string{"status": "reset", "strategy_id": strategyID})
}

// --- Canary ---

// HandleCanaryList returns all canary configs.
func (h *SREHandler) HandleCanaryList(w http.ResponseWriter, r *http.Request, ai *interceptor.AuthInterceptor) {
	if !h.requireAdmin(w, r, ai) {
		return
	}
	writeJSON(w, 200, h.canary.List())
}

// HandleCanarySet creates or updates a canary config.
func (h *SREHandler) HandleCanarySet(w http.ResponseWriter, r *http.Request, ai *interceptor.AuthInterceptor) {
	if !h.requireAdmin(w, r, ai) {
		return
	}
	if r.Method != http.MethodPost {
		http.Error(w, `{"error":"method not allowed"}`, http.StatusMethodNotAllowed)
		return
	}
	var cfg controlplane.CanaryConfig
	if err := json.NewDecoder(r.Body).Decode(&cfg); err != nil {
		http.Error(w, `{"error":"invalid body"}`, http.StatusBadRequest)
		return
	}
	if cfg.StrategyID == "" {
		http.Error(w, `{"error":"strategy_id is required"}`, http.StatusBadRequest)
		return
	}
	h.canary.Set(cfg)
	h.log.Info("canary config set", zap.String("strategy_id", cfg.StrategyID), zap.String("version", cfg.VersionTag))
	writeJSON(w, 200, cfg)
}

// HandleCanaryDelete removes a canary config.
func (h *SREHandler) HandleCanaryDelete(w http.ResponseWriter, r *http.Request, ai *interceptor.AuthInterceptor) {
	if !h.requireAdmin(w, r, ai) {
		return
	}
	if r.Method != http.MethodPost {
		http.Error(w, `{"error":"method not allowed"}`, http.StatusMethodNotAllowed)
		return
	}
	strategyID := r.URL.Query().Get("strategy_id")
	if strategyID == "" {
		http.Error(w, `{"error":"strategy_id is required"}`, http.StatusBadRequest)
		return
	}
	h.canary.Remove(strategyID)
	h.log.Info("canary config removed", zap.String("strategy_id", strategyID))
	writeJSON(w, 200, map[string]string{"status": "removed", "strategy_id": strategyID})
}

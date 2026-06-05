package admin

import (
	"context"
	"time"
	"encoding/json"
	"net/http"

	"go.uber.org/zap"

	"anttrader/internal/controlplane"
	"anttrader/internal/interceptor"
	"connectrpc.com/connect"
	antv1 "anttrader/gen/proto/ant/v1"
	"anttrader/internal/notifier"
)

// SREHandler exposes the control-plane over plain HTTP for the admin UI.
type SREHandler struct {
	killSwitch *controlplane.KillSwitch
	breakers   *controlplane.BreakerRegistry
	canary     *controlplane.CanaryManager
	checker    interceptor.AdminChecker
	notifier   *notifier.EmailNotifier
	log        *zap.Logger
}

// NewSREHandler creates an SREHandler wired to the given control-plane components.
func NewSREHandler(
	ks *controlplane.KillSwitch,
	br *controlplane.BreakerRegistry,
	cm *controlplane.CanaryManager,
	checker interceptor.AdminChecker,
	n *notifier.EmailNotifier,
	log *zap.Logger,
) *SREHandler {
	return &SREHandler{killSwitch: ks, breakers: br, canary: cm, checker: checker, notifier: n, log: log}
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
		if h.notifier != nil {
			h.notifier.KillSwitchAlert(body.Reason, body.Operator)
		}
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

// ── ConnectRPC handlers (push-first, proto binary) ──

func (h *SREHandler) GetKillSwitch(ctx context.Context, req *connect.Request[antv1.GetKillSwitchRequest]) (*connect.Response[antv1.KillSwitchStatus], error) {
	s := h.killSwitch.Status()
	var engagedAtMs int64
	if t, err := time.Parse(time.RFC3339, s.EngagedAt); err == nil { engagedAtMs = t.UnixMilli() }
	return connect.NewResponse(&antv1.KillSwitchStatus{
		Enabled: s.Engaged, Reason: s.Reason, SetAtUnixMs: engagedAtMs, SetBy: s.Operator,
	}), nil
}

func (h *SREHandler) SetKillSwitch(ctx context.Context, req *connect.Request[antv1.SetKillSwitchRequest]) (*connect.Response[antv1.KillSwitchStatus], error) {
	m := req.Msg
	if m.GetEnabled() {
		h.killSwitch.Engage(m.GetReason(), "admin")
	} else {
		h.killSwitch.Disengage()
	}
	return h.GetKillSwitch(ctx, &connect.Request[antv1.GetKillSwitchRequest]{})
}

func (h *SREHandler) ListBreakers(ctx context.Context, req *connect.Request[antv1.ListBreakersRequest]) (*connect.Response[antv1.ListBreakersResponse], error) {
	list := h.breakers.List()
	out := make([]*antv1.BreakerStatus, 0, len(list))
	for _, b := range list {
		out = append(out, &antv1.BreakerStatus{
			Name: b.StrategyID,
			Open: b.State == controlplane.BreakerOpen || b.State == controlplane.BreakerHalfOpen,
			FailureCount: int32(b.TradeCount),
		})
	}
	return connect.NewResponse(&antv1.ListBreakersResponse{Breakers: out}), nil
}

func (h *SREHandler) ResetBreaker(ctx context.Context, req *connect.Request[antv1.ResetBreakerRequest]) (*connect.Response[antv1.BreakerStatus], error) {
	h.breakers.Reset(req.Msg.GetName())
	return connect.NewResponse(&antv1.BreakerStatus{Name: req.Msg.GetName(), Open: false}), nil
}

func (h *SREHandler) GetCanary(ctx context.Context, req *connect.Request[antv1.GetCanaryRequest]) (*connect.Response[antv1.CanaryStatus], error) {
	c := h.canary.List()
	if len(c) > 0 {
		first := c[0]
		dur := int32(first.DurationDays)
		if dur == 0 { dur = 7 }
		return connect.NewResponse(&antv1.CanaryStatus{
			Enabled: true, TrafficPercent: dur, TargetVersion: first.VersionTag,
		}), nil
	}
	return connect.NewResponse(&antv1.CanaryStatus{}), nil
}

func (h *SREHandler) SetCanary(ctx context.Context, req *connect.Request[antv1.SetCanaryRequest]) (*connect.Response[antv1.CanaryStatus], error) {
	m := req.Msg
	h.canary.Set(controlplane.CanaryConfig{
		VersionTag: m.GetTargetVersion(),
		DurationDays: int(m.GetTrafficPercent()),
	})
	return h.GetCanary(ctx, &connect.Request[antv1.GetCanaryRequest]{})
}

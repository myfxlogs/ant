package admin

import (
	"context"
	"time"

	"go.uber.org/zap"

	"anttrader/internal/controlplane"
	"anttrader/internal/interceptor"
	"connectrpc.com/connect"
	antv1 "anttrader/gen/proto/ant/v1"
	"anttrader/internal/notifier"
)

// SREHandler exposes the control-plane over ConnectRPC for the admin UI.
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

// ── ConnectRPC handlers ──

func (h *SREHandler) GetKillSwitch(ctx context.Context, req *connect.Request[antv1.GetKillSwitchRequest]) (*connect.Response[antv1.KillSwitchStatus], error) {
	s := h.killSwitch.Status()
	var engagedAtMs int64
	if t, err := time.Parse(time.RFC3339, s.EngagedAt); err == nil {
		engagedAtMs = t.UnixMilli()
	}
	return connect.NewResponse(&antv1.KillSwitchStatus{
		Enabled: s.Engaged, Reason: s.Reason, SetAtUnixMs: engagedAtMs, SetBy: s.Operator,
	}), nil
}

func (h *SREHandler) SetKillSwitch(ctx context.Context, req *connect.Request[antv1.SetKillSwitchRequest]) (*connect.Response[antv1.KillSwitchStatus], error) {
	m := req.Msg
	if m.GetEnabled() {
		h.killSwitch.Engage(m.GetReason(), "admin")
		h.log.Warn("kill switch engaged", zap.String("reason", m.GetReason()), zap.String("operator", "admin"))
		if h.notifier != nil {
			h.notifier.KillSwitchAlert(m.GetReason(), "admin")
		}
	} else {
		h.killSwitch.Disengage()
		h.log.Warn("kill switch disengaged")
	}
	return h.GetKillSwitch(ctx, &connect.Request[antv1.GetKillSwitchRequest]{})
}

func (h *SREHandler) ListBreakers(ctx context.Context, req *connect.Request[antv1.ListBreakersRequest]) (*connect.Response[antv1.ListBreakersResponse], error) {
	list := h.breakers.List()
	out := make([]*antv1.BreakerStatus, 0, len(list))
	for _, b := range list {
		out = append(out, &antv1.BreakerStatus{
			Name:         b.StrategyID,
			Open:         b.State == controlplane.BreakerOpen || b.State == controlplane.BreakerHalfOpen,
			FailureCount: int32(b.TradeCount),
		})
	}
	return connect.NewResponse(&antv1.ListBreakersResponse{Breakers: out}), nil
}

func (h *SREHandler) ResetBreaker(ctx context.Context, req *connect.Request[antv1.ResetBreakerRequest]) (*connect.Response[antv1.BreakerStatus], error) {
	h.breakers.Reset(req.Msg.GetName())
	h.log.Info("breaker reset", zap.String("strategy_id", req.Msg.GetName()))
	return connect.NewResponse(&antv1.BreakerStatus{Name: req.Msg.GetName(), Open: false}), nil
}

func (h *SREHandler) GetCanary(ctx context.Context, req *connect.Request[antv1.GetCanaryRequest]) (*connect.Response[antv1.CanaryStatus], error) {
	c := h.canary.List()
	if len(c) > 0 {
		first := c[0]
		dur := int32(first.DurationDays)
		if dur == 0 {
			dur = 7
		}
		return connect.NewResponse(&antv1.CanaryStatus{
			Enabled: true, TrafficPercent: dur, TargetVersion: first.VersionTag,
		}), nil
	}
	return connect.NewResponse(&antv1.CanaryStatus{}), nil
}

func (h *SREHandler) SetCanary(ctx context.Context, req *connect.Request[antv1.SetCanaryRequest]) (*connect.Response[antv1.CanaryStatus], error) {
	m := req.Msg
	h.canary.Set(controlplane.CanaryConfig{
		VersionTag:   m.GetTargetVersion(),
		DurationDays: int(m.GetTrafficPercent()),
	})
	h.log.Info("canary config set", zap.String("strategy_id", m.GetTargetVersion()), zap.String("version", m.GetTargetVersion()))
	return h.GetCanary(ctx, &connect.Request[antv1.GetCanaryRequest]{})
}

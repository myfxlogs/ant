// Package service — OMS PreSubmit risk hook (M3-4).
package service

import (
	"context"

	"anttrader/internal/risksvc"
)

// OMSPreSubmitHook evaluates the risksvc.Engine before OMS order submission.
// This is the bridge between the M3 risk engine and the M2 OMS BrokerAdapter path.
// Intended to be called by TradingService.OrderSend before invoking orderSendViaOMS.
type OMSPreSubmitHook struct {
	engine *risksvc.Engine
}

// NewOMSPreSubmitHook creates a hook backed by the given risk engine.
func NewOMSPreSubmitHook(engine *risksvc.Engine) *OMSPreSubmitHook {
	return &OMSPreSubmitHook{engine: engine}
}

// Evaluate runs the risk engine on the order request.
// Returns nil if all rules pass, or the blocking CheckResult.
func (h *OMSPreSubmitHook) Evaluate(ctx context.Context, req *risksvc.CheckRequest) *risksvc.CheckResult {
	if h == nil || h.engine == nil {
		return &risksvc.CheckResult{Passed: true, Rule: "no_engine"}
	}
	return h.engine.Evaluate(ctx, req)
}

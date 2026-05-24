// V1-LEGACY: will be replaced by M7.1-7.4 cards. Do not extend; new code goes alongside.
// Package quantengine — signal-to-OMS bridge.
//
// Bridges strategy signals (from DSL or ONNX) to OMS order submission
// with risk checks, signal_id audit tracking, and broker dispatch.
package quantengine

import (
	"context"
	"fmt"

	"anttrader/internal/oms"
	"anttrader/internal/risksvc"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

// SignalToOMS creates a SignalHandler that converts signals to OMS orders.
//
// The returned handler performs:
//  1. Generate a unique signal_id for audit tracking
//  2. Build an oms.OrderRequest with the canonical symbol
//  3. Run risk checks via the risk engine
//  4. Submit via the BrokerAdapter
//
// Symbol resolution (canonical → broker_symbol_raw) is handled by the
// resolveSymbol callback before submitting to the broker adapter.
func SignalToOMS(
	broker oms.BrokerAdapter,
	risk *risksvc.Engine,
	accountID string,
	resolveSymbol func(canonical string) (string, error),
	log *zap.Logger,
) SignalHandler {
	return func(strategyID, canonical, side string, qty float64, reason string) {
		signalID := uuid.New().String()

		// Convert direction to OMS side
		omsSide := directionToSide(side)
		if omsSide == "" {
			log.Warn("unknown signal direction",
				zap.String("signal_id", signalID),
				zap.String("side", side),
			)
			return
		}

		// Resolve broker symbol
		brokerSymbolRaw, err := resolveSymbol(canonical)
		if err != nil {
			log.Warn("symbol resolution failed",
				zap.String("signal_id", signalID),
				zap.String("canonical", canonical),
				zap.Error(err),
			)
			return
		}

		// ── Risk check ──
		if risk != nil {
			riskResult := risk.Evaluate(context.Background(), &risksvc.CheckRequest{
				AccountID: accountID,
				Symbol:    canonical,
				Side:      omsSide,
				Volume:    qty,
			})
			if !riskResult.Passed {
				log.Warn("risk check blocked",
					zap.String("signal_id", signalID),
					zap.String("rule", riskResult.Rule),
					zap.String("reason", riskResult.Reason),
				)
				return
			}
		}

		// ── Build order request ──
		req := &oms.OrderRequest{
			AccountID:       accountID,
			Symbol:          canonical,
			BrokerSymbolRaw: brokerSymbolRaw,
			Side:            omsSide,
			Volume:          qty,
			StrategyID:      strategyID,
			Comment:         fmt.Sprintf("signal_id=%s reason=%s", signalID, reason),
		}

		// ── Submit to broker ──
		resp, err := broker.Submit(context.Background(), req)
		if err != nil {
			log.Warn("order submit failed",
				zap.String("signal_id", signalID),
				zap.String("canonical", canonical),
				zap.String("side", omsSide),
				zap.Error(err),
			)
			return
		}

		log.Info("order submitted",
			zap.String("signal_id", signalID),
			zap.String("ticket", resp.Ticket),
			zap.String("canonical", canonical),
			zap.String("symbol_raw", brokerSymbolRaw),
			zap.String("side", omsSide),
			zap.Float64("qty", qty),
			zap.String("state", string(resp.State)),
		)
	}
}

// directionToSide maps quantengine direction to OMS side string.
func directionToSide(dir string) string {
	switch dir {
	case "long", "buy":
		return "buy"
	case "short", "sell":
		return "sell"
	default:
		return ""
	}
}

// DefaultSymbolResolver returns a pass-through resolver.
// Symbol resolution is handled externally; this resolver assumes pre-resolved symbols.
func DefaultSymbolResolver() func(canonical string) (string, error) {
	return func(canonical string) (string, error) {
		return canonical, nil
	}
}

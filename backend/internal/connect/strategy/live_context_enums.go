package strategy

import (
	"fmt"

	"alphaforge/internal/mthub"
)

// This file contains enum normalization helpers for live trade contexts.
// VM-TRADE-CONTEXT-6 round 5: unknown enum values must fail-closed,
// not default to "buy"/"fill"/"sell" (which would silently change trade semantics).

// brokerSideFromString validates a broker side string. VM-TRADE-CONTEXT-6
// round 5: unknown side must fail-closed, not default to "buy".
func brokerSideFromString(s string) (string, error) {
	switch s {
	case sideBuy, sideSell:
		return s, nil
	case "":
		return "", fmt.Errorf("broker side is empty")
	default:
		return "", fmt.Errorf("unknown broker side %q", s)
	}
}

// brokerTradeEventTypeString converts a BrokerTradeEventType to the proto
// event_type string. VM-TRADE-CONTEXT-6 round 5: unknown event type must
// fail-closed, not default to "fill".
func brokerTradeEventTypeString(t mthub.BrokerTradeEventType) (string, error) {
	switch t {
	case mthub.BrokerTradeFilled:
		return "fill", nil
	case mthub.BrokerTradeClosed:
		return "close", nil
	case mthub.BrokerTradeModified:
		return "modify", nil
	case mthub.BrokerTradeCancelled:
		return "cancel", nil
	default:
		return "", fmt.Errorf("unknown broker trade event type %d", t)
	}
}

// pendingOrderSide derives the buy/sell side from a pending order type string.
// "buy_limit" → "buy", "sell_stop" → "sell", etc.
// VM-TRADE-CONTEXT-6 round 5: unknown order type must fail-closed, not default
// to "sell" (which would silently change trade semantics).
func pendingOrderSide(orderType string) (string, error) {
	switch orderType {
	case "buy_limit", "buy_stop", "buy_stop_limit", "buy_market":
		return sideBuy, nil
	case "sell_limit", "sell_stop", "sell_stop_limit", "sell_market":
		return sideSell, nil
	case "":
		return "", fmt.Errorf("pending order type is empty")
	default:
		return "", fmt.Errorf("unknown pending order type %q", orderType)
	}
}

package main

import "testing"

// ACCOUNT-TRADE-ALLOWED-DEAD-1: the lifecycle never writes "trade_allowed" —
// a connected master-password session is the truthful trade-capable state.
// The predicate must accept connected (orders provably reach the broker) and
// still accept the reserved trade_allowed value; disconnected/reconnecting/
// empty must fail closed.
//
// Adversarial (M1): reverting to `status == "trade_allowed"` alone makes the
// "connected" case RED — that is exactly the dead-state bug shipped to prod.
func TestMTAccountTradeAllowedStatus(t *testing.T) {
	cases := []struct {
		status string
		want   bool
	}{
		{"connected", true},
		{"trade_allowed", true},
		{"disconnected", false},
		{"reconnecting", false},
		{"binding", false},
		{"", false},
	}
	for _, c := range cases {
		if got := mtAccountTradeAllowedStatus(c.status); got != c.want {
			t.Errorf("mtAccountTradeAllowedStatus(%q) = %v, want %v", c.status, got, c.want)
		}
	}
}

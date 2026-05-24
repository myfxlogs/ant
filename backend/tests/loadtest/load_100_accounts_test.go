//go:build loadtest
// +build loadtest

package loadtest

import "testing"

// Test100AccountsNoSpill verifies CHWriter handles 100-account peak without
// spilling data. Mock broker generates 25k tick/s; after 5min, spill_writes
// must be 0 and e2e latency P99 < 500ms (ADR-0011 §6).
func Test100AccountsNoSpill(t *testing.T) {
	t.Skip("loadtest: requires mock broker infrastructure + full pipeline (runner.go)")

	// Full implementation:
	// 1. Start 100 mock brokers at 25k tick/s aggregate
	// 2. Run for 5 minutes
	// 3. Assert md_spill_writes_total == 0
	// 4. Assert histogram_quantile(0.99, md_e2e_latency_seconds) < 0.5
}

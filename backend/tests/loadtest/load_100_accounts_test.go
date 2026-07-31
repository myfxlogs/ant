//go:build loadtest
// +build loadtest

package loadtest

import (
	"os"
	"testing"
)

// Test100AccountsNoSpill (M10.5-12): 100 mock brokers × 250 tick/s × 5min.
// ADR-0012: ClickHouse removed, test disabled.
func Test100AccountsNoSpill(t *testing.T) {
	t.Parallel()
	t.Skip("ADR-0012: ClickHouse removed, tick persistence disabled")
}

func envOr(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}

//go:build e2e
// +build e2e

package e2e_test

import (
	"os"
	"testing"
)

func TestE2ESmoke(t *testing.T) {
	t.Parallel()
	t.Skip("ADR-0012: ClickHouse removed, tick persistence disabled")
}

func envOr(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}

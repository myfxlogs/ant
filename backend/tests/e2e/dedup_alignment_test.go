//go:build e2e
// +build e2e

package e2e_test

import (
	"os"
	"testing"
)

func TestDedupAlignment(t *testing.T) {
	t.Parallel()
	t.Skip("ADR-0012: ClickHouse removed, tick persistence disabled")
}

func envOrDefault(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

package marketplace

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// mustReadFile reads a file relative to the backend root for source-text assertions.
// marketplace test wd = .../backend/internal/marketplace → backendRoot = wd/../..
func mustReadFileTrust(t *testing.T, relPath string) string {
	t.Helper()
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	backendRoot := filepath.Join(wd, "..", "..")
	data, err := os.ReadFile(filepath.Join(backendRoot, relPath))
	if err != nil {
		t.Fatalf("read %s: %v", relPath, err)
	}
	return string(data)
}

// T5: TestLinkLiveAccount_RejectsDemo guards S5 (LinkLiveAccount rejects non-real
// accounts). RED mutation: delete the account_type check → RED.
func TestLinkLiveAccount_RejectsDemo(t *testing.T) {
	content := mustReadFileTrust(t, "internal/marketplace/live_performance.go")
	idx := strings.Index(content, "func (s *Service) LinkLiveAccount")
	if idx < 0 {
		t.Fatal("LinkLiveAccount not found")
	}
	body := content[idx:]
	if !strings.Contains(body, "only real accounts can link") {
		t.Fatal("LinkLiveAccount must reject non-real accounts (S5)")
	}
}

// T6: TestLeaderboard_FiltersRealOnly guards S6e (leaderboard return query filters
// account_type = 'real'). RED mutation: delete the filter → RED.
func TestLeaderboard_FiltersRealOnly(t *testing.T) {
	content := mustReadFileTrust(t, "internal/marketplace/leaderboard.go")
	if !strings.Contains(content, "lps.account_type = 'real'") {
		t.Fatal("leaderboard return query must filter account_type = 'real' (S6e)")
	}
}

// T7: TestOnProfitUpdate_SkipsDemo guards S6b (OnProfitUpdate skips non-real accounts).
// RED mutation: delete the `entry.AccountType != "real"` check → RED.
func TestOnProfitUpdate_SkipsDemo(t *testing.T) {
	content := mustReadFileTrust(t, "internal/marketplace/live_performance.go")
	idx := strings.Index(content, "func (c *LivePerformanceCollector) OnProfitUpdate")
	if idx < 0 {
		t.Fatal("OnProfitUpdate not found")
	}
	body := content[idx:]
	if !strings.Contains(body, "AccountType") || !strings.Contains(body, `"real"`) {
		t.Fatal("OnProfitUpdate must check AccountType == 'real' (S6b)")
	}
}

// T7b: TestUpsertDailyPerformance_WritesAccountType guards S6c (INSERT writes
// account_type column). RED mutation: remove account_type from INSERT → RED.
func TestUpsertDailyPerformance_WritesAccountType(t *testing.T) {
	content := mustReadFileTrust(t, "internal/marketplace/live_performance.go")
	idx := strings.Index(content, "func (s *Service) UpsertDailyPerformance")
	if idx < 0 {
		t.Fatal("UpsertDailyPerformance not found")
	}
	// Bound to just this function (up to the next top-level func).
	body := content[idx:]
	if next := strings.Index(body[1:], "\nfunc "); next >= 0 {
		body = body[:next+1]
	}
	// The INSERT column list must include account_type.
	if !strings.Contains(body, "winning_trades, account_type)") {
		t.Fatal("UpsertDailyPerformance INSERT must include account_type column (S6c)")
	}
}

package connect

import (
	"os"
	"sync"
	"time"
)

const (
	SessionStatusConnecting   = "CONNECTING"
	SessionStatusSyncing      = "SYNCING"
	SessionStatusRunning      = "RUNNING"
	SessionStatusDegraded     = "DEGRADED"
	SessionStatusDisconnected = "DISCONNECTED"
	SessionStatusMarketClosed = "MARKET_CLOSED"
)

func isProbablyMarketClosed(t time.Time) bool {
	wd := t.Weekday()
	return wd == time.Saturday || wd == time.Sunday
}

var (
	weekendHardOverrideOnce sync.Once
	weekendHardOverrideDur  = 30 * time.Minute
)

func weekendHardOverride() time.Duration {
	weekendHardOverrideOnce.Do(func() {
		if v := os.Getenv("ANTRADER_WEEKEND_HARD_OVERRIDE"); v != "" {
			if d, err := time.ParseDuration(v); err == nil && d > 0 {
				weekendHardOverrideDur = d
			}
		}
	})
	return weekendHardOverrideDur
}

func shouldSuppressHardRestart(now time.Time, profitStale time.Duration, orderStale time.Duration) bool {
	if !isProbablyMarketClosed(now) {
		return false
	}
	thr := weekendHardOverride()
	return profitStale < thr && orderStale < thr
}

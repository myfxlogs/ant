// V1-LEGACY: will be replaced by M7.1-7.4 cards. Do not extend; new code goes alongside.
package mthub

// Metrics integration is stubbed.
// TODO: wire prometheus (or ant's internal/metrics) when needed.
// The original alfq mthub/metrics.go used promauto gauges/counters/histograms
// in an init() function; that has been removed to avoid a hard dependency.

// recordActiveSessions is a no-op stub. Replace with real metrics when ready.
func recordActiveSessions(active map[string]int) {
	// no-op
}

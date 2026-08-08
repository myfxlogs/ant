package marketplace

import (
	"testing"
	"time"
)

// Adversarial proof: shouldScan returns false within 24h of last scan,
// true after 24h. Delete the throttle check line and this test fails red.
func TestDecayMonitor_shouldScan_throttle(t *testing.T) {
	m := &DecayMonitor{
		throttle: make(map[string]time.Time),
	}

	// First scan: always allowed.
	if !m.shouldScan("s1") {
		t.Fatal("first scan should be allowed")
	}

	// Mark scanned now.
	m.markScanned("s1")

	// Within 24h: should be blocked.
	if m.shouldScan("s1") {
		t.Fatal("scan within 24h should be throttled")
	}

	// Simulate 25h later.
	m.mu.Lock()
	m.throttle["s1"] = time.Now().Add(-25 * time.Hour)
	m.mu.Unlock()

	// After 24h: should be allowed.
	if !m.shouldScan("s1") {
		t.Fatal("scan after 24h should be allowed")
	}
}

// Adversarial proof: markScanned records the time. Remove the map write
// and shouldScan will always return true (throttle broken).
func TestDecayMonitor_markScanned_recordsTime(t *testing.T) {
	m := &DecayMonitor{
		throttle: make(map[string]time.Time),
	}

	m.markScanned("s2")

	m.mu.Lock()
	_, exists := m.throttle["s2"]
	m.mu.Unlock()

	if !exists {
		t.Fatal("markScanned should record time in throttle map")
	}
}

// Adversarial proof: different strategies have independent throttle.
func TestDecayMonitor_throttle_perStrategy(t *testing.T) {
	m := &DecayMonitor{
		throttle: make(map[string]time.Time),
	}

	m.markScanned("s1")

	// s1 is throttled, s2 is not.
	if m.shouldScan("s1") {
		t.Fatal("s1 should be throttled")
	}
	if !m.shouldScan("s2") {
		t.Fatal("s2 should not be throttled")
	}
}

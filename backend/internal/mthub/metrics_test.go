package mthub

import (
	"testing"
)

func TestRecordActiveSessions(t *testing.T) {
	// This is a no-op stub; just ensure it doesn't panic.
	active := map[string]int{
		"mt4": 5,
		"mt5": 10,
	}
	recordActiveSessions(active)
}

func TestRecordActiveSessions_Empty(t *testing.T) {
	active := map[string]int{}
	recordActiveSessions(active) // Should not panic
}

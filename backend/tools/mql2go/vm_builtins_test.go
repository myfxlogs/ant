package mql2go

import (
	"testing"
)

// permanentBlindSpots are builtins that intentionally have nil fn.
// They are Object/Chart/File operations that have no meaning in backtest.
var permanentBlindSpots = map[string]bool{
	"ObjectCreate": true, "ObjectDelete": true, "ObjectSet": true,
	"ObjectGet": true, "ObjectSetText": true, "ObjectsTotal": true,
	"ObjectFind": true, "ObjectName": true, "ObjectGetType": true,
	"ChartApplyTemplate": true,
	"FileOpen": true, "FileClose": true, "FileWrite": true,
	"FileRead": true, "FileDelete": true, "FileIsEnding": true,
	"FileSeek": true, "FileTell": true, "FileFlush": true,
	"FileSize": true,
}

// TestAllBuiltinsWired verifies that every entry in builtinRegistry has a non-nil fn,
// except for permanent blind spots (Object/Chart/File).
func TestAllBuiltinsWired(t *testing.T) {
	var unwired []string
	for _, entry := range builtinRegistry {
		if entry.fn == nil && !permanentBlindSpots[entry.name] {
			unwired = append(unwired, entry.name)
		}
	}
	if len(unwired) > 0 {
		t.Errorf("%d builtins have nil fn (not wired, not blind spot): %v", len(unwired), unwired)
	}
}

// TestNoDuplicateBuiltins verifies no duplicate names in builtinRegistry.
func TestNoDuplicateBuiltins(t *testing.T) {
	seen := map[string]int{}
	for _, entry := range builtinRegistry {
		seen[entry.name]++
	}
	var dups []string
	for name, count := range seen {
		if count > 1 {
			dups = append(dups, name)
		}
	}
	if len(dups) > 0 {
		t.Errorf("duplicate builtin names: %v", dups)
	}
}

// TestBuiltinCount verifies the total number of registered builtins.
func TestBuiltinCount(t *testing.T) {
	total := len(builtinRegistry)
	wired := 0
	for _, entry := range builtinRegistry {
		if entry.fn != nil {
			wired++
		}
	}
	t.Logf("builtinRegistry: %d total, %d wired, %d blind spots", total, wired, total-wired)
	if total < 300 {
		t.Errorf("expected at least 300 builtins, got %d", total)
	}
}


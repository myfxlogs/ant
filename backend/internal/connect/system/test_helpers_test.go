package system

import (
	"os"
	"path/filepath"
	"testing"
)

// mustReadFile reads a file relative to the backend module root.
// The test files live in internal/connect/system/, so the backend root
// is ../../ from the test package directory.
func mustReadFile(t *testing.T, relPath string) string {
	t.Helper()
	// Find the backend module root by walking up from the test file.
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	// wd is .../backend/internal/connect/system → backend root is ../../..
	backendRoot := filepath.Join(wd, "..", "..", "..")
	full := filepath.Join(backendRoot, relPath)
	data, err := os.ReadFile(full)
	if err != nil {
		t.Fatalf("read %s: %v", full, err)
	}
	return string(data)
}

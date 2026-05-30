package logger

import (
	"testing"
)

func TestGetLevel(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{"debug", "debug"},
		{"info", "info"},
		{"warn", "warn"},
		{"error", "error"},
		{"default", ""},
		{"unknown", "unknown"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_ = getLevel(tt.input) // must not panic
		})
	}
}

func TestGetEncoder(t *testing.T) {
	_ = getEncoder("json")    // must not panic
	_ = getEncoder("console") // must not panic
	_ = getEncoder("")        // default
}

func TestGetWriteSyncer(t *testing.T) {
	_ = getWriteSyncer("stdout")
	_ = getWriteSyncer("stderr")
	_ = getWriteSyncer("")
}

func TestInit(t *testing.T) {
	err := Init(&Config{Level: "debug", Format: "console", Output: "stdout"})
	if err != nil {
		t.Logf("Init returned: %v (may be already initialized)", err)
	}
	logger := Get()
	if logger == nil {
		t.Error("Get() returned nil after Init")
	}
}

func TestGet_DefaultInit(t *testing.T) {
	logger := Get()
	if logger == nil {
		t.Fatal("Get() returned nil")
	}
	// Should not panic on any log level.
	logger.Debug("test debug")
	logger.Info("test info")
	logger.Warn("test warn")
	logger.Error("test error")
}

func TestSync(t *testing.T) {
	_ = Init(&Config{Level: "info", Format: "json", Output: "stdout"})
	if err := Sync(); err != nil {
		t.Logf("Sync returned: %v", err)
	}
}

func TestDebugInfoWarnError(t *testing.T) {
	// These should not panic even if logger is not initialized.
	Debug("debug msg")
	Info("info msg")
	Warn("warn msg")
	Error("error msg")
}

func TestWith(t *testing.T) {
	l := With()
	if l == nil {
		t.Error("With() returned nil")
	}
}

package user

import (
	"errors"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"
)

// T6: buildShareDecayStatusQuery returns SQL with ORDER BY + LIMIT 1.
// Adversarial: delete ORDER BY or LIMIT 1 from buildShareDecayStatusQuery → test fails → red.
func TestBuildShareDecayStatusQuery_HasOrderByAndLimit(t *testing.T) {
	q := buildShareDecayStatusQuery()
	if !strings.Contains(q, "ORDER BY") {
		t.Error("buildShareDecayStatusQuery missing ORDER BY — non-deterministic with multiple published rows")
	}
	if !strings.Contains(q, "LIMIT 1") {
		t.Error("buildShareDecayStatusQuery missing LIMIT 1 — may return multiple rows")
	}
}

// T7: resolveDecayStatus with ErrNoRows returns "none" WITHOUT logging.
// Adversarial: delete ErrNoRows branch → ErrNoRows falls to log path →
// observer records a log entry → assertion fails → red.
func TestResolveDecayStatus_ErrNoRows(t *testing.T) {
	core, recorded := observer.New(zapcore.WarnLevel)
	log := zap.New(core)

	got := resolveDecayStatus(pgx.ErrNoRows, log, "acc-1", "")
	if got != "none" {
		t.Errorf("resolveDecayStatus(ErrNoRows) = %q, want 'none'", got)
	}
	if recorded.Len() != 0 {
		t.Errorf("resolveDecayStatus(ErrNoRows) logged %d entries — ErrNoRows is expected, should not warn", recorded.Len())
	}
}

// T7b: resolveDecayStatus with other error returns "none" AND logs a warning.
// Adversarial: delete else branch → returns "" → red.
func TestResolveDecayStatus_OtherError(t *testing.T) {
	core, recorded := observer.New(zapcore.WarnLevel)
	log := zap.New(core)

	got := resolveDecayStatus(errors.New("connection refused"), log, "acc-1", "")
	if got != "none" {
		t.Errorf("resolveDecayStatus(otherErr) = %q, want 'none'", got)
	}
	if recorded.Len() != 1 {
		t.Errorf("resolveDecayStatus(otherErr) logged %d entries — should log 1 warning", recorded.Len())
	}
}

// T7c: resolveDecayStatus with nil error returns scanned value.
// Adversarial: delete nil branch → always returns "none" even on success → red.
func TestResolveDecayStatus_NilError(t *testing.T) {
	core, recorded := observer.New(zapcore.WarnLevel)
	log := zap.New(core)

	got := resolveDecayStatus(nil, log, "acc-1", "decaying")
	if got != "decaying" {
		t.Errorf("resolveDecayStatus(nil) = %q, want 'decaying'", got)
	}
	if recorded.Len() != 0 {
		t.Errorf("resolveDecayStatus(nil) logged %d entries — nil error should not warn", recorded.Len())
	}
}

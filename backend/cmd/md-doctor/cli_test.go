package main

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// buildBinary compiles md-doctor to a temp file and returns the path.
func buildBinary(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	bin := filepath.Join(dir, "md-doctor")
	cmd := exec.Command("go", "build", "-o", bin, ".")
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		t.Fatalf("go build md-doctor: %v", err)
	}
	return bin
}

func runMDD(t *testing.T, bin string, env []string, args ...string) (stdout, stderr string, exitCode int) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, bin, args...)
	cmd.Env = append(os.Environ(), env...)
	var so, se bytes.Buffer
	cmd.Stdout, cmd.Stderr = &so, &se
	err := cmd.Run()
	if exitErr, ok := err.(*exec.ExitError); ok {
		exitCode = exitErr.ExitCode()
	}
	return so.String(), se.String(), exitCode
}

// TestMdDoctorHelp verifies the binary prints all 5 subcommands when invoked
// without arguments. This documents the public CLI surface.
func TestMdDoctorHelp(t *testing.T) {
	t.Parallel()
	bin := buildBinary(t)
	stdout, _, _ := runMDD(t, bin, nil)
	for _, want := range []string{"reconcile", "bar-continuity", "canonical-liveness", "dlq-tail", "all"} {
		if !strings.Contains(stdout, want) {
			t.Errorf("help output missing subcommand %q; got:\n%s", want, stdout)
		}
	}
	t.Logf("md-doctor: 5 subcommands listed (reconcile/bar-continuity/canonical-liveness/dlq-tail/all)")
}

// TestMdDoctorReconcileJSON requires a running CH (default credentials from
// CH_USER/CH_PASSWORD/CH_DATABASE env). Verifies the reconcile JSON shape.
func TestMdDoctorReconcileJSON(t *testing.T) {
	t.Parallel()
	if os.Getenv("CH_USER") == "" {
		t.Skip("CH_USER not set; skipping integration test (no CH credentials)")
	}
	bin := buildBinary(t)
	stdout, stderr, code := runMDD(t, bin, nil, "reconcile", "--window", "10m", "--output", "json")
	if code != 0 {
		t.Fatalf("reconcile exit=%d stderr=%s", code, stderr)
	}
	var r struct {
		WindowMs     int64                  `json:"window_ms"`
		ChTotalCount int64                  `json:"ch_total_count"`
		Passed       bool                   `json:"passed"`
		ByBC         map[string]interface{} `json:"by_broker_canonical"`
	}
	if err := json.Unmarshal([]byte(stdout), &r); err != nil {
		t.Fatalf("invalid JSON output: %v\n%s", err, stdout)
	}
	if r.WindowMs != 600000 {
		t.Errorf("window_ms=%d, want 600000", r.WindowMs)
	}
	if r.ChTotalCount < 0 {
		t.Errorf("ch_total_count must be >=0, got %d", r.ChTotalCount)
	}
	t.Logf("reconcile JSON parsed: window_ms=%d ch_total_count=%d passed=%v",
		r.WindowMs, r.ChTotalCount, r.Passed)
}

// TestMdDoctorAllSubcommands smoke-tests that every subcommand exits 0
// (or with a non-fatal failure code <2) on an empty CH. ADR-0010 §3.1.
func TestMdDoctorAllSubcommands(t *testing.T) {
	t.Parallel()
	if os.Getenv("CH_USER") == "" {
		t.Skip("CH_USER not set; skipping integration test")
	}
	bin := buildBinary(t)
	cmds := [][]string{
		{"reconcile", "--window", "10m", "--output", "json"},
		{"bar-continuity", "--window", "10m", "--output", "text"},
		{"canonical-liveness", "--output", "json"},
		{"dlq-tail", "--window", "1h", "--output", "json"},
		{"all", "--window", "10m", "--output", "json"},
	}
	for _, args := range cmds {
		stdout, stderr, code := runMDD(t, bin, nil, args...)
		if code > 1 {
			t.Errorf("md-doctor %v exited %d (>1=fatal)\nstderr=%s", args, code, stderr)
		}
		if !strings.Contains(stdout, "{") && !strings.Contains(stdout, "window") {
			t.Errorf("md-doctor %v: output looks empty:\n%s", args, stdout)
		}
		t.Logf("md-doctor %v: exit=%d, %d bytes stdout", args, code, len(stdout))
	}
}

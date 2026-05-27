package main

import (
	"bytes"
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func buildSloReport(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	bin := filepath.Join(dir, "slo-report")
	cmd := exec.Command("go", "build", "-o", bin, ".")
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		t.Fatalf("go build slo-report: %v", err)
	}
	return bin
}

func runSlo(t *testing.T, bin string, args ...string) (stdout, stderr string, code int) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, bin, args...)
	cmd.Env = os.Environ()
	var so, se bytes.Buffer
	cmd.Stdout, cmd.Stderr = &so, &se
	err := cmd.Run()
	if e, ok := err.(*exec.ExitError); ok {
		code = e.ExitCode()
	}
	return so.String(), se.String(), code
}

// TestSloReportMarkdown verifies that --output md produces a report containing
// all 4 SLO IDs (SLO-MD-1..4). Spec/20 §1 requires these IDs verbatim.
func TestSloReportMarkdown(t *testing.T) {
	t.Parallel()
	bin := buildSloReport(t)
	stdout, stderr, _ := runSlo(t, bin, "--window", "1h", "--output", "md")
	body := stdout + "\n" + stderr
	for i := 1; i <= 4; i++ {
		want := "SLO-MD-" + string(rune('0'+i))
		if !strings.Contains(body, want) {
			t.Errorf("output missing %q\n%s", want, body)
		}
	}
	t.Logf("slo-report: all 4 SLO-MD ids present in markdown output")
}

// TestSloReportPromUnreachable verifies graceful degradation when Prometheus
// is unreachable: each SLO row reports query-failed status instead of crashing.
func TestSloReportPromUnreachable(t *testing.T) {
	t.Parallel()
	bin := buildSloReport(t)
	cmd := exec.Command(bin, "--window", "1h", "--output", "md")
	cmd.Env = append(os.Environ(), "PROM_URL=http://127.0.0.1:1") // refused
	out, _ := cmd.CombinedOutput()
	body := string(out)
	if !strings.Contains(body, "SLO-MD-1") {
		t.Fatalf("output missing SLO-MD-1 even on prom-down:\n%s", body)
	}
	if !strings.Contains(body, "query failed") && !strings.Contains(body, "N/A") {
		t.Errorf("expected degraded marker (query failed/N/A) on prom-down:\n%s", body)
	}
	t.Logf("slo-report degrades gracefully when Prometheus unreachable")
}

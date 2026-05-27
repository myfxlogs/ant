package chmigrate

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// findRepoRoot walks up from CWD until it finds the docs/spec directory.
func findRepoRoot(t *testing.T) string {
	t.Helper()
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	for dir := cwd; dir != "/"; dir = filepath.Dir(dir) {
		if _, err := os.Stat(filepath.Join(dir, "docs", "spec", "13-clickhouse-schema.md")); err == nil {
			return dir
		}
	}
	t.Fatal("could not find repo root containing docs/spec/13-clickhouse-schema.md")
	return ""
}

// TestSpec13Keywords (M10.5-13) asserts that docs/spec/13-clickhouse-schema.md
// has been augmented with the M10 §M10.5-13 mandatory sections:
//   - FINAL query convention (§9 new)
//   - EXCHANGE TABLES migration preconditions (§2.8 note)
//   - md_bars long-term capacity / 3-year TTL / S3 cold storage (§8 new)
//
// Each keyword family must appear at least once. The test prints the matching
// line numbers so the verify log preserves a diff-grade audit trail.
func TestSpec13Keywords(t *testing.T) {
	t.Parallel()
	root := findRepoRoot(t)
	specPath := filepath.Join(root, "docs", "spec", "13-clickhouse-schema.md")
	body, err := os.ReadFile(specPath)
	if err != nil {
		t.Fatalf("read %s: %v", specPath, err)
	}
	text := string(body)

	required := []struct {
		family   string
		patterns []string // any-match within family
	}{
		{"FINAL query", []string{"FINAL"}},
		{"EXCHANGE TABLES precondition", []string{"EXCHANGE TABLES", "EXCHANGE precondition", "前置"}},
		{"long-term capacity / TTL", []string{"3 年 TTL", "3 年", "TTL 1095", "1095"}},
		{"S3 cold storage", []string{"S3 冷存储", "S3", "cold storage"}},
	}

	lines := strings.Split(text, "\n")
	failures := 0
	for _, r := range required {
		hits := []int{}
		for _, p := range r.patterns {
			for i, line := range lines {
				if strings.Contains(line, p) {
					hits = append(hits, i+1)
				}
			}
		}
		if len(hits) == 0 {
			t.Errorf("MISSING %q (any of: %v)", r.family, r.patterns)
			failures++
			continue
		}
		// Cap at 5 line-number reports per family to keep log bounded.
		if len(hits) > 5 {
			hits = hits[:5]
		}
		t.Logf("OK   %s — matched at lines %v", r.family, hits)
	}
	if failures > 0 {
		t.Fatalf("%d required spec/13 keyword families missing", failures)
	}
	t.Logf("spec/13 contains all %d M10.5-13 keyword families (file: %d lines, %d bytes)",
		len(required), len(lines), len(body))
}

// TestSpec13MinimumSize sanity-checks that the spec is non-trivial in length;
// stops accidental empty-file regressions from passing keyword-only checks.
func TestSpec13MinimumSize(t *testing.T) {
	t.Parallel()
	root := findRepoRoot(t)
	info, err := os.Stat(filepath.Join(root, "docs", "spec", "13-clickhouse-schema.md"))
	if err != nil {
		t.Fatal(err)
	}
	const minBytes = 4096
	if info.Size() < minBytes {
		t.Errorf("spec/13 is %d bytes, want >= %d (suspicious truncation)", info.Size(), minBytes)
	}
	t.Logf("spec/13 size = %d bytes", info.Size())
}

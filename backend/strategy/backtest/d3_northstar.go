package backtest

import (
	"fmt"
	"strings"
)

// D3 North Star Dashboard
//
// Aggregates D3 test results into a single dashboard view that shows
// the overall health of the MQL2Go indicator pipeline at a glance.
//
// The dashboard is designed to be printed in test output and CI logs.

// D3NorthStarCategory represents a category of D3 tests.
type D3NorthStarCategory string

const (
	D3CatDifferential D3NorthStarCategory = "Differential"
	D3CatMetamorphic  D3NorthStarCategory = "Metamorphic"
	D3CatCorpus       D3NorthStarCategory = "Corpus"
)

// D3NorthStarMetric holds a single metric in the dashboard.
type D3NorthStarMetric struct {
	Category D3NorthStarCategory
	Name     string
	Passed   bool
	Detail   string
}

// D3NorthStarReport is the complete dashboard.
type D3NorthStarReport struct {
	Metrics []D3NorthStarMetric
}

// PassRate returns the fraction of passing metrics.
func (r *D3NorthStarReport) PassRate() float64 {
	if len(r.Metrics) == 0 {
		return 0
	}
	passed := 0
	for _, m := range r.Metrics {
		if m.Passed {
			passed++
		}
	}
	return float64(passed) / float64(len(r.Metrics))
}

// OverallStatus returns "GREEN" if all pass, "YELLOW" if >80% pass, "RED" otherwise.
func (r *D3NorthStarReport) OverallStatus() string {
	rate := r.PassRate()
	switch {
	case rate >= 1.0:
		return "GREEN"
	case rate >= 0.8:
		return "YELLOW"
	default:
		return "RED"
	}
}

// Format renders the dashboard as a human-readable string.
func (r *D3NorthStarReport) Format() string {
	var sb strings.Builder

	status := r.OverallStatus()
	rate := r.PassRate()

	sb.WriteString("╔══════════════════════════════════════════════════╗\n")
	sb.WriteString("║         D3 North Star Dashboard                  ║\n")
	sb.WriteString("╠══════════════════════════════════════════════════╣\n")
	sb.WriteString(fmt.Sprintf("║  Status: %-7s  Pass Rate: %5.1f%%  (%d/%d)       ║\n",
		status, rate*100, r.countPassed(), len(r.Metrics)))
	sb.WriteString("╠══════════════════════════════════════════════════╣\n")

	// Group by category.
	cats := []D3NorthStarCategory{D3CatDifferential, D3CatMetamorphic, D3CatCorpus}
	for _, cat := range cats {
		items := r.byCategory(cat)
		if len(items) == 0 {
			continue
		}
		sb.WriteString(fmt.Sprintf("║  [%s]%s\n", cat, strings.Repeat(" ", 44-len(string(cat))-2)))
		for _, m := range items {
			mark := "✓"
			if !m.Passed {
				mark = "✗"
			}
			line := fmt.Sprintf("    %s %s", mark, m.Name)
			if m.Detail != "" {
				line += " — " + m.Detail
			}
			sb.WriteString("║" + line + strings.Repeat(" ", max(0, 50-len(line))) + "║\n")
		}
	}

	sb.WriteString("╚══════════════════════════════════════════════════╝\n")
	return sb.String()
}

func (r *D3NorthStarReport) countPassed() int {
	n := 0
	for _, m := range r.Metrics {
		if m.Passed {
			n++
		}
	}
	return n
}

func (r *D3NorthStarReport) byCategory(cat D3NorthStarCategory) []D3NorthStarMetric {
	var out []D3NorthStarMetric
	for _, m := range r.Metrics {
		if m.Category == cat {
			out = append(out, m)
		}
	}
	return out
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

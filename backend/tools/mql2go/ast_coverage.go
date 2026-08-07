package mql2go

import (
	"sort"

	"alphaforge/tools/mql2go/interp"
)

// CoverageResult combines static analysis (from interp.Analyze) with
// bytecode compilation coverage into a single report.
type CoverageResult struct {
	// Score is the coverage ratio (0.0 to 1.0).
	Score float64

	// TotalCalls is the total number of function calls in the MQL source.
	TotalCalls int

	// SupportedCalls is the number of calls that have implementations.
	SupportedCalls int

	// BlindSpots lists unimplemented functions encountered during compilation.
	BlindSpots []CoverageBlindSpot

	// Indicators lists indicator functions detected in the source.
	Indicators []string

	// ExecKind is the execution model: "on_bar", "on_tick", "on_timer".
	ExecKind string

	// EntryRules is the count of entry-order calls.
	EntryRules int

	// ExitRules is the count of exit-order calls.
	ExitRules int

	// Version is the MQL version: "mql4" or "mql5".
	Version string

	// LookaheadViolations lists detected future-data access patterns.
	// Populated by interp.DetectLookahead during AnalyzeCoverage.
	LookaheadViolations []interp.LookaheadViolation

	// DefenseAViolations lists post-parse validation failures (ADR-0028 §4.1).
	// Populated by interp.ValidateDefenseA during AnalyzeCoverage.
	DefenseAViolations []interp.DefenseAViolation
}

// CoverageBlindSpot is a single unimplemented feature.
type CoverageBlindSpot struct {
	Builtin  string
	Severity string
	Count    int
	Source   string // "static" (from IR analysis) or "compile" (from bytecode compilation)
}

// AnalyzeCoverage produces a combined coverage report from both the IR-level
// static analysis and the bytecode compilation pass.
//
// Static analysis catches unimplemented builtins by name (e.g. iCustom, ObjectCreate).
// Compilation coverage catches structural issues (unsupported node types, unknown constants).
func AnalyzeCoverage(ir *interp.IR, bc *Bytecode) *CoverageResult {
	staticRep := interp.Analyze(ir)

	result := &CoverageResult{
		Score:          staticRep.Coverage,
		TotalCalls:     staticRep.TotalCalls,
		SupportedCalls: staticRep.SupportedCalls,
		Indicators:     staticRep.Indicators,
		ExecKind:       staticRep.ExecKind,
		EntryRules:     staticRep.EntryRules,
		ExitRules:      staticRep.ExitRules,
		Version:        staticRep.Version,
	}

	// Merge static blind spots
	seen := make(map[string]bool)
	for _, bs := range staticRep.BlindSpots {
		result.BlindSpots = append(result.BlindSpots, CoverageBlindSpot{
			Builtin:  bs.Builtin,
			Severity: bs.Severity,
			Count:    bs.Count,
			Source:   "static",
		})
		seen[bs.Builtin] = true
	}

	// Merge compilation blind spots (deduplicate against static)
	for _, name := range bc.Coverage.BlindSpots {
		if !seen[name] {
			result.BlindSpots = append(result.BlindSpots, CoverageBlindSpot{
				Builtin:  name,
				Severity: interp.SeverityWarning,
				Count:    1,
				Source:   "compile",
			})
			seen[name] = true
		}
	}

	// Sort blind spots: fatal first, then by count descending
	sort.SliceStable(result.BlindSpots, func(i, j int) bool {
		if result.BlindSpots[i].Severity != result.BlindSpots[j].Severity {
			return severityRank(result.BlindSpots[i].Severity) < severityRank(result.BlindSpots[j].Severity)
		}
		return result.BlindSpots[i].Count > result.BlindSpots[j].Count
	})

	// Detect lookahead (future-data access) violations from IR.
	result.LookaheadViolations = interp.DetectLookahead(ir)

	// Defense A: post-parse validation (ADR-0028 §4.1).
	result.DefenseAViolations = interp.ValidateDefenseA(ir)

	return result
}

func severityRank(s string) int {
	switch s {
	case interp.SeverityFatal:
		return 0
	case interp.SeverityWarning:
		return 1
	case interp.SeverityInfo:
		return 2
	default:
		return 3
	}
}

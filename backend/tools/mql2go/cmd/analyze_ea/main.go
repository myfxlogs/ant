package main

import (
	"fmt"
	"os"
	"sort"
	"strings"

	"anttrader/tools/mql2go"
	"anttrader/tools/mql2go/interp"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "usage: analyze_ea <file.mq4|file.mq5>")
		os.Exit(1)
	}
	source, err := os.ReadFile(os.Args[1])
	if err != nil {
		fmt.Fprintf(os.Stderr, "read file: %v\n", err)
		os.Exit(1)
	}

	ir, err := mql2go.CompileToIR(string(source))
	if err != nil {
		fmt.Fprintf(os.Stderr, "compile: %v\n", err)
		os.Exit(1)
	}

	rep := interp.Analyze(ir)

	fmt.Println("══════════════════════════════════════════════════")
	fmt.Printf("Strategy: %s\n", os.Args[1])
	fmt.Printf("MQL Version: %s\n", rep.Version)
	fmt.Printf("Coverage: %.1f%% (%d/%d calls)\n", rep.Coverage*100, rep.SupportedCalls, rep.TotalCalls)
	fmt.Printf("ExecKind: %s\n", rep.ExecKind)
	fmt.Printf("Entry Rules: %d, Exit Rules: %d\n", rep.EntryRules, rep.ExitRules)
	fmt.Printf("Indicators: %v\n", rep.Indicators)
	fmt.Printf("Params: %d\n", len(rep.Params))
	for _, p := range rep.Params {
		fmt.Printf("  %s %s = %s\n", p.Type, p.Name, interp.EvalExprLiteral(p.Default))
	}

	// Group blind spots by severity
	fatal := []string{}
	warning := []string{}
	info := []string{}
	for _, bs := range rep.BlindSpots {
		switch bs.Severity {
		case interp.SeverityFatal:
			fatal = append(fatal, fmt.Sprintf("  %s (x%d)", bs.Builtin, bs.Count))
		case interp.SeverityWarning:
			warning = append(warning, fmt.Sprintf("  %s (x%d)", bs.Builtin, bs.Count))
		case interp.SeverityInfo:
			info = append(info, fmt.Sprintf("  %s (x%d)", bs.Builtin, bs.Count))
		}
	}
	sort.Strings(fatal)
	sort.Strings(warning)
	sort.Strings(info)

	fmt.Println()
	fmt.Println("── Blind Spots ──")
	fmt.Printf("致命 (Fatal): %d\n", len(fatal))
	for _, s := range fatal {
		fmt.Println(s)
	}
	fmt.Printf("\n警告 (Warning): %d\n", len(warning))
	for _, s := range warning {
		fmt.Println(s)
	}
	fmt.Printf("\n信息 (Info/Permanent): %d\n", len(info))
	for _, s := range info {
		fmt.Println(s)
	}

	// Check for any "other" severity (shouldn't exist)
	other := []string{}
	for _, bs := range rep.BlindSpots {
		if bs.Severity != interp.SeverityFatal &&
			bs.Severity != interp.SeverityWarning &&
			bs.Severity != interp.SeverityInfo {
			other = append(other, fmt.Sprintf("  %s (x%d) severity=%q", bs.Builtin, bs.Count, bs.Severity))
		}
	}
	if len(other) > 0 {
		fmt.Printf("\n⚠️  Unknown severity: %d\n", len(other))
		for _, s := range other {
			fmt.Println(s)
		}
	}

	// Verify all blind spots are expected
	fmt.Println()
	fmt.Println("── Verification ──")
	unexpected := []string{}
	for _, bs := range rep.BlindSpots {
		if strings.HasPrefix(bs.Builtin, "i") && len(bs.Builtin) > 1 && bs.Builtin[1] >= 'A' && bs.Builtin[1] <= 'Z' {
			if bs.Severity != interp.SeverityFatal {
				unexpected = append(unexpected, fmt.Sprintf("Indicator %s should be 致命, got %s", bs.Builtin, bs.Severity))
			}
		}
	}
	if len(unexpected) > 0 {
		fmt.Printf("⚠️  Unexpected classifications: %d\n", len(unexpected))
		for _, s := range unexpected {
			fmt.Println("  " + s)
		}
	} else {
		fmt.Println("✅ All classifications look correct")
	}

	fmt.Println("══════════════════════════════════════════════════")
}

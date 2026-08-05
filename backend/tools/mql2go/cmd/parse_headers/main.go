// Command parse_headers uses tree-sitter to parse MQL .mqh header files
// and extract API symbols (functions, constants, enums, class methods).
//
// Usage:
//
//	go run ./tools/mql2go/cmd/parse_headers <mqh_dir> [--output registry_entries.go]
//
// The output is a text file listing all symbols NOT already in the API registry,
// with their signatures and source files, ready for human review and classification.
package main

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"alphaforge/tools/mql2go"
)

func main() {
	outputFile := flag.String("output", "", "output file (default: stdout)")
	format := flag.String("format", "text", "output format: text or go")
	flag.Parse()

	if flag.NArg() < 1 {
		fmt.Fprintln(os.Stderr, "usage: parse_headers <mqh_dir> [--output file] [--format text|go]")
		os.Exit(1)
	}

	dir := flag.Arg(0)
	symbols, err := mql2go.ParseHeaderDir(dir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	fmt.Fprintf(os.Stderr, "parsed %d symbols from %s\n", len(symbols), dir)

	// Count by kind
	counts := make(map[string]int)
	for _, s := range symbols {
		counts[s.Kind]++
	}
	for k, v := range counts {
		fmt.Fprintf(os.Stderr, "  %s: %d\n", k, v)
	}

	var output string
	switch *format {
	case "go":
		output = mql2go.GenerateRegistryEntries(symbols)
	default:
		output = formatText(symbols)
	}

	if *outputFile != "" {
		if err := os.WriteFile(*outputFile, []byte(output), 0644); err != nil {
			fmt.Fprintf(os.Stderr, "write error: %v\n", err)
			os.Exit(1)
		}
		fmt.Fprintf(os.Stderr, "wrote %s\n", *outputFile)
	} else {
		fmt.Print(output)
	}
}

func formatText(symbols []mql2go.HeaderSymbol) string {
	var b strings.Builder
	for _, s := range symbols {
		b.WriteString(fmt.Sprintf("%s\t%s", s.Name, s.Kind))
		if s.Signature != "" {
			b.WriteString(fmt.Sprintf("\t%s", s.Signature))
		}
		if s.Value != "" {
			b.WriteString(fmt.Sprintf("\tvalue=%s", s.Value))
		}
		b.WriteString(fmt.Sprintf("\t%s\n", s.Source))
	}
	return b.String()
}

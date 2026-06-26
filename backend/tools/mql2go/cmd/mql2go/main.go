// mql2go converts MQL4/MQL5 strategy source to Go SDK code.
//
// Usage:
//
//	mql2go input.mq4 -o output.go
package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"anttrader/tools/mql2go"
)

func main() {
	output := flag.String("o", "", "output Go file (default: stdout)")
	className := flag.String("name", "", "strategy class name")
	flag.Parse()

	if flag.NArg() < 1 {
		fmt.Fprintln(os.Stderr, "Usage: mql2go input.mq4 [-o output.go] [-name StrategyName]")
		os.Exit(1)
	}

	input := filepath.Clean(flag.Arg(0))
	source, err := os.ReadFile(input)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading %s: %v\n", input, err)
		os.Exit(2)
	}

	// Parse MQL and recognize intent.
	intent, err := mql2go.Analyze(string(source))
	if err != nil {
		fmt.Fprintf(os.Stderr, "Analysis failed: %v\n", err)
		os.Exit(3)
	}

	if *className != "" {
		intent.Meta.Name = *className
	}

	// Generate Go code.
	code := mql2go.Generate(intent)

	if *output != "" {
		outPath := filepath.Clean(*output)
		if err := os.WriteFile(outPath, []byte(code), 0600); err != nil {
			fmt.Fprintf(os.Stderr, "Error writing %s: %v\n", outPath, err)
			os.Exit(4)
		}
		fmt.Fprintf(os.Stderr, "Wrote %d lines to %s\n", countLines(code), outPath)
	} else {
		fmt.Print(code)
	}
}

func countLines(s string) int {
	n := 0
	for _, c := range s {
		if c == '\n' {
			n++
		}
	}
	return n
}

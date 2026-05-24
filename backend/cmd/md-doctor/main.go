// md-doctor: data foundation reconciliation CLI.
// Spec: docs/spec/19-md-doctor.md
package main

import (
	"fmt"
	"os"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("md-doctor: data foundation reconciliation")
		fmt.Println("Usage: md-doctor <command> [flags]")
		fmt.Println("Commands: reconcile, bar-continuity, canonical-liveness, dlq-tail, all")
		os.Exit(1)
	}
	switch os.Args[1] {
	case "reconcile", "bar-continuity", "canonical-liveness", "dlq-tail", "all":
		fmt.Printf("md-doctor: %s — stub (CH/NATS connections not wired; see runner.go)\n", os.Args[1])
	default:
		fmt.Fprintf(os.Stderr, "unknown command: %s\n", os.Args[1])
		os.Exit(1)
	}
}

// md-doctor: data foundation reconciliation CLI
// Spec: docs/spec/19-md-doctor.md
package main

import (
	"fmt"
	"os"
	"time"

	"github.com/ClickHouse/clickhouse-go/v2"
	"github.com/nats-io/nats.go"
)

var (
	window     time.Duration
	outputJSON bool
	strict     bool
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "md-doctor: %v\n", err)
		os.Exit(2)
	}
}

func run() error {
	if len(os.Args) < 2 {
		printHelp()
		os.Exit(1)
	}
	cmd := os.Args[1]
	args := os.Args[2:]

	window = 1 * time.Hour
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--window":
			if i+1 < len(args) {
				d, err := time.ParseDuration(args[i+1])
				if err != nil { return fmt.Errorf("bad --window: %w", err) }
				window = d; i++
			}
		case "--output":
			if i+1 < len(args) { outputJSON = args[i+1] == "json"; i++ }
		case "--strict":
			strict = true
		}
	}

	chDSN := os.Getenv("CH_DSN")
	if chDSN == "" { chDSN = "clickhouse://default@localhost:9000/ant" }
	ch, err := clickhouse.Open(&clickhouse.Options{Addr: []string{chDSN}})
	if err != nil { return fmt.Errorf("clickhouse: %w", err) }
	defer ch.Close()

	natsURL := os.Getenv("NATS_URL")
	if natsURL == "" { natsURL = nats.DefaultURL }
	nc, err := nats.Connect(natsURL)
	if err != nil { return fmt.Errorf("nats: %w", err) }
	defer nc.Close()

	switch cmd {
	case "reconcile":
		return doReconcile(ch)
	case "bar-continuity":
		return doBarContinuity(ch)
	case "canonical-liveness":
		return doCanonicalLiveness(ch)
	case "dlq-tail":
		return doDLQTail(ch)
	case "all":
		errs := make([]error, 0, 4)
		for _, fn := range []func(clickhouse.Conn) error{doReconcile, doBarContinuity, doCanonicalLiveness, doDLQTail} {
			if e := fn(ch); e != nil { errs = append(errs, e) }
		}
		if strict && len(errs) > 0 {
			return fmt.Errorf("%d checks failed", len(errs))
		}
		return nil
	default:
		return fmt.Errorf("unknown command: %s", cmd)
	}
}

func printHelp() {
	fmt.Println("md-doctor: data foundation reconciliation")
	fmt.Println("Usage: md-doctor <command> [flags]")
	fmt.Println("Commands: reconcile, bar-continuity, canonical-liveness, dlq-tail, all")
	fmt.Println("Flags: --window <duration> (default 1h), --output json|text, --strict")
}

package main

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"time"

	"github.com/ClickHouse/clickhouse-go/v2"
)

type reconcileRow struct {
	Broker    string `json:"broker"`
	Canonical string `json:"canonical"`
	CHCount   uint64 `json:"ch_count"`
	DiffPct   float64 `json:"diff_pct"`
}

func doReconcile(ch clickhouse.Conn) error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	fromMs := time.Now().Add(-window).UnixMilli()

	rows, err := ch.Query(ctx, `
		SELECT broker, canonical, count()
		FROM md_ticks FINAL
		WHERE arrived_unix_ms > $1
		GROUP BY broker, canonical
		ORDER BY broker, canonical
	`, fromMs)
	if err != nil { return fmt.Errorf("reconcile query: %w", err) }
	defer rows.Close()

	var results []reconcileRow
	var total uint64
	for rows.Next() {
		var r reconcileRow
		if err := rows.Scan(&r.Broker, &r.Canonical, &r.CHCount); err != nil { continue }
		total += r.CHCount
		results = append(results, r)
	}

	type reconcileOut struct {
		WindowMs int64          `json:"window_ms"`
		Total    uint64          `json:"ch_total_count"`
		ByBroker []reconcileRow  `json:"by_broker_canonical"`
		Passed   bool            `json:"passed"`
	}
	out := reconcileOut{
		WindowMs: window.Milliseconds(),
		Total:    total,
		ByBroker: results,
		Passed:   total > 0,
	}

	if outputJSON {
		b, _ := json.MarshalIndent(out, "", "  ")
		fmt.Println(string(b))
	} else {
		fmt.Printf("reconcile: window=%s total=%d\n", window, total)
		for _, r := range results {
			if math.Abs(r.DiffPct) < 0.001 {
				fmt.Printf("  %-15s %-15s ch=%-10d ✓\n", r.Broker, r.Canonical, r.CHCount)
			} else {
				fmt.Printf("  %-15s %-15s ch=%-10d diff=%.4f%% ⚠\n", r.Broker, r.Canonical, r.CHCount, r.DiffPct*100)
			}
		}
		if out.Passed { fmt.Println("PASS") } else { fmt.Println("FAIL") }
	}

	if strict && !out.Passed { return fmt.Errorf("reconcile: no data found in window %s", window) }
	return nil
}

// validateReconcileWindow ensures the time window is at least 1 minute.
func validateReconcileWindow(w time.Duration) error {
	if w < time.Minute {
		return fmt.Errorf("reconcile: window must be >= 1m, got %s", w)
	}
	return nil
}

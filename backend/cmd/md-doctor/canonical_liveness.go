package main

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/ClickHouse/clickhouse-go/v2"
)

func doCanonicalLiveness(ch clickhouse.Conn) error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	fromMs := time.Now().Add(-window).UnixMilli()

	rows, err := ch.Query(ctx, `
		SELECT broker, canonical, max(arrived_unix_ms)
		FROM md_ticks
		WHERE arrived_unix_ms > 0
		GROUP BY broker, canonical
		ORDER BY broker, canonical
	`)
	if err != nil { return fmt.Errorf("canonical-liveness query: %w", err) }
	defer rows.Close()

	type liveRow struct {
		Broker      string `json:"broker"`
		Canonical   string `json:"canonical"`
		LastSeenMs  int64  `json:"last_seen_ms"`
		SecondsAgo  int64  `json:"seconds_ago"`
		Alive       bool   `json:"alive"`
	}
	nowMs := time.Now().UnixMilli()
	var results []liveRow
	allAlive := true
	for rows.Next() {
		var r liveRow
		if err := rows.Scan(&r.Broker, &r.Canonical, &r.LastSeenMs); err != nil { continue }
		r.SecondsAgo = (nowMs - r.LastSeenMs) / 1000
		r.Alive = r.LastSeenMs > fromMs
		results = append(results, r)
		if !r.Alive { allAlive = false }
	}

	out := struct {
		WindowMs int64     `json:"window_ms"`
		Symbols  []liveRow `json:"symbols"`
		Passed   bool      `json:"passed"`
	}{window.Milliseconds(), results, allAlive}

	if outputJSON {
		b, _ := json.MarshalIndent(out, "", "  ")
		fmt.Println(string(b))
	} else {
		fmt.Printf("canonical-liveness: window=%s total_symbols=%d\n", window, len(results))
		for _, r := range results {
			marker := "✓"
			if !r.Alive { marker = "✗ DEAD" }
			fmt.Printf("  %-15s %-15s %ds ago %s\n", r.Broker, r.Canonical, r.SecondsAgo, marker)
		}
		if allAlive { fmt.Println("PASS") } else { fmt.Println("FAIL") }
	}
	if strict && !allAlive { return fmt.Errorf("canonical-liveness: dead symbols found") }
	return nil
}

func isWeekendCheck(ts int64) bool {
	t := time.Unix(ts/1000, 0).UTC()
	wd := t.Weekday()
	hour := t.Hour()
	if wd == time.Saturday && hour >= 22 { return true }
	if wd == time.Sunday && hour < 22 { return true }
	return false
}

func filterWeekendSymbols(results []struct {
	Broker, Canonical string
	LastSeenMs, SecondsAgo int64
	Alive bool
}, fromMs int64) []struct {
	Broker, Canonical string
	LastSeenMs, SecondsAgo int64
	Alive bool
} {
	var filtered []struct {
		Broker, Canonical string
		LastSeenMs, SecondsAgo int64
		Alive bool
	}
	for _, r := range results {
		if !r.Alive && isWeekendCheck(fromMs) { continue }
		filtered = append(filtered, r)
	}
	return filtered
}

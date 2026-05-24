package main

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/ClickHouse/clickhouse-go/v2"
)

type gapRow struct {
	Broker         string `json:"broker"`
	Canonical      string `json:"canonical"`
	Gaps           int    `json:"gaps"`
	TotalGapMin    int    `json:"total_gap_minutes"`
	WorstGapMin    int    `json:"worst_gap_min"`
	WorstGapAt     string `json:"worst_gap_at"`
}

func doBarContinuity(ch clickhouse.Conn) error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	fromMs := time.Now().Add(-window).UnixMilli()

	rows, err := ch.Query(ctx, `
		SELECT broker, canonical, period, close_ts_unix_ms
		FROM md_bars FINAL
		WHERE close_ts_unix_ms > $1 AND period = '1m'
		ORDER BY broker, canonical, close_ts_unix_ms
	`, fromMs)
	if err != nil { return fmt.Errorf("bar-continuity query: %w", err) }
	defer rows.Close()

	type lastSeen struct {
		broker, canonical string
		prev              int64
		gaps, totalMin, worstMin int
		worstAt           int64
	}
	gaps := make(map[string]*lastSeen)
	for rows.Next() {
		var broker, canonical, period string
		var ts int64
		if err := rows.Scan(&broker, &canonical, &period, &ts); err != nil { continue }
		key := broker + ":" + canonical
		ls := gaps[key]
		if ls == nil {
			ls = &lastSeen{broker: broker, canonical: canonical, prev: ts}
			gaps[key] = ls
			continue
		}
		deltaSec := (ts - ls.prev) / 1000
		if deltaSec > 90 { // > 1.5 x 60s period
			ls.gaps++
			gapMin := int(deltaSec / 60)
			ls.totalMin += gapMin
			if gapMin > ls.worstMin { ls.worstMin = gapMin; ls.worstAt = ls.prev }
		}
		ls.prev = ts
	}

	var results []gapRow
	allPassed := true
	for _, ls := range gaps {
		results = append(results, gapRow{
			Broker:      ls.broker,
			Canonical:   ls.canonical,
			Gaps:        ls.gaps,
			TotalGapMin: ls.totalMin,
			WorstGapMin: ls.worstMin,
			WorstGapAt:  time.Unix(ls.worstAt/1000, 0).UTC().Format(time.RFC3339),
		})
		if ls.totalMin > int(window.Minutes())/100 { allPassed = false }
	}

	out := struct {
		WindowMs int64    `json:"window_ms"`
		Results  []gapRow `json:"gaps"`
		Passed   bool     `json:"passed"`
	}{window.Milliseconds(), results, allPassed}

	if outputJSON {
		b, _ := json.MarshalIndent(out, "", "  ")
		fmt.Println(string(b))
	} else {
		fmt.Printf("bar-continuity: window=%s gaps_found=%d\n", window, len(results))
		for _, r := range results {
			if r.Gaps > 0 {
				fmt.Printf("  %-15s %-15s gaps=%d total=%dm worst=%dm at=%s ⚠\n",
					r.Broker, r.Canonical, r.Gaps, r.TotalGapMin, r.WorstGapMin, r.WorstGapAt)
			}
		}
		if allPassed { fmt.Println("PASS") } else { fmt.Println("FAIL") }
	}
	if strict && !allPassed { return fmt.Errorf("bar-continuity: gap minutes exceed 1%% of window") }
	return nil
}

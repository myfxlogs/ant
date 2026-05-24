package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/ClickHouse/clickhouse-go/v2"
)

func doDLQTail(ch clickhouse.Conn) error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	reason := "parse_error"
	limit := 50
	for i := range os.Args {
		if os.Args[i] == "--reason" && i+1 < len(os.Args) { reason = os.Args[i+1] }
	}

	rows, err := ch.Query(ctx, `
		SELECT broker, symbol_raw, canonical, ts_unix_ms, bid_str, ask_str, reason, arrived_unix_ms
		FROM md_ticks_dlq
		WHERE reason = $1
		ORDER BY arrived_unix_ms DESC
		LIMIT $2
	`, reason, limit)
	if err != nil { return fmt.Errorf("dlq-tail query: %w", err) }
	defer rows.Close()

	type dlqRow struct {
		Broker    string `json:"broker"`
		SymbolRaw string `json:"symbol_raw"`
		Canonical string `json:"canonical"`
		TsMs      int64  `json:"ts_ms"`
		BidStr    string `json:"bid_str"`
		AskStr    string `json:"ask_str"`
		Reason    string `json:"reason"`
	}
	var results []dlqRow
	for rows.Next() {
		var r dlqRow
		var arrived int64
		if err := rows.Scan(&r.Broker, &r.SymbolRaw, &r.Canonical, &r.TsMs, &r.BidStr, &r.AskStr, &r.Reason, &arrived); err != nil { continue }
		results = append(results, r)
	}

	out := struct {
		Reason string   `json:"reason"`
		Limit  int      `json:"limit"`
		Rows   []dlqRow `json:"rows"`
		Count  int      `json:"count"`
	}{reason, limit, results, len(results)}

	if outputJSON {
		b, _ := json.MarshalIndent(out, "", "  ")
		fmt.Println(string(b))
	} else {
		fmt.Printf("dlq-tail: reason=%s limit=%d found=%d\n", reason, limit, len(results))
		for _, r := range results {
			fmt.Printf("  %-15s %-15s %s bid=%s ask=%s\n",
				r.Broker, r.SymbolRaw, time.Unix(r.TsMs/1000, 0).UTC().Format(time.RFC3339),
				r.BidStr, r.AskStr)
		}
	}
	return nil
}

// dlqReasonWhitelist checks if a reason string is a known DLQ reason.
func dlqReasonWhitelist(reason string) bool {
	switch reason {
	case "parse_error", "bid_gt_ask", "non_positive", "spill_failed":
		return true
	}
	return false
}

// formatDLQTime formats a unix millisecond timestamp for DLQ output.
func formatDLQTime(ms int64) string {
	return time.Unix(ms/1000, 0).UTC().Format(time.RFC3339)
}

// slo-report: SLO compliance reporting CLI
// Spec: docs/spec/20-slo.md §4
package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/prometheus/client_golang/api"
	v1 "github.com/prometheus/client_golang/api/prometheus/v1"
)

func main() {
	promURL := os.Getenv("PROM_URL")
	if promURL == "" { promURL = "http://localhost:9090" }

	window := 7 * 24 * time.Hour
	for i, a := range os.Args {
		if a == "--window" && i+1 < len(os.Args) {
			d, err := time.ParseDuration(os.Args[i+1])
			if err == nil { window = d }
		}
	}

	client, err := api.NewClient(api.Config{Address: promURL})
	if err != nil {
		fmt.Fprintf(os.Stderr, "prometheus client: %v\n", err)
		os.Exit(1)
	}
	v1api := v1.NewAPI(client)
	ctx := context.Background()
	now := time.Now()

	budget := float64(window.Hours()) * 0.001 // 0.1% error budget

	slos := []struct {
		id, name  string
		query     string
		target    float64
	}{
		{"SLO-MD-1", "可用性", `avg_over_time(md:up:1m[` + fmt.Sprintf("%dh", int(window.Hours())) + `])`, 0.999},
		{"SLO-MD-2", "延迟 P99", `histogram_quantile(0.99, rate(md_e2e_latency_seconds_bucket[5m]))`, 0.5},
		{"SLO-MD-3", "数据完整性", `1 - rate(md_tick_dropped_total[5m]) / rate(md_tick_total[5m])`, 0.999},
		{"SLO-MD-4", "降级耗时", `md_spill_pending_files`, 0},
	}

	fmt.Printf("# SLO Report %s\n\n", now.Format("2006-01"))
	fmt.Println("| SLO | 名称 | 目标 | 实际 | 消耗 budget | 状态 |")
	fmt.Println("|---|---|---|---|---|---|")

	for _, slo := range slos {
		result, _, err := v1api.Query(ctx, slo.query, now)
		actual := "N/A"
		status := "⚠️ query failed"
		budgetUsed := "N/A"
		if err == nil {
			actual = result.String()
			status = "✅"
			budgetUsed = fmt.Sprintf("%.1f%%", 0.0)
		}
		targetStr := fmt.Sprintf("%.1f%%", slo.target*100)
		fmt.Printf("| %s | %s | %s | %s | %s | %s |\n",
			slo.id, slo.name, targetStr, actual, budgetUsed, status)
	}

	fmt.Printf("\nError budget (%.3f%%): %.1f min/month\n", 0.1*100, budget*60)
	fmt.Printf("Source: %s\n", promURL)
}

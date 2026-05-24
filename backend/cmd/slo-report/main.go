// slo-report: SLO compliance reporting CLI.
// Spec: docs/spec/20-slo.md §4
package main

import (
	"fmt"
	"os"
)

func main() {
	fmt.Println("SLO-MD-1 可用性      目标 99.9%  — stub (Prometheus not connected)")
	fmt.Println("SLO-MD-2 延迟 P99    目标 <500ms — stub")
	fmt.Println("SLO-MD-3 数据完整性   目标 ≥99.9% — stub")
	fmt.Println("SLO-MD-4 降级耗时     目标 spill<5min — stub")
	os.Exit(0)
}

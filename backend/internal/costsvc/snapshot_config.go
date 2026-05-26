// Package costsvc provides the SnapshotConfig helper for backtest cost model freezing (M10-BASE-D3).
package costsvc

import (
	"encoding/json"
	"sort"
)

// SnapshotConfig freezes a set of cost models indexed by symbol for deterministic backtest replay.
// The result is a JSON-serializable map[string]CostSnapshot suitable for storage in
// backtest_run.cost_model_snapshot.
func SnapshotConfig(broker string, models map[string]*CostModel) ([]byte, error) {
	symbols := make([]string, 0, len(models))
	for sym := range models {
		symbols = append(symbols, sym)
	}
	sort.Strings(symbols)

	snap := make(map[string]CostSnapshot, len(symbols))
	for _, sym := range symbols {
		m := models[sym]
		m.Broker = broker
		s := m.Snapshot()
		snap[sym] = s
	}
	return json.Marshal(snap)
}

// SnapshotFromList freezes cost models from a slice.
func SnapshotFromList(broker string, models []*CostModel) ([]byte, error) {
	m := make(map[string]*CostModel, len(models))
	for _, model := range models {
		model.Broker = broker
		m[model.Symbol] = model
	}
	return SnapshotConfig(broker, m)
}

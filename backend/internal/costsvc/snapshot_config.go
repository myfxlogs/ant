// Package costsvc provides the SnapshotConfig helper for backtest cost model freezing (M10-BASE-D3).
package costsvc

import (
	"sort"

	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"

	antv1 "alphaforge/gen/proto/ant/v1"
)

// SnapshotConfig freezes a set of cost models indexed by symbol for deterministic backtest replay.
// The result is a proto-serialized CostSnapshotMap suitable for storage in
// backtest_run.cost_model_snapshot (BYTEA).
func SnapshotConfig(broker string, models map[string]*CostModel) ([]byte, error) {
	symbols := make([]string, 0, len(models))
	for sym := range models {
		symbols = append(symbols, sym)
	}
	sort.Strings(symbols)

	m := &antv1.CostSnapshotMap{Entries: make(map[string]*antv1.CostSnapshot, len(symbols))}
	for _, sym := range symbols {
		model := models[sym]
		model.Broker = broker
		s := model.Snapshot()
		m.Entries[sym] = &antv1.CostSnapshot{
			Symbol:            s.Symbol,
			Broker:            s.Broker,
			SpreadPips:        s.SpreadPips.InexactFloat64(),
			PipSize:           s.PipSize.InexactFloat64(),
			PipValue:          s.PipValue.InexactFloat64(),
			CommissionPerLot:  s.CommissionPerLot.InexactFloat64(),
			CommissionBps:     s.CommissionBps.InexactFloat64(),
			SwapLong:          s.SwapLong.InexactFloat64(),
			SwapShort:         s.SwapShort.InexactFloat64(),
			FundingRate:       s.FundingRate.InexactFloat64(),
			FundingIntervalNs: s.FundingInterval,
			SlippageBps:       s.SlippageBps.InexactFloat64(),
			MinCommission:     s.MinCommission.InexactFloat64(),
			FrozenAt:          timestamppb.New(s.FrozenAt),
		}
	}
	return proto.Marshal(m)
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

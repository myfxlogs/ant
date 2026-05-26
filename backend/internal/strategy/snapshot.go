// Package strategy provides strategy state snapshot/restore (M10-BASE-E0).
//
// StrategyState is the protobuf-compatible state container.
// Snapshots are stored in PG strategy_state table and used for hot-reload.
package strategy

import (
	"encoding/json"
	"time"
)

// StrategyState is the canonical serializable state for a running strategy.
// Schema aligned with proto/ant/v1/strategy_state.proto.
type StrategyState struct {
	StrategyName    string            `json:"strategy_name"`
	StrategyVersion string            `json:"strategy_version"`
	InstanceID      string            `json:"instance_id"`
	AccountID       string            `json:"account_id"`
	UserID          string            `json:"user_id"`
	Symbols         []string          `json:"symbols"`
	Parameters      map[string]string `json:"parameters"`
	Positions       []StatePosition   `json:"positions"`
	Metrics         *StateMetrics     `json:"metrics,omitempty"`
	CustomState     []byte            `json:"custom_state,omitempty"` // strategy-specific opaque blob
	SnapshottedAt   time.Time         `json:"snapshotted_at"`
	SchemaVersion   int32             `json:"schema_version"` // incremented on breaking changes
}

// StatePosition is a lightweight position record in strategy state.
type StatePosition struct {
	Symbol     string  `json:"symbol"`
	Side       string  `json:"side"`
	Volume     float64 `json:"volume"`
	EntryPrice float64 `json:"entry_price"`
	OpenTime   int64   `json:"open_time_unix_ms"`
}

// StateMetrics holds strategy-level performance metrics.
type StateMetrics struct {
	TotalTrades    int     `json:"total_trades"`
	WinRate        float64 `json:"win_rate"`
	SharpeRatio    float64 `json:"sharpe_ratio"`
	NetPnL         float64 `json:"net_pnl"`
	GrossPnL       float64 `json:"gross_pnl"`
	MaxDrawdown    float64 `json:"max_drawdown"`
	TotalCost      float64 `json:"total_cost"`
}

// MarshalState serializes strategy state to JSON bytes.
func MarshalState(s *StrategyState) ([]byte, error) {
	return json.Marshal(s)
}

// UnmarshalState deserializes strategy state from JSON bytes.
func UnmarshalState(data []byte) (*StrategyState, error) {
	var s StrategyState
	if err := json.Unmarshal(data, &s); err != nil {
		return nil, err
	}
	return &s, nil
}

// SnapshotStrategy captures a strategy's full state into a StrategyState.
func SnapshotStrategy(s Strategy, instanceID, accountID, userID string) (*StrategyState, error) {
	custom, err := s.Snapshot()
	if err != nil {
		return nil, err
	}
	return &StrategyState{
		StrategyName:    s.Name(),
		StrategyVersion: s.Version(),
		InstanceID:      instanceID,
		AccountID:       accountID,
		UserID:          userID,
		CustomState:     custom,
		SnapshottedAt:   time.Now(),
		SchemaVersion:   1,
	}, nil
}

// RestoreStrategy restores a strategy from a StrategyState.
func RestoreStrategy(s Strategy, state *StrategyState) error {
	return s.Restore(state.CustomState)
}

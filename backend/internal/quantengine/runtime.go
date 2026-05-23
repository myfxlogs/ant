// Package quantengine — Per-strategy runtime with goroutine isolation.
//
// Each strategy runs in its own goroutine with recover() protection.
// Snapshots persist runtime state; hot-reload is triggered by strategy revisions.
package quantengine

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/jmoiron/sqlx"
	"go.uber.org/zap"
)

// RuntimeState is serialized for crash recovery.
type RuntimeState struct {
	StrategyName  string             `json:"strategy_name"`
	RevisionID    string             `json:"revision_id"`
	LastSignal    float64            `json:"last_signal"`
	LastDirection string             `json:"last_direction"`
	PositionState map[string]float64 `json:"position_state"`
	SnapshotAt    int64              `json:"snapshot_at_ms"`
}

// StrategyRuntime bundles a spec with its isolated execution loop.
type StrategyRuntime struct {
	Spec       *StrategySpec
	name       string
	strategyID string // DB UUID, set by RuntimeManager after creation
	runner     *ModelRunner
	onSignal   SignalHandler
	log        *zap.Logger
	stateMu    sync.RWMutex
	state      *RuntimeState
	stopCh     chan struct{}
	barCh      chan struct{} // triggered on each new bar
}

// NewStrategyRuntime creates an isolated runtime for a strategy.
func NewStrategyRuntime(
	spec *StrategySpec,
	onSignal SignalHandler,
	log *zap.Logger,
) (*StrategyRuntime, error) {
	mr, err := NewModelRunner(spec)
	if err != nil {
		return nil, err
	}
	return &StrategyRuntime{
		Spec:     spec,
		name:     spec.Name,
		runner:   mr,
		onSignal: onSignal,
		log:      log.With(zap.String("strategy", spec.Name)),
		state:    &RuntimeState{StrategyName: spec.Name, PositionState: make(map[string]float64)},
		stopCh:   make(chan struct{}),
		barCh:    make(chan struct{}, 1), // buffered to avoid blocking
	}, nil
}

// Start launches the strategy evaluation loop in its own goroutine.
func (rt *StrategyRuntime) Start(ctx context.Context) {
	go func() {
		defer func() {
			if r := recover(); r != nil {
				rt.log.Error("strategy runtime panic recovered",
					zap.Any("panic", r),
				)
			}
		}()
		rt.log.Info("strategy runtime started")
		rt.loop(ctx)
	}()
}

// Stop signals the runtime to stop gracefully.
func (rt *StrategyRuntime) Stop() {
	close(rt.stopCh)
	rt.log.Info("strategy runtime stopped")
}

// OnBar notifies the runtime that a new bar is available.
func (rt *StrategyRuntime) OnBar() {
	select {
	case rt.barCh <- struct{}{}:
	default:
		// channel full — skip this tick, evaluation will catch up
	}
}

// Snapshot returns a copy of the current runtime state.
func (rt *StrategyRuntime) Snapshot() *RuntimeState {
	rt.stateMu.RLock()
	defer rt.stateMu.RUnlock()
	s := *rt.state
	s.SnapshotAt = time.Now().UnixMilli()
	return &s
}

// Restore loads saved state (e.g., after restart).
func (rt *StrategyRuntime) Restore(state *RuntimeState) {
	rt.stateMu.Lock()
	rt.state = state
	rt.stateMu.Unlock()
	rt.log.Info("runtime state restored", zap.Int64("snapshot_at_ms", state.SnapshotAt))
}

// Evaluate runs one inference cycle with the given factor values.
func (rt *StrategyRuntime) Evaluate(ctx context.Context, factorVals map[string]float64) {
	if factorVals == nil {
		return
	}

	signal, err := rt.runner.Predict(ctx, factorVals)
	if err != nil {
		rt.log.Warn("signal eval failed", zap.Error(err))
		return
	}

	dir := Direction(signal)
	if dir == "flat" {
		return
	}

	// Track last signal
	rt.stateMu.Lock()
	rt.state.LastSignal = signal
	rt.state.LastDirection = dir
	rt.stateMu.Unlock()

	if rt.onSignal != nil && len(rt.Spec.CanonicalSymbols) > 0 {
		rt.onSignal(rt.strategyID, rt.Spec.CanonicalSymbols[0], dir, 0.1, rt.name)
	}

	rt.log.Debug("signal generated",
		zap.Float64("signal", signal),
		zap.String("direction", dir),
	)
}

func (rt *StrategyRuntime) loop(ctx context.Context) {
	snapshotTicker := time.NewTicker(30 * time.Second)
	defer snapshotTicker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-rt.stopCh:
			return
		case <-rt.barCh:
			// In production, factor values come from the factor engine.
			// For standalone use, an empty map triggers DSL evaluation.
			rt.Evaluate(ctx, nil)
		case <-snapshotTicker.C:
			// Snapshot written by the manager (calls Snapshot())
		}
	}
}

// ── Runtime Manager ──

// RuntimeManager manages multiple strategy runtimes with snapshot persistence.
type RuntimeManager struct {
	mu       sync.RWMutex
	runtimes map[string]*StrategyRuntime
	db       *sqlx.DB // optional: PG for snapshot persistence
	log      *zap.Logger
}

// NewRuntimeManager creates a runtime manager.
func NewRuntimeManager(log *zap.Logger) *RuntimeManager {
	return &RuntimeManager{
		runtimes: make(map[string]*StrategyRuntime),
		log:      log,
	}
}

// WithDB sets the sqlx.DB for snapshot persistence.
func (m *RuntimeManager) WithDB(db *sqlx.DB) *RuntimeManager {
	m.db = db
	return m
}

// Add registers and starts a new runtime.
func (m *RuntimeManager) Add(ctx context.Context, rt *StrategyRuntime) {
	m.mu.Lock()
	// Stop old runtime if replacing (hot-reload)
	if old, ok := m.runtimes[rt.name]; ok {
		old.Stop()
	}
	m.runtimes[rt.name] = rt
	m.mu.Unlock()
	rt.Start(ctx)
	m.log.Info("runtime added", zap.String("strategy", rt.name))
}

// Remove stops and removes a runtime.
func (m *RuntimeManager) Remove(name string) {
	m.mu.Lock()
	if rt, ok := m.runtimes[name]; ok {
		rt.Stop()
		delete(m.runtimes, name)
	}
	m.mu.Unlock()
}

// OnBar forwards bar events to all active runtimes.
func (m *RuntimeManager) OnBar() {
	m.mu.RLock()
	defer m.mu.RUnlock()
	for _, rt := range m.runtimes {
		rt.OnBar()
	}
}

// OnFactors dispatches factor values to all active runtimes for evaluation.
func (m *RuntimeManager) OnFactors(ctx context.Context, factorVals map[string]float64) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	for _, rt := range m.runtimes {
		rt.Evaluate(ctx, factorVals)
	}
}

// SnapshotAll returns snapshots of all active runtimes.
func (m *RuntimeManager) SnapshotAll() map[string]*RuntimeState {
	m.mu.RLock()
	defer m.mu.RUnlock()
	out := make(map[string]*RuntimeState, len(m.runtimes))
	for name, rt := range m.runtimes {
		out[name] = rt.Snapshot()
	}
	return out
}

// Count returns the number of active runtimes.
func (m *RuntimeManager) Count() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.runtimes)
}

// Get returns a runtime by name, or nil.
func (m *RuntimeManager) Get(name string) *StrategyRuntime {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.runtimes[name]
}

func (m *RuntimeManager) saveSnapshots() {
	if m.db == nil {
		return
	}
	snapshots := m.SnapshotAll()
	if len(snapshots) == 0 {
		return
	}
	for name, s := range snapshots {
		stateJSON, err := json.Marshal(s)
		if err != nil {
			m.log.Warn("snapshot marshal failed", zap.String("strategy", name), zap.Error(err))
			continue
		}
		_, err = m.db.Exec(
			`INSERT INTO strategy_runtime_snapshots (strategy_name, revision_id, state_json, snapshot_at_ms)
			 VALUES ($1, $2, $3, $4)
			 ON CONFLICT DO NOTHING`,
			name, s.RevisionID, stateJSON, s.SnapshotAt)
		if err != nil {
			m.log.Warn("snapshot persist failed", zap.String("strategy", name), zap.Error(err))
		}
	}
}

// RestoreAll loads the latest snapshots from PG and restores all runtimes.
func (m *RuntimeManager) RestoreAll(ctx context.Context) error {
	if m.db == nil {
		return nil
	}
	rows, err := m.db.QueryxContext(ctx,
		`SELECT DISTINCT ON (strategy_name) strategy_name, revision_id, state_json, snapshot_at_ms
		 FROM strategy_runtime_snapshots
		 ORDER BY strategy_name, snapshot_at_ms DESC`)
	if err != nil {
		return fmt.Errorf("restore query: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var name, revID string
		var stateJSON []byte
		var snapMs int64
		if err := rows.Scan(&name, &revID, &stateJSON, &snapMs); err != nil {
			continue
		}
		var state RuntimeState
		if err := json.Unmarshal(stateJSON, &state); err != nil {
			continue
		}
		m.mu.RLock()
		rt, ok := m.runtimes[name]
		m.mu.RUnlock()
		if ok {
			rt.Restore(&state)
			m.log.Info("runtime restored from snapshot",
				zap.String("strategy", name),
				zap.Int64("snapshot_at_ms", snapMs),
			)
		}
	}
	return rows.Err()
}

// StartSnapshotLoop persists snapshots every N seconds.
func (m *RuntimeManager) StartSnapshotLoop(ctx context.Context, interval time.Duration) {
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				m.saveSnapshots()
			}
		}
	}()
}

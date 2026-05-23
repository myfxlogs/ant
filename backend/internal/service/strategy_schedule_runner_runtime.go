package service

import (
	"context"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"anttrader/pkg/logger"
)

func (r *StrategyScheduleRunner) ensureState(id uuid.UUID) {
	if r == nil {
		return
	}
	r.stateMu.Lock()
	defer r.stateMu.Unlock()
	if _, ok := r.states[id]; ok {
		return
	}
	if r.stateStore != nil {
		loaded, ok, err := r.stateStore.Load(r.ctx, id)
		if err == nil && ok && loaded != nil {
			r.states[id] = loaded
			return
		}
	}
	r.states[id] = &ScheduleRuntimeState{ScheduleID: id, StartedAt: time.Now(), Data: map[string]interface{}{}, MartingaleLevel: 0}
}

func (r *StrategyScheduleRunner) updateState(id uuid.UUID, fn func(*ScheduleRuntimeState)) {
	if r == nil || fn == nil {
		return
	}
	r.stateMu.Lock()
	st := r.states[id]
	if st == nil {
		st = &ScheduleRuntimeState{ScheduleID: id, StartedAt: time.Now(), Data: map[string]interface{}{}, MartingaleLevel: 0}
		r.states[id] = st
	}
	fn(st)
	store := r.stateStore
	r.stateMu.Unlock()
	if store != nil {
		if err := store.Save(context.Background(), st); err != nil {
			logger.Warn("schedule runtime state save failed", zap.String("schedule_id", id.String()), zap.Error(err))
		}
	}
}

func (r *StrategyScheduleRunner) persistLastRun(scheduleID uuid.UUID, runErr error) {
	if err := r.scheduleRepo.UpdateLastRun(context.Background(), scheduleID, runErr); err != nil {
		logger.Warn("schedule last_run update failed", zap.String("schedule_id", scheduleID.String()), zap.Error(err))
	}
}

func (r *StrategyScheduleRunner) snapshotExecContext(id uuid.UUID) map[string]interface{} {
	if r == nil {
		return map[string]interface{}{}
	}
	r.stateMu.Lock()
	st := r.states[id]
	var (
		dataCopy map[string]interface{}
		lvl      int
		lastSig  string
		lastAt   time.Time
	)
	if st != nil {
		dataCopy = make(map[string]interface{}, len(st.Data))
		for k, v := range st.Data {
			dataCopy[k] = v
		}
		lvl = st.MartingaleLevel
		lastSig = st.LastSignal
		lastAt = st.LastSignalAt
	}
	r.stateMu.Unlock()

	execCtx := map[string]interface{}{}
	if st != nil {
		execCtx["runtime"] = dataCopy
		execCtx["martingale_level"] = lvl
		execCtx["last_signal"] = lastSig
		execCtx["last_signal_at"] = lastAt.UTC().Format(time.RFC3339Nano)
	}
	return execCtx
}

func toInt64Local(v interface{}) int64 {
	switch t := v.(type) {
	case int64:
		return t
	case int:
		return int64(t)
	case float64:
		return int64(t)
	case float32:
		return int64(t)
	default:
		return 0
	}
}

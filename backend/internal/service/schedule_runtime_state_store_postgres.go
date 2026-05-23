package service

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

type PostgresScheduleRuntimeStateStore struct {
	db *sqlx.DB
}

func NewPostgresScheduleRuntimeStateStore(db *sqlx.DB) *PostgresScheduleRuntimeStateStore {
	return &PostgresScheduleRuntimeStateStore{db: db}
}

func (s *PostgresScheduleRuntimeStateStore) Load(ctx context.Context, scheduleID uuid.UUID) (*ScheduleRuntimeState, bool, error) {
	if s == nil || s.db == nil {
		return nil, false, errors.New("db not available")
	}
	var row struct {
		State json.RawMessage `db:"state"`
	}
	query := `SELECT state FROM strategy_schedule_runtime_states WHERE schedule_id = $1`
	if err := s.db.GetContext(ctx, &row, query, scheduleID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, false, nil
		}
		return nil, false, err
	}
	st, err := unmarshalRuntimeState(row.State)
	if err != nil {
		return nil, false, err
	}
	if st != nil {
		st.ScheduleID = scheduleID
	}
	return st, true, nil
}

func (s *PostgresScheduleRuntimeStateStore) Save(ctx context.Context, state *ScheduleRuntimeState) error {
	if s == nil || s.db == nil {
		return errors.New("db not available")
	}
	if state == nil || state.ScheduleID == uuid.Nil {
		return errors.New("invalid state")
	}
	b, err := marshalRuntimeState(state)
	if err != nil {
		return err
	}
	query := `
		INSERT INTO strategy_schedule_runtime_states (schedule_id, state, created_at, updated_at)
		VALUES ($1, $2, $3, $3)
		ON CONFLICT (schedule_id) DO UPDATE SET state = EXCLUDED.state, updated_at = EXCLUDED.updated_at`
	now := time.Now()
	_, err = s.db.ExecContext(ctx, query, state.ScheduleID, b, now)
	return err
}

func (s *PostgresScheduleRuntimeStateStore) Delete(ctx context.Context, scheduleID uuid.UUID) error {
	if s == nil || s.db == nil {
		return errors.New("db not available")
	}
	query := `DELETE FROM strategy_schedule_runtime_states WHERE schedule_id = $1`
	_, err := s.db.ExecContext(ctx, query, scheduleID)
	return err
}

func marshalRuntimeState(st *ScheduleRuntimeState) ([]byte, error) {
	return json.Marshal(st)
}

func unmarshalRuntimeState(b []byte) (*ScheduleRuntimeState, error) {
	if len(b) == 0 {
		return nil, nil
	}
	var st ScheduleRuntimeState
	if err := json.Unmarshal(b, &st); err != nil {
		return nil, err
	}
	if st.Data == nil {
		st.Data = map[string]interface{}{}
	}
	return &st, nil
}

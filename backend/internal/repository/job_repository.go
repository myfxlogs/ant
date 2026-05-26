package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrJobNotFound = errors.New("job not found")

type Job struct {
	ID             uuid.UUID  `db:"id"`
	UserID         uuid.UUID  `db:"user_id"`
	Kind           string     `db:"kind"`
	Status         string     `db:"status"`
	Progress       float64    `db:"progress"`
	Stage          string     `db:"stage"`
	RequestSummary string     `db:"request_summary"`
	ResultRef      string     `db:"result_ref"`
	ResultSummary  string     `db:"result_summary"`
	ErrorCode      string     `db:"error_code"`
	ErrorMessage   string     `db:"error_message"`
	IdempotencyKey string     `db:"idempotency_key"`
	CreatedAt      time.Time  `db:"created_at"`
	StartedAt      *time.Time `db:"started_at"`
	FinishedAt     *time.Time `db:"finished_at"`
	ExpiresAt      *time.Time `db:"expires_at"`
}

type JobEvent struct {
	ID        uuid.UUID `db:"id"`
	JobID     uuid.UUID `db:"job_id"`
	Seq       int64     `db:"seq"`
	Type      string    `db:"type"`
	Status    string    `db:"status"`
	Progress  float64   `db:"progress"`
	Stage     string    `db:"stage"`
	Message   string    `db:"message"`
	Payload   []byte    `db:"payload"`
	CreatedAt time.Time `db:"created_at"`
}

const jobSelectCols = `id, user_id, kind, status, progress, stage, request_summary, result_ref, result_summary, error_code, error_message, idempotency_key, created_at, started_at, finished_at, expires_at`

type JobRepository struct {
	db *pgxpool.Pool
}

func NewJobRepository(db *pgxpool.Pool) *JobRepository {
	return &JobRepository{db: db}
}

func scanJob(row pgx.Row) (*Job, error) {
	var j Job
	err := row.Scan(&j.ID, &j.UserID, &j.Kind, &j.Status, &j.Progress, &j.Stage,
		&j.RequestSummary, &j.ResultRef, &j.ResultSummary, &j.ErrorCode, &j.ErrorMessage,
		&j.IdempotencyKey, &j.CreatedAt, &j.StartedAt, &j.FinishedAt, &j.ExpiresAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrJobNotFound
		}
		return nil, err
	}
	return &j, nil
}

func (r *JobRepository) CreateJob(ctx context.Context, job *Job) error {
	if job.ID == uuid.Nil {
		job.ID = uuid.New()
	}
	if job.CreatedAt.IsZero() {
		job.CreatedAt = time.Now().UTC()
	}
	if job.Status == "" {
		job.Status = "queued"
	}
	_, err := r.db.Exec(ctx, `
		INSERT INTO jobs (id, user_id, kind, status, progress, stage, request_summary, result_ref, result_summary, error_code, error_message, idempotency_key, created_at, started_at, finished_at, expires_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16)
	`, job.ID, job.UserID, job.Kind, job.Status, job.Progress, job.Stage, job.RequestSummary, job.ResultRef, job.ResultSummary, job.ErrorCode, job.ErrorMessage, job.IdempotencyKey, job.CreatedAt, job.StartedAt, job.FinishedAt, job.ExpiresAt)
	if err != nil {
		return fmt.Errorf("create job: %w", err)
	}
	return nil
}

func (r *JobRepository) GetJob(ctx context.Context, userID, jobID uuid.UUID) (*Job, error) {
	return scanJob(r.db.QueryRow(ctx,
		"SELECT "+jobSelectCols+" FROM jobs WHERE id = $1 AND user_id = $2", jobID, userID))
}

func (r *JobRepository) CancelJob(ctx context.Context, userID, jobID uuid.UUID) (*Job, error) {
	return scanJob(r.db.QueryRow(ctx,
		"UPDATE jobs SET status = 'cancelled', finished_at = COALESCE(finished_at, now()) WHERE id = $1 AND user_id = $2 AND status IN ('queued','running') RETURNING "+jobSelectCols,
		jobID, userID))
}

func (r *JobRepository) AddEvent(ctx context.Context, event *JobEvent) error {
	if event.ID == uuid.Nil {
		event.ID = uuid.New()
	}
	if event.CreatedAt.IsZero() {
		event.CreatedAt = time.Now().UTC()
	}
	if event.Seq <= 0 {
		err := r.db.QueryRow(ctx,
			"SELECT COALESCE(MAX(seq), 0) + 1 FROM job_events WHERE job_id = $1", event.JobID,
		).Scan(&event.Seq)
		if err != nil {
			return fmt.Errorf("add job event: %w", err)
		}
	}
	if len(event.Payload) == 0 {
		event.Payload = []byte(`{}`)
	}
	_, err := r.db.Exec(ctx, `
		INSERT INTO job_events (id, job_id, seq, type, status, progress, stage, message, payload, created_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)
	`, event.ID, event.JobID, event.Seq, event.Type, event.Status, event.Progress, event.Stage, event.Message, event.Payload, event.CreatedAt)
	if err != nil {
		return fmt.Errorf("add job event: %w", err)
	}
	return nil
}

func (r *JobRepository) ListEvents(ctx context.Context, jobID uuid.UUID, afterSeq int64, limit int) ([]JobEvent, error) {
	if limit <= 0 || limit > 500 {
		limit = 100
	}
	rows, err := r.db.Query(ctx,
		"SELECT id, job_id, seq, type, status, progress, stage, message, payload, created_at FROM job_events WHERE job_id = $1 AND seq > $2 ORDER BY seq ASC LIMIT $3",
		jobID, afterSeq, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var events []JobEvent
	for rows.Next() {
		var e JobEvent
		if err := rows.Scan(&e.ID, &e.JobID, &e.Seq, &e.Type, &e.Status, &e.Progress, &e.Stage, &e.Message, &e.Payload, &e.CreatedAt); err != nil {
			return nil, err
		}
		events = append(events, e)
	}
	return events, rows.Err()
}

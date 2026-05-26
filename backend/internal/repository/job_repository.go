package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
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

type JobRepository struct {
	db *sqlx.DB
}

func NewJobRepository(db *sqlx.DB) *JobRepository {
	return &JobRepository{db: db}
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
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO jobs (id, user_id, kind, status, progress, stage, request_summary, result_ref, result_summary, error_code, error_message, idempotency_key, created_at, started_at, finished_at, expires_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16)
	`, job.ID, job.UserID, job.Kind, job.Status, job.Progress, job.Stage, job.RequestSummary, job.ResultRef, job.ResultSummary, job.ErrorCode, job.ErrorMessage, job.IdempotencyKey, job.CreatedAt, job.StartedAt, job.FinishedAt, job.ExpiresAt)
	return fmt.Errorf("create job: %w", err)
}

func (r *JobRepository) GetJob(ctx context.Context, userID, jobID uuid.UUID) (*Job, error) {
	var job Job
	err := r.db.GetContext(ctx, &job, `SELECT * FROM jobs WHERE id = $1 AND user_id = $2`, jobID, userID)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrJobNotFound
	}
	return &job, err
}

func (r *JobRepository) CancelJob(ctx context.Context, userID, jobID uuid.UUID) (*Job, error) {
	var job Job
	err := r.db.GetContext(ctx, &job, `UPDATE jobs SET status = 'cancelled', finished_at = COALESCE(finished_at, now()) WHERE id = $1 AND user_id = $2 AND status IN ('queued','running') RETURNING *`, jobID, userID)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrJobNotFound
	}
	return &job, err
}

func (r *JobRepository) AddEvent(ctx context.Context, event *JobEvent) error {
	if event.ID == uuid.Nil {
		event.ID = uuid.New()
	}
	if event.CreatedAt.IsZero() {
		event.CreatedAt = time.Now().UTC()
	}
	var nextSeq int64
	if event.Seq <= 0 {
		err := r.db.GetContext(ctx, &nextSeq, `SELECT COALESCE(MAX(seq), 0) + 1 FROM job_events WHERE job_id = $1`, event.JobID)
		if err != nil {
			return fmt.Errorf("add job event: %w", err)
		}
		event.Seq = nextSeq
	}
	if len(event.Payload) == 0 {
		event.Payload = []byte(`{}`)
	}
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO job_events (id, job_id, seq, type, status, progress, stage, message, payload, created_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)
	`, event.ID, event.JobID, event.Seq, event.Type, event.Status, event.Progress, event.Stage, event.Message, event.Payload, event.CreatedAt)
	return fmt.Errorf("add job event: %w", err)
}

func (r *JobRepository) ListEvents(ctx context.Context, jobID uuid.UUID, afterSeq int64, limit int) ([]JobEvent, error) {
	if limit <= 0 || limit > 500 {
		limit = 100
	}
	var events []JobEvent
	err := r.db.SelectContext(ctx, &events, `SELECT * FROM job_events WHERE job_id = $1 AND seq > $2 ORDER BY seq ASC LIMIT $3`, jobID, afterSeq, limit)
	return events, err
}

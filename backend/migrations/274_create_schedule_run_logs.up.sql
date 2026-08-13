-- 274: Create schedule_run_logs table.
-- Referenced by log_repository.go and schedule_health_repo.go but never created.
-- Reads (GetScheduleRunLogs, GetScheduleStats) were 500ing because the table didn't exist.
CREATE TABLE IF NOT EXISTS schedule_run_logs (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id       UUID NOT NULL,
    schedule_id   UUID NOT NULL,
    kind          TEXT NOT NULL DEFAULT '',
    action        TEXT NOT NULL DEFAULT '',
    status        TEXT NOT NULL DEFAULT '',
    duration_ms   BIGINT NOT NULL DEFAULT 0,
    error_message TEXT NOT NULL DEFAULT '',
    signal_type   TEXT NOT NULL DEFAULT '',
    signal_volume NUMERIC(20,8) NOT NULL DEFAULT 0,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_schedule_run_logs_user_schedule
    ON schedule_run_logs (user_id, schedule_id, created_at DESC);

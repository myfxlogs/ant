-- Gate evaluation results for backtest runs with auto_gate=true.
-- Persists 7-gate pipeline results and marketplace quality preview
-- so they survive stream reconnection / page refresh.
CREATE TABLE IF NOT EXISTS gate_evaluations (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    backtest_run_id UUID NOT NULL REFERENCES backtest_runs(id) ON DELETE CASCADE,
    user_id         UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    -- Serialized ant.v1.GateEvaluationUpdate (pipeline summary + individual gates)
    gate_result     BYTEA,
    -- Serialized ant.v1.MarketplaceQualityPreview
    quality_preview BYTEA,
    -- Overall pass/fail from the 7-gate pipeline
    passed          BOOLEAN NOT NULL DEFAULT false,
    first_fail      TEXT NOT NULL DEFAULT '',
    summary         TEXT NOT NULL DEFAULT '',
    -- Marketplace publishable flag
    publishable     BOOLEAN NOT NULL DEFAULT false,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- One evaluation per backtest run (upsert pattern).
CREATE UNIQUE INDEX IF NOT EXISTS gate_evaluations_run_id_uniq ON gate_evaluations (backtest_run_id);
CREATE INDEX IF NOT EXISTS gate_evaluations_user_id ON gate_evaluations (user_id);

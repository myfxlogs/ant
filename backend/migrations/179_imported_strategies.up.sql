-- 179_imported_strategies.up.sql
-- User-imported MQL strategies stored as raw source (single source of truth).
-- Execution path: strategyID → fetch source_code → CompileToIR → interp.
-- gen.go Go output is NOT stored — it's a read-only export preview.

CREATE TABLE IF NOT EXISTS imported_strategies (
    id              UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id         UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    name            TEXT NOT NULL,
    source_lang     TEXT NOT NULL DEFAULT 'mql4',   -- "mql4" | "mql5"
    source_code     TEXT NOT NULL,                   -- raw MQL source (fact source)
    params          BYTEA,                           -- interp binary (SerializeParams), NULL if none
    coverage_score  NUMERIC(5,4) NOT NULL DEFAULT 0.0,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_imported_strategies_user ON imported_strategies(user_id);
CREATE INDEX IF NOT EXISTS idx_imported_strategies_user_updated ON imported_strategies(user_id, updated_at DESC);

COMMENT ON TABLE imported_strategies IS 'User-imported MQL strategies. source_code is the single source of truth for execution.';
COMMENT ON COLUMN imported_strategies.params IS 'Serialized ParamDecl list (interp binary format). NOT JSON.';

-- 194_strategy_versions.up.sql
-- Strategy version history: snapshots of strategy code on each save/update.
-- Enables diff, rollback, and audit trail for imported_strategies.

CREATE TABLE IF NOT EXISTS strategy_versions (
    id              UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    strategy_id     UUID NOT NULL REFERENCES imported_strategies(id) ON DELETE CASCADE,
    user_id         UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    version_number  INTEGER NOT NULL,
    source_code     TEXT NOT NULL,
    source_lang     TEXT NOT NULL DEFAULT 'mql4',
    change_summary  TEXT NOT NULL DEFAULT '',
    code_hash       TEXT NOT NULL DEFAULT '',
    created_at      TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(strategy_id, version_number)
);

CREATE INDEX IF NOT EXISTS idx_strategy_versions_strategy ON strategy_versions(strategy_id, version_number DESC);
CREATE INDEX IF NOT EXISTS idx_strategy_versions_user ON strategy_versions(user_id);

COMMENT ON TABLE strategy_versions IS 'Snapshots of strategy source code on each save. Enables diff, rollback, and audit trail.';
COMMENT ON COLUMN strategy_versions.code_hash IS 'SHA-256 hash of source_code for quick change detection.';

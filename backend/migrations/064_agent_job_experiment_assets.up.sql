CREATE TABLE IF NOT EXISTS agent_tokens (
    id UUID PRIMARY KEY,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    token_prefix TEXT NOT NULL,
    token_hash TEXT NOT NULL UNIQUE,
    scopes TEXT[] NOT NULL DEFAULT '{}',
    account_allowlist TEXT[] NOT NULL DEFAULT '{}',
    symbol_allowlist TEXT[] NOT NULL DEFAULT '{}',
    paper_only BOOLEAN NOT NULL DEFAULT TRUE,
    rate_limit_per_min INTEGER NOT NULL DEFAULT 60,
    expires_at TIMESTAMPTZ,
    status TEXT NOT NULL DEFAULT 'active',
    last_used_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_agent_tokens_user_id ON agent_tokens(user_id);
CREATE INDEX IF NOT EXISTS idx_agent_tokens_status ON agent_tokens(status);

CREATE TABLE IF NOT EXISTS agent_audit_logs (
    id UUID PRIMARY KEY,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    agent_token_id UUID REFERENCES agent_tokens(id) ON DELETE SET NULL,
    agent_name TEXT NOT NULL DEFAULT '',
    rpc_service TEXT NOT NULL,
    rpc_method TEXT NOT NULL,
    scope TEXT NOT NULL DEFAULT '',
    status_code TEXT NOT NULL DEFAULT '',
    idempotency_key TEXT NOT NULL DEFAULT '',
    risk_decision TEXT NOT NULL DEFAULT '',
    request_summary TEXT NOT NULL DEFAULT '',
    response_summary TEXT NOT NULL DEFAULT '',
    duration_ms BIGINT NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_agent_audit_logs_user_id_created_at ON agent_audit_logs(user_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_agent_audit_logs_agent_token_id ON agent_audit_logs(agent_token_id);

CREATE TABLE IF NOT EXISTS jobs (
    id UUID PRIMARY KEY,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    kind TEXT NOT NULL,
    status TEXT NOT NULL,
    progress DOUBLE PRECISION NOT NULL DEFAULT 0,
    stage TEXT NOT NULL DEFAULT '',
    request_summary TEXT NOT NULL DEFAULT '',
    result_ref TEXT NOT NULL DEFAULT '',
    result_summary TEXT NOT NULL DEFAULT '',
    error_code TEXT NOT NULL DEFAULT '',
    error_message TEXT NOT NULL DEFAULT '',
    idempotency_key TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    started_at TIMESTAMPTZ,
    finished_at TIMESTAMPTZ,
    expires_at TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS idx_jobs_user_id_created_at ON jobs(user_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_jobs_status ON jobs(status);
CREATE UNIQUE INDEX IF NOT EXISTS idx_jobs_user_kind_idempotency ON jobs(user_id, kind, idempotency_key) WHERE idempotency_key <> '';

CREATE TABLE IF NOT EXISTS job_events (
    id UUID PRIMARY KEY,
    job_id UUID NOT NULL REFERENCES jobs(id) ON DELETE CASCADE,
    seq BIGINT NOT NULL,
    type TEXT NOT NULL,
    status TEXT NOT NULL DEFAULT '',
    progress DOUBLE PRECISION NOT NULL DEFAULT 0,
    stage TEXT NOT NULL DEFAULT '',
    message TEXT NOT NULL DEFAULT '',
    payload JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE(job_id, seq)
);

CREATE INDEX IF NOT EXISTS idx_job_events_job_id_seq ON job_events(job_id, seq);

CREATE TABLE IF NOT EXISTS strategy_experiments (
    id UUID PRIMARY KEY,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    base_template_id UUID,
    status TEXT NOT NULL,
    parameter_space JSONB NOT NULL DEFAULT '{}'::jsonb,
    search_method TEXT NOT NULL DEFAULT 'grid',
    max_candidates INTEGER NOT NULL DEFAULT 20,
    objective TEXT NOT NULL DEFAULT '',
    market_regime_ref TEXT NOT NULL DEFAULT '',
    best_candidate_id UUID,
    job_id UUID REFERENCES jobs(id) ON DELETE SET NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    finished_at TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS idx_strategy_experiments_user_id_created_at ON strategy_experiments(user_id, created_at DESC);

CREATE TABLE IF NOT EXISTS strategy_experiment_candidates (
    id UUID PRIMARY KEY,
    experiment_id UUID NOT NULL REFERENCES strategy_experiments(id) ON DELETE CASCADE,
    parameters JSONB NOT NULL DEFAULT '{}'::jsonb,
    draft_code_ref TEXT NOT NULL DEFAULT '',
    backtest_run_id UUID,
    score DOUBLE PRECISION NOT NULL DEFAULT 0,
    grade TEXT NOT NULL DEFAULT '',
    score_components JSONB NOT NULL DEFAULT '{}'::jsonb,
    rank INTEGER NOT NULL DEFAULT 0,
    summary TEXT NOT NULL DEFAULT '',
    recommendation TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_strategy_experiment_candidates_experiment_id_rank ON strategy_experiment_candidates(experiment_id, rank);

CREATE TABLE IF NOT EXISTS strategy_assets (
    id UUID PRIMARY KEY,
    owner_user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    source_template_id UUID NOT NULL,
    source_version INTEGER NOT NULL DEFAULT 1,
    name TEXT NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    visibility TEXT NOT NULL DEFAULT 'private',
    review_status TEXT NOT NULL DEFAULT 'pending',
    rating_summary TEXT NOT NULL DEFAULT '',
    clone_count INTEGER NOT NULL DEFAULT 0,
    latest_version INTEGER NOT NULL DEFAULT 1,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_strategy_assets_owner_user_id ON strategy_assets(owner_user_id);
CREATE INDEX IF NOT EXISTS idx_strategy_assets_visibility_review_status ON strategy_assets(visibility, review_status);

CREATE TABLE IF NOT EXISTS strategy_asset_clones (
    id UUID PRIMARY KEY,
    asset_id UUID NOT NULL REFERENCES strategy_assets(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    cloned_template_id UUID NOT NULL,
    source_version INTEGER NOT NULL DEFAULT 1,
    sync_available BOOLEAN NOT NULL DEFAULT FALSE,
    last_sync_check_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_strategy_asset_clones_user_id ON strategy_asset_clones(user_id);
CREATE INDEX IF NOT EXISTS idx_strategy_asset_clones_asset_id ON strategy_asset_clones(asset_id);

CREATE TABLE IF NOT EXISTS market_regimes (
    id UUID PRIMARY KEY,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    account_id UUID NOT NULL,
    symbol TEXT NOT NULL,
    timeframe TEXT NOT NULL,
    regime TEXT NOT NULL,
    confidence DOUBLE PRECISION NOT NULL DEFAULT 0,
    features JSONB NOT NULL DEFAULT '{}'::jsonb,
    segments JSONB NOT NULL DEFAULT '[]'::jsonb,
    strategy_families TEXT[] NOT NULL DEFAULT '{}',
    from_time TIMESTAMPTZ,
    to_time TIMESTAMPTZ,
    model_version TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_market_regimes_user_symbol_created_at ON market_regimes(user_id, symbol, created_at DESC);

CREATE TABLE IF NOT EXISTS kline_cache_metadata (
    id UUID PRIMARY KEY,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    cache_key TEXT NOT NULL UNIQUE,
    source TEXT NOT NULL,
    account_id UUID,
    symbol TEXT NOT NULL,
    timeframe TEXT NOT NULL,
    start_time TIMESTAMPTZ,
    end_time TIMESTAMPTZ,
    hit BOOLEAN NOT NULL DEFAULT FALSE,
    ttl_seconds INTEGER NOT NULL DEFAULT 0,
    fetched_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    expired_at TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS idx_kline_cache_metadata_user_symbol ON kline_cache_metadata(user_id, symbol, timeframe);

-- 221_marketplace_trials.up.sql
-- Free trial tracking for marketplace strategies.

CREATE TABLE IF NOT EXISTS marketplace_trials (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    strategy_id UUID NOT NULL REFERENCES marketplace_strategies(strategy_id) ON DELETE CASCADE,
    started_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    expires_at TIMESTAMPTZ NOT NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'active', -- active | expired | cancelled
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- One trial per user per strategy.
CREATE UNIQUE INDEX idx_marketplace_trials_user_strategy ON marketplace_trials(user_id, strategy_id);
CREATE INDEX idx_marketplace_trials_status ON marketplace_trials(status);
CREATE INDEX idx_marketplace_trials_expires ON marketplace_trials(expires_at);

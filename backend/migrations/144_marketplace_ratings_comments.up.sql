-- 144: marketplace ratings + comments tables.
CREATE TABLE IF NOT EXISTS marketplace_ratings (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    strategy_id UUID NOT NULL,          -- references marketplace_strategies.id
    user_id UUID NOT NULL,
    rating INTEGER NOT NULL CHECK (rating >= 1 AND rating <= 5),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (strategy_id, user_id)
);

CREATE INDEX IF NOT EXISTS idx_marketplace_ratings_strategy
    ON marketplace_ratings(strategy_id);

CREATE TABLE IF NOT EXISTS marketplace_comments (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    strategy_id UUID NOT NULL,          -- references marketplace_strategies.id
    user_id UUID NOT NULL,
    content TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_marketplace_comments_strategy
    ON marketplace_comments(strategy_id, created_at);

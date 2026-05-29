-- User watchlist for favorite trading symbols (B-4.2).
-- Phase 1: schema only. RPC + UI wiring deferred to B-4.3.
CREATE TABLE IF NOT EXISTS user_watchlists (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    symbol_canonical VARCHAR(20) NOT NULL,
    sort_order INT NOT NULL DEFAULT 0,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(user_id, symbol_canonical)
);

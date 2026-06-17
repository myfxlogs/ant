-- 162_share_tokens.up.sql
-- Public performance sharing: generates expiring tokens for read-only access.

CREATE TABLE IF NOT EXISTS share_tokens (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    account_id UUID NOT NULL,
    token VARCHAR(32) UNIQUE NOT NULL,
    description TEXT,
    expires_at TIMESTAMPTZ NOT NULL DEFAULT (NOW() + INTERVAL '7 days'),
    view_count INT NOT NULL DEFAULT 0,
    max_views INT,  -- NULL = unlimited
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_share_tokens_token ON share_tokens(token);
CREATE INDEX IF NOT EXISTS idx_share_tokens_user_id ON share_tokens(user_id);

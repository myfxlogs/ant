-- In-app user notification inbox (B-2.6).
-- Schema this round; ConnectRPC handler + UI deferred to P1.
CREATE TABLE IF NOT EXISTS user_notifications (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    type VARCHAR(50) NOT NULL,       -- margin_call | risk_alert | system | trade
    title VARCHAR(255) NOT NULL,
    message TEXT NOT NULL DEFAULT '',
    data JSONB DEFAULT '{}',
    is_read BOOLEAN NOT NULL DEFAULT false,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_user_notifications_user_unread
    ON user_notifications(user_id, is_read, created_at DESC);

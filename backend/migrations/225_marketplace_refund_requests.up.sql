-- 225_marketplace_refund_requests.up.sql
CREATE TABLE IF NOT EXISTS marketplace_refund_requests (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id),
    subscription_id UUID NOT NULL REFERENCES user_subscriptions(id),
    reason TEXT NOT NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'pending',
    reviewed_by UUID REFERENCES users(id),
    review_note TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    reviewed_at TIMESTAMPTZ
);
CREATE INDEX IF NOT EXISTS idx_refund_requests_status ON marketplace_refund_requests(status);
CREATE INDEX IF NOT EXISTS idx_refund_requests_user ON marketplace_refund_requests(user_id);

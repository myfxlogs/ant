-- 215_provider_verification.up.sql
-- Provider identity verification for marketplace.

ALTER TABLE users ADD COLUMN IF NOT EXISTS verified_provider BOOLEAN NOT NULL DEFAULT false;
ALTER TABLE users ADD COLUMN IF NOT EXISTS provider_type VARCHAR(20) NOT NULL DEFAULT 'human';

CREATE TABLE IF NOT EXISTS provider_verification_requests (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    provider_type VARCHAR(20) NOT NULL DEFAULT 'human',
    status VARCHAR(20) NOT NULL DEFAULT 'pending',
    reviewed_by UUID REFERENCES users(id),
    review_note TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    reviewed_at TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS idx_provider_verification_user ON provider_verification_requests(user_id);
CREATE INDEX IF NOT EXISTS idx_provider_verification_status ON provider_verification_requests(status);

-- 196_email_verification.up.sql
-- P3.4: Email verification flow — add email_verified_at to users + verification tokens table.

ALTER TABLE users ADD COLUMN IF NOT EXISTS email_verified_at TIMESTAMPTZ;

COMMENT ON COLUMN users.email_verified_at IS 'When the user verified their email. NULL = not verified.';

CREATE TABLE IF NOT EXISTS email_verification_tokens (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id     UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    token_hash  VARCHAR(64) NOT NULL UNIQUE,
    expires_at  TIMESTAMPTZ NOT NULL DEFAULT (now() + INTERVAL '24 hours'),
    used_at     TIMESTAMPTZ,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_email_verif_tokens_user ON email_verification_tokens(user_id);
CREATE INDEX idx_email_verif_tokens_hash ON email_verification_tokens(token_hash) WHERE used_at IS NULL;

COMMENT ON TABLE email_verification_tokens IS 'Email verification tokens for P3.4 registration flow';

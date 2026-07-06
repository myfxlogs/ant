-- 188_audit_log: account operation audit trail
-- Date: 2026-07-06

CREATE TABLE IF NOT EXISTS account_audit_log (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    account_id  UUID NOT NULL,
    user_id     UUID NOT NULL,
    action      VARCHAR(32) NOT NULL,  -- create, delete, update, connect, disconnect, freeze, unfreeze
    detail      VARCHAR(512),          -- human-readable summary
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_audit_log_account ON account_audit_log (account_id);
CREATE INDEX idx_audit_log_user ON account_audit_log (user_id);
CREATE INDEX idx_audit_log_created ON account_audit_log (created_at);

-- 092_risk_events.up.sql
-- M2-5: risk_events audit table — records every risk check result for compliance.

CREATE TABLE IF NOT EXISTS risk_events (
    id          UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    order_id    UUID REFERENCES orders(id) ON DELETE SET NULL,
    account_id  UUID NOT NULL,
    rule_name   VARCHAR(64) NOT NULL,        -- e.g. max_position, daily_loss, margin
    result      VARCHAR(16) NOT NULL,         -- PASS, BLOCK, WARN
    detail      JSONB,                        -- rule-specific detail (e.g. {"current": 5, "limit": 3})
    created_at  timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_risk_events_order ON risk_events(order_id);
CREATE INDEX IF NOT EXISTS idx_risk_events_account ON risk_events(account_id);
CREATE INDEX IF NOT EXISTS idx_risk_events_created ON risk_events(created_at);

COMMENT ON TABLE risk_events IS 'Audit log for every risk rule evaluation — immutable compliance record';

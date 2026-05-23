-- 096_factor_definitions.up.sql
-- Factor definitions table for the DSL-based factor engine (M7.3).
-- Each row defines a named factor expression that can be referenced
-- by strategies and other factors.

CREATE TABLE IF NOT EXISTS factor_definitions (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name         TEXT NOT NULL UNIQUE,
    expression   TEXT NOT NULL,
    description  TEXT NOT NULL DEFAULT '',
    symbols      TEXT[] NOT NULL DEFAULT '{}',
    owner_user_id UUID NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    active       BOOLEAN NOT NULL DEFAULT true,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_factor_definitions_owner ON factor_definitions (owner_user_id);
CREATE INDEX idx_factor_definitions_active ON factor_definitions (active) WHERE active = true;

-- Seed 5 built-in factors
INSERT INTO factor_definitions (name, expression, description, owner_user_id, active)
SELECT
    'momentum_5',  'pct_change($close, 5)',  '5-period price momentum', id, true
FROM users WHERE email = (SELECT email FROM users ORDER BY created_at ASC LIMIT 1)
UNION ALL
SELECT
    'volatility_20', 'std(pct_change($close, 1), 20)', '20-period return volatility', id, true
FROM users WHERE email = (SELECT email FROM users ORDER BY created_at ASC LIMIT 1)
UNION ALL
SELECT
    'rsi_14',    'rsi($close, 14)',            '14-period RSI', id, true
FROM users WHERE email = (SELECT email FROM users ORDER BY created_at ASC LIMIT 1)
UNION ALL
SELECT
    'ma_ratio',  'ema($close, 20) / ema($close, 60) - 1', 'EMA 20/60 ratio', id, true
FROM users WHERE email = (SELECT email FROM users ORDER BY created_at ASC LIMIT 1)
UNION ALL
SELECT
    'bb_position', '($close - bb_lower($close, 20, 2)) / (bb_upper($close, 20, 2) - bb_lower($close, 20, 2))', 'Bollinger Band position', id, true
FROM users WHERE email = (SELECT email FROM users ORDER BY created_at ASC LIMIT 1);

-- 110_platform_tables.up.sql
-- M8.0-1: C2C platform architecture — shared + user-private tables (ADR-0006)

-- Platform shared (no user_id, all users can read)
CREATE TABLE IF NOT EXISTS platform_strategies (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name            TEXT NOT NULL,
    description     TEXT,
    expression      TEXT NOT NULL,
    canonicals      TEXT[] NOT NULL DEFAULT '{}',
    periods         TEXT[] NOT NULL DEFAULT '{1m,5m,15m,1h,4h,1d}',
    category        TEXT NOT NULL DEFAULT 'custom',
    is_official     BOOLEAN NOT NULL DEFAULT false,
    published_by    UUID REFERENCES users(id),
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS platform_factors (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name            TEXT NOT NULL UNIQUE,
    expression      TEXT NOT NULL,
    description     TEXT,
    canonicals      TEXT[] NOT NULL DEFAULT '{}',
    periods         TEXT[] NOT NULL DEFAULT '{}',
    lookback        INT NOT NULL DEFAULT 100,
    category        TEXT NOT NULL DEFAULT 'builtin',
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS platform_ai_agents (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name            TEXT NOT NULL UNIQUE,
    role            TEXT NOT NULL,
    system_prompt   TEXT NOT NULL,
    model           TEXT NOT NULL DEFAULT 'deepseek-chat',
    temperature     DOUBLE PRECISION NOT NULL DEFAULT 0.7,
    is_default      BOOLEAN NOT NULL DEFAULT false,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS admins (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id         UUID REFERENCES users(id),
    scope           TEXT NOT NULL DEFAULT 'platform:admin',
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- User private (must have user_id + RLS)
CREATE TABLE IF NOT EXISTS user_subscriptions (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    subscriber_user_id  UUID NOT NULL REFERENCES users(id),
    target_user_id      UUID REFERENCES users(id),
    target_strategy_id  UUID,
    kind                TEXT NOT NULL DEFAULT 'strategy',  -- strategy | copy_trade
    active              BOOLEAN NOT NULL DEFAULT true,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (subscriber_user_id, target_strategy_id, kind)
);

CREATE TABLE IF NOT EXISTS user_strategy_publishes (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id         UUID NOT NULL REFERENCES users(id),
    platform_strategy_id UUID NOT NULL REFERENCES platform_strategies(id),
    royalty_pct     INT NOT NULL DEFAULT 0,
    published_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (user_id, platform_strategy_id)
);

CREATE TABLE IF NOT EXISTS copy_trade_links (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    subscription_id     UUID NOT NULL REFERENCES user_subscriptions(id),
    from_account_id     UUID NOT NULL,
    to_account_id       UUID NOT NULL,
    ratio               DOUBLE PRECISION NOT NULL DEFAULT 1.0,
    max_lots            DOUBLE PRECISION NOT NULL DEFAULT 10.0,
    active              BOOLEAN NOT NULL DEFAULT true,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS user_factor_overrides (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id         UUID NOT NULL REFERENCES users(id),
    factor_name     TEXT NOT NULL,
    expression      TEXT,
    canonicals      TEXT[],
    periods         TEXT[],
    lookback        INT,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (user_id, factor_name)
);

CREATE TABLE IF NOT EXISTS user_ai_agents (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id         UUID NOT NULL REFERENCES users(id),
    agent_name      TEXT NOT NULL,
    system_prompt   TEXT,
    model           TEXT,
    temperature     DOUBLE PRECISION,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (user_id, agent_name)
);

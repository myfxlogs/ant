-- 113_paper_trading: paper trading tables (ADR-0015)
-- Paper accounts, orders, and strategies are isolated from live trading data.

CREATE TABLE IF NOT EXISTS paper_accounts (
    id              UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id         UUID NOT NULL,
    name            VARCHAR(100) NOT NULL,
    initial_balance NUMERIC(20,8) NOT NULL DEFAULT 10000,
    current_balance NUMERIC(20,8) NOT NULL DEFAULT 10000,
    equity          NUMERIC(20,8) NOT NULL DEFAULT 10000,
    currency        VARCHAR(10) NOT NULL DEFAULT 'USD',
    created_at      TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    archived        BOOLEAN NOT NULL DEFAULT false
);

CREATE TABLE IF NOT EXISTS paper_orders (
    id                UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    paper_account_id  UUID NOT NULL REFERENCES paper_accounts(id) ON DELETE CASCADE,
    strategy_id       UUID,
    symbol            VARCHAR(20) NOT NULL,
    side              VARCHAR(10) NOT NULL,
    volume            NUMERIC(10,2) NOT NULL,
    fill_price        NUMERIC(18,8),
    slippage_bps      NUMERIC(10,2) DEFAULT 0,
    state             VARCHAR(20) NOT NULL DEFAULT 'PENDING',
    created_at        TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    closed_at         TIMESTAMPTZ
);
CREATE INDEX idx_paper_orders_account ON paper_orders(paper_account_id);
CREATE INDEX idx_paper_orders_strategy ON paper_orders(strategy_id);

CREATE TABLE IF NOT EXISTS paper_strategies (
    id               UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id          UUID NOT NULL,
    name             VARCHAR(200) NOT NULL,
    description      TEXT,
    category         VARCHAR(50),
    dsl_code         TEXT NOT NULL,
    backtest_metrics JSONB,
    promoted_at      TIMESTAMPTZ,
    created_at       TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    archived         BOOLEAN NOT NULL DEFAULT false
);
CREATE INDEX idx_paper_strategies_user ON paper_strategies(user_id);

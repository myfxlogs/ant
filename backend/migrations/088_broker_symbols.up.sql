-- 088_broker_symbols.up.sql
-- M1 broker symbol 映射 — broker-specific symbol → canonical
-- 每个 broker 连接后会拉取 symbol 列表写入此表

CREATE TABLE IF NOT EXISTS broker_symbols (
    broker_id        uuid NOT NULL,                  -- references accounts.id (broker connection)
    symbol_raw       text NOT NULL,                  -- broker-native symbol (e.g. BTCUSDm, EURUSD.x)
    canonical        text REFERENCES canonical_symbols(canonical),
    digits           smallint,                       -- price digits
    point            numeric,
    tick_size        numeric,
    contract_size    numeric,
    min_lot          numeric,
    max_lot          numeric,
    lot_step         numeric,
    swap_long        numeric,
    swap_short       numeric,
    trade_mode       smallint NOT NULL DEFAULT 0,   -- 0=disabled, 1=long_only, 2=short_only, 3=full
    description      text,
    sessions_quote   jsonb,
    sessions_trade   jsonb,
    raw_payload      jsonb,                          -- original broker response for debugging
    partial          boolean NOT NULL DEFAULT false, -- true if broker returned incomplete data
    updated_at       timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (broker_id, symbol_raw)
);

CREATE INDEX IF NOT EXISTS idx_broker_symbols_canonical ON broker_symbols(canonical);
CREATE INDEX IF NOT EXISTS idx_broker_symbols_broker ON broker_symbols(broker_id);

COMMENT ON TABLE broker_symbols IS 'Broker-native symbol list — populated by symbolsync on broker connect';

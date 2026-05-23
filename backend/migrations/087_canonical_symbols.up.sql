-- 087_canonical_symbols.up.sql
-- M1 canonical symbol 基础设施 — 归一化品种字典
-- 所有 broker-specific symbol 都映射到一个 canonical symbol

CREATE TABLE IF NOT EXISTS canonical_symbols (
    canonical    text PRIMARY KEY,
    asset_class  text NOT NULL,       -- forex, crypto, commodity, index, stock
    base_ccy     text NOT NULL,       -- e.g. BTC, EUR, XAU
    quote_ccy    text NOT NULL,       -- e.g. USD, JPY
    description  text NOT NULL,       -- human-readable name
    enabled      boolean NOT NULL DEFAULT true,
    created_at   timestamptz NOT NULL DEFAULT now()
);

COMMENT ON TABLE canonical_symbols IS 'Normalised symbol dictionary — every broker symbol maps to exactly one canonical';

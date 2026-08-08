-- KB-P0: Knowledge base compatibility tables (K1 facts + L0 fixes).
-- Pure typed columns, zero JSONB. Single source for MQL compat knowledge.

-- K1: Compatibility facts (constants / functions / indicators with status + severity)
CREATE TABLE IF NOT EXISTS kb_compat_fact (
    id             BIGSERIAL PRIMARY KEY,
    identifier     TEXT NOT NULL,               -- "clrGreen" / "iCustom" / "OrderSend"
    kind           TEXT NOT NULL,               -- constant | function | indicator
    status         TEXT NOT NULL,               -- supported | unsupported | partial
    severity       TEXT NOT NULL DEFAULT 'info',-- fatal | warning | info
    value_text     TEXT,                        -- constant value (text) / bool
    value_numeric  NUMERIC,                     -- constant value (numeric)
    mapping_target TEXT,                        -- alias target (clrGreen → Green)
    source         TEXT NOT NULL DEFAULT 'seed',-- seed | manual | auto-verified
    verified_at    TIMESTAMPTZ,
    created_at     TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (identifier, kind)
);

CREATE INDEX IF NOT EXISTS idx_kb_compat_fact_id ON kb_compat_fact(identifier);

-- L0: Deterministic fixes (problem pattern → transparent transform rule)
CREATE TABLE IF NOT EXISTS kb_compat_fix (
    id               BIGSERIAL PRIMARY KEY,
    pattern          TEXT NOT NULL,             -- trigger identifier/pattern
    fix_type         TEXT NOT NULL,             -- alias | rename | normalize
    resolution_target TEXT NOT NULL,            -- deterministic resolution target
    source           TEXT NOT NULL DEFAULT 'manual',
    created_at       TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (pattern)
);

COMMENT ON TABLE kb_compat_fact IS 'KB-P0: MQL compatibility facts (K1) — constants, functions, indicators with status/severity.';
COMMENT ON TABLE kb_compat_fix IS 'KB-P0: Deterministic compatibility fixes (L0) — alias/rename/normalize rules.';

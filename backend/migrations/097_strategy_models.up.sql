-- 097_strategy_models.up.sql
-- ONNX model registry for quantengine (M7.5).
-- Each row maps a strategy to an ONNX model artifact.

CREATE TABLE IF NOT EXISTS strategy_models (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    strategy_id  UUID NOT NULL REFERENCES strategies (id) ON DELETE CASCADE,
    version      INT NOT NULL DEFAULT 1,
    onnx_uri     TEXT NOT NULL,
    inputs       JSONB NOT NULL DEFAULT '[]',
    outputs      JSONB NOT NULL DEFAULT '[]',
    owner_user_id UUID NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    active       BOOLEAN NOT NULL DEFAULT true,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (strategy_id, version)
);

CREATE INDEX idx_strategy_models_strategy ON strategy_models (strategy_id);
CREATE INDEX idx_strategy_models_active ON strategy_models (active) WHERE active = true;

-- 139: AI confidence calibration table.
-- Tracks prediction accuracy per confidence bucket to enable
-- automatic threshold optimization over time (QuantDinger ai_calibration.py).

CREATE TABLE IF NOT EXISTS ai_calibrations (
    id BIGSERIAL PRIMARY KEY,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    confidence_bucket INT NOT NULL CHECK (confidence_bucket BETWEEN 0 AND 100),
    total_predictions INT NOT NULL DEFAULT 0,
    correct_predictions INT NOT NULL DEFAULT 0,
    accuracy DOUBLE PRECISION NOT NULL DEFAULT 0,
    calibrated_threshold DOUBLE PRECISION,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (user_id, confidence_bucket)
);

CREATE TABLE IF NOT EXISTS ai_predictions (
    id BIGSERIAL PRIMARY KEY,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    decision VARCHAR(10) NOT NULL CHECK (decision IN ('BUY','SELL','HOLD')),
    raw_confidence DOUBLE PRECISION NOT NULL,
    predicted_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    symbol VARCHAR(20),
    actual_return_pct DOUBLE PRECISION,
    was_correct BOOLEAN,
    validated_at TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS idx_ai_predictions_user ON ai_predictions(user_id, predicted_at DESC);
CREATE INDEX IF NOT EXISTS idx_ai_predictions_unvalidated ON ai_predictions(user_id, predicted_at) WHERE validated_at IS NULL;

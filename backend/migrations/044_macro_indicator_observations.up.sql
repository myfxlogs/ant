-- 044_macro_indicator_observations.up.sql
-- Time series storage for key macro indicators (CPI, unemployment, Fed funds, GDP).

CREATE TABLE macro_indicator_observations (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    indicator_code VARCHAR(50) NOT NULL,
    obs_date DATE NOT NULL,
    value NUMERIC,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(indicator_code, obs_date)
);

CREATE INDEX idx_macro_indicator_obs_indicator_date
    ON macro_indicator_observations(indicator_code, obs_date DESC);

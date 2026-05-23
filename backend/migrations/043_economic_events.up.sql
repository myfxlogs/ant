-- 043_economic_events.up.sql
-- Economic calendar events table for storing external macro release data (FRED, etc.).

CREATE TABLE economic_events (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    source VARCHAR(50) NOT NULL,
    external_id VARCHAR(100) NOT NULL,
    date VARCHAR(10) NOT NULL,
    time VARCHAR(20),
    country VARCHAR(10),
    event VARCHAR(255),
    impact VARCHAR(50),
    actual VARCHAR(50),
    previous VARCHAR(50),
    estimate VARCHAR(50),
    unit VARCHAR(20),
    currency VARCHAR(10),
    timestamp BIGINT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE UNIQUE INDEX ux_economic_events_source_external_id_date
    ON economic_events(source, external_id, date);

CREATE INDEX idx_economic_events_date
    ON economic_events(date DESC);

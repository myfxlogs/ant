-- 100_factor_definitions_v2.up.sql
-- Rebuild factor_definitions with v2 schema (canonicals, periods, lookback).
-- First line drops v1 table (migration 096) to replace with v2 schema.

DROP TABLE IF EXISTS factor_definitions CASCADE;

CREATE TABLE IF NOT EXISTS factor_definitions (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id       UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    name          TEXT NOT NULL,
    expression    TEXT NOT NULL,
    canonicals    TEXT[] NOT NULL,
    periods       TEXT[] NOT NULL,
    lookback      INT NOT NULL DEFAULT 100,
    is_active     BOOLEAN NOT NULL DEFAULT true,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (user_id, name)
);

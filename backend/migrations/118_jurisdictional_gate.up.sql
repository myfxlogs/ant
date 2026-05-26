-- 118_jurisdictional_gate.up.sql
-- M11-16: Jurisdictional Gate — KYC, GeoIP country, disclaimer, risk questionnaire, sanctioned countries.
-- Creates two new tables independent of users/user_risk_profiles.

CREATE TABLE user_jurisdiction (
    user_id                    UUID PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    kyc_status                 VARCHAR(20) NOT NULL DEFAULT 'unverified',
    kyc_verified_at            TIMESTAMPTZ,
    kyc_verified_by            UUID REFERENCES users(id),
    country_code               VARCHAR(2),
    country_source             VARCHAR(20),
    disclaimer_accepted_at     TIMESTAMPTZ,
    disclaimer_version         VARCHAR(50),
    questionnaire_completed_at TIMESTAMPTZ,
    questionnaire_version      VARCHAR(50),
    risk_score                 INT,
    sanctioned_override        BOOLEAN NOT NULL DEFAULT FALSE,
    created_at                 TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at                 TIMESTAMPTZ NOT NULL DEFAULT now()
);

COMMENT ON TABLE user_jurisdiction IS 'Per-user KYC and jurisdictional compliance status for M11-16 gate';
COMMENT ON COLUMN user_jurisdiction.kyc_status IS 'unverified, pending, verified, rejected';
COMMENT ON COLUMN user_jurisdiction.sanctioned_override IS 'Admin override to allow a user from a sanctioned country';

CREATE INDEX idx_user_jurisdiction_kyc ON user_jurisdiction(kyc_status);

CREATE TABLE sanctioned_countries (
    country_code VARCHAR(2) PRIMARY KEY,
    label        VARCHAR(100) NOT NULL,
    added_by     UUID REFERENCES users(id),
    added_at     TIMESTAMPTZ NOT NULL DEFAULT now()
);

COMMENT ON TABLE sanctioned_countries IS 'Sanctioned country list (ISO 3166-1 alpha-2) managed by admins';

INSERT INTO sanctioned_countries (country_code, label) VALUES
    ('IR', 'Iran'), ('KP', 'North Korea'), ('SY', 'Syria'),
    ('CU', 'Cuba'), ('VE', 'Venezuela');

SELECT create_updated_at_trigger('user_jurisdiction');

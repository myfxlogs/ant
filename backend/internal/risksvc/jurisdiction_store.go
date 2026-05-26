package risksvc

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// PgJurisdictionStore implements JurisdictionStore backed by PostgreSQL.
// Used by JurisdictionGate in the trade pipeline for quick lookups.
type PgJurisdictionStore struct {
	pool *pgxpool.Pool
}

// NewPgJurisdictionStore creates a PG-backed jurisdiction store.
func NewPgJurisdictionStore(pool *pgxpool.Pool) *PgJurisdictionStore {
	return &PgJurisdictionStore{pool: pool}
}

func (s *PgJurisdictionStore) GetStatus(ctx context.Context, userID string) (*JurisdictionStatus, error) {
	var st JurisdictionStatus
	err := s.pool.QueryRow(ctx, `
		SELECT COALESCE(uj.user_id, $1),
		       COALESCE(uj.kyc_status, 'unverified'),
		       COALESCE(uj.country_code, ''),
		       COALESCE(uj.country_source, ''),
		       COALESCE(uj.disclaimer_accepted_at IS NOT NULL, false),
		       COALESCE(uj.questionnaire_completed_at IS NOT NULL, false),
		       COALESCE(uj.risk_score, 0),
		       COALESCE(uj.sanctioned_override, false),
		       uj.kyc_verified_at,
		       uj.disclaimer_accepted_at,
		       uj.questionnaire_completed_at
		FROM (SELECT $1::uuid AS user_id) AS u
		LEFT JOIN user_jurisdiction uj ON uj.user_id = u.user_id
	`, userID).Scan(
		&st.UserID, &st.KYCStatus,
		&st.CountryCode, &st.CountrySource,
		&st.DisclaimerAccepted, &st.QuestionnaireDone,
		&st.RiskScore, &st.SanctionedOverride,
		&st.KYCVerifiedAt, &st.DisclaimerAcceptedAt, &st.QuestionnaireDoneAt,
	)
	if err != nil {
		return &JurisdictionStatus{UserID: userID, KYCStatus: "unverified"}, nil
	}
	return &st, nil
}

func (s *PgJurisdictionStore) SetKYCStatus(ctx context.Context, userID, status, verifiedBy string) error {
	_, err := s.pool.Exec(ctx, `
		INSERT INTO user_jurisdiction (user_id, kyc_status, kyc_verified_at, kyc_verified_by, updated_at)
		VALUES ($1, $2, CASE WHEN $2 = 'verified' THEN now() ELSE NULL END, CASE WHEN $2 = 'verified' THEN $3::uuid ELSE NULL END, now())
		ON CONFLICT (user_id) DO UPDATE SET
		    kyc_status = EXCLUDED.kyc_status,
		    kyc_verified_at = CASE WHEN EXCLUDED.kyc_status = 'verified' THEN now() ELSE user_jurisdiction.kyc_verified_at END,
		    kyc_verified_by = CASE WHEN EXCLUDED.kyc_status = 'verified' THEN EXCLUDED.kyc_verified_by ELSE user_jurisdiction.kyc_verified_by END,
		    updated_at = now()
	`, userID, status, verifiedBy)
	return err
}

func (s *PgJurisdictionStore) RecordCountry(ctx context.Context, userID, countryCode, source string) error {
	_, err := s.pool.Exec(ctx, `
		INSERT INTO user_jurisdiction (user_id, country_code, country_source, updated_at)
		VALUES ($1, $2, $3, now())
		ON CONFLICT (user_id) DO UPDATE SET
		    country_code = EXCLUDED.country_code,
		    country_source = EXCLUDED.country_source,
		    updated_at = now()
	`, userID, countryCode, source)
	return err
}

func (s *PgJurisdictionStore) IsDisclaimerAccepted(ctx context.Context, userID string) (bool, error) {
	var accepted bool
	err := s.pool.QueryRow(ctx, `
		SELECT COALESCE(disclaimer_accepted_at IS NOT NULL, false)
		FROM user_jurisdiction WHERE user_id = $1
	`, userID).Scan(&accepted)
	if err != nil {
		return false, nil // no row = not accepted
	}
	return accepted, nil
}

func (s *PgJurisdictionStore) AcceptDisclaimer(ctx context.Context, userID, version string) error {
	_, err := s.pool.Exec(ctx, `
		INSERT INTO user_jurisdiction (user_id, disclaimer_accepted_at, disclaimer_version, updated_at)
		VALUES ($1, now(), $2, now())
		ON CONFLICT (user_id) DO UPDATE SET
		    disclaimer_accepted_at = now(),
		    disclaimer_version = EXCLUDED.disclaimer_version,
		    updated_at = now()
	`, userID, version)
	return err
}

func (s *PgJurisdictionStore) IsQuestionnaireCompleted(ctx context.Context, userID string) (bool, error) {
	var done bool
	err := s.pool.QueryRow(ctx, `
		SELECT COALESCE(questionnaire_completed_at IS NOT NULL, false)
		FROM user_jurisdiction WHERE user_id = $1
	`, userID).Scan(&done)
	if err != nil {
		return false, nil
	}
	return done, nil
}

func (s *PgJurisdictionStore) SubmitQuestionnaire(ctx context.Context, userID, version string, riskScore int) error {
	_, err := s.pool.Exec(ctx, `
		INSERT INTO user_jurisdiction (user_id, questionnaire_completed_at, questionnaire_version, risk_score, updated_at)
		VALUES ($1, now(), $2, $3, now())
		ON CONFLICT (user_id) DO UPDATE SET
		    questionnaire_completed_at = now(),
		    questionnaire_version = EXCLUDED.questionnaire_version,
		    risk_score = EXCLUDED.risk_score,
		    updated_at = now()
	`, userID, version, riskScore)
	return err
}

func (s *PgJurisdictionStore) IsSanctioned(ctx context.Context, countryCode string) (bool, error) {
	var exists bool
	err := s.pool.QueryRow(ctx, `
		SELECT EXISTS(SELECT 1 FROM sanctioned_countries WHERE country_code = $1)
	`, countryCode).Scan(&exists)
	if err != nil {
		return false, err
	}
	return exists, nil
}

// AcceptDisclaimerAt is a convenience method to read the timestamp.
func (s *PgJurisdictionStore) AcceptDisclaimerAt(ctx context.Context, userID string) (*time.Time, error) {
	var t *time.Time
	err := s.pool.QueryRow(ctx, `
		SELECT disclaimer_accepted_at FROM user_jurisdiction WHERE user_id = $1
	`, userID).Scan(&t)
	if err != nil {
		return nil, nil
	}
	return t, nil
}

// QuestionnaireCompletedAt reads the completion timestamp.
func (s *PgJurisdictionStore) QuestionnaireCompletedAt(ctx context.Context, userID string) (*time.Time, error) {
	var t *time.Time
	err := s.pool.QueryRow(ctx, `
		SELECT questionnaire_completed_at FROM user_jurisdiction WHERE user_id = $1
	`, userID).Scan(&t)
	if err != nil {
		return nil, nil
	}
	return t, nil
}

package repository

import (
	"context"
	"time"
)

// --- Jurisdiction queries on AdminRepository ---

// GetJurisdictionStatus returns the full compliance snapshot for a user.
func (r *AdminRepository) GetJurisdictionStatus(ctx context.Context, userID string) (*JurisdictionStatus, error) {
	var s JurisdictionStatus
	err := r.db.QueryRow(ctx, `
		SELECT uj.user_id, uj.kyc_status,
		       COALESCE(uj.country_code, ''),
		       COALESCE(uj.country_source, ''),
		       COALESCE(uj.disclaimer_accepted_at IS NOT NULL, false),
		       COALESCE(uj.questionnaire_completed_at IS NOT NULL, false),
		       COALESCE(uj.risk_score, 0),
		       COALESCE(uj.sanctioned_override, false),
		       uj.kyc_verified_at,
		       uj.disclaimer_accepted_at,
		       uj.questionnaire_completed_at
		FROM user_jurisdiction uj
		WHERE uj.user_id = $1
	`, userID).Scan(
		&s.UserID, &s.KYCStatus,
		&s.CountryCode, &s.CountrySource,
		&s.DisclaimerAccepted, &s.QuestionnaireDone,
		&s.RiskScore, &s.SanctionedOverride,
		&s.KYCVerifiedAt, &s.DisclaimerAcceptedAt, &s.QuestionnaireDoneAt,
	)
	if err != nil {
		if err.Error() == "no rows in result set" {
			return &JurisdictionStatus{UserID: userID, KYCStatus: "unverified"}, nil
		}
		return nil, err
	}
	return &s, nil
}

// SetKYCStatus upserts the KYC status for a user.
func (r *AdminRepository) SetKYCStatus(ctx context.Context, userID, status, verifiedBy string) error {
	_, err := r.db.Exec(ctx, `
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

// ListSanctionedCountries returns all sanctioned country codes.
func (r *AdminRepository) ListSanctionedCountries(ctx context.Context) ([]SanctionedCountry, error) {
	rows, err := r.db.Query(ctx, `
		SELECT sc.country_code, sc.label,
		       COALESCE(u.email, ''), sc.added_at
		FROM sanctioned_countries sc
		LEFT JOIN users u ON u.id = sc.added_by
		ORDER BY sc.added_at DESC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []SanctionedCountry
	for rows.Next() {
		var c SanctionedCountry
		if err := rows.Scan(&c.CountryCode, &c.Label, &c.AddedBy, &c.AddedAt); err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, nil
}

// AddSanctionedCountry adds a country to the sanctioned list.
func (r *AdminRepository) AddSanctionedCountry(ctx context.Context, code, label, addedBy string) error {
	_, err := r.db.Exec(ctx, `
		INSERT INTO sanctioned_countries (country_code, label, added_by)
		VALUES ($1, $2, $3::uuid)
		ON CONFLICT (country_code) DO UPDATE SET label = EXCLUDED.label, added_by = EXCLUDED.added_by, added_at = now()
	`, code, label, addedBy)
	return err
}

// RemoveSanctionedCountry removes a country from the sanctioned list.
func (r *AdminRepository) RemoveSanctionedCountry(ctx context.Context, code string) error {
	_, err := r.db.Exec(ctx, `
		DELETE FROM sanctioned_countries WHERE country_code = $1
	`, code)
	return err
}

// ListUsersByKYCStatus returns users filtered by KYC status with pagination.
func (r *AdminRepository) ListUsersByKYCStatus(ctx context.Context, kycStatus string, page, pageSize int) ([]UserKYCItem, int64, error) {
	page, pageSize = normalizePage(page, pageSize)
	offset := (page - 1) * pageSize

	var total int64
	countQuery := `SELECT COUNT(*) FROM users u`
	args := []interface{}{}

	if kycStatus != "" {
		countQuery += ` LEFT JOIN user_jurisdiction uj ON uj.user_id = u.id
			WHERE COALESCE(uj.kyc_status, 'unverified') = $1`
		args = append(args, kycStatus)
	}
	if err := r.db.QueryRow(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	query := `
		SELECT u.id, u.email,
		       COALESCE(uj.kyc_status, 'unverified'),
		       COALESCE(uj.country_code, ''),
		       COALESCE(uj.disclaimer_accepted_at IS NOT NULL, false),
		       COALESCE(uj.questionnaire_completed_at IS NOT NULL, false),
		       COALESCE(uj.risk_score, 0),
		       COALESCE(uj.sanctioned_override, false)
		FROM users u
		LEFT JOIN user_jurisdiction uj ON uj.user_id = u.id`

	queryArgs := make([]interface{}, 0, 3)
	if kycStatus != "" {
		query += ` WHERE COALESCE(uj.kyc_status, 'unverified') = $1`
		queryArgs = append(queryArgs, kycStatus)
	}
	query += ` ORDER BY u.created_at DESC LIMIT $` + paramNum(len(queryArgs)+1) + ` OFFSET $` + paramNum(len(queryArgs)+2)
	queryArgs = append(queryArgs, pageSize, offset)

	rows, err := r.db.Query(ctx, query, queryArgs...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var users []UserKYCItem
	for rows.Next() {
		var u UserKYCItem
		if err := rows.Scan(&u.UserID, &u.Email, &u.KYCStatus, &u.CountryCode,
			&u.DisclaimerAccepted, &u.QuestionnaireDone, &u.RiskScore, &u.SanctionedOverride); err != nil {
			return nil, 0, err
		}
		users = append(users, u)
	}
	return users, total, nil
}

// SetSanctionedOverride toggles the admin override for a sanctioned-country user.
func (r *AdminRepository) SetSanctionedOverride(ctx context.Context, userID string, override bool) error {
	_, err := r.db.Exec(ctx, `
		INSERT INTO user_jurisdiction (user_id, sanctioned_override, updated_at)
		VALUES ($1, $2, now())
		ON CONFLICT (user_id) DO UPDATE SET
		    sanctioned_override = EXCLUDED.sanctioned_override,
		    updated_at = now()
	`, userID, override)
	return err
}

// IsSanctioned checks whether a country code is in the sanctioned list.
func (r *AdminRepository) IsSanctioned(ctx context.Context, countryCode string) (bool, error) {
	var exists bool
	err := r.db.QueryRow(ctx, `
		SELECT EXISTS(SELECT 1 FROM sanctioned_countries WHERE country_code = $1)
	`, countryCode).Scan(&exists)
	if err != nil {
		return false, err
	}
	return exists, nil
}

// RecordCountry upserts the geoip-detected country for a user.
func (r *AdminRepository) RecordCountry(ctx context.Context, userID, countryCode, source string) error {
	_, err := r.db.Exec(ctx, `
		INSERT INTO user_jurisdiction (user_id, country_code, country_source, updated_at)
		VALUES ($1, $2, $3, now())
		ON CONFLICT (user_id) DO UPDATE SET
		    country_code = EXCLUDED.country_code,
		    country_source = EXCLUDED.country_source,
		    updated_at = now()
	`, userID, countryCode, source)
	return err
}

// paramNum is a helper for numbered query parameters.
func paramNum(n int) string {
	digits := []byte{'0', '1', '2', '3', '4', '5', '6', '7', '8', '9'}
	if n < 10 {
		return string(digits[n])
	}
	return string([]byte{digits[n/10], digits[n%10]})
}

// --- Repository-level model types (used by admin handler) ---

type JurisdictionStatus struct {
	UserID               string
	KYCStatus            string
	CountryCode          string
	CountrySource        string
	IsSanctioned         bool
	DisclaimerAccepted   bool
	QuestionnaireDone    bool
	RiskScore            int
	SanctionedOverride   bool
	KYCVerifiedAt        *time.Time
	DisclaimerAcceptedAt *time.Time
	QuestionnaireDoneAt  *time.Time
}

type SanctionedCountry struct {
	CountryCode string
	Label       string
	AddedBy     string
	AddedAt     time.Time
}

type UserKYCItem struct {
	UserID               string
	Email                string
	KYCStatus            string
	CountryCode          string
	DisclaimerAccepted   bool
	QuestionnaireDone    bool
	RiskScore            int
	SanctionedOverride   bool
}

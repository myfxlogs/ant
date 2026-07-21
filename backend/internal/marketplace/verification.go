package marketplace

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// RequestVerification creates a provider verification request.
// Returns the request ID and status ("pending").
func (s *Service) RequestVerification(ctx context.Context, userID, providerType, note string) (string, string, error) {
	uid, err := uuid.Parse(userID)
	if err != nil {
		return "", "", fmt.Errorf("marketplace: invalid user_id: %w", err)
	}

	if providerType != "human" && providerType != "ai" && providerType != "hybrid" {
		return "", "", fmt.Errorf("marketplace: invalid provider_type %q", providerType)
	}

	// Check if user already has a pending or approved request.
	var existingStatus string
	err = s.pg.QueryRow(ctx,
		`SELECT status FROM provider_verification_requests
		 WHERE user_id = $1 AND status IN ('pending','approved')
		 ORDER BY created_at DESC LIMIT 1`,
		uid,
	).Scan(&existingStatus)
	if err == nil {
		if existingStatus == "approved" {
			return "", "", fmt.Errorf("marketplace: already verified")
		}
		return "", "", fmt.Errorf("marketplace: verification request already pending")
	}

	id := uuid.New()
	_, err = s.pg.Exec(ctx,
		`INSERT INTO provider_verification_requests (id, user_id, provider_type, status, review_note, created_at)
		 VALUES ($1, $2, $3, 'pending', $4, $5)`,
		id, uid, providerType, note, time.Now().UTC())
	if err != nil {
		return "", "", fmt.Errorf("marketplace: create verification request: %w", err)
	}

	return id.String(), "pending", nil
}

// ProcessVerification allows an admin to approve or reject a provider verification request.
func (s *Service) ProcessVerification(ctx context.Context, adminID, requestID string, approve bool, note string) error {
	aid, err := uuid.Parse(adminID)
	if err != nil {
		return fmt.Errorf("marketplace: invalid admin_id: %w", err)
	}
	rid, err := uuid.Parse(requestID)
	if err != nil {
		return fmt.Errorf("marketplace: invalid request_id: %w", err)
	}

	tx, err := s.pg.Begin(ctx)
	if err != nil {
		return fmt.Errorf("marketplace: begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var uid uuid.UUID
	var providerType, status string
	err = tx.QueryRow(ctx,
		`SELECT user_id, provider_type, status FROM provider_verification_requests WHERE id = $1 FOR UPDATE`,
		rid,
	).Scan(&uid, &providerType, &status)
	if err != nil {
		return fmt.Errorf("marketplace: verification request not found: %w", err)
	}
	if status != "pending" {
		return fmt.Errorf("marketplace: verification request already processed")
	}

	newStatus := "rejected"
	if approve {
		newStatus = "approved"
	}

	_, err = tx.Exec(ctx,
		`UPDATE provider_verification_requests
		 SET status = $1, reviewed_by = $2, review_note = $3, reviewed_at = $4
		 WHERE id = $5`,
		newStatus, aid, note, time.Now().UTC(), rid,
	)
	if err != nil {
		return fmt.Errorf("marketplace: update verification request: %w", err)
	}

	if approve {
		_, err = tx.Exec(ctx,
			`UPDATE users SET verified_provider = true, provider_type = $1 WHERE id = $2`,
			providerType, uid,
		)
		if err != nil {
			return fmt.Errorf("marketplace: update user verification: %w", err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("marketplace: commit process verification: %w", err)
	}
	s.pubCache.clear()
	return nil
}

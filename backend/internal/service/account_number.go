package service

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"math/big"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

// AccountNumberService generates and validates 5/6-digit account numbers.
// Rules: 5 or 6 digits, no leading zero, no 4 or 7.
// Valid chars: first {1,2,3,5,6,8,9} (7), rest {0,1,2,3,5,6,8,9} (8).
// Capacity: 7 × 8⁵ = 229,376 (6-digit default for new users).
// Legacy 5-digit numbers (28,672 pool) remain valid.
type AccountNumberService struct {
	pg *pgxpool.Pool
}

// ErrAccountNumberExhausted is returned when all valid account numbers are taken.
var ErrAccountNumberExhausted = fmt.Errorf("account number space exhausted: all 229,376 valid numbers taken")

// NumberLength is the default length for newly generated numbers.
const NumberLength = 6

// validFirst is the set of allowed first digits (no 0, 4, 7).
var validFirst = []byte{'1', '2', '3', '5', '6', '8', '9'}

// validRest is the set of allowed digits for positions 2-6 (no 4, 7).
var validRest = []byte{'0', '1', '2', '3', '5', '6', '8', '9'}

// totalCapacity is 7 × 8⁵ = 229,376
const totalCapacity = 7 * 8 * 8 * 8 * 8 * 8

func NewAccountNumberService(pg *pgxpool.Pool) *AccountNumberService {
	return &AccountNumberService{pg: pg}
}

// GenerateAccountNumber finds an available account number using crypto/rand.
// Retries up to 100 times with random candidates, then falls back to a
// batch-optimized sequential scan of the entire number space.
func (s *AccountNumberService) GenerateAccountNumber(ctx context.Context) (string, error) {
	// Phase 1: random generation (100 attempts)
	for i := 0; i < 100; i++ {
		num, err := randomAccountNumber()
		if err != nil {
			continue
		}
		avail, err := s.IsAccountNumberAvailable(ctx, num)
		if err != nil {
			return "", fmt.Errorf("account number: availability check: %w", err)
		}
		if avail {
			return num, nil
		}
	}

	// Phase 2: batch-optimized sequential scan of the entire space.
	return s.scanAll(ctx)
}

// ValidateAccountNumber checks the account number against all format rules.
// Accepts 5-digit (legacy) or 6-digit (current) numbers.
func ValidateAccountNumber(num string) error {
	if len(num) != 5 && len(num) != 6 {
		return fmt.Errorf("account number must be 5 or 6 digits, got %d", len(num))
	}
	if num[0] == '0' {
		return fmt.Errorf("account number cannot start with 0")
	}
	for i := 0; i < len(num); i++ {
		c := num[i]
		if c < '0' || c > '9' {
			return fmt.Errorf("account number contains non-digit character '%c'", c)
		}
		if c == '4' || c == '7' {
			return fmt.Errorf("account number cannot contain digit '%c'", c)
		}
	}
	return nil
}

// IsAccountNumberAvailable checks if a specific number is not yet taken.
func (s *AccountNumberService) IsAccountNumberAvailable(ctx context.Context, num string) (bool, error) {
	return s.IsAccountNumberAvailableExcluding(ctx, num, "")
}

// IsAccountNumberAvailableExcluding checks availability, ignoring the given user ID.
// Pass empty userID to check against all users.
func (s *AccountNumberService) IsAccountNumberAvailableExcluding(ctx context.Context, num, excludeUserID string) (bool, error) {
	var exists bool
	var err error
	if excludeUserID == "" {
		err = s.pg.QueryRow(ctx,
			"SELECT EXISTS(SELECT 1 FROM users WHERE account_number = $1)", num,
		).Scan(&exists)
	} else {
		err = s.pg.QueryRow(ctx,
			"SELECT EXISTS(SELECT 1 FROM users WHERE account_number = $1 AND id != $2)", num, excludeUserID,
		).Scan(&exists)
	}
	if err != nil {
		return false, err
	}
	return !exists, nil
}

// IsUniqueViolation returns true if the error is a PostgreSQL unique-constraint
// violation (SQLSTATE 23505). Use this instead of string-matching error messages.
func IsUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}

// IsAccountNumberViolation returns true if the error is a UNIQUE violation
// specifically on the account_number column. Use this to distinguish between
// account_number collisions (retry) and email collisions (reject).
func IsAccountNumberViolation(err error) bool {
	var pgErr *pgconn.PgError
	if !errors.As(err, &pgErr) || pgErr.Code != "23505" {
		return false
	}
	return pgErr.ConstraintName == "idx_users_account_number"
}

// randomAccountNumber generates a random valid 6-digit account number using crypto/rand.
func randomAccountNumber() (string, error) {
	buf := make([]byte, NumberLength)

	// First digit
	n, err := rand.Int(rand.Reader, big.NewInt(int64(len(validFirst))))
	if err != nil {
		return "", err
	}
	buf[0] = validFirst[n.Int64()]

	// Remaining digits
	for i := 1; i < NumberLength; i++ {
		n, err := rand.Int(rand.Reader, big.NewInt(int64(len(validRest))))
		if err != nil {
			return "", err
		}
		buf[i] = validRest[n.Int64()]
	}
	return string(buf), nil
}

// batchSize is the number of candidates to check per DB round-trip in scanAll.
const batchSize = 2000

// scanAll iterates the entire valid number space in batches, using a single
// NOT EXISTS query per batch instead of 229k individual queries.
func (s *AccountNumberService) scanAll(ctx context.Context) (string, error) {
	batch := make([]string, 0, batchSize)

	// 6 nested loops over valid digit sets → 229,376 total candidates.
	for _, a := range validFirst {
		for _, b := range validRest {
			for _, c := range validRest {
				for _, d := range validRest {
					for _, e := range validRest {
						for _, f := range validRest {
							num := string([]byte{a, b, c, d, e, f})
							batch = append(batch, num)

							if len(batch) >= batchSize {
								if found := s.findFirstAvailable(ctx, batch); found != "" {
									return found, nil
								}
								batch = batch[:0]
							}
						}
					}
				}
			}
		}
	}

	// Final partial batch.
	if len(batch) > 0 {
		if found := s.findFirstAvailable(ctx, batch); found != "" {
			return found, nil
		}
	}

	return "", ErrAccountNumberExhausted
}

// findFirstAvailable returns the first candidate that is not in the users table.
func (s *AccountNumberService) findFirstAvailable(ctx context.Context, candidates []string) string {
	rows, err := s.pg.Query(ctx,
		`SELECT c.num FROM unnest($1::text[]) AS c(num)
		 WHERE NOT EXISTS (SELECT 1 FROM users WHERE account_number = c.num)
		 LIMIT 1`,
		candidates,
	)
	if err != nil {
		return ""
	}
	defer rows.Close()
	if rows.Next() {
		var num string
		if err := rows.Scan(&num); err != nil {
			return ""
		}
		return num
	}
	return ""
}

// AssignAccountNumber validates and checks availability of an admin-chosen number.
func (s *AccountNumberService) AssignAccountNumber(ctx context.Context, num string) error {
	if err := ValidateAccountNumber(num); err != nil {
		return err
	}
	avail, err := s.IsAccountNumberAvailable(ctx, num)
	if err != nil {
		return err
	}
	if !avail {
		return fmt.Errorf("account number %s is already taken", num)
	}
	return nil
}

// SetAccountNumber updates the user's account_number column.
func (s *AccountNumberService) SetAccountNumber(ctx context.Context, userID, num string) error {
	if err := ValidateAccountNumber(num); err != nil {
		return err
	}
	_, err := s.pg.Exec(ctx,
		"UPDATE users SET account_number = $1, updated_at = CURRENT_TIMESTAMP WHERE id = $2",
		num, userID)
	if err != nil {
		return fmt.Errorf("set account number: %w", err)
	}
	return nil
}

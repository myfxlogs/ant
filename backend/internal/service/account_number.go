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

// AccountNumberService generates and validates 5-digit account numbers.
// Rules: exactly 5 digits, no leading zero, no 4 or 7.
// Valid chars: first {1,2,3,5,6,8,9} (7), rest {0,1,2,3,5,6,8,9} (8).
// Total possible: 7 × 8⁴ = 28,672
type AccountNumberService struct {
	pg *pgxpool.Pool
}

// ErrAccountNumberExhausted is returned when all valid account numbers are taken.
var ErrAccountNumberExhausted = fmt.Errorf("account number exhausted: all 28,672 valid numbers are taken")

// validFirst is the set of allowed first digits (no 0, 4, 7).
var validFirst = []byte{'1', '2', '3', '5', '6', '8', '9'}

// validRest is the set of allowed digits for positions 2-5 (no 4, 7).
var validRest = []byte{'0', '1', '2', '3', '5', '6', '8', '9'}

func NewAccountNumberService(pg *pgxpool.Pool) *AccountNumberService {
	return &AccountNumberService{pg: pg}
}

// GenerateAccountNumber finds an available 5-digit account number.
// Uses random generation with retries, falling back to sequential scan if needed.
func (s *AccountNumberService) GenerateAccountNumber(ctx context.Context) (string, error) {
	// Phase 1: random generation with up to 100 retries
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

	// Phase 2: sequential scan of the entire space
	return s.scanAll(ctx)
}

// ValidateAccountNumber checks the account number against all format rules.
func ValidateAccountNumber(num string) error {
	if len(num) != 5 {
		return fmt.Errorf("account number must be exactly 5 digits, got %d", len(num))
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

// randomAccountNumber generates a random valid 5-digit account number.
func randomAccountNumber() (string, error) {
	buf := make([]byte, 5)

	// First digit
	n, err := rand.Int(rand.Reader, big.NewInt(int64(len(validFirst))))
	if err != nil {
		return "", err
	}
	buf[0] = validFirst[n.Int64()]

	// Remaining digits
	for i := 1; i < 5; i++ {
		n, err := rand.Int(rand.Reader, big.NewInt(int64(len(validRest))))
		if err != nil {
			return "", err
		}
		buf[i] = validRest[n.Int64()]
	}
	return string(buf), nil
}

// scanAll iterates through every valid account number in order to find the first available.
func (s *AccountNumberService) scanAll(ctx context.Context) (string, error) {
	for _, a := range validFirst {
		for _, b := range validRest {
			for _, c := range validRest {
				for _, d := range validRest {
					for _, e := range validRest {
						num := string([]byte{a, b, c, d, e})
						avail, err := s.IsAccountNumberAvailable(ctx, num)
						if err != nil {
							return "", err
						}
						if avail {
							return num, nil
						}
					}
				}
			}
		}
	}
	return "", ErrAccountNumberExhausted
}

// AssignAccountNumber assigns a specific (admin-chosen) account number to a user.
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

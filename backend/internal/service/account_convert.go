package service

import (
	"fmt"
	"math"
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"

	"anttrader/internal/repository"
)

// --- type conversion helpers (sqlc pgtype <-> Go native) ---

func uuidToPgUUID(id uuid.UUID) pgtype.UUID {
	return pgtype.UUID{Bytes: id, Valid: true}
}

func stringToPgUUID(s string) (pgtype.UUID, error) {
	uid, err := uuid.Parse(s)
	if err != nil {
		return pgtype.UUID{}, err
	}
	return pgtype.UUID{Bytes: uid, Valid: true}, nil
}

func pgUUIDToString(u pgtype.UUID) string {
	if !u.Valid {
		return ""
	}
	return uuid.UUID(u.Bytes).String()
}

func float64ToPgNumeric(v float64) (pgtype.Numeric, error) {
	if math.IsNaN(v) || math.IsInf(v, 0) {
		return pgtype.Numeric{}, fmt.Errorf("float64ToPgNumeric: cannot convert NaN or Inf")
	}
	var n pgtype.Numeric
	// Full-precision formatting preserves values that "%.8f" would truncate
	// (e.g. 0.000000001 → "0.00000000" → zero).
	if err := n.Scan(strconv.FormatFloat(v, 'f', -1, 64)); err != nil {
		return n, fmt.Errorf("float64ToPgNumeric: %w", err)
	}
	return n, nil
}

func pgNumericToFloat64(n pgtype.Numeric) (float64, bool) {
	if !n.Valid {
		return 0, false
	}
	f8, err := n.Float64Value()
	if err != nil || !f8.Valid {
		return 0, false
	}
	return f8.Float64, true
}

// pgNumericToFloat64Ignore calls pgNumericToFloat64 and discards the valid flag
// for callers that treat NULL as 0.
func pgNumericToFloat64Ignore(n pgtype.Numeric) float64 {
	v, _ := pgNumericToFloat64(n)
	return v
}


func pgInt4ToInt32(i pgtype.Int4) int32 {
	if !i.Valid {
		return 0
	}
	return i.Int32
}

func pgTextToString(t pgtype.Text) string {
	if !t.Valid {
		return ""
	}
	return t.String
}

func pgTimestampToString(t pgtype.Timestamp) string {
	if !t.Valid {
		return ""
	}
	return t.Time.UTC().Format(time.RFC3339)
}

func pgBoolToBool(b pgtype.Bool) bool {
	return b.Valid && b.Bool
}

// mtAccountToDTO maps a sqlc-generated MtAccount to the service DTO.
func mtAccountToDTO(a repository.MtAccount) AccountDTO {
	return AccountDTO{
		ID:              pgUUIDToString(a.ID),
		UserID:          pgUUIDToString(a.UserID),
		Platform:        a.MtType,
		Broker:          pgTextToString(a.BrokerCompany),
		Login:           a.Login,
		Server:          pgTextToString(a.BrokerServer),
		IsDisabled:      pgBoolToBool(a.IsDisabled),
		Status:          a.AccountStatus,
		Balance:         pgNumericToFloat64Ignore(a.Balance),
		Equity:          pgNumericToFloat64Ignore(a.Equity),
		Credit:          pgNumericToFloat64Ignore(a.Credit),
		Margin:          pgNumericToFloat64Ignore(a.Margin),
		FreeMargin:      pgNumericToFloat64Ignore(a.FreeMargin),
		MarginLevel:     pgNumericToFloat64Ignore(a.MarginLevel),
		Leverage:        pgInt4ToInt32(a.Leverage),
		Currency:        pgTextToString(a.Currency),
		LastError:       pgTextToString(a.LastError),
		LastConnectedAt: pgTimestampToString(a.LastConnectedAt),
		IsInvestor:      pgBoolToBool(a.IsInvestor),
	}
}

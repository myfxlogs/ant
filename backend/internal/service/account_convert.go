package service

import (
	"fmt"
	"math"
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
	if err := n.Scan(fmt.Sprintf("%.8f", v)); err != nil {
		return n, fmt.Errorf("float64ToPgNumeric: %w", err)
	}
	return n, nil
}

func pgNumericToFloat64(n pgtype.Numeric) float64 {
	if !n.Valid {
		return 0
	}
	f8, err := n.Float64Value()
	if err != nil || !f8.Valid {
		return 0
	}
	return f8.Float64
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
		Balance:         pgNumericToFloat64(a.Balance),
		Equity:          pgNumericToFloat64(a.Equity),
		Credit:          pgNumericToFloat64(a.Credit),
		Margin:          pgNumericToFloat64(a.Margin),
		FreeMargin:      pgNumericToFloat64(a.FreeMargin),
		MarginLevel:     pgNumericToFloat64(a.MarginLevel),
		Leverage:        pgInt4ToInt32(a.Leverage),
		Currency:        pgTextToString(a.Currency),
		LastError:       pgTextToString(a.LastError),
		LastConnectedAt: pgTimestampToString(a.LastConnectedAt),
		IsInvestor:      pgBoolToBool(a.IsInvestor),
	}
}

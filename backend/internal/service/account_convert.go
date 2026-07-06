package service

import (
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/shopspring/decimal"

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

func decimalToPgNumeric(d decimal.Decimal) (pgtype.Numeric, error) {
	var n pgtype.Numeric
	if err := n.Scan(d.String()); err != nil {
		return n, fmt.Errorf("decimalToPgNumeric: %w", err)
	}
	return n, nil
}

func pgNumericToDecimal(n pgtype.Numeric) decimal.Decimal {
	if !n.Valid {
		return decimal.Zero
	}
	v, err := n.Value()
	if err != nil {
		return decimal.Zero
	}
	s, ok := v.(string)
	if !ok {
		return decimal.Zero
	}
	d, err := decimal.NewFromString(s)
	if err != nil {
		return decimal.Zero
	}
	return d
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
		BrokerHost:      a.BrokerHost,
		Status:          a.AccountStatus,
		Balance:         pgNumericToDecimal(a.Balance),
		Equity:          pgNumericToDecimal(a.Equity),
		Credit:          pgNumericToDecimal(a.Credit),
		Margin:          pgNumericToDecimal(a.Margin),
		FreeMargin:      pgNumericToDecimal(a.FreeMargin),
		MarginLevel:     pgNumericToDecimal(a.MarginLevel),
		Leverage:        pgInt4ToInt32(a.Leverage),
		Currency:        pgTextToString(a.Currency),
		LastError:       pgTextToString(a.LastError),
		LastConnectedAt: pgTimestampToString(a.LastConnectedAt),
		IsInvestor:      pgBoolToBool(a.IsInvestor),
	}
}

package marketplace

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/shopspring/decimal"
)

const couponSelectCols = `id::text, code, discount_type, discount_value,
	        min_purchase_amount, max_uses, used_count, expires_at, enabled`

const couponApplicableClause = ` AND (COALESCE(cardinality(applicable_strategy_ids), 0) = 0 OR $2::uuid = ANY(applicable_strategy_ids))`

// scanCouponRow scans a coupon row from a pgx.Row/Rows scanner.
func scanCouponRow(row interface{ Scan(...any) error }) (CouponRow, error) {
	var r CouponRow
	var expiresAt *time.Time
	if err := row.Scan(&r.ID, &r.Code, &r.DiscountType, &r.DiscountValue,
		&r.MinPurchase, &r.MaxUses, &r.UsedCount, &expiresAt, &r.Enabled); err != nil {
		return r, err
	}
	r.ExpiresAt = expiresAt
	return r, nil
}

// computeCouponDiscount applies the coupon discount logic to the given amount.
// Returns the discount amount and final amount. Pure function, no I/O.
func computeCouponDiscount(row CouponRow, amount decimal.Decimal) (*CouponResult, error) {
	if !row.Enabled {
		return &CouponResult{ErrorMessage: "coupon is disabled"}, nil
	}
	if row.ExpiresAt != nil && time.Now().After(*row.ExpiresAt) {
		return &CouponResult{ErrorMessage: "coupon has expired"}, nil
	}
	if row.MaxUses > 0 && row.UsedCount >= row.MaxUses {
		return &CouponResult{ErrorMessage: "coupon usage limit reached"}, nil
	}
	if amount.LessThan(row.MinPurchase) {
		return &CouponResult{ErrorMessage: fmt.Sprintf("minimum purchase amount is %s", row.MinPurchase.String())}, nil
	}

	var discount decimal.Decimal
	switch row.DiscountType {
	case "percentage":
		if row.DiscountValue.GreaterThan(decimal.NewFromInt(100)) || row.DiscountValue.LessThanOrEqual(decimal.Zero) {
			return &CouponResult{ErrorMessage: "invalid percentage discount value (must be 0-100)"}, nil
		}
		discount = amount.Mul(row.DiscountValue).Div(decimal.NewFromInt(100))
	case "fixed":
		discount = row.DiscountValue
		if discount.GreaterThan(amount) {
			discount = amount
		}
	default:
		return &CouponResult{ErrorMessage: "invalid discount type"}, nil
	}

	finalAmount := amount.Sub(discount)
	if finalAmount.LessThan(decimal.Zero) {
		finalAmount = decimal.Zero
	}

	return &CouponResult{
		ID:             row.ID,
		Valid:          true,
		DiscountType:   row.DiscountType,
		DiscountAmount: discount,
		FinalAmount:    finalAmount,
	}, nil
}

// CouponRow represents a row in marketplace_coupons.
type CouponRow struct {
	ID                  string
	Code                string
	DiscountType        string
	DiscountValue       decimal.Decimal
	MinPurchase         decimal.Decimal
	MaxUses             int32
	UsedCount           int32
	ExpiresAt           *time.Time
	Enabled             bool
	ApplicableStrategyIDs []string
}

// CouponResult holds the validation result for a coupon.
type CouponResult struct {
	ID             string
	Valid          bool
	DiscountType   string
	DiscountAmount decimal.Decimal
	FinalAmount    decimal.Decimal
	ErrorMessage   string
}

// ValidateCoupon validates a coupon code and computes the discounted amount.
func (s *Service) ValidateCoupon(ctx context.Context, code, strategyID, amountStr string) (*CouponResult, error) {
	amount, err := decimal.NewFromString(amountStr)
	if err != nil {
		return nil, fmt.Errorf("marketplace: invalid amount: %w", err)
	}
	sid, err := uuid.Parse(strategyID)
	if err != nil {
		return &CouponResult{ErrorMessage: "invalid strategy_id"}, nil
	}

	row, err := scanCouponRow(s.pg.QueryRow(ctx,
		`SELECT `+couponSelectCols+`
		 FROM marketplace_coupons
		 WHERE code = $1`+couponApplicableClause,
		code, sid,
	))
	if err != nil {
		return &CouponResult{ErrorMessage: "coupon not found or not applicable"}, nil
	}
	return computeCouponDiscount(row, amount)
}

// validateCouponTx validates a coupon within a database transaction with FOR UPDATE lock,
// preventing TOCTOU races between validation and consumption.
func (s *Service) validateCouponTx(ctx context.Context, tx pgx.Tx, code string, sid uuid.UUID, amount decimal.Decimal) (*CouponResult, error) {
	row, err := scanCouponRow(tx.QueryRow(ctx,
		`SELECT `+couponSelectCols+`
		 FROM marketplace_coupons
		 WHERE code = $1`+couponApplicableClause+`
		 FOR UPDATE`,
		code, sid,
	))
	if err != nil {
		return &CouponResult{ErrorMessage: "coupon not found or not applicable"}, nil
	}
	return computeCouponDiscount(row, amount)
}

// CreateCoupon creates a new coupon (admin only).
func (s *Service) CreateCoupon(ctx context.Context, adminID, code, discountType, discountValue, minPurchase string, maxUses int32, expiresAtStr string, applicableStrategyIDs []string) (string, error) {
	dv, err := decimal.NewFromString(discountValue)
	if err != nil {
		return "", fmt.Errorf("marketplace: invalid discount_value: %w", err)
	}
	mp, err := decimal.NewFromString(minPurchase)
	if err != nil {
		return "", fmt.Errorf("marketplace: invalid min_purchase_amount: %w", err)
	}

	var expiresAt interface{}
	if expiresAtStr != "" {
		t, err := time.Parse(time.RFC3339, expiresAtStr)
		if err != nil {
			return "", fmt.Errorf("marketplace: invalid expires_at: %w", err)
		}
		expiresAt = t
	}

	if len(applicableStrategyIDs) > 0 {
		for _, sid := range applicableStrategyIDs {
			if _, err := uuid.Parse(sid); err != nil {
				return "", fmt.Errorf("marketplace: invalid applicable_strategy_id %q: %w", sid, err)
			}
		}
	}

	aid, err := uuid.Parse(adminID)
	if err != nil {
		return "", fmt.Errorf("marketplace: invalid admin_id: %w", err)
	}

	appStr := strings.Join(applicableStrategyIDs, ",")

	var id string
	err = s.pg.QueryRow(ctx,
		`INSERT INTO marketplace_coupons (code, discount_type, discount_value, min_purchase_amount, max_uses, expires_at, applicable_strategy_ids, created_by)
		 VALUES ($1, $2, $3, $4, $5, $6, string_to_array($7, ',')::uuid[], $8)
		 RETURNING id::text`,
		code, discountType, dv, mp, maxUses, expiresAt, appStr, aid,
	).Scan(&id)
	if err != nil {
		return "", fmt.Errorf("marketplace: create coupon: %w", err)
	}

	return id, nil
}

// ListCoupons lists all coupons (admin only).
func (s *Service) ListCoupons(ctx context.Context, enabledOnly bool) ([]CouponRow, error) {
	query := `SELECT id::text, code, discount_type, discount_value,
	          min_purchase_amount, max_uses, used_count, expires_at, enabled,
	          array_to_string(applicable_strategy_ids, ',')
	          FROM marketplace_coupons`
	if enabledOnly {
		query += " WHERE enabled = true"
	}
	query += " ORDER BY created_at DESC"

	rows, err := s.pg.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("marketplace: list coupons: %w", err)
	}
	defer rows.Close()

	var result []CouponRow
	for rows.Next() {
		var r CouponRow
		var app string
		if err := rows.Scan(&r.ID, &r.Code, &r.DiscountType, &r.DiscountValue,
			&r.MinPurchase, &r.MaxUses, &r.UsedCount, &r.ExpiresAt, &r.Enabled, &app); err != nil {
			return nil, fmt.Errorf("marketplace: scan coupon: %w", err)
		}
		if app != "" {
			r.ApplicableStrategyIDs = strings.Split(app, ",")
		}
		result = append(result, r)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return result, nil
}

// DisableCoupon disables a coupon (admin only).
func (s *Service) DisableCoupon(ctx context.Context, couponID string) error {
	cid, err := uuid.Parse(couponID)
	if err != nil {
		return fmt.Errorf("marketplace: invalid coupon_id: %w", err)
	}

	_, err = s.pg.Exec(ctx,
		`UPDATE marketplace_coupons SET enabled = false WHERE id = $1`,
		cid)
	if err != nil {
		return fmt.Errorf("marketplace: disable coupon: %w", err)
	}
	return nil
}

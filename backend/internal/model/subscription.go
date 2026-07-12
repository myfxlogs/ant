package model

import (
	"time"

	"github.com/google/uuid"
)

// SubscriptionPlan represents a platform subscription plan (Free/Pro/Enterprise).
type SubscriptionPlan struct {
	ID                    uuid.UUID `db:"id"`
	Name                  string    `db:"name"`
	DisplayName           string    `db:"display_name"`
	PriceMonthly          string    `db:"price_monthly"`
	PriceYearly           string    `db:"price_yearly"`
	MaxAITokensMonthly    int       `db:"max_ai_tokens_monthly"`
	MaxStrategies         int       `db:"max_strategies"`
	MaxBacktestsDaily     int       `db:"max_backtests_daily"`
	MaxLiveStrategies     int       `db:"max_live_strategies"`
	MaxSymbolsPerStrategy int       `db:"max_symbols_per_strategy"`
	CapabilityTier        int       `db:"capability_tier"`
	Features              string    `db:"features"`
	SortOrder             int       `db:"sort_order"`
	IsActive              bool      `db:"is_active"`
	CreatedAt             time.Time `db:"created_at"`
	UpdatedAt             time.Time `db:"updated_at"`
}

// UserPlatformSubscription represents a user's active platform subscription.
type UserPlatformSubscription struct {
	ID                  uuid.UUID  `db:"id"`
	UserID              uuid.UUID  `db:"user_id"`
	PlanID              uuid.UUID  `db:"plan_id"`
	Status              string     `db:"status"`
	BillingCycle        string     `db:"billing_cycle"`
	CurrentPeriodStart  time.Time  `db:"current_period_start"`
	CurrentPeriodEnd    time.Time  `db:"current_period_end"`
	AutoRenew           bool       `db:"auto_renew"`
	CancelledAt         *time.Time `db:"cancelled_at"`
	WalletTransactionID *uuid.UUID `db:"wallet_transaction_id"`
	CreatedAt           time.Time  `db:"created_at"`
	UpdatedAt           time.Time  `db:"updated_at"`
}

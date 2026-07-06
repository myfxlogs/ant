package model

import (
	"github.com/shopspring/decimal"
	"time"

	"github.com/google/uuid"
)

type MTAccount struct {
	ID              uuid.UUID  `json:"id" db:"id"`
	UserID          uuid.UUID  `json:"user_id" db:"user_id"`
	MTType          string     `json:"mt_type" db:"mt_type"`
	BrokerCompany   string     `json:"broker_company" db:"broker_company"`
	BrokerServer    string     `json:"broker_server" db:"broker_server"`
	BrokerHost      string     `json:"broker_host" db:"broker_host"`
	Login           string     `json:"login" db:"login"`
	Password        string     `json:"-" db:"password"`
	Alias           string     `json:"alias" db:"alias"`
	Balance         decimal.Decimal    `json:"balance" db:"balance"`
	Credit          decimal.Decimal    `json:"credit" db:"credit"`
	Equity          decimal.Decimal    `json:"equity" db:"equity"`
	Margin          decimal.Decimal    `json:"margin" db:"margin"`
	FreeMargin      decimal.Decimal    `json:"free_margin" db:"free_margin"`
	MarginLevel     decimal.Decimal    `json:"margin_level" db:"margin_level"`
	Leverage        int        `json:"leverage" db:"leverage"`
	Currency        string     `json:"currency" db:"currency"`
	AccountMethod   string     `json:"account_method" db:"account_method"`
	AccountType     string     `json:"account_type" db:"account_type"`
	IsInvestor      bool       `json:"is_investor" db:"is_investor"`
	AccountStatus   string     `json:"account_status" db:"account_status"`
	MTToken              string     `json:"-" db:"mt_token"`
	BrokerMarginCallPct  decimal.Decimal    `json:"broker_margin_call_pct" db:"broker_margin_call_pct"`
	BrokerStopOutPct     decimal.Decimal    `json:"broker_stop_out_pct" db:"broker_stop_out_pct"`
	LastError            string     `json:"last_error" db:"last_error"`
	LastConnectedAt *time.Time `json:"last_connected_at" db:"last_connected_at"`
	LastCheckedAt   *time.Time `json:"last_checked_at" db:"last_checked_at"`
	CreatedAt       time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at" db:"updated_at"`
}

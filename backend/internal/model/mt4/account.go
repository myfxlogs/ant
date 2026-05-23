package mt4

import (
	"time"

	"github.com/google/uuid"
)

type AccountType int32

const (
	AccountTypeReal    AccountType = 0
	AccountTypeContest AccountType = 1
	AccountTypeDemo    AccountType = 2
)

type Account struct {
	ID              uuid.UUID   `json:"id"`
	UserID          uuid.UUID   `json:"user_id"`
	MTType          string      `json:"mt_type"`
	BrokerCompany   string      `json:"broker_company"`
	BrokerServer    string      `json:"broker_server"`
	BrokerHost      string      `json:"broker_host"`
	Login           string      `json:"login"`
	Password        string      `json:"-"`
	Alias           string      `json:"alias"`
	IsDisabled      bool        `json:"is_disabled"`
	Balance         float64     `json:"balance"`
	Credit          float64     `json:"credit"`
	Profit          float64     `json:"profit"`
	Equity          float64     `json:"equity"`
	Margin          float64     `json:"margin"`
	FreeMargin      float64     `json:"free_margin"`
	MarginLevel     float64     `json:"margin_level"`
	Leverage        int32       `json:"leverage"`
	Currency        string      `json:"currency"`
	Type            AccountType `json:"type"`
	IsInvestor      bool        `json:"is_investor"`
	AccountStatus   string      `json:"account_status"`
	StreamStatus    string      `json:"stream_status"`
	MTToken         string      `json:"-"`
	LastError       string      `json:"last_error"`
	LastConnectedAt *time.Time  `json:"last_connected_at"`
	LastCheckedAt   *time.Time  `json:"last_checked_at"`
	CreatedAt       time.Time   `json:"created_at"`
	UpdatedAt       time.Time   `json:"updated_at"`
}

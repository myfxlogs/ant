package model

import (
	"github.com/shopspring/decimal"
	"time"

	"github.com/google/uuid"
)

type AdminLog struct {
	ID            uuid.UUID              `json:"id" db:"id"`
	AdminID       *uuid.UUID             `json:"admin_id" db:"admin_id"`
	Module        string                 `json:"module" db:"module"`
	ActionType    string                 `json:"action_type" db:"action_type"`
	TargetType    string                 `json:"target_type" db:"target_type"`
	TargetID      string                 `json:"target_id" db:"target_id"`
	IPAddress     string                 `json:"ip_address" db:"ip_address"`
	UserAgent     string                 `json:"user_agent" db:"user_agent"`
	RequestMethod string                 `json:"request_method" db:"request_method"`
	RequestPath   string                 `json:"request_path" db:"request_path"`
	Details       map[string]interface{} `json:"details" db:"details"`
	Success       bool                   `json:"success" db:"success"`
	ErrorMessage  string                 `json:"error_message" db:"error_message"`
	CreatedAt     time.Time              `json:"created_at" db:"created_at"`
}

type Permission struct {
	ID          uuid.UUID `json:"id" db:"id"`
	Code        string    `json:"code" db:"code"`
	Name        string    `json:"name" db:"name"`
	Description string    `json:"description" db:"description"`
	CreatedAt   time.Time `json:"created_at" db:"created_at"`
}

type RolePermission struct {
	Role         string    `json:"role" db:"role"`
	PermissionID uuid.UUID `json:"permission_id" db:"permission_id"`
	GrantedAt    time.Time `json:"granted_at" db:"granted_at"`
}

type SystemConfig struct {
	Key         string    `json:"key" db:"key"`
	Value       string    `json:"value" db:"value"`
	Description string    `json:"description" db:"description"`
	Enabled     *bool     `json:"enabled" db:"enabled"`
	CreatedAt   time.Time `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time `json:"updated_at" db:"updated_at"`
}

type DashboardStats struct {
	TotalUsers     int64   `json:"total_users"`
	ActiveUsers    int64   `json:"active_users"`
	TotalAccounts  int64   `json:"total_accounts"`
	OnlineAccounts int64   `json:"online_accounts"`
	TodayTrades    int64   `json:"today_trades"`
	TodayVolume    decimal.Decimal `json:"today_volume"`
	TodayProfit    decimal.Decimal `json:"today_profit"`
	SystemLoad     decimal.Decimal `json:"system_load"`
}

type UserListParams struct {
	Page          int    `form:"page"`
	PageSize      int    `form:"page_size"`
	Search        string `form:"search"`
	Status        string `form:"status"`
	Role          string `form:"role"`
	DeletedFilter string `form:"deleted_filter"` // "" = active, "deleted", "all"
}

type AccountListParams struct {
	Page     int    `form:"page"`
	PageSize int    `form:"page_size"`
	Search   string `form:"search"`
	Status   string `form:"status"`
	MTType   string `form:"mt_type"`
	UserID   string `form:"user_id"`
}

type LogListParams struct {
	Page       int    `form:"page"`
	PageSize   int    `form:"page_size"`
	Module     string `form:"module"`
	ActionType string `form:"action_type"`
	StartDate  string `form:"start_date"`
	EndDate    string `form:"end_date"`
	AdminID    string `form:"admin_id"`
}

type TradingSummary struct {
	Period struct {
		StartDate string `json:"start_date"`
		EndDate   string `json:"end_date"`
	} `json:"period"`
	Overview struct {
		TotalUsers        int64 `json:"total_users"`
		ActiveUsers       int64 `json:"active_users"`
		TotalAccounts     int64 `json:"total_accounts"`
		ConnectedAccounts int64 `json:"connected_accounts"`
	} `json:"overview"`
	Trading struct {
		TotalOrders   int64   `json:"total_orders"`
		ClosedOrders  int64   `json:"closed_orders"`
		PendingOrders int64   `json:"pending_orders"`
		TotalVolume   decimal.Decimal `json:"total_volume"`
		TotalProfit   decimal.Decimal `json:"total_profit"`
		TotalLoss     decimal.Decimal `json:"total_loss"`
		NetProfit     decimal.Decimal `json:"net_profit"`
	} `json:"trading"`
	ByPlatform map[string]PlatformSummary `json:"by_platform"`
}

type PlatformSummary struct {
	Accounts int64   `json:"accounts"`
	Orders   int64   `json:"orders"`
	Volume   float64 `json:"volume"`
}

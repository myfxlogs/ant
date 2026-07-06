package repository

import (
	"github.com/shopspring/decimal"
	"context"
	"database/sql"
	"errors"
	"time"
	"github.com/jackc/pgx/v5/pgxpool"

	"anttrader/internal/model"
)

var (
	ErrUserNotFound     = errors.New("user not found")
	ErrConfigNotFound   = errors.New("config not found")
	ErrLogNotFound      = errors.New("log not found")
	ErrPermissionDenied = errors.New("permission denied")
)

type AdminRepository struct {
	db *pgxpool.Pool
}

func NewAdminRepository(db *pgxpool.Pool) *AdminRepository {
	return &AdminRepository{db: db}
}

func (r *AdminRepository) GetDashboardStats(ctx context.Context) (*model.DashboardStats, error) {
	stats := &model.DashboardStats{SystemLoad: decimal.Zero}
	if err := r.fetchDashboardUsers(ctx, stats); err != nil { return nil, err }
	if err := r.fetchDashboardAccounts(ctx, stats); err != nil { return nil, err }
	if err := r.fetchDashboardToday(ctx, stats); err != nil { return nil, err }
	return stats, nil
}

func (r *AdminRepository) fetchDashboardUsers(ctx context.Context, s *model.DashboardStats) error {
	if err := r.db.QueryRow(ctx, `SELECT COUNT(*) FROM users`).Scan(&s.TotalUsers); err != nil { return err }
	return r.db.QueryRow(ctx, `SELECT COUNT(*) FROM users WHERE status='active'`).Scan(&s.ActiveUsers)
}

func (r *AdminRepository) fetchDashboardAccounts(ctx context.Context, s *model.DashboardStats) error {
	if err := r.db.QueryRow(ctx, `SELECT COUNT(*) FROM mt_accounts`).Scan(&s.TotalAccounts); err != nil { return err }
	return r.db.QueryRow(ctx, `SELECT COUNT(*) FROM mt_accounts WHERE account_status='connected'`).Scan(&s.OnlineAccounts)
}

func (r *AdminRepository) fetchDashboardToday(ctx context.Context, s *model.DashboardStats) error {
	today := time.Now().Format("2006-01-02")
	qInt := func(query string, dest *int64) error {
		err := r.db.QueryRow(ctx, query, today).Scan(dest)
		if errors.Is(err, sql.ErrNoRows) { return nil }
		return err
	}
	qDec := func(query string, dest *decimal.Decimal) error {
		var v float64
		err := r.db.QueryRow(ctx, query, today).Scan(&v)
		if errors.Is(err, sql.ErrNoRows) { return nil }
		if err != nil { return err }
		*dest = decimal.NewFromFloat(v)
		return nil
	}
	if err := qInt(`SELECT COUNT(*) FROM trade_records WHERE DATE(close_time)=$1`, &s.TodayTrades); err != nil { return err }
	if err := qDec(`SELECT COALESCE(SUM(volume),0) FROM trade_records WHERE DATE(close_time)=$1`, &s.TodayVolume); err != nil { return err }
	if err := qDec(`SELECT COALESCE(SUM(profit),0) FROM trade_records WHERE DATE(close_time)=$1`, &s.TodayProfit); err != nil { return err }
	return nil
}

func normalizePage(page, pageSize int) (int, int) {
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 20
	}
	if pageSize > 100 {
		pageSize = 100
	}
	return page, pageSize
}

func (r *AdminRepository) GetTradingSummary(ctx context.Context, startDate, endDate string) (*model.TradingSummary, error) {
	s := &model.TradingSummary{}
	s.Period.StartDate, s.Period.EndDate = startDate, endDate
	fetchTradingSummaryOverview(ctx, r, s)
	fetchTradingSummaryTrading(ctx, r, startDate, endDate, s)
	s.Trading.NetProfit = s.Trading.TotalProfit.Add(s.Trading.TotalLoss)
	fetchTradingSummaryByPlatform(ctx, r, startDate, endDate, s)
	return s, nil
}

func fetchTradingSummaryOverview(ctx context.Context, r *AdminRepository, s *model.TradingSummary) {
	_ = r.db.QueryRow(ctx, `SELECT COUNT(*) FROM users`).Scan(&s.Overview.TotalUsers)
	_ = r.db.QueryRow(ctx, `SELECT COUNT(*) FROM users WHERE status='active'`).Scan(&s.Overview.ActiveUsers)
	_ = r.db.QueryRow(ctx, `SELECT COUNT(*) FROM mt_accounts`).Scan(&s.Overview.TotalAccounts)
	_ = r.db.QueryRow(ctx, `SELECT COUNT(*) FROM mt_accounts WHERE account_status='connected'`).Scan(&s.Overview.ConnectedAccounts)
}

func fetchTradingSummaryTrading(ctx context.Context, r *AdminRepository, start, end string, s *model.TradingSummary) {
	_ = r.db.QueryRow(ctx, `SELECT COUNT(*), COALESCE(SUM(volume),0), COALESCE(SUM(profit),0) FROM trade_records WHERE DATE(close_time) BETWEEN $1 AND $2`, start, end).
		Scan(&s.Trading.ClosedOrders, &s.Trading.TotalVolume, &s.Trading.TotalProfit)
	_ = r.db.QueryRow(ctx, `SELECT COALESCE(SUM(CASE WHEN profit<0 THEN profit ELSE 0 END),0) FROM trade_records WHERE DATE(close_time) BETWEEN $1 AND $2`, start, end).
		Scan(&s.Trading.TotalLoss)
	_ = r.db.QueryRow(ctx, `SELECT COUNT(*) FROM trade_records WHERE close_time IS NULL`).Scan(&s.Trading.PendingOrders)
	_ = r.db.QueryRow(ctx, `SELECT COUNT(*) FROM trade_records WHERE DATE(close_time) BETWEEN $1 AND $2`, start, end).
		Scan(&s.Trading.TotalOrders)
}

func fetchTradingSummaryByPlatform(ctx context.Context, r *AdminRepository, start, end string, s *model.TradingSummary) {
	rows, err := r.db.Query(ctx, `SELECT COALESCE(ma.mt_type,'unknown'), COUNT(DISTINCT ma.id), COUNT(tr.id), COALESCE(SUM(tr.volume),0) FROM mt_accounts ma LEFT JOIN trade_records tr ON tr.account_id=ma.id AND DATE(tr.close_time) BETWEEN $1 AND $2 GROUP BY ma.mt_type`, start, end)
	if err != nil { return }
	defer rows.Close()
	s.ByPlatform = make(map[string]model.PlatformSummary)
	for rows.Next() {
		var pf string; var ps model.PlatformSummary
		if err := rows.Scan(&pf, &ps.Accounts, &ps.Orders, &ps.Volume); err == nil { s.ByPlatform[pf] = ps }
	}
}

func (r *AdminRepository) Ping(ctx context.Context) error {
	return r.db.Ping(ctx)
}

// HasPermission checks if a role has a specific permission.
// Permission matrix (expandable):
//   super_admin → all permissions
//   operation   → strategy:review, strategy:publish, user:read, order:read
//   customer_service → user:read, order:read
//   audit       → read-only (user:read, order:read, strategy:read, log:read)
//   user        → no admin permissions
func (r *AdminRepository) HasPermission(ctx context.Context, role, permissionCode string) (bool, error) {
	switch role {
	case "super_admin":
		return true, nil
	case "operation":
		allowed := map[string]bool{
			"strategy:review": true, "strategy:publish": true,
			"user:read": true, "order:read": true,
		}
		return allowed[permissionCode], nil
	case "customer_service":
		allowed := map[string]bool{"user:read": true, "order:read": true}
		return allowed[permissionCode], nil
	case "audit":
		return isReadPermission(permissionCode), nil
	default:
		return false, nil
	}
}

func isReadPermission(code string) bool {
	switch code {
	case "user:read", "order:read", "strategy:read", "log:read":
		return true
	}
	return false
}

// AuditLogEntry represents a single account audit event.
type AuditLogEntry struct {
	ID        string    `json:"id"`
	AccountID string    `json:"account_id"`
	UserID    string    `json:"user_id"`
	Action    string    `json:"action"`
	Detail    string    `json:"detail"`
	CreatedAt time.Time `json:"created_at"`
}

// GetAuditLogs returns audit entries for the given account, newest first.
func (r *AdminRepository) GetAuditLogs(ctx context.Context, accountID string, limit int) ([]*AuditLogEntry, error) {
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	rows, err := r.db.Query(ctx,
		`SELECT id::text, account_id::text, user_id::text, action, detail, created_at
		 FROM account_audit_log WHERE account_id = $1
		 ORDER BY created_at DESC LIMIT $2`, accountID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var entries []*AuditLogEntry
	for rows.Next() {
		e := &AuditLogEntry{}
		if err := rows.Scan(&e.ID, &e.AccountID, &e.UserID, &e.Action, &e.Detail, &e.CreatedAt); err != nil {
			return nil, err
		}
		entries = append(entries, e)
	}
	return entries, rows.Err()
}

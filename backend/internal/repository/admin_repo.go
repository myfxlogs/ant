package repository

import (
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
	stats := &model.DashboardStats{}

	err := r.db.QueryRow(ctx, `SELECT COUNT(*) FROM users`).Scan(&stats.TotalUsers)
	if err != nil {
		return nil, err
	}

	err = r.db.QueryRow(ctx, `SELECT COUNT(*) FROM users WHERE status = 'active'`).Scan(&stats.ActiveUsers)
	if err != nil {
		return nil, err
	}

	err = r.db.QueryRow(ctx, `SELECT COUNT(*) FROM mt_accounts`).Scan(&stats.TotalAccounts)
	if err != nil {
		return nil, err
	}

	err = r.db.QueryRow(ctx, `SELECT COUNT(*) FROM mt_accounts WHERE account_status = 'connected'`).Scan(&stats.OnlineAccounts)
	if err != nil {
		return nil, err
	}

	today := time.Now().Format("2006-01-02")
	err = r.db.QueryRow(ctx, `
		SELECT COUNT(*) FROM trade_records 
		WHERE DATE(close_time) = $1
	`, today).Scan(&stats.TodayTrades)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return nil, err
	}

	err = r.db.QueryRow(ctx, `
		SELECT COALESCE(SUM(volume), 0) FROM trade_records 
		WHERE DATE(close_time) = $1
	`, today).Scan(&stats.TodayVolume)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return nil, err
	}

	err = r.db.QueryRow(ctx, `
		SELECT COALESCE(SUM(profit), 0) FROM trade_records 
		WHERE DATE(close_time) = $1
	`, today).Scan(&stats.TodayProfit)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return nil, err
	}

	stats.SystemLoad = 0.0

	return stats, nil
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
	summary := &model.TradingSummary{}
	summary.Period.StartDate = startDate
	summary.Period.EndDate = endDate

	_ = r.db.QueryRow(ctx, `SELECT COUNT(*) FROM users`).Scan(&summary.Overview.TotalUsers)
	_ = r.db.QueryRow(ctx, `SELECT COUNT(*) FROM users WHERE status = 'active'`).Scan(&summary.Overview.ActiveUsers)
	_ = r.db.QueryRow(ctx, `SELECT COUNT(*) FROM mt_accounts`).Scan(&summary.Overview.TotalAccounts)
	_ = r.db.QueryRow(ctx, `SELECT COUNT(*) FROM mt_accounts WHERE account_status = 'connected'`).Scan(&summary.Overview.ConnectedAccounts)

	_ = r.db.QueryRow(ctx, `
		SELECT COUNT(*), COALESCE(SUM(volume), 0), COALESCE(SUM(profit), 0)
		FROM trade_records
		WHERE DATE(close_time) BETWEEN $1 AND $2`, startDate, endDate,
	).Scan(&summary.Trading.ClosedOrders, &summary.Trading.TotalVolume, &summary.Trading.TotalProfit)

	_ = r.db.QueryRow(ctx, `
		SELECT COALESCE(SUM(CASE WHEN profit < 0 THEN profit ELSE 0 END), 0)
		FROM trade_records
		WHERE DATE(close_time) BETWEEN $1 AND $2`, startDate, endDate,
	).Scan(&summary.Trading.TotalLoss)

	_ = r.db.QueryRow(ctx, `
		SELECT COUNT(*) FROM trade_records
		WHERE close_time IS NULL`, // pending orders
	).Scan(&summary.Trading.PendingOrders)

	_ = r.db.QueryRow(ctx, `
		SELECT COUNT(*)
		FROM trade_records
		WHERE DATE(close_time) BETWEEN $1 AND $2`, startDate, endDate,
	).Scan(&summary.Trading.TotalOrders)

	summary.Trading.NetProfit = summary.Trading.TotalProfit + summary.Trading.TotalLoss

	// ByPlatform breakdown
	rows, err := r.db.Query(ctx, `
		SELECT COALESCE(ma.platform, 'unknown'), COUNT(DISTINCT ma.id), COUNT(tr.id), COALESCE(SUM(tr.volume), 0)
		FROM mt_accounts ma
		LEFT JOIN trade_records tr ON tr.account_id = ma.id
			AND DATE(tr.close_time) BETWEEN $1 AND $2
		GROUP BY ma.platform`, startDate, endDate)
	if err == nil {
		defer rows.Close()
		summary.ByPlatform = make(map[string]model.PlatformSummary)
		for rows.Next() {
			var platform string
			var plat model.PlatformSummary
			if err := rows.Scan(&platform, &plat.Accounts, &plat.Orders, &plat.Volume); err == nil {
				summary.ByPlatform[platform] = plat
			}
		}
	}

	return summary, nil
}

func (r *AdminRepository) Ping(ctx context.Context) error {
	return r.db.Ping(ctx)
}

func (r *AdminRepository) HasPermission(ctx context.Context, role, permissionCode string) (bool, error) {
	return role == "admin", nil
}
